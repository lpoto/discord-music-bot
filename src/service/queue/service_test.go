package queue_test

import (
	"discord-music-bot/model"
	"discord-music-bot/service/queue"
	"testing"

	"github.com/stretchr/testify/suite"
)

type QueueServiceTestSuite struct {
	suite.Suite
	service *queue.QueueService
}

// SetupSuite runs on suit init and creates
// the queue service.
func (s *QueueServiceTestSuite) SetupSuite() {
	s.service = queue.NewQueueService()
}

// TestUnitIncrementQueueOffset tests that
// IncrementQueueOffset() properly increments the queue's offset.
func (s *QueueServiceTestSuite) TestUnitIncrementQueueOffset() {
	queue := &model.Queue{
		ClientID:  "CLIENT-ID-TEST",
		GuildID:   "GUILD-ID-TEST",
		MessageID: "MESSAGE-ID-TEST",
		ChannelID: "CHANNEL-ID-TEST",
		Size:      13,
		Limit:     10,
		Offset:    0,
	}
	s.service.IncrementQueueOffset(queue)
	// No other fields should be changed
	s.Equal(queue.ClientID, "CLIENT-ID-TEST")
	s.Equal(queue.GuildID, "GUILD-ID-TEST")
	s.Equal(queue.ChannelID, "CHANNEL-ID-TEST")
	s.Equal(queue.MessageID, "MESSAGE-ID-TEST")

	s.Equal(10, queue.Offset)

	s.service.IncrementQueueOffset(queue)
	s.Equal(0, queue.Offset)

	// Wraps on 21 instead of 20 as
	// it counts as head song + 20 songs
	// HeadSong is displayed separately.
	queue.Size = 21
	s.service.IncrementQueueOffset(queue)
	s.service.IncrementQueueOffset(queue)
	s.Equal(0, queue.Offset)

	queue.Size = 22
	s.service.IncrementQueueOffset(queue)
	s.service.IncrementQueueOffset(queue)
	s.Equal(20, queue.Offset)

	queue.Limit = 1
	s.service.IncrementQueueOffset(queue)
	s.Equal(0, queue.Offset)

}

// TestUnitDecrementQueueOffset tests that
// DecrementQueueOffset() properly decrements the queue's offset.
func (s *QueueServiceTestSuite) TestUnitDecrementQueueOffset() {
	queue := &model.Queue{
		ClientID:  "CLIENT-ID-TEST",
		GuildID:   "GUILD-ID-TEST",
		MessageID: "MESSAGE-ID-TEST",
		ChannelID: "CHANNEL-ID-TEST",
		Size:      13,
		Limit:     10,
		Offset:    0,
	}
	s.service.DecrementQueueOffset(queue)
	// No other fields should be changed
	s.Equal(queue.ClientID, "CLIENT-ID-TEST")
	s.Equal(queue.GuildID, "GUILD-ID-TEST")
	s.Equal(queue.ChannelID, "CHANNEL-ID-TEST")
	s.Equal(queue.MessageID, "MESSAGE-ID-TEST")
	s.Equal(10, queue.Offset)

	s.service.DecrementQueueOffset(queue)
	s.Equal(0, queue.Offset)

	queue.Size = 11
	s.service.DecrementQueueOffset(queue)
	// Wraps to 0 on 1 as its counted as head song + 10 songs
	// Head song is displayed separately
	s.Equal(0, queue.Offset)

	queue.Size = 22
	s.service.DecrementQueueOffset(queue)
	s.Equal(20, queue.Offset)
	s.service.DecrementQueueOffset(queue)
	s.Equal(10, queue.Offset)

	queue.Limit = 1
	s.service.DecrementQueueOffset(queue)
	s.Equal(9, queue.Offset)

	queue.Offset = 0
	queue.Size = 20
	s.service.DecrementQueueOffset(queue)
	s.Equal(18, queue.Offset)

}

// TestQueueServiceTestSuite runs all tests under
// the QueueServiceTestSuite
func TestQueueServiceTestSuite(t *testing.T) {
	suite.Run(t, new(QueueServiceTestSuite))
}
