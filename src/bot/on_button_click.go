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
func (bot *Bot) onButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	label := bot.builder.GetButtonLabelFromComponentData(i.MessageComponentData())
	bot.WithField("GuildID", i.GuildID).Tracef("Button clicked (%s)", label)

	switch label {
	case bot.builder.Config.Components.AddSongs:
		bot.onAddSongsCommand(s, i)
		return
	case bot.builder.Config.Components.Backward:
		bot.backwardButtonClick(s, i)
		return
	case bot.builder.Config.Components.Forward:
		bot.forwardButtonClick(s, i)
		return
	case bot.builder.Config.Components.Loop:
		bot.loopButtonClick(s, i)
		return
	case bot.builder.Config.Components.Pause:
		bot.pauseButtonClick(s, i)
		return
	case bot.builder.Config.Components.Skip:
		bot.skipButtonClick(s, i)
		return
	case bot.builder.Config.Components.Previous:
		bot.previousButtonClick(s, i)
		return
	case bot.builder.Config.Components.Replay:
		bot.replayButtonClick(s, i)
		return
	case bot.builder.Config.Components.Join:
		bot.joinButtonClick(s, i)
		return
	default:
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
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
func (bot *Bot) forwardButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(s, i.Interaction)

	queue, _ := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	bot.service.IncrementQueueOffset(queue)
	if err := bot.datastore.UpdateQueue(queue); err != nil {
		bot.Errorf("Error on forward button click: %v", err)
		return
	}
	bot.queueUpdater.NeedsUpdate(i.GuildID)
	bot.queueUpdater.Update(s, i.GuildID)
}

// backwardButtonClick decrements the queue's offset, updates it
// and then updates the queue message
func (bot *Bot) backwardButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(s, i.Interaction)

	queue, _ := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	bot.service.DecrementQueueOffset(queue)
	if err := bot.datastore.UpdateQueue(queue); err != nil {
		bot.Errorf("Error on backward button click: %v", err)
		return
	}
	bot.queueUpdater.NeedsUpdate(i.GuildID)
	bot.queueUpdater.Update(s, i.GuildID)
}

// pauseButtonClick adds or removes the queue's Paused option, updates it
// and then updates the queue message
func (bot *Bot) pauseButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(s, i.Interaction)

	if _, ok := bot.blockedButtons[i.GuildID]["PAUSE"]; ok {
		return
	}
	defer func() {
		if m, ok := bot.blockedButtons[i.GuildID]; ok {
			delete(m, "PAUSE")
		}
	}()
	time.Sleep(300 * time.Millisecond)

	queue, _ := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	if bot.builder.QueueHasOption(queue, model.Paused) {
		bot.datastore.RemoveQueueOptions(
			queue.ClientID,
			queue.GuildID,
			model.Paused,
		)
		if ap, ok := bot.audioplayers.Get(i.GuildID); ok {
			ap.Unpause()
		}
	} else {
		bot.datastore.PersistQueueOptions(
			queue.ClientID,
			queue.GuildID,
			model.PausedOption(),
		)
		if ap, ok := bot.audioplayers.Get(i.GuildID); ok {
			ap.Pause()
		}

	}
	bot.queueUpdater.NeedsUpdate(i.GuildID)
	bot.queueUpdater.Update(s, i.GuildID)
}

// loopButtonClick adds or removes the queue's Loop option, updates it
// and then updates the queue message
func (bot *Bot) loopButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(s, i.Interaction)
	if _, ok := bot.blockedButtons[i.GuildID]["LOOP"]; ok {
		return
	}
	defer func() {
		if m, ok := bot.blockedButtons[i.GuildID]; ok {
			delete(m, "LOOP")
		}
	}()
	time.Sleep(300 * time.Millisecond)

	queue, _ := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	if bot.builder.QueueHasOption(queue, model.Loop) {
		bot.datastore.RemoveQueueOptions(
			queue.ClientID,
			queue.GuildID,
			model.Loop,
		)
	} else {
		bot.datastore.PersistQueueOptions(
			queue.ClientID,
			queue.GuildID,
			model.LoopOption(),
		)
	}
	bot.queueUpdater.NeedsUpdate(i.GuildID)
	bot.queueUpdater.Update(s, i.GuildID)
}

