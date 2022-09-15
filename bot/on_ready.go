package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onReady is a handler function called when discord emits
// READY event
func (bot *Bot) onReady(s *discordgo.Session, r *discordgo.Ready) {
	bot.Infof("Bot %s #%s ready!", r.User.Username, r.User.Discriminator)
}
