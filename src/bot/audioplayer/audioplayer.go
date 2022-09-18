package audioplayer

import (
	"context"
	"discord-music-bot/model"
	"errors"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
)

type AudioPlayer struct {
	client           *youtube.Client
	guildID          string
	session          *discordgo.Session
	streamingSession *dca.StreamingSession
	encodingSession  *dca.EncodeSession
	streaming        bool
}

// NewAudioPlayer constructs an object that handles playing
// audio in a discord's voice channel
func NewAudioPlayer(session *discordgo.Session, guildID string) *AudioPlayer {
	return &AudioPlayer{
		client:           &youtube.Client{},
		guildID:          guildID,
		session:          session,
		streamingSession: nil,
		encodingSession:  nil,
		streaming:        false,
	}
}

// IsPlaying returns true if the audioplayer is currenthly
// streaming some audio, false otherwise
func (ap *AudioPlayer) IsPlaying() bool {
	return ap.streaming
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
		return err
	}
	formats := video.Formats.WithAudioChannels()
	url, err := ap.client.GetStreamURL(video, &formats[0])
	if err != nil {
		return err
	}

	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	ap.encodingSession, err = dca.EncodeFile(url, options)
	if err != nil {
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

	for {
		select {
		case <-done:
		case <-streamingDone:
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

// PlaybackPosition returns the duration of the currently playing
// stream already streamed, rounded to seconds. Returns 0 if nothing
// is playing.
func (ap *AudioPlayer) PlaybackPosition() time.Duration {
	if ap.streamingSession == nil {
		return 0
	}
	return ap.streamingSession.PlaybackPosition().Truncate(time.Second)

}
