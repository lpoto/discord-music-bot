package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onSelectMenu is a handler function called when a user
// selects from the select menu on a message authored by the bot.
// This is not emitted through the discord websocket, but is rather
// called from the INTERACTION_CREATE event when the interaction type
// is select menu and the message author is bot
func (bot *Bot) onSelectMenu(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Menu selected")
}
