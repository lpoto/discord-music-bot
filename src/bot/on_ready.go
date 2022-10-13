package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// onReady is a handler function called when discord emits
// READY event
func (bot *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	s.UpdateListeningStatus(
		"/" + bot.config.SlashCommands.Help.Name,
	)
	bot._ready = true

	// check if any queues should be removed from datastore
	bot.cleanDiscordMusicQueues(s)

	time.Sleep(500 * time.Millisecond)

	// NOTE: mark the bot as ready, so the
	// other handlers start working
	bot.WithFields(log.Fields{
		"Username": r.User.Username + " #" + r.User.Discriminator,
	}).Info("Bot ready")

	bot.ready = true

}
