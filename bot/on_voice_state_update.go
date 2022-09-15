package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onVoiceStateUpdate is a handler function called when discord emits
// VoiceStateUpdate event. It determines whether voice sate update occured
// in the music bot's voice channel and whether the bot has
// enough active listeners to continue playing music.
func (bot *Bot) onVoiceStateUpdate(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	bot.WithField("GuildID", i.GuildID).Trace("Voice state update")
}
