package audioplayer

import (
	"context"
	"discord-music-bot/model"
	"discord-music-bot/youtube"
	"discord-music-bot/youtube/stream"
	"errors"
	"io"
	"time"

	"github.com/bwmarrin/discordgo"
)

type AudioPlayer struct {
	youtube         *youtube.Youtube
	streamSession   *stream.Session
	durationSeconds int
	subscriptions   *Subscriptions
	stop            bool
}

// NewAudioPlayer constructs an object that handles playing
// audio in a discord's voice channel
func NewAudioPlayer(yt *youtube.Youtube) *AudioPlayer {
	ap := &AudioPlayer{
		youtube:         yt,
		streamSession:   nil,
		stop:            false,
		subscriptions:   NewSubscriptions(),
		durationSeconds: 0,
	}
	return ap
}

func (ap *AudioPlayer) Subscriptions() *Subscriptions {
	return ap.subscriptions
}

// Stop stops the audioplayer's stream session
func (ap *AudioPlayer) Stop() {
	ap.stop = true
	if ap.streamSession == nil {
		return
	}
	ap.streamSession.Stop()
}

// Pause pauses the audioplayer's stream session
func (ap *AudioPlayer) Pause() {
	if ap.streamSession == nil {
		return
	}
	ap.streamSession.SetPaused(true)
}

// Unpause unpauses the audioplayer's stream session
func (ap *AudioPlayer) Unpause() {
	if ap.streamSession == nil {
		return
	}
	ap.streamSession.SetPaused(false)
}

// IsPaused returns true if the audioplayer is currenthly
// paused, false otherwise
func (ap *AudioPlayer) IsPaused() bool {
	if ap.streamSession == nil {
		return false
	}
	return ap.streamSession.Paused()
}

// PlaybackPosition returns the duration of the currently playing
// stream already streamed. Returns 0 if nothing
// is playing.
func (ap *AudioPlayer) PlaybackPosition() time.Duration {
	if ap.streamSession == nil {
		return 0
	}
	if ap.streamSession.Finished() {
		return 0
	}
	return ap.streamSession.PlaybackPosition()
}

// Cleanup sets the audioplayer's data back to default.
func (ap *AudioPlayer) Cleanup() {
	ap.stop = false
	ap.durationSeconds = 0
	if ap.streamSession != nil {
		ap.streamSession.Cleanup()
		ap.streamSession = nil
	}
}

// Play starts playing the provided song in the bot's
// current voice channel. Returns error if the bot is not connected.
func (ap *AudioPlayer) Play(ctx context.Context, song *model.Song, vc *discordgo.VoiceConnection) (int, error) {
	defer ap.Cleanup()

	vc.Speaking(true)
	defer vc.Speaking(false)

	var potError error = nil
streamingLoop:
	// NOTE: try to run the stream 3 times in case
	// something went wrong with encoding and the stream finished
	// with an error in less than a second
	for i := 0; i < 3; i++ {
		if ap.stop {
			ap.stop = false
			return 2, nil
		}
		streamSession, err := ap.youtube.Stream().GetSession(
			song.Url,
			vc,
		)
		if err != nil {
			potError = err
			continue streamingLoop
		}
		ap.streamSession = streamSession

		streamDone := ap.streamSession.StreamDone()

		done := ctx.Done()
		t := time.Now()

		for {
			select {
			case <-done:
				return 3, nil
			case err = <-streamDone:
				if err.Error() == "Voice connection closed" {
					// NOTE: if voice connection has been closed,
					// just stop without calling any defer functions
					return 1, errors.New("Voice connection closed")
				}
				// NOTE: the stream finished, if it lasted
				// less than a second, retry it
				if song.DurationSeconds > 3 && time.Since(t) < 3*time.Second {
					if ap.streamSession != nil {
						ap.streamSession.Cleanup()
						ap.streamSession = nil
					}
					potError = errors.New("Ended unexpectedly")
					continue streamingLoop
				}
				if err != nil && err != io.EOF {
					return 1, err
				}
				// NOTE: the stream finished successfully
				return 0, nil
			default:
				if vc.Ready == false {
					return 1, errors.New("Voice connection closed")
				}
				if ap.stop {
					ap.stop = false
					return 2, nil
				}
			}
		}
	}
	return 1, potError
}
