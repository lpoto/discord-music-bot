package audioplayer

import (
	"context"

	"github.com/bwmarrin/discordgo"
	"layeh.com/gopus"
)

type PCM struct {
	maxBytes        int
	buffer          <-chan []int16
	voiceConnection *discordgo.VoiceConnection
	apConfig        *Configuration
}

// NewPCM constructs a new object that handles
// sending opus packets to Discord
func NewPCM(apConfig *Configuration, pcmBuffer <-chan []int16, vc *discordgo.VoiceConnection) *PCM {
	return &PCM{
		maxBytes:        (apConfig.FrameSize * 2) * 2,
		buffer:          pcmBuffer,
		voiceConnection: vc,
		apConfig:        apConfig,
	}

}

// Run is a long lived worker that encoded the pcm
// packets recieved from the audioplayer and sends them
// to Discord
func (pcm *PCM) Run(ctx context.Context) {
	opusEncoder, err := gopus.NewEncoder(
		pcm.apConfig.FrameRate,
		pcm.apConfig.Channels,
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
				pcm.apConfig.FrameSize,
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
