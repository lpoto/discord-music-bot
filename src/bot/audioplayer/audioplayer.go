package audioplayer

import (
	"context"
	"discord-music-bot/model"
	"errors"
	"io"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
)

type AudioPlayer struct {
	guildID          string
	session          *discordgo.Session
	client           *youtube.Client
	streamingSession *dca.StreamingSession
	encodingSession  *dca.EncodeSession
	streaming        bool
	defaultDeferFunc func(*discordgo.Session, string)
	errorDeferFunc   func(*discordgo.Session, string)
	deferFuncBuffer  chan func(*discordgo.Session, string)
}

// NewAudioPlayer constructs an object that handles playing
// audio in a discord's voice channel
func NewAudioPlayer(session *discordgo.Session, guildID string, defaultDeferFunc func(*discordgo.Session, string), errorDeferFunc func(*discordgo.Session, string)) *AudioPlayer {
	return &AudioPlayer{
		client:           &youtube.Client{},
		guildID:          guildID,
		session:          session,
		streamingSession: nil,
		encodingSession:  nil,
		streaming:        false,
		defaultDeferFunc: defaultDeferFunc,
		errorDeferFunc:   errorDeferFunc,
		deferFuncBuffer:  make(chan func(*discordgo.Session, string), 10),
	}
}

// IsPlaying returns true if the audioplayer is currenthly
// streaming some audio, false otherwise
func (ap *AudioPlayer) IsPlaying() bool {
	return ap.streaming
}

// IsPaused returns true if the audioplayer is currenthly
// paused, false otherwise
func (ap *AudioPlayer) IsPaused() bool {
	return ap.streamingSession.Paused()
}

// Stop stops the current stream, if there is any
func (ap *AudioPlayer) Stop() {
	if ap.encodingSession == nil {
		return
	}
	ap.encodingSession.Stop()
}

// Play starts playing the provided song in the bot's
// current voice channel. Returns error if the bot is not connected.
func (ap *AudioPlayer) Play(ctx context.Context, song *model.Song) error {
	ap.streaming = true
	defer func() { ap.streaming = false }()

	voiceConnection, ok := ap.session.VoiceConnections[ap.guildID]
	if !ok {
		return errors.New("Not connected to voice")
	}

	video, err := ap.client.GetVideo(song.Url)

	if err != nil {
		ap.errorDeferFunc(ap.session, ap.guildID)
		return err
	}
	formats := video.Formats.WithAudioChannels()
	url, err := ap.client.GetStreamURL(video, &formats[0])
	if err != nil {
		ap.errorDeferFunc(ap.session, ap.guildID)
		return err
	}

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	ap.encodingSession, err = dca.EncodeFile(url, options)
	if err != nil {
		ap.errorDeferFunc(ap.session, ap.guildID)
		return err
	}
	defer func() {
		ap.encodingSession.Cleanup()
		ap.encodingSession = nil
	}()

	streamingDone := make(chan error)
	ap.streamingSession = dca.NewStream(
		ap.encodingSession,
		voiceConnection,
		streamingDone,
	)
	defer func() { ap.streamingSession = nil }()

	done := ctx.Done()

	t := time.Now()
	for {
		select {
		case <-done:
		case err := <-streamingDone:
			if err != io.EOF && err != io.ErrUnexpectedEOF ||
				time.Since(t) <= time.Second {
				ap.errorDeferFunc(ap.session, ap.guildID)
			} else {
				ap.getDefferFunc()(ap.session, ap.guildID)
			}
			return nil
		}
	}
}

// Pause pauses the currently streaming session if any
func (ap *AudioPlayer) Pause() {
	if ap.streamingSession == nil {
		return
	}
	ap.streamingSession.SetPaused(true)
}

// Pause unpauses the currently streaming session if any
func (ap *AudioPlayer) Unpause() {
	if ap.streamingSession == nil {
		return
	}
	ap.streamingSession.SetPaused(false)
}

// AddDeferFunc adds the provided function to the deferFuncBuffer.
// Functions in this buffer are then called when the player finishes,
// instead of the default defer func.
func (ap *AudioPlayer) AddDeferFunc(f func(*discordgo.Session, string)) {
	select {
	case ap.deferFuncBuffer <- f:
	default:
	}
}

// PlaybackPosition returns the duration of the currently playing
// stream already streamed, rounded to seconds. Returns 0 if nothing
// is playing.
func (ap *AudioPlayer) PlaybackPosition() time.Duration {
	if ap.streamingSession == nil {
		return 0
	}
	return ap.streamingSession.PlaybackPosition().Truncate(time.Second)

}

// getDefferFunc returns the function that should be called when the
// audioplayer finishes. If any functions were added to the
// deferFuncBuffer, all those are used, if none were added, the
// defaultFunc, added in the constructor, is returned.
func (ap *AudioPlayer) getDefferFunc() func(*discordgo.Session, string) {
	return func(s *discordgo.Session, guildID string) {
		if len(ap.deferFuncBuffer) == 0 {
			ap.defaultDeferFunc(s, guildID)
			return
		}
		for i := 0; i < len(ap.deferFuncBuffer); i++ {
			select {
			case f, ok := <-ap.deferFuncBuffer:
				if !ok {
					return
				}
				f(s, guildID)
			}
		}
	}
}
