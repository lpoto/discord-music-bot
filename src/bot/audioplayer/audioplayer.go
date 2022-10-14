package audioplayer

import (
	"context"
	"discord-music-bot/model"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
)

type AudioPlayer struct {
	guildID          string
	session          *discordgo.Session
	client           *youtube.Client
	funcs            *DeferFunctions
	encodingSession  *dca.EncodeSession
	streamingSession *dca.StreamingSession
	durationSeconds  int
	stop             bool
	Continue         bool
}

type DeferFunctions struct {
	defaultOnSuccess func(*discordgo.Session, string)
	onFailure        func(*discordgo.Session, string)
	onSuccessBuffer  chan func(*discordgo.Session, string)
}

// NewAudioPlayer constructs an object that handles playing
// audio in a discord's voice channel
func NewAudioPlayer(session *discordgo.Session, guildID string, funcs *DeferFunctions) *AudioPlayer {
	return &AudioPlayer{
		client:           &youtube.Client{},
		guildID:          guildID,
		session:          session,
		funcs:            funcs,
		encodingSession:  nil,
		streamingSession: nil,
		stop:             false,
		Continue:         true,
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
	if ap.streamingSession == nil {
		return
	}
	ap.streamingSession.SetPaused(true)
}

// Pause resumes the current stream if there is any
func (ap *AudioPlayer) Unpause() {
	if ap.streamingSession == nil {
		return
	}
	ap.streamingSession.SetPaused(false)
}

// IsPaused returns true if the audioplayer is currenthly
// paused, false otherwise
func (ap *AudioPlayer) IsPaused() bool {
	if ap.streamingSession == nil {
		return false
	}
	return ap.streamingSession.Paused()
}

// Stop stops the current stream, if there is any
func (ap *AudioPlayer) Stop() {
	ap.stop = true
	if ap.encodingSession == nil {
		return
	}
	ap.encodingSession.Stop()
}

// PlaybackPosition returns the duration of the currently playing
// stream already streamed. Returns 0 if nothing
// is playing.
func (ap *AudioPlayer) PlaybackPosition() time.Duration {
	if ap.streamingSession == nil {
		return 0
	}
	if finished, err := ap.streamingSession.Finished(); err != nil || finished {
		return 0
	}
	return ap.streamingSession.PlaybackPosition()
}

// TimeLeft returns the duration before the
// current stream finishes. 0 if there is no stream.
func (ap *AudioPlayer) TimeLeft() time.Duration {
	if ap.streamingSession == nil {
		return 0
	}
	if finished, err := ap.streamingSession.Finished(); err != nil || finished {
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
	if _, ok := ap.session.VoiceConnections[ap.guildID]; !ok {
		return errors.New("Not connected to voice when starting 'Play'")
	}

	if ap.stop {
		ap.funcs.getDeferFunc()(ap.session, ap.guildID)
		return nil
	}
	ap.durationSeconds = song.DurationSeconds

	// NOTE: try to get the best audio opus format for
	// the youtube video belonging to the song's url,
	// then pass the video's stream url to dca
	// (it then encodes it with FFMPEG and sends it to discord)
	streamUrl := ""
	var err error
	var format *youtube.Format = nil
	var video *youtube.Video = nil

	// NOTE: try fetching the format 2 times, in case something
	// went wrong
	for i := 0; i < 3; i++ {
		if ap.stop {
			ap.funcs.getDeferFunc()(ap.session, ap.guildID)
			ap.durationSeconds = 0
			return nil
		}
		video, format, err = ap.getStreamFormat(song.Url)
		if err == nil {
			streamUrl, err = ap.client.GetStreamURL(video, format)
		}
	}
	if err != nil {
		ap.funcs.onFailure(ap.session, ap.guildID)
		ap.durationSeconds = 0
		return err
	}
	options := dca.StdEncodeOptions
	options.RawOutput = true
	options.Bitrate = 96
	options.Application = "lowdelay"

	ap.encodingSession, err = dca.EncodeFile(streamUrl, options)
	if err != nil {
		return err
	}

	f, err := ap.startStream(ctx, song, 0, nil)

	if ap.encodingSession != nil {
		ap.encodingSession.Cleanup()
	}
	ap.encodingSession = nil
	ap.streamingSession = nil
	ap.durationSeconds = 0

	if f != nil {
		f(ap.session, ap.guildID)
	}

	return err
}

// startStream uses the audioplayer's encodingSession and starts the stream,
// then returns the defer function that should be called on finish and error if any
func (ap *AudioPlayer) startStream(ctx context.Context, song *model.Song, retry int, err error) (func(*discordgo.Session, string), error) {
	if ap.stop {
		return ap.funcs.getDeferFunc(), nil
	}
	if retry > 3 {
		return ap.funcs.onFailure, err
	}

	done := ctx.Done()

	voiceConnection, ok := ap.session.VoiceConnections[ap.guildID]
	if !ok {
		err = errors.New("Not connected to voice, when restarting stream")
		return nil, err
	}
	voiceConnection.Speaking(true)
	defer voiceConnection.Speaking(false)

	streamDone := make(chan error)
	ap.streamingSession = dca.NewStream(
		ap.encodingSession,
		voiceConnection,
		streamDone,
	)

	t := time.Now()

	for {
		select {
		case <-done:
			return ap.funcs.getDeferFunc(), nil
		case err = <-streamDone:
			if err.Error() == "Voice connection closed" {
				// NOTE: if voice connection has been closed,
				// just stop without calling any defer functions

				time.Sleep(100 * time.Millisecond)
				return ap.startStream(ctx, song, retry+1, err)
			}
			// NOTE: the stream finished, if it lasted
			// less than a second, retry it
			if song.DurationSeconds > 3 && time.Since(t) < 3*time.Second {
				if ap.encodingSession != nil {
					ap.encodingSession.Cleanup()
					ap.encodingSession = nil
				}
				return ap.startStream(ctx, song, retry+1, err)
			}
			if err != nil && err != io.EOF {
				return ap.funcs.onFailure, err
			}
			// NOTE: the stream finished successfully
			return ap.funcs.getDeferFunc(), nil
		default:
			if ap.stop {
				return ap.funcs.getDeferFunc(), nil
			}
		}
	}
}

// getStreamFormat gets the youtube video belonging to the provided
// url and returns it's format that best fits the music bot.
// This tries to return the format with audio mimetype, opus codec, high audio
// quality and low video quality.
func (ap *AudioPlayer) getStreamFormat(url string) (*youtube.Video, *youtube.Format, error) {
	video, err := ap.client.GetVideo(url)
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
	formats = ap.filterStreamFormatsByMimeType(formats)
	// NOTE: try to get the best possible audio quality
	formats = ap.filterStreamFormatsByAudioQuality(formats)
	// NOTE: try to get the smallest possible video size
	// as video quality is unimportant
	formats = ap.filterStreamFOrmatsByQuality(formats)
	if len(formats) == 0 {
		return nil, nil, errors.New("No formats found")
	}
	return video, &formats[0], nil
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

func (ap *AudioPlayer) filterStreamFormatsByAudioQuality(formats youtube.FormatList) youtube.FormatList {
	formats2 := make(youtube.FormatList, 0)
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
		return formats2
	}
	return formats
}

func (ap *AudioPlayer) filterStreamFormatsByMimeType(formats youtube.FormatList) youtube.FormatList {
	formats2 := make(youtube.FormatList, 0)
	for _, f := range formats {
		t := f.MimeType
		if strings.Contains(t, "opus") && strings.Contains(t, "audio") {
			formats2 = append(formats2, f)
		}
	}
	if len(formats2) > 0 {
		return formats2
	}
	return formats
}

func (ap *AudioPlayer) filterStreamFOrmatsByQuality(formats youtube.FormatList) youtube.FormatList {
	if formats2 := formats.Quality("tiny"); len(formats2) > 0 {
		formats = formats2
	} else if formats2 := formats.Quality("small"); len(formats2) > 0 {
		formats = formats2
	} else if formats2 := formats.Quality("medium"); len(formats2) > 0 {
		formats = formats2
	}
	return formats
}
