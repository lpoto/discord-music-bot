package bot

import "github.com/bwmarrin/discordgo"

// onStopMessageCommand is a handler function called when the name of interaction's
// application command data matches the registered Stop global message command.
func (bot *Bot) onStopMessageCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
	// NOTE: delete the queue message, the queue should then be deleted
	// in the message_deleted handler
	if err := s.ChannelMessageDelete(queue.ChannelID, queue.MessageID); err != nil {
		bot.Errorf("Error when deleting channel message: %v", err)
	}
	// NOTE: end the audioplayer if there is any
	if ap, ok := bot.audioplayers.Get(i.GuildID); ok && ap != nil {
		ap.Continue = false
		ap.Stop()
	}
	// NOTE: disconnect from voice
	if vc, ok := s.VoiceConnections[i.GuildID]; ok {
		vc.Disconnect()
	}
	// NOTE: Notify the user that the queue has been deleted with
	// an ephemeral message
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "The queue has been successfully deleted.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
