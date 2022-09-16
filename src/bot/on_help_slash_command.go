package bot

import "github.com/bwmarrin/discordgo"

// onHelpSlashCommand is a handler function called when the bot's help slash
// command is called in the discord channel, this is not emmited through the
// discord's websocket, but is rather called from INTERACTION_CREATE event when
// the interaction's command data name matches the help slash command's name.
func (bot *Bot) onHelpSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Help slash command")
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: bot.help(),
			Flags:   1 << 6, // 1 << 6 flags marks the response ephemeral
		},
	})
}

func (bot *Bot) help() string {
	return "There is currently no help available :("
}
