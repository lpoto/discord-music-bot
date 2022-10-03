package bot

import "github.com/bwmarrin/discordgo"

// onResendMessageCommand is a handler function called when the name of interaction's
// application command data matches the registered Resend global message command.
func (bot *Bot) onResendMessageCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	queue, err := bot.datastore.GetQueue(
		s.State.User.ID,
		i.GuildID,
	)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "There is no music queue in this server!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}
	toDeleteMessageID := queue.MessageID
	toDeleteChannelID := queue.ChannelID

	embed := bot.builder.MapQueueToEmbed(queue, 0)
	components := bot.builder.GetMusicQueueComponents(queue)

	err = s.InteractionRespond(
		i.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})
	if err != nil {
		bot.WithField("GuildID", i.GuildID).Errorf(
			"Error when resending queue: %v",
			err,
		)
		return
	}
	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		bot.Errorf(
			"Error when fetching interaction response message: %v",
			err,
		)
		return
	}
	queue.MessageID = msg.ID
	queue.ChannelID = msg.ChannelID
	if err := bot.datastore.UpdateQueue(queue); err != nil {
		bot.Errorf("Error when updating queue: %v", err)
	}
	if err := s.ChannelMessageDelete(
		toDeleteChannelID,
		toDeleteMessageID,
	); err != nil {
		bot.Errorf("Error when deleting channel message: %v", err)
	}
}
