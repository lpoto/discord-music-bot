package audioplayer

import (
	"context"
	"discord-music-bot/model"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kkdai/youtube/v2"
)

type Configuration struct {
	Channels  int `yaml:"Channels" validate:"gt=0"`
	FrameRate int `yaml:"FrameRate" validate:"gt=0"`
	FrameSize int `yaml:"FrameSize" validate:"gt=0"`
	Retry     int `yaml:"Retry" validate:"gte=0"`
}

type AudioPlayer struct {
	config  *Configuration
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
func NewAudioPlayer(session *discordgo.Session, guildID string, config *Configuration, funcs *DeferFunctions) *AudioPlayer {
	return &AudioPlayer{
		config:  config,
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

	ctx, cancel := context.WithCancel(ctx)

	pcmChannel := make(chan []int16, 2)

	ap.pcm = NewPCM(ap.config, pcmChannel, voiceConnection)
	go ap.pcm.Run(ctx)

	defer func() {
		ap.pcm = nil
		cancel()
		voiceConnection.Speaking(false)
	}()

	body, err := ap.getStreamBody(song.Url)

	if err != nil {
		ap.funcs.onFailure(ap.session, ap.guildID)
		return err
	}
	defer body.Close()

	var streamAudio func(int) error

	streamAudio = func(retry int) error {
		if ap.stop {
			ap.funcs.getDeferFunc()(ap.session, ap.guildID)
			return nil
		}
		if retry >= 3 {
			ap.funcs.onFailure(ap.session, ap.guildID)
			return nil
		}
		stdout, err := ap.runFFmpegCommand(body)
		if err != nil {
			ap.funcs.onFailure(ap.session, ap.guildID)
			return err
		}

		done := ctx.Done()

		// buffer used during loop below
		audiobuf := make([]int16, ap.config.FrameSize*ap.config.Channels)
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
func (ap *AudioPlayer) getStreamBody(url string) (io.ReadCloser, error) {
	video, err := ap.client.GetVideo(url)
	if err != nil {
		return nil, err
	}
	var f func(int) (io.ReadCloser, error)

	f = func(retries int) (io.ReadCloser, error) {
		if retries > 2 {
			return nil, errors.New("Status code != 200")
		}
		formats := video.Formats.WithAudioChannels()
		url, err = ap.client.GetStreamURL(video, &formats[0])
		resp, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			return f(retries + 1)
		}
		return resp.Body, nil
	}
	return f(0)
}

// runFFmpegCommand runs ffmpeg with the provided body
// and the audio player's configuration
func (ap *AudioPlayer) runFFmpegCommand(body io.ReadCloser) (io.ReadCloser, error) {
	run := exec.Command(
		"ffmpeg",
		"-i", "-", "-f", "s16le", "-ar",
		strconv.Itoa(ap.config.FrameRate), "-ac",
		strconv.Itoa(ap.config.Channels), "pipe:1",
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
