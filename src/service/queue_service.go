package service

import (
	"discord-music-bot/model"
	"math"
)

// IncrementQueueOffset increments the provided queue's
// offset by it's limit. If the new offset is larger than
// the size of the queue, the offset is wrapped back to 0.
// The provided queue is expected to have all the data fetched.
func (service *Service) IncrementQueueOffset(queue *model.Queue) {
	queue.Offset += queue.Limit
	if queue.Offset >= queue.Size {
		queue.Offset = 0
	}
}

// DecrementQueueOffset decrements the provided queue's
// offset by it's limit. If the new offset is less than 0,
// the offset is maximized.
func (service *Service) DecrementQueueOffset(queue *model.Queue) {
	queue.Offset -= queue.Limit
	if queue.Offset < 0 {
		y := queue.Size
		if queue.Size%queue.Limit == 0 {
			y = queue.Size - 1
		}
		x := int(math.Round(float64(y) / float64(queue.Limit)))
		queue.Offset = x * queue.Limit
	}
}

// AddOrRemoveQueueOption adds the provided option to the provide queue
// if the queue does not already have the option set, if it has, the
// option is removed
func (service *Service) AddOrRemoveQueueOption(queue *model.Queue, option model.QueueOption) {
	opts := make([]model.QueueOption, 0)
	for _, o := range queue.Options {
		if o != option {
			opts = append(opts, o)
		}
	}
	if len(opts) == len(queue.Options) {
		opts = append(opts, option)
	}
	queue.Options = opts
}
