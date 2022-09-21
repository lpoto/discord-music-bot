package model

import (
	"errors"
	"fmt"
	"strings"
)

type QueueOption string

const (
	Loop     QueueOption = "loop"     // When loop option is set, songs are pushed to the back  of the queue instead of being removed
	Paused   QueueOption = "paused"   // When paused option is set, the queue's audioplayer is paused
	Inactive QueueOption = "inactive" // When inactive, only join button is displayed
)

type Queue struct {
	ClientID     string        `json:"client_id"`     // Id of the bot that created the queue
	GuildID      string        `json:"guild_id"`      // Id of the discord server in which the queue has been created
	MessageID    string        `json:"message_id"`    // Id of the queue's message in a discord channel
	ChannelID    string        `json:"channel_id"`    // Id of the channel in which the queue has been created
	Offset       int           `json:"offset"`        // Current offset of the displayed songs in the queue
	Limit        int           `json:"limit"`         // Number of songs displayed at once
	Options      []QueueOption `json:"options"`       // A list of options currently added to the queue
	Songs        []*Song       `json:"songs"`         // Currently displayed songs
	HeadSong     *Song         `json:"head_song"`     // A queue's song with the minimum position
	PreviousSong *Song         `json:"previous_song"` // Latest song that has been removed from the queue
	Size         int           `json:"size"`          // Total number of songs that belong to the queue
}

// ParseQueueOption converts the provided
// string to QueueOption if s is one of
// the possible options
func ParseQueueOption(s string) (QueueOption, error) {
	opts := GetQueueOptions()
	s2 := strings.ToLower(s)
	for _, o := range opts {
		if string(o) == s2 {
			return QueueOption(s2), nil
		}
	}
	return QueueOption(""), errors.New(
		fmt.Sprintf(
			"The only allowed queue options are: %v",
			opts,
		),
	)
}

// GetQueueOptions returns a slice of all
// the possible queue options
func GetQueueOptions() []QueueOption {
	return []QueueOption{
		Loop,
		Paused,
	}
}

// QueueOptionsToStringSlice converts the
// provided slice of QueueOptions to a slice of strings
func QueueOptionsToStringSlice(opts []QueueOption) []string {
	x := make([]string, 0, len(opts))
	for _, o := range opts {
		x = append(x, string(o))
	}
	return x
}

// QueueOption converts the provided slice of string to
// a slice of QueueOptions
func StringSliceToQueueOptions(opts []string) []QueueOption {
	x := make([]QueueOption, 0, len(opts))
	for _, o := range opts {
		x = append(x, QueueOption(o))
	}
	return x
}
