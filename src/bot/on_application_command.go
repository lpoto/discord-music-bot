package bot

import (
	"discord-music-bot/bot/transaction"
	"strings"
)

// onApplicationCommand is a handler function called when discord emits
// INTERACTION_CREATE event and the interaction's type is applicationCommand.
func (bot *DiscordEventHandler) onApplicationCommand(t *transaction.Transaction) {
	util := &Util{bot.Bot}

	// NOTE: an application command has been used,
	// determine which one.

	name := strings.TrimSpace(
		t.Interaction().ApplicationCommandData().Name,
	)
	switch name {
	case strings.TrimSpace(bot.config.SlashCommands.Music.Name):
		// music slash command has been used
		if !util.checkVoice(t) {
			// should check voice connection when starting
			// a music queue
			return
		}
		bot.onMusicSlashCommand(t)
		return
	case strings.TrimSpace(bot.config.SlashCommands.Stop.Name):
		bot.onStopSlashCommand(t)
		return
	case strings.TrimSpace(bot.config.SlashCommands.Help.Name):
		// help slash command has been used
		bot.onHelpSlashCommand(t)
		return
	}
}
