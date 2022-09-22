package audioplayer

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

type PCM struct {
	maxBytes        int
	frameRate       int
	frameSize       int
	channels        int
	buffer          <-chan []int16
	voiceConnection *discordgo.VoiceConnection
}

// NewPCM constructs a new object that handles
// sending opus packets to Discord
func NewPCM(frameRate int, frameSize int, channel int, pcmBuffer <-chan []int16, vc *discordgo.VoiceConnection) *PCM {
	return &PCM{
		maxBytes:        (frameSize * 2) * 2,
		buffer:          pcmBuffer,
		voiceConnection: vc,
		frameRate:       frameRate,
		frameSize:       frameSize,
		channels:        channel,
	}

}

// Run is a long lived worker that encoded the pcm
// packets recieved from the audioplayer and sends them
// to Discord
func (pcm *PCM) Run(ctx context.Context) {
	opusEncoder, err := gopus.NewEncoder(
		pcm.frameRate,
		pcm.channels,
		gopus.Audio,
	)
	if err != nil {
		return
	}

	done := ctx.Done()

	for {
		select {
		case <-done:
			return
		default:
			// read pcm from chan, exit if channel is closed.
			recv, ok := <-pcm.buffer
			if !ok {
				return
			}

			// try encoding pcm frame with Opus
			opus, err := opusEncoder.Encode(
				recv,
				pcm.frameSize,
				pcm.maxBytes,
			)
			if err != nil {
				return
			}

			if pcm.voiceConnection.Ready == false ||
				pcm.voiceConnection.OpusSend == nil {
				return
			}
			// send encoded opus data to the sendOpus channel
			pcm.voiceConnection.OpusSend <- opus
		}
	}
}
