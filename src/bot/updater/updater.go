package updater

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/builder"
	"discord-music-bot/datastore"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
	"golang.org/x/net/context"
)

type QueueUpdater struct {
	log                 *log.Logger
	mutex               sync.RWMutex
	interactionsBuffer  map[string]chan *discordgo.Interaction
	builder             *builder.Builder
	datastore           *datastore.Datastore
	audioplayers        *audioplayer.AudioPlayersMap
	maxAloneTime        time.Duration
	ready               func() bool
	queueRequests       map[string]chan struct{}
	queueRequestsMutex  sync.Mutex
	queueRunning        map[string]struct{}
	queueRunningMutex   sync.Mutex
	onFailureFuncs      map[string]chan func()
	onFailureFuncsMutex sync.Mutex
}

// NewQueueUpdater constructs a new object that
// handles updating queues
func NewQueueUpdater(log *log.Logger, builder *builder.Builder, maxAloneTime time.Duration, datastore *datastore.Datastore, audioplayers *audioplayer.AudioPlayersMap, ready func() bool) *QueueUpdater {
	return &QueueUpdater{
		mutex:               sync.RWMutex{},
		log:                 log,
		interactionsBuffer:  make(map[string]chan *discordgo.Interaction),
		builder:             builder,
		datastore:           datastore,
		audioplayers:        audioplayers,
		maxAloneTime:        maxAloneTime,
		ready:               ready,
		queueRunning:        make(map[string]struct{}),
		queueRequests:       make(map[string]chan struct{}),
		queueRunningMutex:   sync.Mutex{},
		queueRequestsMutex:  sync.Mutex{},
		onFailureFuncs:      make(map[string]chan func()),
		onFailureFuncsMutex: sync.Mutex{},
	}
}

// AddInteraction adds the provided interaction to the
// bot's queueUpdateInteractionsBuffer, if it is not already full.
// The provided interaction is deffered after some time, so the
// interaction failed error is not thrown in discord channel.
// This interaction may then be used when updating the queue, if it
// was not yet deffered.
func (updater *QueueUpdater) AddInteraction(s *discordgo.Session, i *discordgo.Interaction) {
	go func() {
		time.Sleep(discordgo.InteractionDeadline - (300 * time.Millisecond))
		s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
	}()
	updater.mutex.Lock()
	defer updater.mutex.Unlock()

	if _, ok := updater.interactionsBuffer[i.GuildID]; !ok {
		updater.interactionsBuffer[i.GuildID] = make(
			chan *discordgo.Interaction, 10,
		)
	}
	select {
	case updater.interactionsBuffer[i.GuildID] <- i:
	default:
	}
}

// Update fetches the queue from the datastore and updates it.
// It first tries to update it from the interactions stored in the
// bot's queueUpdateInteractionsBuffer, and if not successful from
// the queue's channelID and messageID.
// Queue will be updated after timeout, if timeout is negative,
// queue will just be marked as "needs update", but won't be updated.
// WARNING: If queue has already been updated during the timeout,
// it won't be updated again.
func (updater *QueueUpdater) Update(s *discordgo.Session, guildID string, timeout time.Duration, onFailure func()) {

	updater.log.WithFields(log.Fields{
		"GuildID": guildID,
		"Timeout": timeout,
	}).Trace("Queue update requested")

	// NOTE: check if a queue is already running
	// with it's requests buffer, if so, and it's buffer is empty
	// add a struct to it, otherwise do nothing.
	// (makes no sense to update more than once with same data)
	updater.queueRequestsMutex.Lock()
	c, ok := updater.queueRequests[guildID]
	if !ok {
		c = make(chan struct{}, 10)
		updater.queueRequests[guildID] = c
	}
	updater.queueRequestsMutex.Unlock()
	if len(c) == 0 {
		select {
		case c <- struct{}{}:
		default:
		}
	}
	updater.onFailureFuncsMutex.Lock()
	c2, ok := updater.onFailureFuncs[guildID]
	if !ok {
		c2 = make(chan func(), 100)
		updater.onFailureFuncs[guildID] = c2
	}
	updater.onFailureFuncsMutex.Unlock()
	if onFailure != nil {
		select {
		case c2 <- onFailure:
		default:
		}
	}
	// NOTE: wait the timeout and then start the updating
	// queue if it is not being updated already, or the
	// requests buffer hasn't already been cleared while waiting.
	if timeout > 0 {
		time.Sleep(timeout)
	}
	go updater.runUpdaterQueue(s, guildID)
}

