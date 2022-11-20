package updater

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/builder"
	"discord-music-bot/datastore"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/net/context"
)

type QueueUpdater struct {
	mutex              sync.Mutex
	lastUpdated        map[string]time.Time
	needUpdate         map[string]struct{}
	interactionsBuffer map[string]chan *discordgo.Interaction
	builder            *builder.Builder
	datastore          *datastore.Datastore
	audioplayers       *audioplayer.AudioPlayersMap
	maxAloneTime       time.Duration
	ready              func() bool
}

// NewQueueUpdater constructs a new object that
// handles updating queues
func NewQueueUpdater(builder *builder.Builder, maxAloneTime time.Duration, datastore *datastore.Datastore, audioplayers *audioplayer.AudioPlayersMap, ready func() bool) *QueueUpdater {
	return &QueueUpdater{
		mutex:              sync.Mutex{},
		lastUpdated:        make(map[string]time.Time),
		needUpdate:         make(map[string]struct{}),
		interactionsBuffer: make(map[string]chan *discordgo.Interaction),
		builder:            builder,
		datastore:          datastore,
		audioplayers:       audioplayers,
		maxAloneTime:       maxAloneTime,
		ready:              ready,
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

// NeedsUpdate marks that the queue in the guild identified
// by the provided ID needs to be updated
func (updater *QueueUpdater) NeedsUpdate(guildID string) {
	updater.mutex.Lock()
	defer updater.mutex.Unlock()

	if _, ok := updater.needUpdate[guildID]; !ok {
		updater.needUpdate[guildID] = struct{}{}
	}
}

// Update fetches the queue from the datastore and updates it.
// It first tries to update it from the interactions stored in the
// bot's queueUpdateInteractionsBuffer, and if not successful from
// the queue's channelID and messageID.
// TODO: this function needs refactoring
func (updater *QueueUpdater) Update(s *discordgo.Session, guildID string) error {
	updater.mutex.Lock()

	// NOTE: the queue no longer needs to be updated
	if _, ok := updater.needUpdate[guildID]; !ok {
		updater.mutex.Unlock()
		return nil
	}
	updater.mutex.Unlock()

	defer func() {
		// NOTE: queue has been updated, so it no longer
		// needs an update
		updater.mutex.Lock()
		delete(updater.needUpdate, guildID)
		updater.mutex.Unlock()
	}()

	clientID := s.State.User.ID

	queue, err := updater.datastore.GetQueue(clientID, guildID)
	if err != nil {
		return err
	}
	state := builder.QueueStateInactive
	if !updater.ready() {
		state = builder.QueueStateOffline
	} else {
		if vc, ok := s.VoiceConnections[guildID]; ok && len(vc.ChannelID) > 0 {
			state = builder.QueueStateDefault
		}
	}
	embed := updater.builder.MapQueueToEmbed(queue)
	components := updater.builder.GetMusicQueueComponents(queue, state)

	err = nil

	updater.mutex.Lock()
	buffer, ok := updater.interactionsBuffer[guildID]
	updater.mutex.Unlock()

	if !ok {
		_, err = s.ChannelMessageEditComplex(
			&discordgo.MessageEdit{
				ID:         queue.MessageID,
				Channel:    queue.ChannelID,
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			})
	} else {

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
				break updateLoop
			default:
				_, err = s.ChannelMessageEditComplex(
					&discordgo.MessageEdit{
						ID:         queue.MessageID,
						Channel:    queue.ChannelID,
						Embeds:     []*discordgo.MessageEmbed{embed},
						Components: components,
					})
				break updateLoop
			}
		}
	}
	if err == nil {
		updater.mutex.Lock()
		updater.lastUpdated[guildID] = time.Now()
		updater.mutex.Unlock()
	}
	return err
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
			for _, guildID := range updater.audioplayers.Keys() {
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
		ap.AddDeferFunc(func(*discordgo.Session, string) {})
		ap.Continue = false
		ap.Stop()
	}
	updater.NeedsUpdate(guildID)

	if vc, ok := s.VoiceConnections[guildID]; ok {
		vc.Disconnect()
	}
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
