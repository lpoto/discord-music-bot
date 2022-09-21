package builder

import (
	"discord-music-bot/model"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// NewQueue constructs an object that represents a music queue
// in a discord server. It is identified by the clientID and guildID.
func (builder *Builder) NewQueue(clientID string, guildID string, messageID string, channelID string) *model.Queue {
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
func (builder *Builder) MapQueueToEmbed(queue *model.Queue) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       builder.Config.Title,
		Fields:      make([]*discordgo.MessageEmbedField, 0),
		Description: builder.Config.Description,
		Footer: &discordgo.MessageEmbedFooter{
			Text: builder.Config.Footer,
		},
	}
	spacer := "> "
	spacer2 := spacer + "ㅤ"
	if queue.HeadSong != nil {
		embed.Color = queue.HeadSong.Color
		// TODO: add song loader
		// TODO: wrap head song to lines of length 30
		// TODO: use canvas to shorten song names
		headSong := builder.WrapName(queue.HeadSong.Name)
		headSong = fmt.Sprintf(
			"%s\n**%s**\u3000%s\n%s",
			spacer,
			queue.HeadSong.DurationString,
			headSong,
			spacer2,
		)
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

// GetMusicQueueComponents constructs a list od message components
// that belong to the provided queue, they may vary based on
// the queue's options
func (builder *Builder) GetMusicQueueComponents(queue *model.Queue) []discordgo.MessageComponent {
	if builder.QueueHasOption(queue, model.Inactive) {
		return []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: []discordgo.MessageComponent{
					builder.newButton(builder.Config.Components.Join, discordgo.SecondaryButton, false),
				},
			},
		}
	}

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
				builder.newButton(builder.Config.Components.Backward, discordgo.SecondaryButton, queue.Size <= queue.Limit),
				builder.newButton(builder.Config.Components.Forward, discordgo.SecondaryButton, queue.Size <= queue.Limit),
				builder.newButton(builder.Config.Components.Previous, discordgo.SecondaryButton, queue.InactiveSize == 0 && !(queue.Size > 1 && builder.QueueHasOption(queue, model.Loop)) || builder.QueueHasOption(queue, model.Paused)),
				builder.newButton(builder.Config.Components.Skip, discordgo.SecondaryButton, queue.HeadSong == nil || builder.QueueHasOption(queue, model.Paused) || (queue.Size == 1 && builder.QueueHasOption(queue, model.Loop))),
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				builder.newButton(builder.Config.Components.AddSongs, discordgo.SecondaryButton, false),
				builder.newButton(builder.Config.Components.Loop, loopStyle, false),
				builder.newButton(builder.Config.Components.Pause, pauseStyle, queue.HeadSong == nil),
				builder.newButton(builder.Config.Components.Replay, discordgo.SecondaryButton, queue.HeadSong == nil || builder.QueueHasOption(queue, model.Paused)),
			},
		},
	}
}

// QueueHasOption checks if the provided queue
// has the provided option set
func (builder *Builder) QueueHasOption(queue *model.Queue, option model.QueueOptionName) bool {
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
func (builder *Builder) GetButtonLabelFromComponentData(data discordgo.MessageComponentInteractionData) string {
	return strings.Split(data.CustomID, "<split>")[0]
}

func (builder *Builder) newButton(label string, style discordgo.ButtonStyle, disabled bool) discordgo.Button {
	return discordgo.Button{
		CustomID: label + "<split>" + uuid.NewString(),
		Label:    label,
		Style:    style,
		Disabled: disabled,
	}
}
