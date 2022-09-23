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
}

// NewQueueUpdater constructs a new object that
// handles updating queues
func NewQueueUpdater(builder *builder.Builder, datastore *datastore.Datastore, audioplayers *audioplayer.AudioPlayersMap) *QueueUpdater {
	return &QueueUpdater{
		mutex:              sync.Mutex{},
		lastUpdated:        make(map[string]time.Time),
		needUpdate:         make(map[string]struct{}),
		interactionsBuffer: make(map[string]chan *discordgo.Interaction),
		builder:            builder,
		datastore:          datastore,
		audioplayers:       audioplayers,
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
	position := 0
	if ap, ok := updater.audioplayers.Get(guildID); ok {
		position = int(ap.PlaybackPosition().Truncate(time.Second).Seconds())
	}
	embed := updater.builder.MapQueueToEmbed(queue, position)
	components := updater.builder.GetMusicQueueComponents(queue)

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
	if err != nil {
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

// RunIntervalUpdater is a long lived worker that updated all the queues at the provided
// interval. Only queues with active audioplayers are updated.
// If a queue was updated less than the interval ago, it is not updated again.
func (updater *QueueUpdater) RunIntervalUpdater(ctx context.Context, session *discordgo.Session, interval time.Duration) {
	done := ctx.Done()

	ticker := time.NewTicker(interval)

	skip := make(map[string]struct{})
	skipMutex := sync.Mutex{}

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			for _, guildID := range updater.audioplayers.Keys() {
				// NOTE: for each guildID that has an audioplayer
				// check if the queue belonging to the guildiD should be updated
				go func(guildID string) {
					// NOTE: if guildID is in skip, don't update
					skipMutex.Lock()
					if _, ok := skip[guildID]; ok {
						delete(skip, guildID)
						skipMutex.Unlock()
						return
					}
					skipMutex.Unlock()
					if ap, ok := updater.audioplayers.Get(
						guildID,
					); !ok || ap.IsPaused() || ap.TimeLeft()+(time.Second) < interval {
						return
					}
					updater.mutex.Lock()
					if t, ok := updater.lastUpdated[guildID]; ok {
						if time.Since(t) < interval {
							updater.mutex.Unlock()
							return
						}
					}
					updater.mutex.Unlock()

					skipMutex.Lock()
					// NOTE: add guildiD to skip
					// if the update was fast, remove it,
					// else the discord is blocking us on the message,
					// so we leave it in the skip, and we skip an update
					// on that message and relieve the discord's limit
					skip[guildID] = struct{}{}
					skipMutex.Unlock()
					updater.NeedsUpdate(guildID)
					t := time.Now()
					updater.Update(session, guildID)

					if time.Since(t) < time.Second {
						skipMutex.Lock()
						delete(skip, guildID)
						skipMutex.Unlock()
					}
				}(guildID)
			}
		}
	}
}
