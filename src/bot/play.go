package bot

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/bot/transaction"
	"discord-music-bot/model"
	"time"
)

type AudioplayerEventHandler struct {
	*Bot
}

// play searches for a queue that belongs to the provided guildID
// and starts playing it's headSong if no song is currently playing.
func (bot *Bot) play(t *transaction.Transaction, channelID string) {
	bot.log.WithField("GuildID", t.Interaction().GuildID).Trace(
		"Play requested ...",
	)

	// NOTE: make sure there is no audioplayer already active
	if _, ok := bot.audioplayers.Get(t.GuildID()); ok {
		// NOTE: audio has already been started from
		// another source, do not continue
		time.Sleep(300 * time.Millisecond)
		if _, ok := bot.audioplayers.Get(t.GuildID()); ok {
			// NOTE: should still update even when
			// returning as there must be a reason
			// for the play request
			bot.log.WithField("GuildID", t.Interaction().GuildID).Trace(
				"Audioplayer already exists",
			)
			t.UpdateQueue(100 * time.Millisecond)
			return
		}
	}

	bot.log.WithField("GuildID", t.Interaction().GuildID).Trace(
		"No existing audioplayer, creating one ...",
	)

	events := &AudioplayerEventHandler{bot}
	util := &Util{bot}

	// NOTE: try to join the voice channel, if client
	// is already in the one identified by the channelID,
	// it will not connect again
	if err := util.joinVoice(t, channelID); err != nil {
		t.UpdateQueue(100 * time.Millisecond)
		return
	}

	ap := audioplayer.NewAudioPlayer(bot.youtube)

	// NOTE: handle all external logic for audioplayer
	// with subscriptions. That way we can easily trigger
	// custom audioplayer events when clicking on buttons etc.
	events.handleSubscriptions(t, ap)

	bot.audioplayers.Add(t.GuildID(), ap)

	// NOTE: try to start playing, and then
	// update the queue, as it should be updated if
	// audioplayer started succesfully or if it failed.
	events.startPlayingSong(t, ap)

	t.UpdateQueue(100 * time.Millisecond)

}

func (bot *AudioplayerEventHandler) handleSubscriptions(t *transaction.Transaction, ap *audioplayer.AudioPlayer) {
	bot.log.WithField("GuildID", t.Interaction().GuildID).Trace(
		"Handling audioplayer subscriptions",
	)
	ap.Subscriptions().Subscribe("stop", func() {
		ap.Stop()
	})
	ap.Subscriptions().Subscribe("pause", func() {
		ap.Unpause()
		t.Refresh()
		t.UpdateQueue(100 * time.Millisecond)
	})
	ap.Subscriptions().Subscribe("unpause", func() {
		ap.Pause()
		t.Refresh()
		t.UpdateQueue(100 * time.Millisecond)
	})
	ap.Subscriptions().Subscribe("finished", func() {
		bot.handleHeadSongRemoval(t)
		bot.startPlayingSong(t, ap)
		t.Refresh()
		t.UpdateQueue(100 * time.Millisecond)
	})
	ap.Subscriptions().Subscribe("replay", func() {
		ap.Subscriptions().Emit("stop")
		bot.startPlayingSong(t, ap)
	})
	ap.Subscriptions().Subscribe("skip", func() {
		ap.Subscriptions().Emit("stop")
		ap.Subscriptions().Emit("finished")
	})
	ap.Subscriptions().Subscribe("skipToPrevious", func() {
		ap.Subscriptions().Emit("stop")
		bot.handleReverseHeadSongRemoval(t)
		bot.startPlayingSong(t, ap)
		t.Refresh()
		t.UpdateQueue(100 * time.Millisecond)
	})
	ap.Subscriptions().Subscribe("error", func() {
		bot.handleAudioplayerError(t.GuildID())
		bot.startPlayingSong(t, ap)
		t.Refresh()
		t.UpdateQueue(100 * time.Millisecond)
	})
	ap.Subscriptions().Subscribe("delete", func() {
		bot.audioplayers.Remove(t.GuildID())
	})
	ap.Subscriptions().Subscribe("terminate", func() {
		ap.Subscriptions().Emit("stop")
		ap.Subscriptions().Emit("delete")
	})
}

