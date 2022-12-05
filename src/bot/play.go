package bot

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/bot/transaction"
	"discord-music-bot/model"
	"errors"
	"time"

	log "github.com/sirupsen/logrus"
)

type AudioplayerEventHandler struct {
	*Bot
}

// play searches for a queue that belongs to the provided guildID
// and starts playing it's headSong if no song is currently playing.
func (bot *Bot) play(t *transaction.Transaction, channelID string) {
	if _, ok := bot.audioplayers.Get(t.GuildID()); ok {
		// NOTE: audio has already been started from
		// another source, do not continue
		time.Sleep(300 * time.Millisecond)
		if _, ok := bot.audioplayers.Get(t.GuildID()); ok {
			// NOTE: should still update even when
			// returning as there must be a reason
			// for the play request
			t.UpdateQueue(100 * time.Millisecond)
			return
		}
	}
	events := &AudioplayerEventHandler{bot}

	if err := events.joinVoice(t, channelID); err != nil {
		t.UpdateQueue(100 * time.Millisecond)
		return
	}

	ap := audioplayer.NewAudioPlayer(bot.youtube)

	ap.Subscriptions().Subscribe("stop", func() {
		events.audioplayerOnStop(t)
	})
	ap.Subscriptions().Subscribe("replay", func() {
		ap.Subscriptions().Emit("stop")
	})
	ap.Subscriptions().Subscribe("error", func() {
		events.audioplayerOnError(t.GuildID())
	})
	ap.Subscriptions().Subscribe("skip", func() {
		ap.Subscriptions().Emit("stop")
	})

	bot.audioplayers.Add(t.GuildID(), ap)

	events.startPlayingSong(t, ap)
	t.UpdateQueue(100 * time.Millisecond)

}

func (bot *AudioplayerEventHandler) startPlayingSong(t *transaction.Transaction, ap *audioplayer.AudioPlayer) error {
	song, err := bot.datastore.Song().GetHeadSongForQueue(
		bot.session.State.User.ID,
		t.GuildID(),
	)
	if err != nil {
		return err
	}
	voice, ok := bot.session.VoiceConnections[t.GuildID()]
	if ok == false || !voice.Ready {
		time.Sleep(300 * time.Second)
		voice, ok = bot.session.VoiceConnections[t.GuildID()]
		if ok == false || !voice.Ready {
			return errors.New("Failed to connect to voide")
		}
	}
	go ap.Play(bot.ctx, song, voice)
	return nil
}

// audioplayerOnStop is a function called when the audioplayer emits
// stop event
func (bot *AudioplayerEventHandler) audioplayerOnStop(t *transaction.Transaction) {
	defer func() {
		t.Refresh()
		t.UpdateQueue(100 * time.Millisecond)
	}()

	// NOTE: if loop is enabled in the queue
	// push it's headSong to the back of the queue
	// else just
	if bot.datastore.Queue().QueueHasOption(
		bot.session.State.User.ID,
		t.GuildID(),
		model.Loop,
	) {
		bot.log.WithField("GuildID", t.GuildID()).Info(
			"Bot has loop enabled, pushing head song to back",
		)
		if err := bot.datastore.Song().PushHeadSongToBack(
			bot.session.State.User.ID,
			t.GuildID(),
		); err != nil {
			bot.log.Errorf(
				"Error when pushing first song to back during play: %v",
				err,
			)
		}
	} else {
		bot.log.WithField("GuildID", t.GuildID()).Info(
			"Bot does not have loop enabled, removing head song",
		)
		headsong, err := bot.datastore.Song().GetHeadSongForQueue(
			bot.session.State.User.ID,
			t.GuildID(),
		)
		if err != nil {
			bot.log.Errorf(
				"Error when fetching song during play: %v", err,
			)
			return
		}
		// NOTE: persist queue's headSong to inactive song table
		if err := bot.datastore.Song().PersistInactiveSongs(
			bot.session.State.User.ID,
			t.GuildID(),
			headsong,
		); err != nil {
			bot.log.Info(
				"Error when removing song during play: %v", err,
			)
		}
		if err := bot.datastore.Song().RemoveHeadSong(
			// NOTE: the finished song should be removed
			// from the queue
			bot.session.State.User.ID,
			t.GuildID(),
		); err != nil {
			bot.log.Errorf(
				"Error when removing song during play: %v", err,
			)
		}
	}
}

// audioplayerOnError is the a function called
// when the audioplayer finishes with error.
func (bot *AudioplayerEventHandler) audioplayerOnError(guildID string) {
	// NOTE: if error occured when playing the head song,
	// it should be removed

	if err := bot.datastore.Song().RemoveHeadSong(
		// NOTE: the finished song should be removed
		// from the queue
		bot.session.State.User.ID,
		guildID,
	); err != nil {
		bot.log.Errorf(
			"Error when removing song during play: %v", err,
		)
	}
}

// joinVoice connects to the voice channel identified by the provided guilID and
// channelID, returns error on failure. If the client is already connected to the
// voice channel, it does not connect again.
func (bot *AudioplayerEventHandler) joinVoice(t *transaction.Transaction, channelID string) error {
	bot.log.WithFields(log.Fields{
		"GuildID":   t.GuildID(),
		"ChannelID": channelID,
	}).Trace("Joining voice")

	vc, ok := bot.session.VoiceConnections[t.GuildID()]
	if ok && vc.ChannelID == channelID {
		return nil
	}
	if _, err := bot.session.ChannelVoiceJoin(
		t.GuildID(),
		channelID,
		false,
		false,
	); err != nil {
		bot.log.Debugf("Could not join voice: %v", err)
		return err
	}
	return nil
}
