package audioplayer

import (
	"discord-music-bot/model"
	"io"

	"github.com/bwmarrin/discordgo"
	"github.com/jonas747/dca"
	"github.com/kkdai/youtube/v2"
)

type AudioPlayer struct {
	client          *youtube.Client
	voiceConnection *discordgo.VoiceConnection
}

func NewAudioPlayer(vc *discordgo.VoiceConnection) *AudioPlayer {
	return &AudioPlayer{
		client:          &youtube.Client{},
		voiceConnection: vc,
	}
}

func (ap *AudioPlayer) Play(song *model.Song) error {
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

	session, err := dca.EncodeFile(url, options)
	if err != nil {
		return err
	}
	defer session.Cleanup()

	done := make(chan error)
	dca.NewStream(session, ap.voiceConnection, done)
	if err := <-done; err != nil && err != io.EOF {
		return err
	}
	return nil
}
