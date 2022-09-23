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
	time.Sleep(time.Second)
	bot.WithFields(log.Fields{
		"Username": r.User.Username + " #" + r.User.Discriminator,
	}).Info("Bot ready")

	// NOTE: mark the bot as ready, so the
	// other handlers start working
	bot.ready = true
}
