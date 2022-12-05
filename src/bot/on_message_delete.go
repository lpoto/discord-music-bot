package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onMessageDelete is a handler function called when discord emits
// MESSAGE_DELETE event. It determines whether the delted message
// was a music bot's queue message and if so, it deletes the queue.
func (bot *DiscordEventHandler) onMessageDelete(m *discordgo.MessageDelete) {
	bot.log.WithField("GuildID", m.GuildID).Trace("Message deleted")

	util := &Util{bot.Bot}
	util.deleteQueue(bot.session, m.GuildID, []string{m.ID})
}

// onBulkMessageDelete is a handler function called when discord emits
// MESSAGE_DELETE_BULK event. It determines whether any of the delted messages
// was a music bot's queue message and if so, it deletes the queue.
func (bot *DiscordEventHandler) onBulkMessageDelete(m *discordgo.MessageDeleteBulk) {
	util := &Util{bot.Bot}
	util.deleteQueue(bot.session, m.GuildID, m.Messages)
}
