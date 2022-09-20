package model

import (
	"errors"
	"fmt"
	"strings"
)

type QueueOption string

const (
	Loop   QueueOption = "loop"
	Paused QueueOption = "paused"
	Test   QueueOption = "test"
)

type Queue struct {
	ClientID     string        `json:"client_id"`
	GuildID      string        `json:"guild_id"`
	MessageID    string        `json:"message_id"`
	ChannelID    string        `json:"channel_id"`
	Offset       int           `json:"offset"`
	Limit        int           `json:"limit"`
	Options      []QueueOption `json:"options"`
	Songs        []*Song       `json:"songs"`
	HeadSong     *Song         `json:"head_song"`
	PreviousSong *Song         `json:"previous_song"`
	Size         int           `json:"size"`
}

// ParseQueueOption converts the provided
// string to QueueOption if s is one of
// the possible options
func ParseQueueOption(s string) (QueueOption, error) {
	opts := GetQueueOptions()
	s2 := strings.ToLower(s)
	for _, o := range opts {
		if string(o) == s2 || s2 == string(Test) {
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
