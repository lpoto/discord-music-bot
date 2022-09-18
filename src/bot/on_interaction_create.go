package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// onInteractionCreate is a handler function called when discord emits
// INTERACTION_CREATE event. It determines the type of interaction
// and which handler should be called.
func (bot *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if i.GuildID == "" || i.Interaction.AppID != s.State.User.ID {
		// NOTE: only listen for interactions in guilds.
		// Interaction's appID should be equal to the bot' user id, so we
		// respond only to application commands authored by the bot and
		// interactions on messages authored by the bot
		return
	}
	t := time.Now()
	bot.WithField(
		"GuildID", i.GuildID,
	).Tracef("Interaction created (%s)", i.ID)

	defer func() {
		bot.WithField(
			"Latency", time.Since(t),
		).Tracef("Interaction handled (%s)", i.ID)

	}()

	if i.Type == discordgo.InteractionApplicationCommand {
		// NOTE: an application command has been used,
		// determine which one

		switch i.ApplicationCommandData().Name {
		case bot.applicationCommandsConfig.Music.Name:
			// music slash command has been used
			bot.onMusicSlashCommand(s, i)
		case bot.applicationCommandsConfig.Help.Name:
			// help slash command has been used
			bot.onHelpSlashCommand(s, i)
		}
	} else if i.Type == discordgo.InteractionModalSubmit {
		// NOTE: a user has submited a modal in the discord server
		// determine which modal has been submitted
		switch bot.getModalName(i.Interaction.ModalSubmitData()) {
		case bot.applicationCommandsConfig.AddSongs.Name:
			// add songs modal has been submited
			bot.onAddSongsModalSubmit(s, i)
		}

	} else if i.Type == discordgo.InteractionMessageComponent {

		//NOTE: all message component id's authored by the bot start with the same prefix
		// that way we know bot is the author

		switch i.Interaction.MessageComponentData().ComponentType {
		case discordgo.ButtonComponent:
			// a button has been clicked
			bot.onButtonClick(s, i)
		}
	}
}
