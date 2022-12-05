package bot

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/bot/modal"
	"discord-music-bot/bot/transaction"
	"discord-music-bot/model"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

type ButtonClickHandler struct {
	*Bot
}

// onButtonClick is a handler function called when a user
// clicks a button on a message owned by the bot.
// This is not emitted through the discord websocket, but is rather
// called from the INTERACTIONCREATE event when the interaction type
// is button click and the message author is bot
func (bot *DiscordEventHandler) onButtonClick(t *transaction.Transaction) {
	label := bot.builder.Queue().GetButtonLabelFromComponentData(
		t.Interaction().MessageComponentData(),
	)
	bot.log.WithField("GuildID", t.Interaction().GuildID).Tracef(
		"Button clicked (%s)", label,
	)

	button := &ButtonClickHandler{bot.Bot}

	channelID := ""
	if userState, _ := bot.session.State.VoiceState(
		t.GuildID(),
		t.Interaction().Member.User.ID,
	); userState != nil {
		channelID = userState.ChannelID
	}

	switch label {
	case bot.builder.Queue().ButtonsConfig().AddSongs:
		button.addSongsButtonClick(t)
		return
	case bot.builder.Queue().ButtonsConfig().Backward:
		button.backwardButtonClick(t)
		return
	case bot.builder.Queue().ButtonsConfig().Forward:
		button.forwardButtonClick(t)
		return
	case bot.builder.Queue().ButtonsConfig().Loop:
		button.loopButtonClick(t)
		return
	case bot.builder.Queue().ButtonsConfig().Pause:
		button.pauseButtonClick(t)
		return
	case bot.builder.Queue().ButtonsConfig().Skip:
		button.skipButtonClick(t, channelID)
		return
	case bot.builder.Queue().ButtonsConfig().Previous:
		button.previousButtonClick(t, channelID)
		return
	case bot.builder.Queue().ButtonsConfig().Replay:
		button.replayButtonClick(t, channelID)
		return
	case bot.builder.Queue().ButtonsConfig().Join:
		button.joinButtonClick(t, channelID)
		return
	default:
		bot.session.InteractionRespond(t.Interaction(),
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Sorry, something went wrong ...",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
	}
}

// forwardButtonClick increments the queue's offset, updates it
// and then updates the queue message
func (bot *ButtonClickHandler) forwardButtonClick(t *transaction.Transaction) {
	queue, _ := bot.datastore.Queue().GetQueue(
		bot.session.State.User.ID,
		t.GuildID(),
	)
	queue, _ = bot.datastore.Song().UpdateQueueWithSongs(queue)

	bot.service.Queue().IncrementQueueOffset(queue)
	if err := bot.datastore.Queue().UpdateQueue(queue); err != nil {
		bot.log.Errorf("log.Error on forward button click: %v", err)
		return
	}
	t.UpdateQueue(100 * time.Millisecond)
}

// backwardButtonClick decrements the queue's offset, updates it
// and then updates the queue message
func (bot *ButtonClickHandler) backwardButtonClick(t *transaction.Transaction) {
	queue, _ := bot.datastore.Queue().GetQueue(
		bot.session.State.User.ID,
		t.GuildID(),
	)
	queue, _ = bot.datastore.Song().UpdateQueueWithSongs(queue)

	bot.service.Queue().DecrementQueueOffset(queue)
	if err := bot.datastore.Queue().UpdateQueue(queue); err != nil {
		bot.log.Errorf("log.Error on backward button click: %v", err)
		return
	}
	t.UpdateQueue(100 * time.Millisecond)
}

// pauseButtonClick adds or removes the queue's Paused option, updates it
// and then updates the queue message
func (bot *ButtonClickHandler) pauseButtonClick(t *transaction.Transaction) {
	bot.blockAndGetAudioplayer("PAUSE", t.GuildID(), func(ap *audioplayer.AudioPlayer) {
		time.Sleep(300 * time.Millisecond)

		if bot.datastore.Queue().QueueHasOption(
			bot.session.State.User.ID,
			t.GuildID(),
			model.Paused,
		) {
			bot.datastore.Queue().RemoveQueueOptions(
				bot.session.State.User.ID,
				t.GuildID(),
				model.Paused,
			)
			if ap, ok := bot.audioplayers.Get(t.GuildID()); ok {
				ap.Subscriptions().Emit("unpause")
			}
		} else {
			bot.datastore.Queue().PersistQueueOptions(
				bot.session.State.User.ID,
				t.GuildID(),
				model.PausedOption(),
			)
			if ap, ok := bot.audioplayers.Get(t.GuildID()); ok {
				ap.Subscriptions().Emit("pause")
			}

		}
		t.UpdateQueue(100 * time.Millisecond)
	})
}

// loopButtonClick adds or removes the queue's Loop option, updates it
// and then updates the queue message
func (bot *ButtonClickHandler) loopButtonClick(t *transaction.Transaction) {
	bot.blockAndGetAudioplayer("LOOP", t.GuildID(), func(ap *audioplayer.AudioPlayer) {
		time.Sleep(300 * time.Millisecond)

		if bot.datastore.Queue().QueueHasOption(
			bot.session.State.User.ID,
			t.GuildID(),
			model.Loop,
		) {
			bot.datastore.Queue().RemoveQueueOptions(
				bot.session.State.User.ID,
				t.GuildID(),
				model.Loop,
			)
		} else {
			bot.datastore.Queue().PersistQueueOptions(
				bot.session.State.User.ID,
				t.GuildID(),
				model.LoopOption(),
			)
		}
		t.UpdateQueue(100 * time.Millisecond)
	})
}

// skipButtonClick skips the currently playing song if any
func (bot *ButtonClickHandler) skipButtonClick(t *transaction.Transaction, channelID string) {
	bot.blockAndGetAudioplayer("SKIP", t.GuildID(), func(ap *audioplayer.AudioPlayer) {
		if ap == nil {
			bot.play(t, channelID)

		} else if ap.IsPaused() {
			return
		}
		time.Sleep(700 * time.Millisecond)
		if ap != nil {
			ap.Subscriptions().Emit("stop")
		}
	})
}

// replayButtonClick adds a different defer func to the audioplayer, that does not remove,
// the queue's current headSong, and restarts the audioplayer.
func (bot *ButtonClickHandler) replayButtonClick(t *transaction.Transaction, channelID string) {
	bot.blockAndGetAudioplayer("REPLAY", t.GuildID(), func(ap *audioplayer.AudioPlayer) {
		if ap == nil {
			bot.play(t, channelID)
			return
		} else if ap.IsPaused() {
			return
		}
		time.Sleep(700 * time.Millisecond)
		if ap != nil {
			ap.Subscriptions().Emit("replay")
		}
	})
}

// previousButtonClick adds a different defer func to the audioplayer, that adds
// the queue's previous  song as its head song, and restarts the player.
func (bot *ButtonClickHandler) previousButtonClick(t *transaction.Transaction, channelID string) {
	bot.blockAndGetAudioplayer("PREVIOUS", t.GuildID(), func(ap *audioplayer.AudioPlayer) {
		if ap != nil && ap.IsPaused() {
			return
		}

		queue, err := bot.datastore.Queue().GetQueue(
			bot.session.State.User.ID,
			t.GuildID(),
		)
		if err != nil {
			bot.log.Errorf("log.Error on previous button click: %v", err)
			return
		}
		queue, err = bot.datastore.Song().UpdateQueueWithSongs(queue)
		if err != nil {
			bot.log.Errorf("log.Error on previous button click: %v", err)
			return
		}
		if queue.InactiveSize == 0 && !(queue.Size > 1 &&
			bot.datastore.Queue().QueueHasOption(
				bot.session.State.User.ID,
				t.GuildID(),
				model.Loop,
			)) {

			return
		}
		// NOTE: when audioplayer finishes, add previous song as the headSong
		// and update the queue.
		// If loop is enabled, the previous song is last song in the queue,
		// else it is the last removed song
		if bot.datastore.Queue().QueueHasOption(
			bot.session.State.User.ID,
			t.GuildID(),
			model.Loop,
		) {
			bot.datastore.Song().PushLastSongToFront(
				bot.session.State.User.ID,
				t.GuildID(),
			)
		} else {
			song, err := bot.datastore.Song().PopLatestInactiveSong(
				bot.session.State.User.ID,
				t.GuildID(),
			)
			if err != nil {
				bot.log.Errorf("log.Error on previous song button click: %v", err)
				t.UpdateQueue(500 * time.Millisecond)
				return
			}
			if err := bot.datastore.Song().PersistSongToFront(
				bot.session.State.User.ID,
				t.GuildID(),
				song,
			); err != nil {
				bot.log.Errorf("log.Error on previous song button click: %v", err)
			}
		}

		if ap == nil {
			bot.play(t, channelID)
			return
		}
		time.Sleep(700 * time.Millisecond)
		ap.Subscriptions().Emit("skipToPrevious")
	})
}

// joinButtonClick removes the inactive option from the queue, updates
// it and then updates the queue message
func (bot *ButtonClickHandler) joinButtonClick(t *transaction.Transaction, channelID string) {
	bot.play(t, channelID)
}

// addSongs responds to the provided interaction with the
// add songs modal.
func (bot *ButtonClickHandler) addSongsButtonClick(t *transaction.Transaction) error {
	bot.log.WithField("GuildID", t.Interaction().GuildID).Trace(
		"Send add songs modal",
	)
	t.Defer()

	textInput := discordgo.TextInput{
		CustomID:    uuid.NewString(),
		Label:       bot.config.Modals.AddSongs.Label,
		Placeholder: bot.config.Modals.AddSongs.Placeholder,
		Style:       discordgo.TextInputParagraph,
		MinLength:   1,
		MaxLength:   4000,
		Required:    true,
	}

	m := modal.GetModal(
		bot.config.Modals.AddSongs.Name,
		[]discordgo.MessageComponent{textInput},
	)
	if err := bot.session.InteractionRespond(
		t.Interaction(),
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Components: m.Components,
				CustomID:   m.CustomID,
				Title:      bot.config.Modals.AddSongs.Name,
			},
		},
	); err != nil {
		bot.log.Errorf(
			"log.Error when responding with add songs modal: %v",
			err,
		)
		return err
	}
	return nil
}

func (bot *ButtonClickHandler) blockAndGetAudioplayer(blockKey string, guildID string, f func(*audioplayer.AudioPlayer)) {
	if bot.blockedCommands.IsBlocked(guildID, blockKey) {
		return
	}
	bot.blockedCommands.Block(guildID, blockKey)
	defer bot.blockedCommands.Unblock(guildID, blockKey)

	ap, ok := bot.audioplayers.Get(guildID)
	if ok && ap.IsPaused() {
		return
	}
	f(ap)
}
