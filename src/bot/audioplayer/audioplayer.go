package audioplayer

import (
	"context"
	"discord-music-bot/model"
	"errors"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
)

type AudioPlayer struct {
	client           *youtube.Client
	guildID          string
	session          *discordgo.Session
	pauseBuffer      chan struct{}
	streamingSession *dca.StreamingSession
}

// NewAudioPlayer constructs an object that handles playing
// audio in a discord's voice channel
func NewAudioPlayer(session *discordgo.Session, guildID string) *AudioPlayer {
	return &AudioPlayer{
		client:           &youtube.Client{},
		guildID:          guildID,
		session:          session,
		pauseBuffer:      make(chan struct{}, 5),
		streamingSession: nil,
	}
}

// Play starts playing the provided song in the bot's
// current voice channel. Returns error if the bot is not connected.
func (ap *AudioPlayer) Play(ctx context.Context, song *model.Song) error {
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

	encodingSession, err := dca.EncodeFile(url, options)
	if err != nil {
		return err
	}
	defer encodingSession.Cleanup()

	streamingDone := make(chan error)
	ap.streamingSession = dca.NewStream(
		encodingSession,
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
		case _, ok := <-ap.pauseBuffer:
			// NOTE: pause if not paused, unpause if paused
			if !ok {
				log.Panic("Pause buffer full!")
			}
			ap.streamingSession.SetPaused(ap.streamingSession.Paused())
		}
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
