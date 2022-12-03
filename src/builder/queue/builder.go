package queue

import (
	"discord-music-bot/model"
	"discord-music-bot/builder/song"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

type Configuration struct {
	Title       string         `yaml:"Title" validate:"required"`
	Description string         `yaml:"Description"`
	Footer      string         `yaml:"Footer"`
	Buttons     *ButtonsConfig `yaml:"Buttons" validate:"required"`
}
type ButtonsConfig struct {
	Backward string `yaml:"Backward" validate:"required"`
	Forward  string `yaml:"Forward" validate:"required"`
	Pause    string `yaml:"Pause" validate:"required"`
	Skip     string `yaml:"Skip" validate:"required"`
	Previous string `yaml:"Previous" validate:"required"`
	Replay   string `yaml:"Replay" validate:"required"`
	AddSongs string `yaml:"AddSongs" validate:"required"`
	Loop     string `yaml:"Loop" validate:"required"`
	Join     string `yaml:"Join" validate:"required"`
	Offline  string `yaml:"Offline" validate:"required"`
}

type QueueBuilder struct {
	config *Configuration
    songBuilder *song.SongBuilder
}

// NewQueueBuidler constructs an object that handles
// building queues and mapping them to embeds.
func NewQueueBuidler(config *Configuration, songBuilder *song.SongBuilder) *QueueBuilder {
	return &QueueBuilder{
		config: config,
        songBuilder: songBuilder,
	}
}

// ButtonsConfig returns the builder's buttons config.
func (builder *QueueBuilder) ButtonsConfig() *ButtonsConfig {
    return builder.config.Buttons
}

// NewQueue constructs an object that represents a music queue
// in a discord server. It is identified by the clientID and guildID.
func (builder *QueueBuilder) NewQueue(clientID string, guildID string, messageID string, channelID string) *model.Queue {
	queue := new(model.Queue)
	queue.GuildID = guildID
	queue.ClientID = clientID
	queue.MessageID = messageID
	queue.ChannelID = channelID
	queue.Size = 0
	queue.Offset = 0
	queue.HeadSong = nil
	queue.InactiveSize = 0
	queue.Limit = 10
	queue.Songs = make([]*model.Song, 0)
	return queue
}

// MapQueueToEmbed maps the provided queue to a message embed.
// The embed has the first song name in the first field
// and name of the songs,
// limited by queue's limit and offset, in the second field.
// It has buttons for all of the available commands and
// a text input, through which the songs may be added.
func (builder *QueueBuilder) MapQueueToEmbed(queue *model.Queue) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       builder.config.Title,
		Fields:      make([]*discordgo.MessageEmbedField, 0),
		Description: builder.config.Description,
		Footer: &discordgo.MessageEmbedFooter{
			Text: builder.config.Footer,
		},
	}
	spacer := "> "
	spacer2 := spacer + "ㅤ"
	if queue.HeadSong != nil {
		embed.Color = queue.HeadSong.Color
		headSong := builder.songBuilder.WrapName(queue.HeadSong.Name)
		headSong = fmt.Sprintf(
			"**%s**\u3000%s\n%s",
			queue.HeadSong.DurationString, headSong, spacer2,
		)
		headSong = fmt.Sprintf("%s\n%s", spacer, headSong)
		embed.Fields = append(embed.Fields,
			&discordgo.MessageEmbedField{
				Name:  "Now",
				Value: "\u2000" + headSong,
			},
		)
	}
	if len(queue.Songs) > 0 {
		songs := make([]string, 0)
		for i, s := range queue.Songs {
			songs = append(songs, fmt.Sprintf(
				"***%d***\u3000%s",
				i+queue.Offset+1,
				s.ShortName,
			),
			)
		}
		sngs := strings.Join(songs, "\n")
		if len(songs) < queue.Limit && queue.Size > queue.Limit+1 {
			sngs += strings.Repeat("\n"+spacer, queue.Limit-len(songs))
		}
		sngs += fmt.Sprintf(
			"\n%s\n%s%sSongs in queue: ***%d***",
			spacer, spacer,
			strings.Repeat("\u3000", 3),
			queue.Size-1,
		)
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:  "Next",
			Value: sngs,
		})
	}
	return embed
}

// GetInactiveQueueComponents constructs a slice of  message components
// that belong to the provided queue when the bot is about to go offline.
func (builder *QueueBuilder) GetOfflineQueueComponents(queue *model.Queue) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				builder.newButton(builder.config.Buttons.Offline, discordgo.SecondaryButton, true),
			},
		},
	}
}

// GetInactiveQueueComponents constructs a slice of  message components
// that belong to the provided queue when it is considered inactive.
func (builder *QueueBuilder) GetInactiveQueueComponents(queue *model.Queue) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				builder.newButton(builder.config.Buttons.Join, discordgo.SecondaryButton, false),
			},
		},
	}
}

// GetMusicQueueComponents constructs a slice of message components
// that belong to the provided queue, they may vary based on
// the queue's options
func (builder *QueueBuilder) GetMusicQueueComponents(queue *model.Queue) []discordgo.MessageComponent {
	loopStyle := discordgo.SecondaryButton
	pauseStyle := discordgo.SecondaryButton
	if builder.QueueHasOption(queue, model.Loop) {
		loopStyle = discordgo.SuccessButton
	}
	if builder.QueueHasOption(queue, model.Paused) {
		pauseStyle = discordgo.SuccessButton
	}
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				builder.newButton(builder.config.Buttons.Backward, discordgo.SecondaryButton, queue.Size <= queue.Limit),
				builder.newButton(builder.config.Buttons.Forward, discordgo.SecondaryButton, queue.Size <= queue.Limit),
				builder.newButton(builder.config.Buttons.Previous, discordgo.SecondaryButton, queue.InactiveSize == 0 && !(queue.Size > 1 && builder.QueueHasOption(queue, model.Loop)) || builder.QueueHasOption(queue, model.Paused)),
				builder.newButton(builder.config.Buttons.Skip, discordgo.SecondaryButton, queue.HeadSong == nil || builder.QueueHasOption(queue, model.Paused)),
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				builder.newButton(builder.config.Buttons.AddSongs, discordgo.SecondaryButton, false),
				builder.newButton(builder.config.Buttons.Loop, loopStyle, queue.Size == 0 && !builder.QueueHasOption(queue, model.Loop)),
				builder.newButton(builder.config.Buttons.Pause, pauseStyle, queue.HeadSong == nil),
				builder.newButton(builder.config.Buttons.Replay, discordgo.SecondaryButton, queue.HeadSong == nil || builder.QueueHasOption(queue, model.Paused)),
			},
		},
	}
}

// QueueHasOption checks if the provided queue
// has the provided option set
func (builder *QueueBuilder) QueueHasOption(queue *model.Queue, option model.QueueOptionName) bool {
	if queue == nil || queue.Options == nil {
		return false
	}
	for _, o := range queue.Options {
		if option == o.Name {
			return true
		}
	}
	return false
}

//GetButtonLabelFromComponentData returns the button's label from
//it's customID
func (builder *QueueBuilder) GetButtonLabelFromComponentData(data discordgo.MessageComponentInteractionData) string {
	return strings.Split(data.CustomID, "<split>")[0]
}

func (builder *QueueBuilder) newButton(label string, style discordgo.ButtonStyle, disabled bool) discordgo.Button {
	return discordgo.Button{
		CustomID: label + "<split>" + uuid.NewString(),
		Label:    label,
		Style:    style,
		Disabled: disabled,
	}
}
