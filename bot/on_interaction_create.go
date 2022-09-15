package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
)

// onInteractionCreate is a handler function called when discord emits
// INTERACTION_CREATE event. If the interaction is application command and
// the name matches the bot's music or help command, it calls either
// onMusicSlashCommand or onHelpSlashCommand. Otherwise if the interaction's
// type is messageComponent and the customID start's with the bot's components'
// customID prefix, it calls either onButtonClick, onSelectMenu or onTextInput
func (bot *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if i.GuildID == "" {
		return
	}
	bot.WithField("GuildID", i.GuildID).Trace("Interaction created")
	if i.Type == discordgo.InteractionApplicationCommand {
		if i.ApplicationCommandData().Name ==
			bot.slashCommandsConfig.Music.Name {
			// NOTE: recieved interaction is a music slash command
			bot.onMusicSlashCommand(s, i)
		} else if i.ApplicationCommandData().Name ==
			bot.slashCommandsConfig.Help.Name {
			// NOTE: recieved interaction is a help slash command
			bot.onHelpSlashCommand(s, i)
		}
	} else if i.Type == discordgo.InteractionMessageComponent &&
		strings.HasPrefix(i.MessageComponentData().CustomID, bot.customIDPrefix) {
		//NOTE: all message component id's authored by the bot start with the same prefix
		// that way we know bot is the author

		if i.Interaction.MessageComponentData().ComponentType ==
			discordgo.ButtonComponent {
			// NOTE: a user has clicked a button on a message owned
			// by the bot
			bot.onButtonClick(s, i)
		} else if i.Interaction.MessageComponentData().ComponentType ==
			discordgo.SelectMenuComponent {
			// NOTE: a user has selected something from a select menu
			// on a message owned by the bot
			bot.onSelectMenu(s, i)
		} else if i.Interaction.MessageComponentData().ComponentType ==
			discordgo.TextInputComponent {
			// NOTE: a user has typed something into a text input
			// on a message owned by the bot
			bot.onTextInput(s, i)
		}
	}
}
