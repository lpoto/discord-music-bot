package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onMessageDelete is a handler function called when discord emits
// MESSAGE_DELETE event. It determines whether the delted message
// was a music bot's queue message and if so, it deletes the queue.
func (bot *Bot) onMessageDelete(s *discordgo.Session, m *discordgo.MessageDelete) {
	if m.GuildID == "" {
		return
	}
	bot.WithField("GuildID", m.GuildID).Trace("Message deleted")
}

// onBulkMessageDelete is a handler function called when discord emits
// MESSAGE_DELETE_BULK event. It determines whether any of the delted messages
// was a music bot's queue message and if so, it deletes the queue.
func (bot *Bot) onBulkMessageDelete(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {
	if m.GuildID == "" {
		return
	}
	bot.WithField("GuildID", m.GuildID).Trace("Messages bulk deleted")
}
