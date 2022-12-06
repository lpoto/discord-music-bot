package bot

import (
	"discord-music-bot/bot/transaction"
	"discord-music-bot/model"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// onVoiceStateUpdate is a handler function called when discord emits
// VoiceStateUpdate event. It determines whether voice sate update occured
// in the music bot's voice channel and whether the bot has
// enough active listeners to continue playing music.
func (bot *DiscordEventHandler) onVoiceStateUpdate(t *transaction.Transaction, i *discordgo.VoiceStateUpdate) {
	defer t.Defer()

	// NOTE: for us only relevant updates are when client
	// switches channels or leaves channel
	if i.BeforeUpdate == nil || len(i.BeforeUpdate.ChannelID) == 0 {
		return
	}
	if len(i.ChannelID) == 0 {
		bot.log.WithFields(log.Fields{
			"GuildID": t.GuildID(),
			"From":    i.BeforeUpdate.ChannelID,
		}).Trace("Client has left the channel")

		if ap, ok := bot.audioplayers.Get(i.GuildID); ok && ap != nil {
			ap.Subscriptions().Emit("terminate")
		}

		time.Sleep(1 * time.Second)
		_, e := bot.datastore.Queue().GetQueue(
			bot.session.State.User.ID,
			t.GuildID(),
		)
		if e == nil {
			// NOTE: remove paused option, so that on reconnect the
			// bot is ready to play
			bot.datastore.Queue().RemoveQueueOptions(
				bot.session.State.User.ID,
				i.GuildID,
				model.Paused,
			)
			t.UpdateQueue(0)
		} else {
			t.Defer()
		}
	} else {
		// WARNING: this is here only due to the bug in
		// godiscord that cancels voice connection when switching
		// channels, this should be removed once the fix is merged.
		// And the channel switching handled properly.
		bot.log.WithFields(log.Fields{
			"GuildID": t.GuildID(),
			"From":    i.BeforeUpdate.ChannelID,
			"TO":      i.ChannelID,
		}).Trace("Client has switched channels")

		voice, ok := bot.session.VoiceConnections[i.GuildID]
		time.Sleep(1 * time.Second)
		if v, ok := bot.session.VoiceConnections[i.GuildID]; ok && v.Ready {
			return
		}
		if ok {
			voice.Disconnect()
		}
		bot._ready = false
		t.UpdateQueue(500 * time.Millisecond)
		time.Sleep(4 * time.Second)
		bot._ready = true
		t.Refresh()
		t.UpdateQueue(500 * time.Millisecond)
	}
}
