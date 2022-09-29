package bot

import (
	"discord-music-bot/model"

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
			// NOTE: the bot has been disconnected from voice channel
			// mark the queue inactive
			if err := bot.datastore.PersistQueueOptions(
				s.State.User.ID,
				i.GuildID,
				model.InactiveOption(),
			); err != nil {
				bot.Errorf("Error when persisting inactive option: %v", err)
			} else {
				bot.queueUpdater.NeedsUpdate(i.GuildID)
				bot.queueUpdater.Update(s, i.GuildID)
			}
		}
	}
}
