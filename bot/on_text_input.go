package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onTextInput is a handler function called when a user
// inputs a text to a text input on a message authored by the bot.
// This is not emitted through the discord websocket, but is rather
// called from the INTERACTION_CREATE event when the interaction type
// is text input and the message author is bot
func (bot *Bot) onTextInput(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Text input")
}
