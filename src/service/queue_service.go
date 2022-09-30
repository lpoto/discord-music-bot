package service

import (
	"discord-music-bot/model"
)

// IncrementQueueOffset increments the provided queue's
// offset by it's limit. If the new offset is larger than
// the size of the queue, the offset is wrapped back to 0.
// The provided queue is expected to have all the data fetched.
func (service *Service) IncrementQueueOffset(queue *model.Queue) {
	queue.Offset += queue.Limit
	if queue.Offset+1 >= queue.Size {
		queue.Offset = 0
	}
}

// DecrementQueueOffset decrements the provided queue's
// offset by it's limit. If the new offset is less than 0,
// the offset is maximized.
func (service *Service) DecrementQueueOffset(queue *model.Queue) {
	queue.Offset -= queue.Limit
	if queue.Offset < 0 {
		i := queue.Size - 1
		j := i % queue.Limit
		if j == 0 {
			j = queue.Limit
		}
		queue.Offset = i - j
	}
}
