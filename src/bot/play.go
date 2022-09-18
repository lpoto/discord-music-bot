package bot

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/model"

	"github.com/bwmarrin/discordgo"
)

// play searches for a queue that belongs to the provided guildID
// and starts playing it's headSong if no song is currently playing.
func (bot *Bot) play(s *discordgo.Session, guildID string, channelID string) {
	if len(channelID) == 0 {
		return
	}
	_, ok := bot.audioplayers[guildID]
	if ok {
		// NOTE: audio has already been started from
		// another source, do not continue
		return
	}

	bot.WithField("GuildID", guildID).Trace("Play request")

	ap := audioplayer.NewAudioPlayer(s, guildID)

	bot.audioplayers[guildID] = ap
	defer delete(bot.audioplayers, guildID)

	queue, err := bot.datastore.GetQueue(s.State.User.ID, guildID)
	if err != nil {
		return
	}
	if queue.HeadSong == nil {
		return
	}
	_, err = s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		bot.Errorf("Could not join voice: %v", err)
		return
	}

	// NOTE: always play the queue's headSong
	song := queue.HeadSong

	bot.WithField(
		"GuildID", guildID,
	).Tracef("Playing song: %s", song.Name)

	if err := ap.Play(bot.ctx, song); err != nil {
		bot.Errorf("Error when playing: %v", err)
	}
	if err := bot.datastore.RemoveSongs(
		// NOTE: the finished song should be removed from the queue
		s.State.User.ID,
		guildID,
		[]uint{song.ID},
	); err != nil {
		bot.Errorf(
			"Error when removing song during play: %v", err,
		)

	}
	// NOTE: when loop option is set, the song should be pushed
	// to the back of the queue instead of removed
	if bot.builder.QueueHasOption(queue, model.Loop) {
		if err := bot.datastore.PersistSongs(
			s.State.User.ID,
			guildID,
			[]*model.Song{song},
		); err != nil {
			bot.Errorf(
				"Error when persisting song during play: %v", err,
			)
		}
	}

	// NOTE: update the queue after the song has been removed
	go bot.onUpdateQueueFromGuildID(s, guildID)

	// NOTE: audioplayer has successfully stopped streaming.
	// play the next song, if any
	select {
	case <-bot.ctx.Done():
		return
	default:
		delete(bot.audioplayers, guildID)
		bot.play(s, guildID, channelID)
	}
}
