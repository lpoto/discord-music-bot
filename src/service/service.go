package service

import (
	"discord-music-bot/service/queue"
	"discord-music-bot/service/song"
)

type Service struct {
	queue *queue.QueueService
	song  *song.SongService
}

// NewService constructs an object that holds different
// services for handling some of the logic.
func NewService() *Service {
	return &Service{
		queue: queue.NewQueueService(),
		song:  song.NewSongService(),
	}
}

// Queue returns the service for handling the logic
// behind manipulating queues.
func (service *Service) Queue() *queue.QueueService {
	return service.queue
}

// Song returns the service for handling the logic
// behind manipulating songs.
func (service *Service) Song() *song.SongService {
	return service.song
}
