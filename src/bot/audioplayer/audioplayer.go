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
	guildID         string
	session         *discordgo.Session
	youtube         *youtube.Youtube
	funcs           *DeferFunctions
	streamSession   *stream.Session
	durationSeconds int
	stop            bool
	Continue        bool
}

type DeferFunctions struct {
	defaultOnSuccess func(*discordgo.Session, string)
	onFailure        func(*discordgo.Session, string)
	onSuccessBuffer  chan func(*discordgo.Session, string)
}

// NewAudioPlayer constructs an object that handles playing
// audio in a discord's voice channel
func NewAudioPlayer(session *discordgo.Session, yt *youtube.Youtube, guildID string, funcs *DeferFunctions) *AudioPlayer {
	return &AudioPlayer{
		youtube:       yt,
		guildID:       guildID,
		session:       session,
		funcs:         funcs,
		streamSession: nil,
		stop:          false,
		Continue:      true,
	}
}

// NewDeferFunctions constructs an object that holds functions called
// when the audioplayer finishes. If pleyer finishes with an error, the onFailure
// function is called, else if it successfully finishes, if any functions are present
// in the onSuccessBuffer, those are called, otherwise the defaultOnSuccess is called
func NewDeferFunctions(success func(*discordgo.Session, string), err func(*discordgo.Session, string)) *DeferFunctions {
	return &DeferFunctions{
		defaultOnSuccess: success,
		onFailure:        err,
		onSuccessBuffer:  make(chan func(*discordgo.Session, string), 5),
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

// Stop stops the current stream, if there is any
func (ap *AudioPlayer) Stop() {
	ap.stop = true
	if ap.streamSession == nil {
		return
	}
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

// TimeLeft returns the duration before the
// current stream finishes. 0 if there is no stream.
func (ap *AudioPlayer) TimeLeft() time.Duration {
	if ap.streamSession == nil {
		return 0
	}
	if ap.streamSession.Finished() {
		return 0
	}
	return time.Duration(ap.durationSeconds*int(time.Second)) - ap.PlaybackPosition()
}

// AddDeferFunc adds the provided function to the deferFuncBuffer.
// Functions in this buffer are then called when the player finishes,
// instead of the default defer func.
func (ap *AudioPlayer) AddDeferFunc(f func(*discordgo.Session, string)) {
	select {
	case ap.funcs.onSuccessBuffer <- f:
	default:
	}
}

// Play starts playing the provided song in the bot's
// current voice channel. Returns error if the bot is not connected.
func (ap *AudioPlayer) Play(ctx context.Context, song *model.Song) error {
	voiceConnection, ok := ap.session.VoiceConnections[ap.guildID]
	if !ok {
		return errors.New("Not connected to voice")
	}

	if ap.stop {
		ap.funcs.getDeferFunc()(ap.session, ap.guildID)
		return nil
	}
	ap.durationSeconds = song.DurationSeconds

	voiceConnection.Speaking(true)
	defer voiceConnection.Speaking(false)

	var err error = nil
	f := ap.funcs.onFailure

streamingLoop:
	// NOTE: try to run the stream 3 times in case
	// something went wrong with encoding and the stream finished
	// with an error in less than a second
	for i := 0; i < 3; i++ {
		if ap.stop {
			f = ap.funcs.getDeferFunc()
			break streamingLoop
		}

		streamSession, err2 := ap.youtube.Stream().GetSession(
			song.Url,
			voiceConnection,
		)
		if err2 != nil {
			err = err2
			continue streamingLoop
		}
		ap.streamSession = streamSession

		streamDone := ap.streamSession.StreamDone()

		done := ctx.Done()
		t := time.Now()

		for {
			select {
			case <-done:
				err, f = nil, ap.funcs.getDeferFunc()
				break streamingLoop
			case err = <-streamDone:
				if err.Error() == "Voice connection closed" {
					// NOTE: if voice connection has been closed,
					// just stop without calling any defer functions
					f = nil
					break streamingLoop
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
					f = ap.funcs.onFailure
					break streamingLoop
				}
				// NOTE: the stream finished successfully
				err, f = nil, ap.funcs.getDeferFunc()
				break streamingLoop
			default:
				if ap.stop {
					err, f = nil, ap.funcs.getDeferFunc()
					break streamingLoop
				}
			}
		}
	}
	if ap.streamSession != nil {
		ap.streamSession.Cleanup()
	}
	ap.streamSession = nil
	ap.durationSeconds = 0

	if f != nil {
		f(ap.session, ap.guildID)
	}

	return err
}

// getDefferFunc returns the function that should be called when the
// audioplayer finishes. If any functions were added to the
// deferFuncBuffer, all those are used, if none were added, the
// defaultFunc, added in the constructor, is returned.
func (f *DeferFunctions) getDeferFunc() func(*discordgo.Session, string) {
	return func(s *discordgo.Session, guildID string) {
		callDefault := true
		for i := 0; i < len(f.onSuccessBuffer); i++ {
			select {
			case f, ok := <-f.onSuccessBuffer:
				if !ok {
					return
				}
				f(s, guildID)
				callDefault = false
			}
		}
		if callDefault {
			f.defaultOnSuccess(s, guildID)
			return
		}
	}
}