// skipButtonClick skips the currently playing song if any
func (bot *Bot) skipButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(s, i.Interaction)

	if _, ok := bot.blockedButtons[i.GuildID]["SKIP"]; ok {
		// NOTE: someone already pressed the sip button
		// in the server.
		// Don't allow multiple people clicking at the same
		// time, to prevent skipping multiple songs at once
		return
	}
	defer func() {
		if m, ok := bot.blockedButtons[i.GuildID]; ok {
			delete(m, "SKIP")
		}
	}()

	time.Sleep(750 * time.Millisecond)

	if ap, ok := bot.audioplayers.Get(i.GuildID); ok && !ap.IsPaused() {
		ap.Stop()
	}
}

// replayButtonClick adds a different defer func to the audioplayer, that does not remove,
// the queue's current headSong, and restarts the audioplayer.
func (bot *Bot) replayButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(s, i.Interaction)

	if _, ok := bot.blockedButtons[i.GuildID]["REPLAY"]; ok {
		return
	}
	defer func() {
		if m, ok := bot.blockedButtons[i.GuildID]; ok {
			delete(m, "REPLAY")
		}
	}()

	ap, ok := bot.audioplayers.Get(i.GuildID)
	if ok && ap.IsPaused() {
		return
	}
	done := make(chan struct{}, 2)
	f := func(s *discordgo.Session, guildID string) {
		select {
		case done <- struct{}{}:
			time.Sleep(800 * time.Millisecond)
		default:
		}
		bot.queueUpdater.NeedsUpdate(i.GuildID)
		bot.queueUpdater.Update(s, guildID)
	}
	if ap == nil {
		f(s, i.GuildID)
	} else {
		ap.AddDeferFunc(f)
		ap.Stop()
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
func (bot *Bot) previousButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(s, i.Interaction)

	if _, ok := bot.blockedButtons[i.GuildID]["PREVIOUS"]; ok {
		return
	}
	defer func() {
		if m, ok := bot.blockedButtons[i.GuildID]; ok {
			delete(m, "PREVIOUS")
		}
	}()

	ap, ok := bot.audioplayers.Get(i.GuildID)
	if ok && ap.IsPaused() {
		return
	}
	queue, err := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	if err != nil {
		bot.Errorf("Error on previous button click: %v", err)
		return
	}
	if queue.InactiveSize == 0 && !(queue.Size > 1 && bot.builder.QueueHasOption(queue, model.Loop)) {
		return
	}
	done := make(chan struct{}, 2)
	// NOTE: when audioplayer finishes, add previous song as the headSong
	// and update the queue.
	// If loop is enabled, the previous song is last song in the queue,
	// else it is the last removed song
	f := func(s *discordgo.Session, guildID string) {

		if bot.builder.QueueHasOption(queue, model.Loop) {
			bot.datastore.PushLastSongToFront(s.State.User.ID, guildID)
		} else {
			song, err := bot.datastore.PopLatestInactiveSong(
				s.State.User.ID, guildID,
			)
			if err != nil {
				bot.Errorf("Error on previous song button click: %v", err)
				bot.queueUpdater.NeedsUpdate(i.GuildID)
				bot.queueUpdater.Update(s, guildID)
				return
			}
			if err := bot.datastore.PersistSongToFront(
				s.State.User.ID, guildID, song,
			); err != nil {
				bot.Errorf("Error on previous song button click: %v", err)
			}
		}
		select {
		case done <- struct{}{}:
			time.Sleep(800 * time.Millisecond)
		default:
		}
		bot.queueUpdater.NeedsUpdate(i.GuildID)
		bot.queueUpdater.Update(s, guildID)
	}
	if ap == nil {
		f(s, i.GuildID)
	} else {
		ap.AddDeferFunc(f)
		ap.Stop()
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
func (bot *Bot) joinButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.queueUpdater.AddInteraction(s, i.Interaction)

	if err := bot.datastore.RemoveQueueOptions(
		s.State.User.ID,
		i.GuildID,
		model.Inactive,
	); err != nil {
		bot.Errorf("Error on join button click: %v", err)
		return
	}
	bot.queueUpdater.NeedsUpdate(i.GuildID)
	bot.queueUpdater.Update(s, i.GuildID)
}
