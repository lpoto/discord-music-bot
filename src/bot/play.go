package bot

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/model"
	"time"

	"github.com/bwmarrin/discordgo"
)

// play searches for a queue that belongs to the provided guildID
// and starts playing it's headSong if no song is currently playing.
func (bot *Bot) play(s *discordgo.Session, guildID string, channelID string) error {
	if len(channelID) == 0 {
		return nil
	}
	if _, ok := bot.audioplayers.Get(guildID); ok {
		// NOTE: audio has already been started from
		// another source, do not continue
		time.Sleep(300 * time.Millisecond)
		if _, ok := bot.audioplayers.Get(guildID); ok {
			return nil
		}
		return nil
	}

	bot.WithField("GuildID", guildID).Trace("Play request")

	ap := audioplayer.NewAudioPlayer(
		s, guildID,
		audioplayer.NewDeferFunctions(
			bot.audioplayerDefaultDefer,
			bot.audioplayerDefaultErrorDefer,
		),
	)

	bot.audioplayers.Add(guildID, ap)
	defer bot.audioplayers.Remove(guildID)

	queue, err := bot.datastore.GetQueue(s.State.User.ID, guildID)
	if err != nil {
		return err
	}
	if queue.HeadSong == nil {
		return nil
	}

	_, err = s.ChannelVoiceJoin(guildID, channelID, false, false)
	if err != nil {
		bot.Tracef("Could not join voice: %v", err)
		if buffer, ok := bot.queueUpdater.GetInteractionsBuffer(guildID); ok {
		responseLoop:
			for i := 0; i < len(buffer); i++ {
				select {
				case interaction := <-buffer:
					err := s.InteractionRespond(interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Cannot join the channel, I may be missing permissions",
							Flags:   1 << 6, //Ephemeral
						},
					})
					if err != nil {
						continue responseLoop
					}
					break responseLoop
				}
			}
		}
		return nil
	}

	// NOTE: always play the queue's headSong,
	// as it is the song with the minimum position
	// in the queue
	song := queue.HeadSong

	bot.WithField(
		"GuildID", guildID,
	).Tracef("Playing song: %s", song.Name)

	if err := ap.Play(bot.ctx, song); err != nil {
		if err.Error() == "Voice connection closed" {
			// NOTE: stop playing since bot has been
			// disconnected from the voice channel
			return nil
		}
		bot.Errorf("Error when playing: %v", err)
	}

	select {
	case <-bot.ctx.Done():
		return nil
	default:
		continuePlaying := ap.Continue
		bot.audioplayers.Remove(guildID)
		if continuePlaying {
			return bot.play(s, guildID, channelID)
		}
	}
	return nil
}

// audioplayerDefaultDefer is the default function called
// when the audioplayer finishes. This will only be called if
// no other functions were passed into the audioplayer's deferfuncbuffer
func (bot *Bot) audioplayerDefaultDefer(s *discordgo.Session, guildID string) {
	queue, err := bot.datastore.GetQueue(
		s.State.User.ID,
		guildID,
	)
	if err != nil {
		return
	}

	// NOTE: if loop is enabled in the queue
	// push it's headSong to the back of the queue
	// else just
	if bot.builder.QueueHasOption(queue, model.Loop) {
		if err := bot.datastore.PushHeadSongToBack(
			s.State.User.ID,
			guildID,
		); err != nil {
			bot.Errorf(
				"Error when pushing first song to back during play: %v",
				err,
			)
		}
	} else {
		// NOTE: persist queue's headSong to inactive song table
		if err := bot.datastore.PersistInactiveSongs(
			s.State.User.ID,
			guildID,
			queue.HeadSong,
		); err != nil {
			bot.Errorf(
				"Error when removing song during play: %v", err,
			)
		}
		if err := bot.datastore.RemoveHeadSong(
			// NOTE: the finished song should be removed
			// from the queue
			s.State.User.ID,
			guildID,
		); err != nil {
			bot.Errorf(
				"Error when removing song during play: %v", err,
			)
		}
	}

	bot.queueUpdater.NeedsUpdate(guildID)
	bot.queueUpdater.Update(s, guildID)
}

// audioplayerDefaultErrorDefer is the default function called
// when the audioplayer finishes with error.
func (bot *Bot) audioplayerDefaultErrorDefer(s *discordgo.Session, guildID string) {
	// NOTE: if error occured when playing the head song,
	// it should be removed

	if err := bot.datastore.RemoveHeadSong(
		// NOTE: the finished song should be removed
		// from the queue
		s.State.User.ID,
		guildID,
	); err != nil {
		bot.Errorf(
			"Error when removing song during play: %v", err,
		)
	}
	bot.queueUpdater.NeedsUpdate(guildID)
	bot.queueUpdater.Update(s, guildID)
}