// GetInteractionsBuffer returns the interaction buffer for the provided guildID
func (updater *QueueUpdater) GetInteractionsBuffer(guildID string) (chan *discordgo.Interaction, bool) {
	b, ok := updater.interactionsBuffer[guildID]
	return b, ok
}

// RunInactiveQueueUpdater is a long lived worker that check for inactive queues
// at interval. Queue is inactive when it had no listeners for the MaxAloneTime duration
// specified in the config.
func (updater *QueueUpdater) RunInactiveQueueUpdater(ctx context.Context, session *discordgo.Session) {
	done := ctx.Done()

	t := updater.maxAloneTime / 10
	if t < 10 {
		t = 10
	}
	ticker := time.NewTicker(t)

	inactive := make(map[string]time.Time)

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			// NOTE: for each guildID that has an audioplayer
			// check if the queue belonging to the guildiD should be updated
			// Also check if there are no undefened listeners in that channel.
			// If there aren't any for some time, dc the bot.
			for guildID := range session.VoiceConnections {
				inactiveSince, ok := inactive[guildID]

				// NOTE: the bot has been alone in the channel
				// for too long, mark the queue inactive and dc
				if ok && time.Since(inactiveSince) >= updater.maxAloneTime {
					delete(inactive, guildID)
					updater.markQueueInactive(session, guildID)
					return
				}

				// NOTE: check if the bot has no listeners
				// if so, if no already set, set the
				// inactive time, so we now how long the bot has been inactive
				if updater.hasNoListeners(
					ctx, session, guildID,
				) {
					// NOTE: only set time.Now if it is
					// not set already, so we don't override
					// the previous set
					if _, ok := inactive[guildID]; !ok {
						inactive[guildID] = time.Now()
					}
				} else {
					// NOTE: if there are listeners, remove
					// the inactive time
					delete(inactive, guildID)
				}
			}
		}
	}
}

// markQueueInactive persists inactive option to the queue identified
// by the provided guildID and session's clientID, removes it's pause option
// and disconects the client from the voiceChannel in the guild identified
// by the provided guildID
func (updater *QueueUpdater) markQueueInactive(s *discordgo.Session, guildID string) {
	if ap, ok := updater.audioplayers.Get(guildID); ok {
		ap.StopTerminate()
	}
	//updater.Update(s, guildID, -1, nil)

	if vc, ok := s.VoiceConnections[guildID]; ok {
		vc.Disconnect()
	}

	updater.log.WithField("GuildID", guildID).Trace(
		"Marked queue as inactive",
	)
}

// hasNoListeners checks whethere there are no undefened members
// in the same channel as the client
func (updater *QueueUpdater) hasNoListeners(ctx context.Context, s *discordgo.Session, guildID string) bool {
	clientState, err := s.State.VoiceState(guildID, s.State.User.ID)
	if err != nil {
		return true
	}
	maxMembersFetch := 1000
	done := ctx.Done()
	after := ""
outerMemberLoop:
	for i := 0; i < 100; i++ {
		members, err := s.GuildMembers(guildID, after, maxMembersFetch)
		if err != nil {
			return true
		}
	innerMemberLoop:
		for _, m := range members {
			select {
			case <-done:
				return true
			default:
				if m.User.ID == s.State.User.ID {
					continue innerMemberLoop
				}
				memberState, err := s.State.VoiceState(guildID, m.User.ID)
				if err != nil {
					continue innerMemberLoop
				}
				if memberState.ChannelID == clientState.ChannelID &&
					!memberState.Deaf && !memberState.SelfDeaf {
					return false
				}
			}
		}
		if len(members) < maxMembersFetch {
			break outerMemberLoop
		}
	}
	return true
}

