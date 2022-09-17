package model

import (
	"errors"
	"fmt"
	"strings"
)

type QueueOption string

const (
	Loop            QueueOption = "loop"
	Paused          QueueOption = "paused"
	Expanded        QueueOption = "expanded"
	Editing         QueueOption = "editing"
	StopSelected    QueueOption = "stop_selected"
	ForwardSelected QueueOption = "forward_selected"
	RemoveSelected  QueueOption = "remove_selected"
	ClearSelected   QueueOption = "clear_selected"
	Test            QueueOption = "test"
)

type Queue struct {
	ClientID  string        `json:"client_id"`
	GuildID   string        `json:"guild_id"`
	MessageID string        `json:"message_id"`
	ChannelID string        `json:"channel_id"`
	Offset    int           `json:"offset"`
	Limit     int           `json:"limit"`
	Options   []QueueOption `json:"options"`
	Songs     []*Song       `json:"songs"`
	HeadSong  *Song         `json:"head_song"`
	Size      int           `json:"size"`
}

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

func GetQueueOptions() []QueueOption {
	return []QueueOption{
		Loop,
		Paused,
		Expanded,
		Editing,
		StopSelected,
		ForwardSelected,
		RemoveSelected,
		ClearSelected,
	}
}

func QueueOptionsToStringSlice(opts []QueueOption) []string {
	x := make([]string, 0, len(opts))
	for _, o := range opts {
		x = append(x, string(o))
	}
	return x
}

func StringSliceToQueueOptions(opts []string) []QueueOption {
	x := make([]QueueOption, 0, len(opts))
	for _, o := range opts {
		x = append(x, QueueOption(o))
	}
	return x
}
