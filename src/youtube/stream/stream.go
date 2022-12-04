package stream

import (
	"errors"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
)

type Stream struct {
	yt *youtube.Client
}

type Session struct {
	streamUrl        string
	encodingSession  *dca.EncodeSession
	streamingSession *dca.StreamingSession
	streamDone       chan error
}

// NewStream constructs an object that handles
// the creation of youtube encoding and streaming sessions
func NewStream() *Stream {
	return &Stream{
		yt: &youtube.Client{},
	}
}

// GetSession creates a new streaming session from
// the provided url
func (s *Stream) GetSession(url string, vc *discordgo.VoiceConnection) (*Session, error) {
	streamUrl, err := s.getStreamUrl(url)
	if err != nil {
		return nil, err
	}
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	encodingSession, err := dca.EncodeFile(streamUrl, options)
	if err != nil {
		return nil, err
	}
	streamDone := make(chan error)
	streamingSession := dca.NewStream(
		encodingSession,
		vc,
		streamDone,
	)
	return &Session{
		streamUrl:        streamUrl,
		streamingSession: streamingSession,
		encodingSession:  encodingSession,
		streamDone:       streamDone,
	}, nil
}

// SetPaused provides paused/unpaused functionality
// for the session's streaming session
func (s *Session) SetPaused(p bool) {
	s.streamingSession.SetPaused(p)
}

// Paused returns true if the session's
// streaming session is paused.
func (s *Session) Paused() bool {
	return s.streamingSession.Paused()
}

// Finished returns true if the session's
// streaming session is Finished.
func (s *Session) Finished() bool {
	f, err := s.streamingSession.Finished()
	if err != nil {
		return true
	}
	return f
}

func (s *Session) StreamDone() chan error {
	return s.streamDone
}

// PlaybackPosition returns the session's
// streaming session's playback position
func (s *Session) PlaybackPosition() time.Duration {
	return s.streamingSession.PlaybackPosition()
}

// Cleanup cleans up the session's encoding session.
func (s *Session) Cleanup() {
	s.encodingSession.Cleanup()
}

// Stop stops the session's encoding session.
func (s *Session) Stop() {
	s.encodingSession.Stop()
}

// getStreamUrl converts the provided url into a stream url
func (s *Stream) getStreamUrl(url string) (string, error) {
	var gErr error = nil
	// try 3 times
	for i := 0; i < 3; i++ {
		video, format, err := s.getStreamFormat(url)
		if err == nil {
			streamUrl, err := s.yt.GetStreamURL(video, format)
			if err == nil {
				return streamUrl, nil
			} else {
				gErr = err
			}
		} else {
			gErr = err
		}
	}
	return "", gErr
}

// getStreamFormat gets the youtube video belonging to the provided
// url and returns it's format that best fits the music bot.
// This tries to return the format with audio mimetype, opus codec, high audio
// quality and low video quality.
func (s *Stream) getStreamFormat(url string) (*youtube.Video, *youtube.Format, error) {
	video, err := s.yt.GetVideo(url)
	if err != nil {
		return nil, nil, err
	}
	// NOTE: filter the formats, so we get the smallest video
	// with best audio quality
	formats := video.Formats
	if formats2 := formats.WithAudioChannels(); len(formats2) > 0 {
		formats = formats2
	}
	// NOTE: try to get audio formats with opus codecs
	formats2 := make(youtube.FormatList, 0)
	for _, f := range formats {
		t := f.MimeType
		if strings.Contains(t, "opus") && strings.Contains(t, "audio") {
			formats2 = append(formats2, f)
		}
	}
	formats = formats2
	// NOTE: try to get the best possible audio quality
	formats2 = make(youtube.FormatList, 0)
	for _, f := range formats {
		if f.AudioQuality == "AUDIO_QUALITY_HIGH" {
			formats2 = append(formats2, f)
		}
	}
	if len(formats2) == 0 {
		for _, f := range formats {
			if f.AudioQuality == "AUDIO_QUALITY_MEDIUM" {
				formats2 = append(formats2, f)
			}
		}
	}
	if len(formats2) > 0 {
		formats = formats2
	}
	// NOTE: try to get the smallest possible video size
	// as video quality is unimportant
	if formats2 := formats.Quality("tiny"); len(formats2) > 0 {
		formats = formats2
	} else if formats2 := formats.Quality("small"); len(formats2) > 0 {
		formats = formats2
	} else if formats2 := formats.Quality("medium"); len(formats2) > 0 {
		formats = formats2
	}
	if len(formats) == 0 {
		return nil, nil, errors.New("No formats found")
	}
	return video, &formats[0], nil
}
