package audioplayer

import (
	"context"
	"discord-music-bot/model"
	"discord-music-bot/youtube"
	"discord-music-bot/youtube/stream"
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
	ap.subscriptions.Subscribe("pause", func() {
		if ap.streamSession == nil {
			return
		}
		ap.streamSession.SetPaused(true)
	})
	ap.subscriptions.Subscribe("unpause", func() {
		if ap.streamSession == nil {
			return
		}
		ap.streamSession.SetPaused(true)
	})
	ap.subscriptions.Subscribe("stop", func() {
		ap.stop = true
		if ap.streamSession == nil {
			return
		}
		ap.streamSession.Stop()
	})
	return ap
}

func (ap *AudioPlayer) Subscriptions() *Subscriptions {
	return ap.subscriptions
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
	if ap.streamSession != nil {
		ap.streamSession.Cleanup()
		ap.streamSession = nil
	}
	ap.stop = false
	ap.durationSeconds = 0
}

// Play starts playing the provided song in the bot's
// current voice channel. Returns error if the bot is not connected.
func (ap *AudioPlayer) Play(ctx context.Context, song *model.Song, vc *discordgo.VoiceConnection) {
	defer ap.Cleanup()

	vc.Speaking(true)
	defer vc.Speaking(false)

streamingLoop:
	// NOTE: try to run the stream 3 times in case
	// something went wrong with encoding and the stream finished
	// with an error in less than a second
	for i := 0; i < 3; i++ {
		if ap.stop {
			return
		}
		streamSession, err := ap.youtube.Stream().GetSession(
			song.Url,
			vc,
		)
		if err != nil {
			continue streamingLoop
		}
		ap.streamSession = streamSession

		streamDone := ap.streamSession.StreamDone()

		done := ctx.Done()
		t := time.Now()

		for {
			select {
			case <-done:
				return
			case err = <-streamDone:
				if err.Error() == "Voice connection closed" {
					// NOTE: if voice connection has been closed,
					// just stop without calling any defer functions
					return
				}
				// NOTE: the stream finished, if it lasted
				// less than a second, retry it
				if song.DurationSeconds > 3 && time.Since(t) < 3*time.Second {
					if ap.streamSession != nil {
						ap.streamSession.Cleanup()
						ap.streamSession = nil
					}
					continue streamingLoop
				}
				if err != nil && err != io.EOF {
					return
				}
				// NOTE: the stream finished successfully
				return
			default:
				if vc.Ready == false {
					return
				}
				if ap.stop {
					return
				}
			}
		}
	}
}
