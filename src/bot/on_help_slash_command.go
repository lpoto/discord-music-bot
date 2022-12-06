package bot

import (
	"discord-music-bot/bot/transaction"

	"github.com/bwmarrin/discordgo"
)

// onHelpSlashCommand is a handler function called when the bot's help slash
// command is called in the discord channel, this is not emmited through the
// discord's websocket, but is rather called from INTERACTION_CREATE event when
// the interaction's command data name matches the help slash command's name.
func (bot *DiscordEventHandler) onHelpSlashCommand(t *transaction.Transaction) {
	defer t.Defer()

	help := bot.helpContent
	if len(help) == 0 {
		help = "Sorry, there is currently no help available."
	}
	if err := bot.session.InteractionRespond(
		t.Interaction(),
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: help,
				Flags: discordgo.MessageFlagsEphemeral +
					discordgo.MessageFlagsSupressEmbeds,
			},
		}); err != nil {
		bot.log.WithField("GuildID", t.GuildID()).Errorf(
			"Error when responding to help command: %v",
			err,
		)
	}
}
