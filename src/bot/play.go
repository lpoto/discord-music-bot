package bot

import (
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/model"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// play searches for a queue that belongs to the provided guildID
// and starts playing it's headSong if no song is currently playing.
func (bot *Bot) play(guildID string, channelID string) error {
	if len(channelID) == 0 {
		return nil
	}

	if _, ok := bot.audioplayers.Get(guildID); ok {
		// NOTE: audio has already been started from
		// another source, do not continue
		time.Sleep(300 * time.Millisecond)
		if _, ok := bot.audioplayers.Get(guildID); ok {
			// NOTE: should still update even when
			// returning as there must be a reason
			// for the play request
			bot.queueUpdater.Update(
				bot.session,
				guildID,
				100*time.Millisecond,
				nil,
			)
			return nil
		}
	}

	bot.WithField("GuildID", guildID).Trace("Play request")

	ap := audioplayer.NewAudioPlayer(bot.youtube)

	bot.audioplayers.Add(guildID, ap)
	defer bot.audioplayers.Remove(guildID)

	var err error = nil

	// NOTE: make sure there is a ready voice connection
	// for the audioplayer
	voice, ok := bot.session.VoiceConnections[guildID]
	if ok == false || voice.Ready == false {
		err = bot.joinVoice(guildID, channelID)
	}
	bot.queueUpdater.Update(
		bot.session,
		guildID,
		100*time.Millisecond,
		nil,
	)
	if err != nil {
		return nil
	}
playLoop:
	for {
		// NOTE: always play the queue's headSong,
		// as it is the song with the minimum position
		// in the queue.
		song, err := bot.datastore.Song().GetHeadSongForQueue(
			bot.session.State.User.ID,
			guildID,
		)
		// Return if there is none
		if err != nil {
			bot.queueUpdater.Update(
				bot.session,
				guildID,
				100*time.Millisecond,
				nil,
			)
			return nil
		}
		voice, ok = bot.session.VoiceConnections[guildID]
		if ok == false || !voice.Ready {
			time.Sleep(300 * time.Second)
			voice, ok = bot.session.VoiceConnections[guildID]
			if ok == false || !voice.Ready {
				bot.WithField("GuildID", guildID).Trace("Voice not ready")
				bot.queueUpdater.Update(
					bot.session,
					guildID,
					300*time.Millisecond,
					nil,
				)
				return nil
			}
		}

		bot.WithField(
			"GuildID", guildID,
		).Tracef("Playing song: %s", song.Url)

		reason, err := ap.Play(bot.ctx, song, voice)
		if err != nil {
			bot.Errorf("Error when playing: %v")
			bot.audioplayerOnError(bot.session, guildID)
		}
		switch reason {
		case audioplayer.FinishedVoiceClosed:
			bot.WithField("GuildID", guildID).Trace("Audioplayer Voice Closed")
			return nil
		case audioplayer.FinishedOK:
			bot.WithField("GuildID", guildID).Trace("Audioplayer finished OK")
			bot.audioplayerOnOK(bot.session, guildID)
			continue playLoop
		case audioplayer.FinishedTerminated:
			bot.WithField("GuildID", guildID).Trace("Audioplayer Terminated")
			return nil
		default:
			continue playLoop
		}
	}
}

// audioplayerOnOK is a function called when the audioplayer
// finishes with OK finish reason.
func (bot *Bot) audioplayerOnOK(s *discordgo.Session, guildID string) {
	// NOTE: if loop is enabled in the queue
	// push it's headSong to the back of the queue
	// else just
	if bot.datastore.Queue().QueueHasOption(
		s.State.User.ID,
		guildID,
		model.Loop,
	) {
		bot.WithField("GuildID", guildID).Trace(
			"Bot has loop enabled, pushing head song to back",
		)
		if err := bot.datastore.Song().PushHeadSongToBack(
			s.State.User.ID,
			guildID,
		); err != nil {
			bot.Errorf(
				"Error when pushing first song to back during play: %v",
				err,
			)
		}
	} else {
		bot.WithField("GuildID", guildID).Trace(
			"Bot does not have loop enabled, removing head song",
		)
		headsong, err := bot.datastore.Song().GetHeadSongForQueue(
			s.State.User.ID,
			guildID,
		)
		if err != nil {
			bot.Errorf(
				"Error when fetching song during play: %v", err,
			)
			return
		}
		// NOTE: persist queue's headSong to inactive song table
		if err := bot.datastore.Song().PersistInactiveSongs(
			s.State.User.ID,
			guildID,
			headsong,
		); err != nil {
			bot.Errorf(
				"Error when removing song during play: %v", err,
			)
		}
		if err := bot.datastore.Song().RemoveHeadSong(
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
}

// audioplayerOnError is the a function called
// when the audioplayer finishes with error.
func (bot *Bot) audioplayerOnError(s *discordgo.Session, guildID string) {
	// NOTE: if error occured when playing the head song,
	// it should be removed

	if err := bot.datastore.Song().RemoveHeadSong(
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

// joinVoice connects to the voice channel identified by the provided guilID and
// channelID, returns error on failure. If the client is already connected to the
// voice channel, it does not connect again.
func (bot *Bot) joinVoice(guildID string, channelID string) error {
	bot.WithFields(log.Fields{
		"GuildID":   guildID,
		"ChannelID": channelID,
	}).Trace("Joining voice")

	vc, ok := bot.session.VoiceConnections[guildID]
	if ok && vc.ChannelID == channelID {
		return nil
	}
	if _, err := bot.session.ChannelVoiceJoin(guildID, channelID, false, false); err != nil {
		bot.Tracef("Could not join voice: %v", err)
		if buffer, ok := bot.queueUpdater.GetInteractionsBuffer(guildID); ok {
		responseLoop:
			for i := 0; i < len(buffer); i++ {
				select {
				case interaction := <-buffer:
					err := bot.session.InteractionRespond(interaction, &discordgo.InteractionResponse{
						Type: discordgo.InteractionResponseChannelMessageWithSource,
						Data: &discordgo.InteractionResponseData{
							Content: "Cannot join the channel, I may be missing permissions",
							Flags:   discordgo.MessageFlagsEphemeral,
						},
					})
					if err != nil {
						continue responseLoop
					}
					break responseLoop
				}
			}
		}
		return err
	}
	return nil
}
