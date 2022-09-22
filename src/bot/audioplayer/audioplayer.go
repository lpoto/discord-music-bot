package audioplayer

import (
	"context"
	"discord-music-bot/model"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
)

type AudioPlayer struct {
	guildID string
	session *discordgo.Session
	client  *youtube.Client
	funcs   *DeferFunctions
	pcm     *PCM
	stop    bool
	pause   bool
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
		client:  &youtube.Client{},
		guildID: guildID,
		session: session,
		funcs:   funcs,
		pcm:     nil,
		stop:    false,
		pause:   false,
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

// Play starts playing the provided song in the bot's
// current voice channel. Returns error if the bot is not connected.
func (ap *AudioPlayer) Play(ctx context.Context, song *model.Song) error {
	voiceConnection, ok := ap.session.VoiceConnections[ap.guildID]
	if !ok {
		return errors.New("Not connected to voice")
	}

	voiceConnection.Speaking(true)

	var body io.ReadCloser
	var err error
	var format *youtube.Format = nil
	var video *youtube.Video = nil
	body, err = nil, errors.New("Failed to get stream body")

	for i := 0; i < 3; i++ {
		video, format, err = ap.getStreamFormat(song.Url)
		if err == nil {
			body, err = ap.getStreamBody(video, format)
		}
		if err == nil {
			break
		}
	}

	if err != nil {
		ap.funcs.onFailure(ap.session, ap.guildID)
		return err
	}
	defer body.Close()

	// NOTE: determine the frameRate, size and channels
	// from the format...
	frameRate, err := strconv.Atoi(format.AudioSampleRate)
	if err != nil {
		frameRate = 48000
	}
	frameSize := frameRate / 50
	channels := format.AudioChannels

	ctx, cancel := context.WithCancel(ctx)

	pcmChannel := make(chan []int16, 2)

	ap.pcm = NewPCM(frameRate, frameSize, channels, pcmChannel, voiceConnection)
	go ap.pcm.Run(ctx)

	defer func() {
		ap.pcm = nil
		cancel()
		voiceConnection.Speaking(false)
	}()

	var streamAudio func(int) error

	// NOTE: try to stream 3 times in a row on error
	// in case something goes wrong
	streamAudio = func(retry int) error {

		if ap.stop {
			ap.funcs.getDeferFunc()(ap.session, ap.guildID)
			return nil
		}
		if retry >= 3 {
			ap.funcs.onFailure(ap.session, ap.guildID)
			return nil
		}
		stdout, err := ap.runFFmpegCommand(body, channels, frameRate)
		if err != nil {
			ap.funcs.onFailure(ap.session, ap.guildID)
			return err
		}

		done := ctx.Done()

		// buffer used during loop below
		audiobuf := make([]int16, frameSize*channels)
	streamLoop:
		for {
			select {
			case <-done:
				return nil
			default:
				if ap.stop {
					ap.funcs.getDeferFunc()(ap.session, ap.guildID)
					return nil
				}
				if ap.pause {
					continue streamLoop
				}
				// read data from ffmpeg stdout
				err = binary.Read(stdout, binary.LittleEndian, &audiobuf)
				if err == io.EOF || err == io.ErrUnexpectedEOF {
					ap.funcs.getDeferFunc()(ap.session, ap.guildID)
					return nil
				}
				if err != nil {
					return streamAudio(retry + 1)
				}
				pcmChannel <- audiobuf
			}
		}
	}
	return streamAudio(0)
}

// Pause pauses the currently streaming session if any
func (ap *AudioPlayer) Pause() {
	ap.pause = true
}

// Pause unpauses the currently streaming session if any
func (ap *AudioPlayer) Unpause() {
	ap.pause = false
}

// IsPaused returns true if the audioplayer is currenthly
// paused, false otherwise
func (ap *AudioPlayer) IsPaused() bool {
	return ap.pause
}

// Stop stops the current stream, if there is any
func (ap *AudioPlayer) Stop() {
	ap.stop = true
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

// PlaybackPosition returns the duration of the currently playing
// stream already streamed, rounded to seconds. Returns 0 if nothing
// is playing.
func (ap *AudioPlayer) PlaybackPosition() time.Duration {
	return 0
}

// getStreamBody gets the stream url from youtube from
// the provided url, then calls a http request and fetches the body
// for the stream
func (ap *AudioPlayer) getStreamBody(video *youtube.Video, format *youtube.Format) (io.ReadCloser, error) {
	url, err := ap.client.GetStreamURL(video, format)
	if err != nil {
		return nil, err
	}
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, errors.New(fmt.Sprintf("Response status code: %v", resp.StatusCode))
	}
	return resp.Body, nil
}

// getStreamFormat gets the youtube video belonging to the provided
// url and returns it's format that best fits the music bot
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
	if formats2 := formats.Type(`audio/mp4; codecs="opus"`); len(formats2) > 0 {
		formats = formats2
	} else if formats2 := formats.Type(`audio/webm; codecs="opus"`); len(formats2) > 0 {
		formats = formats2
	}
	// NOTE: try to get the best possible audio quality
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

// runFFmpegCommand runs ffmpeg with the provided body
// and the audio player's configuration
func (ap *AudioPlayer) runFFmpegCommand(body io.ReadCloser, channels int, frameRate int) (io.ReadCloser, error) {
	run := exec.Command(
		"ffmpeg",
		"-i", "-", "-f", "s16le", "-ar",
		strconv.Itoa(frameRate), "-ac",
		strconv.Itoa(channels), "pipe:1",
	)
	run.Stdin = body
	stdout, err := run.StdoutPipe()
	if err != nil {
		ap.funcs.onFailure(ap.session, ap.guildID)
		return nil, err
	}
	err = run.Start()
	if err != nil {
		ap.funcs.onFailure(ap.session, ap.guildID)
		return nil, err
	}
	return stdout, nil

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
