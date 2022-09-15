package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onInteractionCreate() is a handler function called when discord emits
// INTERACTION_CREATE event. It determines the type of interaction
// and whether it is relevant to the music bot. If so it calls a
// function based on the interaction type
func (bot *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
}
