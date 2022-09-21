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

	bot.deleteQueue(s.State.User.ID, m.GuildID, []string{m.ID})
}

// onBulkMessageDelete is a handler function called when discord emits
// MESSAGE_DELETE_BULK event. It determines whether any of the delted messages
// was a music bot's queue message and if so, it deletes the queue.
func (bot *Bot) onBulkMessageDelete(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {
	if m.GuildID == "" {
		return
	}
	bot.deleteQueue(s.State.User.ID, m.GuildID, m.Messages)
}

func (bot *Bot) deleteQueue(clientID string, guildID string, messageIDs []string) {
	queue, err := bot.datastore.FindQueue(
		clientID,
		guildID,
	)
	if err != nil {
		return
	}
	ok := false
	for _, v := range messageIDs {
		if queue.MessageID == v {
			ok = true
			break
		}
	}
	if !ok {
		bot.Trace("The queue message was not deleted")
		return
	}
	bot.Trace("The queue message was deleted, removing the queue")
	if err := bot.datastore.RemoveQueue(
		clientID,
		queue.GuildID,
	); err != nil {
		bot.Errorf(
			"Error when removing queue after message delete: %v",
			err,
		)
	}
}