func (updater *QueueUpdater) runUpdaterQueue(s *discordgo.Session, guildID string) {
	updater.queueRunningMutex.Lock()
	_, ok := updater.queueRunning[guildID]
	if !ok {
		updater.queueRunning[guildID] = struct{}{}
	}
	updater.queueRunningMutex.Unlock()
	if ok {
		updater.log.WithField("GuildID", guildID).Trace(
			"Queue updater queue already running",
		)
		return
	}
	updater.log.WithField("GuildID", guildID).Trace(
		"Starting queue updater queue",
	)

	updater.queueRequestsMutex.Lock()
	c, ok := updater.queueRequests[guildID]
	updater.queueRequestsMutex.Unlock()

	defer func() {
		updater.queueRequestsMutex.Lock()
		delete(updater.queueRequests, guildID)
		updater.queueRequestsMutex.Unlock()
		updater.queueRunningMutex.Lock()
		delete(updater.queueRunning, guildID)
		updater.queueRunningMutex.Unlock()
		updater.onFailureFuncsMutex.Lock()
		delete(updater.onFailureFuncs, guildID)
		updater.onFailureFuncsMutex.Unlock()
	}()

	for {
		select {
		case _, ok := <-c:
			if !ok {
				return
			}
			if err := updater.updateQueue(s, guildID); err != nil {
				updater.onFailureFuncsMutex.Lock()
				c2, ok := updater.onFailureFuncs[guildID]
				updater.onFailureFuncsMutex.Unlock()
				if ok && c2 != nil {
				onFailureLoop:
					for {
						select {
						case f, ok := <-c2:
							if !ok {
								break onFailureLoop
							}
							f()
						default:
							break onFailureLoop
						}
					}
				}
			}
		default:
			return
		}
	}
}

func (updater *QueueUpdater) updateQueue(s *discordgo.Session, guildID string) error {
	updater.log.WithField("GuildID", guildID).Trace(
		"Updating queue",
	)
	clientID := s.State.User.ID

	queue, err := updater.datastore.Queue().GetQueue(clientID, guildID)
	if err != nil {
		updater.log.WithField("GuildID", guildID).Trace(
			"No queue found, cannot update message",
		)
		return err
	}
	queue, err = updater.datastore.Song().UpdateQueueWithSongs(queue)
	if err != nil {
		updater.log.WithField("GuildID", guildID).Trace(
			"No queue data, cannot update message",
		)
		return err
	}
	var components []discordgo.MessageComponent
	if !updater.ready() {
		components = updater.builder.Queue().GetOfflineQueueComponents(queue)
	} else if vc, ok := s.VoiceConnections[guildID]; ok && vc.Ready {
		components = updater.builder.Queue().GetMusicQueueComponents(queue)
	} else {
		components = updater.builder.Queue().GetInactiveQueueComponents(queue)

	}
	embed := updater.builder.Queue().MapQueueToEmbed(queue)

	err = nil

	updater.mutex.RLock()
	buffer, ok := updater.interactionsBuffer[guildID]
	updater.mutex.RUnlock()

	if !ok {
		buffer = make(chan *discordgo.Interaction)
	}
updateLoop:
	for {
		select {
		case i := <-buffer:
			err = s.InteractionRespond(i, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: components,
				},
			})
			if err != nil {
				continue updateLoop
			}
			updater.log.WithField("GuildID", guildID).Trace(
				"Queue updated from interaction",
			)
			break updateLoop
		default:
			_, err = s.ChannelMessageEditComplex(
				&discordgo.MessageEdit{
					ID:         queue.MessageID,
					Channel:    queue.ChannelID,
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: components,
				})
			if err == nil {
				updater.log.WithField("GuildID", guildID).Trace(
					"Queue updated by GuildID and ChannelID",
				)
			} else {
				updater.log.WithField("GuildID", guildID).Tracef(
					"Failed when updating queue: %v",
					err,
				)
				return err
			}
			break updateLoop
		}
	}
	return nil
}
