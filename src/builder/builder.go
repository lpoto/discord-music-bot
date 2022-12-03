package builder

import (
	"discord-music-bot/builder/queue"
	"discord-music-bot/builder/song"
)

type Configuration struct {
	Queue *queue.Configuration `yaml:"Queue" validate:"required"`
}

type Builder struct {
	queue *queue.QueueBuilder
	song  *song.SongBuilder
}

// NewBuilder constructs an object that handles building
// the queue's embed, components, ... based on it's current state
func NewBuilder(config *Configuration) *Builder {
	b := &Builder{
		song: song.NewSongBuilder(),
	}
	b.queue = queue.NewQueueBuidler(config.Queue, b.song)
	return b
}

// Queue returns an object that handles building
// queue objects and mapping them to embeds.
func (builder *Builder) Queue() *queue.QueueBuilder {
	return builder.queue
}

// Song return an object  that handles building
// songs and formatting their names etc.
func (builder *Builder) Song() *song.SongBuilder {
	return builder.song
}
