package bot

import "github.com/bwmarrin/discordgo"

// onStopSlashCommand is a handler function called when the bot's stop slash
// command is called in the discord channel, this is not emmited through the
// discord's websocket, but is rather called from INTERACTION_CREATE event when
// the interaction's command data name matches the stop slash command's name.
func (bot *Bot) onStopSlashCommand(i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Stop slash command")

	queue, err := bot.datastore.Queue().GetQueue(
		bot.session.State.User.ID, i.GuildID,
	)
	if err != nil {
		if err := bot.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "There is no active music queue!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			bot.WithField("GuildID", i.GuildID).Errorf(
				"Error when responding to help command: %v",
				err,
			)
		}
		return
	}
	if err := bot.session.ChannelMessageDelete(queue.ChannelID, queue.MessageID); err != nil {
		if err := bot.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Something went wrong!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
			bot.WithField("GuildID", i.GuildID).Errorf(
				"Error when responding to help command: %v",
				err,
			)
		}
	}
	bot.deleteQueue(bot.session, i.GuildID, []string{queue.MessageID})

	if err := bot.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Music has been stopped!",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	}); err != nil {
		bot.WithField("GuildID", i.GuildID).Errorf(
			"Error when responding to help command: %v",
			err,
		)
	}
}
