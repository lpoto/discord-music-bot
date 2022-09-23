package updater

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/builder"
	"discord-music-bot/datastore"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
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
	defer updater.mutex.Unlock()

	// NOTE: the queue no longer needs to be updated
	if _, ok := updater.needUpdate[guildID]; !ok {
		return nil
	}

	defer func() {
		// NOTE: queue has been updated, so it no longer
		// needs an update
		delete(updater.needUpdate, guildID)
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

updateLoop:
	for {
		select {
		case i := <-updater.interactionsBuffer[guildID]:
			if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: components,
				},
			}); err != nil {
				continue updateLoop
			}
			return nil
		default:
			if _, err := s.ChannelMessageEditComplex(
				&discordgo.MessageEdit{
					ID:         queue.MessageID,
					Channel:    queue.ChannelID,
					Embeds:     []*discordgo.MessageEmbed{embed},
					Components: components,
				}); err != nil {
				return err
			} else {
				return nil
			}
		}
	}
}

// GetInteractionsBuffer returns the interaction buffer for the provided guildID
func (updater *QueueUpdater) GetInteractionsBuffer(guildID string) (chan *discordgo.Interaction, bool) {
	b, ok := updater.interactionsBuffer[guildID]
	return b, ok
}