func (bot *AudioplayerEventHandler) startPlayingSong(t *transaction.Transaction, ap *audioplayer.AudioPlayer) {
	bot.log.WithField("GuildID", t.GuildID()).Trace("Try to start playing")

	song, err := bot.datastore.Song().GetHeadSongForQueue(
		bot.session.State.User.ID,
		t.GuildID(),
	)
	if err != nil {
		ap.Subscriptions().Emit("delete")
		bot.log.WithField("GuildID", t.GuildID()).Trace(
			"No head song found, cannot start playing",
		)
		return
	}
	voice, ok := bot.session.VoiceConnections[t.GuildID()]
	if ok == false || !voice.Ready {
		time.Sleep(300 * time.Second)
		voice, ok = bot.session.VoiceConnections[t.GuildID()]
		if ok == false || !voice.Ready {
			ap.Subscriptions().Emit("delete")
			bot.log.WithField("GuildID", t.GuildID()).Debug(
				"Failed to connect to voice",
			)
			return
		}
	}
	go func() {
		bot.log.WithField("GuildID", t.GuildID()).Tracef(
			"Playing song: %v", song.Name,
		)
		code, err := ap.Play(bot.ctx, song, voice)
		switch code {
		case 0:
			bot.log.WithField("GuildID", t.GuildID()).Tracef(
				"Audioplayer finished on itself",
			)
			ap.Subscriptions().Emit("finished")
			return
		case 1:
			bot.log.WithField("GuildID", t.GuildID()).Tracef(
				"Audioplayer finished with error: %v",
				err,
			)
			ap.Subscriptions().Emit("error")
			return
		default:
			bot.log.WithField("GuildID", t.Interaction().GuildID).Tracef(
				"Audioplayer exited with code: %d", code,
			)
		}
	}()
	return
}

func (bot *AudioplayerEventHandler) handleAudioplayerError(guildID string) {
	bot.log.WithField("GuildID", guildID).Trace(
		"Removing queue's head song",
	)
	bot.removeHeadSong(guildID)
}

func (bot *AudioplayerEventHandler) handleHeadSongRemoval(t *transaction.Transaction) {
	if bot.datastore.Queue().QueueHasOption(
		bot.session.State.User.ID,
		t.GuildID(),
		model.Loop,
	) {
		bot.log.WithField("GuildID", t.GuildID()).Trace(
			"Bot has loop enabled, pushing head song to back",
		)
		if err := bot.datastore.Song().PushHeadSongToBack(
			bot.session.State.User.ID,
			t.GuildID(),
		); err != nil {
			bot.log.WithField("GuildID", t.GuildID()).Errorf(
				"Error when pushing head song to back during play: %v",
				err,
			)
		}
		return
	}
	bot.log.WithField("GuildID", t.GuildID()).Trace(
		"Bot does not have loop enabled, removing head song",
	)
	headsong, err := bot.datastore.Song().GetHeadSongForQueue(
		bot.session.State.User.ID,
		t.GuildID(),
	)
	if err != nil {
		bot.log.WithField("GuildID", t.GuildID()).Errorf(
			"Error when fetching head song during play: %v",
			err,
		)
	} else {
		// NOTE: persist queue's headSong to inactive song table
		if err = bot.datastore.Song().PersistInactiveSongs(
			bot.session.State.User.ID,
			t.GuildID(),
			headsong,
		); err != nil {
			bot.log.WithField("GuildID", t.GuildID()).Errorf(
				"Error when persisting inactive song during play: %v",
				err,
			)

		}
		bot.removeHeadSong(t.GuildID())
	}
}

func (bot *AudioplayerEventHandler) handleReverseHeadSongRemoval(t *transaction.Transaction) {
	if bot.datastore.Queue().QueueHasOption(
		bot.session.State.User.ID,
		t.GuildID(),
		model.Loop,
	) {
		bot.log.WithField("GuildID", t.GuildID()).Trace(
			"Bot has loop enabled, pushing last song to front",
		)
		if err := bot.datastore.Song().PushLastSongToFront(
			bot.session.State.User.ID,
			t.GuildID(),
		); err != nil {
			bot.log.WithField("GuildID", t.GuildID()).Errorf(
				"Error when pushing last song to fron during play: %v",
				err,
			)
		}
		return
	}

	bot.log.WithField("GuildID", t.GuildID()).Trace(
		"Bot does not have loop enabled, popping latest inactive song",
	)
	song, err := bot.datastore.Song().PopLatestInactiveSong(
		bot.session.State.User.ID,
		t.GuildID(),
	)
	if err != nil {
		bot.log.WithField("GuildID", t.GuildID()).Errorf(
			"Error when popping latest inactive song during play: %v",
			err,
		)
		return
	}
	if err := bot.datastore.Song().PersistSongToFront(
		bot.session.State.User.ID,
		t.GuildID(),
		song,
	); err != nil {
		bot.log.WithField("GuildID", t.GuildID()).Errorf(
			"Error when persisting song to fron during play: %v",
			err,
		)
	}
}

func (bot *AudioplayerEventHandler) removeHeadSong(guildID string) {
	if err := bot.datastore.Song().RemoveHeadSong(
		bot.session.State.User.ID,
		guildID,
	); err != nil {
		bot.log.Errorf(
			"Error when removing song during play: %v", err,
		)
	}
}
