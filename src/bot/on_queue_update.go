package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onUpdateQueueFromInteraction fetches the queue from the datastore
// based on the provided interaction's guildID and session state's user id,
// then fetches the message that belongs to the queue's messageID and updates
// it with the fetched queue.
// This is called from other handler functions and is expected to have
// an unexpired, unresponded interaction.
func (bot *Bot) onUpdateQueueFromInteraction(s *discordgo.Session, i *discordgo.Interaction) error {
	clientID := s.State.User.ID
	guildID := i.GuildID

	queue, err := bot.datastore.GetQueue(clientID, guildID)
	if err != nil {
		return err
	}
	embed := bot.builder.MapQueueToEmbed(queue)
	components := bot.builder.GetMusicQueueComponents(queue)

	if err := s.InteractionRespond(i, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	}); err != nil {
		return err
	}
	return nil
}

// onUpdateQueueFromGuildID fetches the queue from the datastore
// based on the provided guildID and session state's user id,
// then fetches the message that belongs to the queue's messageID and updates
// it with the fetched queue.
// Returns error if the queue message does not exist.
func (bot *Bot) onUpdateQueueFromGuildID(s *discordgo.Session, guildID string) error {
	clientID := s.State.User.ID

	queue, err := bot.datastore.GetQueue(clientID, guildID)
	if err != nil {
		bot.Errorf("Error when updating queue from interaction: %v", err)
		return nil
	}
	embed := bot.builder.MapQueueToEmbed(queue)
	components := bot.builder.GetMusicQueueComponents(queue)

	// Try to update the queue message, return err on failure
	// (if the message no longer exists)
	if _, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		ID:         queue.MessageID,
		Channel:    queue.ChannelID,
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}); err != nil {
		bot.Tracef("Failed to update queue message: %v", err)
		return err
	}
	return nil
}
