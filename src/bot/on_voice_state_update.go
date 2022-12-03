package bot

import (
	"discord-music-bot/model"
	"time"

	"github.com/bwmarrin/discordgo"
)

// onVoiceStateUpdate is a handler function called when discord emits
// VoiceStateUpdate event. It determines whether voice sate update occured
// in the music bot's voice channel and whether the bot has
// enough active listeners to continue playing music.
func (bot *Bot) onVoiceStateUpdate(s *discordgo.Session, i *discordgo.VoiceStateUpdate) {
	if !bot.ready {
		return
	}
	bot.WithField("GuildID", i.GuildID).Trace("Voice state update")

	if i.UserID == s.State.User.ID {
		if i.BeforeUpdate != nil &&
			len(i.BeforeUpdate.ChannelID) > 0 && len(i.ChannelID) == 0 {
			// NOTE: remove paused option, so that on reconnect the
			// bot is ready to play
			bot.datastore.Queue().RemoveQueueOptions(
				s.State.User.ID,
				i.GuildID,
				model.Paused,
			)
			// NOTE: delete audioplayer if any
			if ap, ok := bot.audioplayers.Get(i.GuildID); ok && ap != nil {
				ap.Continue = false
				ap.Stop()
			}
			bot.queueUpdater.NeedsUpdate(i.GuildID)
			time.Sleep(1 * time.Second)
			bot.queueUpdater.Update(s, i.GuildID)
		}
	}
}
