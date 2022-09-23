package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// interactionToQueueUpdateBuffer adds the provided interaction to the
// bot's queueUpdateInteractionsBuffer, if it is not already full.
// The provided interaction is deffered after some time, so the
// interaction failed error is not thrown in discord channel.
// This interaction may then be used when updating the queue, if it
// was not yet deffered.
func (bot *Bot) interactionToQueueUpdateBuffer(s *discordgo.Session, i *discordgo.Interaction) {
	go func() {
		time.Sleep(discordgo.InteractionDeadline - (300 * time.Millisecond))
		s.InteractionRespond(i, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseDeferredMessageUpdate,
		})
	}()
	if _, ok := bot.queueUpdateInteractionsBuffer[i.GuildID]; !ok {
		bot.queueUpdateInteractionsBuffer[i.GuildID] = make(
			chan *discordgo.Interaction, 10,
		)
	}
	select {
	case bot.queueUpdateInteractionsBuffer[i.GuildID] <- i:
	default:
	}
}

// updateQueue fetches the queue from the datastore and updates it.
// It first tries to update it from the interactions stored in the
// bot's queueUpdateInteractionsBuffer, and if not successful from
// the queue's channelID and messageID.
func (bot *Bot) updateQueue(s *discordgo.Session, guildID string) error {
	clientID := s.State.User.ID

	queue, err := bot.datastore.GetQueue(clientID, guildID)
	if err != nil {
		return err
	}
	position := 0
	if ap, ok := bot.audioplayers[guildID]; ok {
		position = int(ap.PlaybackPosition().Truncate(time.Second))
	}
	embed := bot.builder.MapQueueToEmbed(queue, position)
	components := bot.builder.GetMusicQueueComponents(queue)
updateLoop:
	for {
		select {
		case i := <-bot.queueUpdateInteractionsBuffer[guildID]:
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
