package bot

import (
	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// onReady is a handler function called when discord emits
// READY event
func (bot *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	bot.WithFields(log.Fields{
		"Username": r.User.Username + " #" + r.User.Discriminator,
	}).Info("Bot ready")
}