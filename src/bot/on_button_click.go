package bot

import (
	"discord-music-bot/model"
	"time"

	"github.com/bwmarrin/discordgo"
)

// onButtonClick is a handler function called when a user
// clicks a button on a message owned by the bot.
// This is not emitted through the discord websocket, but is rather
// called from the INTERACTION_CREATE event when the interaction type
// is button click and the message author is bot
func (bot *Bot) onButtonClick(i *discordgo.InteractionCreate) {
	label := bot.builder.Queue().GetButtonLabelFromComponentData(i.MessageComponentData())
	bot.WithField("GuildID", i.GuildID).Tracef("Button clicked (%s)", label)

	switch label {
	case bot.builder.Queue().ButtonsConfig().AddSongs:
		bot.addSongs(i)
		return
	case bot.builder.Queue().ButtonsConfig().Backward:
		bot.backwardButtonClick(i)
		return
	case bot.builder.Queue().ButtonsConfig().Forward:
		bot.forwardButtonClick(i)
		return
	case bot.builder.Queue().ButtonsConfig().Loop:
		bot.loopButtonClick(i)
		return
	case bot.builder.Queue().ButtonsConfig().Pause:
		bot.pauseButtonClick(i)
		return
	case bot.builder.Queue().ButtonsConfig().Skip:
		bot.skipButtonClick(i)
		return
	case bot.builder.Queue().ButtonsConfig().Previous:
		bot.previousButtonClick(i)
		return
	case bot.builder.Queue().ButtonsConfig().Replay:
		bot.replayButtonClick(i)
		return
	case bot.builder.Queue().ButtonsConfig().Join:
		bot.joinButtonClick(i)
		return
	default:
		bot.session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
func (bot *Bot) forwardButtonClick(i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(bot.session, i.Interaction)

	queue, _ := bot.datastore.Queue().GetQueue(
		bot.session.State.User.ID,
		i.GuildID,
	)
	queue, _ = bot.datastore.Song().UpdateQueueWithSongs(queue)

	bot.service.Queue().IncrementQueueOffset(queue)
	if err := bot.datastore.Queue().UpdateQueue(queue); err != nil {
		bot.Errorf("Error on forward button click: %v", err)
		return
	}
	bot.queueUpdater.Update(
		bot.session,
		i.GuildID,
		100*time.Millisecond,
		nil,
	)
}

// backwardButtonClick decrements the queue's offset, updates it
// and then updates the queue message
func (bot *Bot) backwardButtonClick(i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(bot.session, i.Interaction)

	queue, _ := bot.datastore.Queue().GetQueue(
		bot.session.State.User.ID,
		i.GuildID,
	)
	queue, _ = bot.datastore.Song().UpdateQueueWithSongs(queue)

	bot.service.Queue().DecrementQueueOffset(queue)
	if err := bot.datastore.Queue().UpdateQueue(queue); err != nil {
		bot.Errorf("Error on backward button click: %v", err)
		return
	}
	bot.queueUpdater.Update(
		bot.session,
		i.GuildID,
		100*time.Millisecond,
		nil,
	)
}

// pauseButtonClick adds or removes the queue's Paused option, updates it
// and then updates the queue message
func (bot *Bot) pauseButtonClick(i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(bot.session, i.Interaction)

	if bot.blockedCommands.IsBlocked(i.GuildID, "PAUSE") {
		return
	}
	bot.blockedCommands.Block(i.GuildID, "PAUSE")
	defer bot.blockedCommands.Unblock(i.GuildID, "PAUSE")

	time.Sleep(300 * time.Millisecond)

	if bot.datastore.Queue().QueueHasOption(
		bot.session.State.User.ID,
		i.GuildID,
		model.Paused,
	) {
		bot.datastore.Queue().RemoveQueueOptions(
			bot.session.State.User.ID,
			i.GuildID,
			model.Paused,
		)
		if ap, ok := bot.audioplayers.Get(i.GuildID); ok {
			ap.Unpause()
		}
	} else {
		bot.datastore.Queue().PersistQueueOptions(
			bot.session.State.User.ID,
			i.GuildID,
			model.PausedOption(),
		)
		if ap, ok := bot.audioplayers.Get(i.GuildID); ok {
			ap.Pause()
		}

	}
	bot.queueUpdater.Update(
		bot.session,
		i.GuildID,
		100*time.Millisecond,
		nil,
	)
}

// loopButtonClick adds or removes the queue's Loop option, updates it
// and then updates the queue message
func (bot *Bot) loopButtonClick(i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(bot.session, i.Interaction)
	if bot.blockedCommands.IsBlocked(i.GuildID, "LOOP") {
		return
	}
	bot.blockedCommands.Block(i.GuildID, "LOOP")
	defer bot.blockedCommands.Unblock(i.GuildID, "LOOP")

	time.Sleep(300 * time.Millisecond)

	if bot.datastore.Queue().QueueHasOption(
		bot.session.State.User.ID,
		i.GuildID,
		model.Loop,
	) {
		bot.datastore.Queue().RemoveQueueOptions(
			bot.session.State.User.ID,
			i.GuildID,
			model.Loop,
		)
	} else {
		bot.datastore.Queue().PersistQueueOptions(
			bot.session.State.User.ID,
			i.GuildID,
			model.LoopOption(),
		)
	}
	bot.queueUpdater.Update(
		bot.session,
		i.GuildID,
		100*time.Millisecond,
		nil,
	)
}

// skipButtonClick skips the currently playing song if any
func (bot *Bot) skipButtonClick(i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(bot.session, i.Interaction)

	if bot.blockedCommands.IsBlocked(i.GuildID, "SKIP") {
		return
	}
	bot.blockedCommands.Block(i.GuildID, "SKIP")
	defer bot.blockedCommands.Unblock(i.GuildID, "SKIP")

	time.Sleep(750 * time.Millisecond)

	if ap, ok := bot.audioplayers.Get(i.GuildID); ok && !ap.IsPaused() {
		ap.StopOK()
	}
}

// replayButtonClick adds a different defer func to the audioplayer, that does not remove,
// the queue's current headSong, and restarts the audioplayer.
func (bot *Bot) replayButtonClick(i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(bot.session, i.Interaction)

	if bot.blockedCommands.IsBlocked(i.GuildID, "REPLAY") {
		return
	}
	bot.blockedCommands.Block(i.GuildID, "REPLAY")
	defer bot.blockedCommands.Unblock(i.GuildID, "REPLAY")

	ap, ok := bot.audioplayers.Get(i.GuildID)
	if ok && ap.IsPaused() {
		return
	}
	done := make(chan struct{}, 2)
	if ap != nil {
		ap.StopTerminate()
	}
	select {
	case done <- struct{}{}:
		bot.queueUpdater.Update(
			bot.session,
			i.GuildID,
			800*time.Millisecond,
			nil,
		)
	default:
	}
	t := time.Now()
	for {
		select {
		case <-done:
			return
		default:
			if time.Since(t) >= time.Second {
				return
			}
		}
	}
}

// previousButtonClick adds a different defer func to the audioplayer, that adds
// the queue's previous  song as its head song, and restarts the player.
func (bot *Bot) previousButtonClick(i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(bot.session, i.Interaction)

	if bot.blockedCommands.IsBlocked(i.GuildID, "PREVIOUS") {
		return
	}
	bot.blockedCommands.Block(i.GuildID, "PREVIOUS")
	defer bot.blockedCommands.Unblock(i.GuildID, "PREVIOUS")

	ap, ok := bot.audioplayers.Get(i.GuildID)
	if ok && ap.IsPaused() {
		return
	}
	queue, err := bot.datastore.Queue().GetQueue(
		bot.session.State.User.ID,
		i.GuildID,
	)
	if err != nil {
		bot.Errorf("Error on previous button click: %v", err)
		return
	}
	queue, err = bot.datastore.Song().UpdateQueueWithSongs(queue)
	if err != nil {
		bot.Errorf("Error on previous button click: %v", err)
		return
	}
	if queue.InactiveSize == 0 && !(queue.Size > 1 &&
		bot.datastore.Queue().QueueHasOption(
			bot.session.State.User.ID,
			i.GuildID,
			model.Loop,
		)) {

		return
	}
	done := make(chan struct{}, 2)
	// NOTE: when audioplayer finishes, add previous song as the headSong
	// and update the queue.
	// If loop is enabled, the previous song is last song in the queue,
	// else it is the last removed song
	if ap != nil {
		ap.StopTerminate()
	}
	if bot.datastore.Queue().QueueHasOption(
		bot.session.State.User.ID,
		i.GuildID,
		model.Loop,
	) {
		bot.datastore.Song().PushLastSongToFront(
			bot.session.State.User.ID,
			i.GuildID,
		)
	} else {
		song, err := bot.datastore.Song().PopLatestInactiveSong(
			bot.session.State.User.ID, i.GuildID,
		)
		if err != nil {
			bot.Errorf("Error on previous song button click: %v", err)
			bot.queueUpdater.Update(
				bot.session,
				i.GuildID,
				500*time.Millisecond,
				nil,
			)
			return
		}
		if err := bot.datastore.Song().PersistSongToFront(
			bot.session.State.User.ID, i.GuildID, song,
		); err != nil {
			bot.Errorf("Error on previous song button click: %v", err)
		}
	}

	select {
	case done <- struct{}{}:
		bot.queueUpdater.Update(
			bot.session,
			i.GuildID,
			800*time.Millisecond,
			nil,
		)
	default:
	}

	t := time.Now()
	for {
		select {
		case <-done:
			return
		default:
			if time.Since(t) >= time.Second {
				return
			}
		}
	}
}

// joinButtonClick removes the inactive option from the queue, updates
// it and then updates the queue message
func (bot *Bot) joinButtonClick(i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(bot.session, i.Interaction)

	go func() {
		bot.queueUpdater.Update(
			bot.session,
			i.GuildID,
			1*time.Second,
			nil,
		)
	}()
}
