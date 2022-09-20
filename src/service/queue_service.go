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
		i := queue.Size - 1
		j := i % queue.Limit
		if j == 0 {
			j = queue.Limit
		}
		queue.Offset = i - j
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

// RemoveQueueOption removes the provied options from the queue
func (service *Service) RemoveQueueOption(queue *model.Queue, option model.QueueOption) {
	opts := make([]model.QueueOption, 0)
	for _, o := range queue.Options {
		if o != option {
			opts = append(opts, o)
		}
	}
	queue.Options = opts
}

// AddQueueOption adds the provied option to the queue,
// if it does not already contain it
func (service *Service) AddQueueOption(queue *model.Queue, option model.QueueOption) {
	opts := make([]model.QueueOption, 0)
	for _, o := range queue.Options {
		if o != option {
			opts = append(opts, o)
		}
	}
	opts = append(opts, option)
	queue.Options = opts
}
