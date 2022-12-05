package bot

import (
	"discord-music-bot/model"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// onVoiceStateUpdate is a handler function called when discord emits
// VoiceStateUpdate event. It determines whether voice sate update occured
// in the music bot's voice channel and whether the bot has
// enough active listeners to continue playing music.
func (bot *Bot) onVoiceStateUpdate(i *discordgo.VoiceStateUpdate) {
	if !bot.ready {
		return
	}
	if i.UserID == bot.session.State.User.ID {
		fields := log.Fields{
			"GuildID": i.GuildID,
			"To":      i.ChannelID,
		}
		if i.BeforeUpdate != nil {
			if len(i.BeforeUpdate.ChannelID) > 0 {
				if len(i.ChannelID) == 0 {

					// NOTE: remove paused option, so that on reconnect the
					// bot is ready to play
					bot.datastore.Queue().RemoveQueueOptions(
						bot.session.State.User.ID,
						i.GuildID,
						model.Paused,
					)
					if ap, ok := bot.audioplayers.Get(i.GuildID); ok && ap != nil {
						ap.StopVoiceClosed()
					}
					bot.queueUpdater.Update(
						bot.session,
						i.GuildID,
						500*time.Millisecond,
						nil,
					)
				} else {
					// WARNING: this is here only due to the bug in
					// godiscord that cancels voice connection when switching
					// channels, this should be removed once the fix is merged.
					// And the channel switching handled properly.

					voice, ok := bot.session.VoiceConnections[i.GuildID]
					time.Sleep(1 * time.Second)
					if v, ok := bot.session.VoiceConnections[i.GuildID]; ok && v.Ready {
						return
					}
					if ok {
						voice.Disconnect()
					}
					bot._ready = false
					bot.queueUpdater.Update(
						bot.session,
						i.GuildID,
						500*time.Millisecond,
						nil,
					)
					time.Sleep(4 * time.Second)
					bot._ready = true
					bot.queueUpdater.Update(
						bot.session,
						i.GuildID,
						500*time.Millisecond,
						nil,
					)
				}
			}
		}

		bot.WithFields(fields).Trace(
			"Voice state update",
		)
	}
}
