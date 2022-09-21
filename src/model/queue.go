package model

type QueueOptionName string

const (
	Loop     QueueOptionName = "loop"     // When loop option is set, songs are pushed to the back  of the queue instead of being removed
	Paused   QueueOptionName = "paused"   // When paused option is set, the queue's audioplayer is paused
	Inactive QueueOptionName = "inactive" // When inactive, only join button is displayed
)

type QueueOption struct {
	Name QueueOptionName `json:"name"` // Name of the option, set for the queue
}

type Queue struct {
	ClientID     string         `json:"client_id"`     // Id of the bot that created the queue
	GuildID      string         `json:"guild_id"`      // Id of the discord server in which the queue has been created
	MessageID    string         `json:"message_id"`    // Id of the queue's message in a discord channel
	ChannelID    string         `json:"channel_id"`    // Id of the channel in which the queue has been created
	Offset       int            `json:"offset"`        // Current offset of the displayed songs in the queue
	Limit        int            `json:"limit"`         // Number of songs displayed at once
	Options      []*QueueOption `json:"options"`       // A list of options currently added to the queue
	Songs        []*Song        `json:"songs"`         // Currently displayed songs
	HeadSong     *Song          `json:"head_song"`     // A queue's song with the minimum position
	InactiveSize int            `json:"inactive_size"` // Total number of inactive songs (removed from the queue) that belong to the queue
	Size         int            `json:"size"`          // Total number of songs that belong to the queue
}

func LoopOption() *QueueOption {
	return &QueueOption{
		Name: Loop,
	}
}

func PausedOption() *QueueOption {
	return &QueueOption{
		Name: Paused,
	}
}

func InactiveOption() *QueueOption {
	return &QueueOption{
		Name: Inactive,
	}
}
