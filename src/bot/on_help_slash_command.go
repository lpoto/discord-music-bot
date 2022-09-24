package bot

import "github.com/bwmarrin/discordgo"

// onHelpSlashCommand is a handler function called when the bot's help slash
// command is called in the discord channel, this is not emmited through the
// discord's websocket, but is rather called from INTERACTION_CREATE event when
// the interaction's command data name matches the help slash command's name.
func (bot *Bot) onHelpSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Help slash command")
	help := bot.helpContent
	if len(help) == 0 {
		help = "Sorry, there is currently no help available."
	}
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: help,
			Flags:   1 << 6, // 1 << 6 flags marks the response ephemeral
		},
	}); err != nil {
		bot.WithField("GuildID", i.GuildID).Errorf(
			"Error when responding to help command: %v",
			err,
		)
	}
}
