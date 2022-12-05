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

const (
	FinishedError       string = "Error"
	FinishedOK          string = "Ok"
	FinishedTerminated  string = "Terminated"
	FinishedVoiceClosed string = "VoiceClosed"
)

type AudioPlayer struct {
	youtube         *youtube.Youtube
	streamSession   *stream.Session
	stopReason      string
	durationSeconds int
	stop            bool
}

// NewAudioPlayer constructs an object that handles playing
// audio in a discord's voice channel
func NewAudioPlayer(yt *youtube.Youtube) *AudioPlayer {
	return &AudioPlayer{
		youtube:         yt,
		streamSession:   nil,
		stop:            false,
		stopReason:      FinishedOK,
		durationSeconds: 0,
	}
}

// Pause pauses the current stream if there is any
func (ap *AudioPlayer) Pause() {
	if ap.streamSession == nil {
		return
	}
	ap.streamSession.SetPaused(true)
}

// Pause resumes the current stream if there is any
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

// StopOK stops the current stream with FinishedOK reason.
func (ap *AudioPlayer) StopOK() {
	ap.Stop(FinishedOK)
}

// StopOK stops the current stream with FinishedTerminated reason.
func (ap *AudioPlayer) StopTerminate() {
	ap.Stop(FinishedTerminated)
}

// StopOK stops the current stream with FinishedVoiceClosed reason.
func (ap *AudioPlayer) StopVoiceClosed() {
	ap.Stop(FinishedVoiceClosed)
}

// Stop stops the current stream, if there is any.
func (ap *AudioPlayer) Stop(reason string) {
	ap.stop = true
	if ap.streamSession == nil {
		return
	}
	ap.stopReason = reason
	ap.streamSession.Stop()
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
	ap.stopReason = FinishedOK
	ap.stop = false
	ap.durationSeconds = 0
}

// Play starts playing the provided song in the bot's
// current voice channel. Returns error if the bot is not connected.
func (ap *AudioPlayer) Play(ctx context.Context, song *model.Song, vc *discordgo.VoiceConnection) (finishReason string, err error) {
	defer ap.Cleanup()

	if ap.stop {
		return ap.stopReason, nil
	}
	ap.durationSeconds = song.DurationSeconds

	vc.Speaking(true)
	defer vc.Speaking(false)

	var potentialError error = nil

streamingLoop:
	// NOTE: try to run the stream 3 times in case
	// something went wrong with encoding and the stream finished
	// with an error in less than a second
	for i := 0; i < 3; i++ {
		if ap.stop {
			return ap.stopReason, nil
		}
		streamSession, err := ap.youtube.Stream().GetSession(
			song.Url,
			vc,
		)
		if err != nil {
			potentialError = err
			continue streamingLoop
		}
		ap.streamSession = streamSession

		streamDone := ap.streamSession.StreamDone()

		done := ctx.Done()
		t := time.Now()

		for {
			select {
			case <-done:
				return FinishedTerminated, nil
			case err = <-streamDone:
				if err.Error() == "Voice connection closed" {
					// NOTE: if voice connection has been closed,
					// just stop without calling any defer functions
					return FinishedVoiceClosed, nil
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
					return FinishedError, err
				}
				// NOTE: the stream finished successfully
				return FinishedOK, nil
			default:
				if vc.Ready == false {
					return FinishedVoiceClosed, nil
				}
				if ap.stop {
					return ap.stopReason, nil
				}
			}
		}
	}
	if potentialError == nil {
		return FinishedError, potentialError
	}
	return FinishedOK, potentialError
}
