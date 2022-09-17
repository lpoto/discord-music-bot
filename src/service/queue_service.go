package service

import (
	"discord-music-bot/model"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// NewQueue constructs an object that represents a music queue
// in a discord server. It is identified by the clientID and guildID.
func (service *Service) NewQueue(clientID string, guildID string, messageID string, channelID string) *model.Queue {
	queue := new(model.Queue)
	queue.GuildID = guildID
	queue.ClientID = clientID
	queue.MessageID = messageID
	queue.ChannelID = channelID
	queue.Size = 0       // number of songs in the queue
	queue.Offset = 0     // index of songs displayed at first position
	queue.HeadSong = nil // currently playing song
	queue.Limit = 10     // Number of songs per page
	queue.Options = make([]model.QueueOption, 0)
	queue.Songs = make([]*model.Song, 0)
	return queue
}

// MapQueueToEmbed maps the provided queue to a message embed.
// The embed has the first song name in the first field
// and name of the songs,
// limited by queue's limit and offset, in the second field.
// It has buttons for all of the available commands and
// a text input, through which the songs may be added.
func (service *Service) MapQueueToEmbed(queue *model.Queue, footer string) *discordgo.MessageEmbed {
	embed := &discordgo.MessageEmbed{
		Title:       "Music queue",
		Fields:      make([]*discordgo.MessageEmbedField, 0),
		Description: "",
		Footer: &discordgo.MessageEmbedFooter{
			Text: footer,
		},
	}
	spacer := "> "
	spacer2 := spacer + "ㅤ"
	if queue.HeadSong != nil {
		embed.Color = queue.HeadSong.Color
		// TODO: add song loader
		// TODO: wrap head song to lines of length 30
		// TODO: use canvas to shorten song names
		headSong := queue.HeadSong.Info.TrimmedName
		headSong = fmt.Sprintf(
			"%s\n**%s**\u3000%s\n%s",
			spacer,
			queue.HeadSong.Info.DurationString,
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
				s.Info.TrimmedName,
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
