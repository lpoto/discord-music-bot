package queue_test

import (
	"database/sql"
	"discord-music-bot/datastore/queue"
	"discord-music-bot/model"
	"fmt"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type QueueStoreTestSuite struct {
	db    *sql.DB
	store *queue.QueueStore
	suite.Suite
}

// SetupSuite runs when the suite is initialized and
// connects to the database and initialized the queue store.
func (s *QueueStoreTestSuite) SetupSuite() {
	db, err := sql.Open(
		"postgres",
		"host=postgres port=5432 user=postgres password=postgres "+
			"dbname=discord_bot_test sslmode=disable",
	)
	s.NoError(err)

	s.db = db
	s.store = queue.NewQueueStore(db, logrus.StandardLogger())
}

// SetupTest runs before every test and initializes the store.
func (s *QueueStoreTestSuite) SetupTest() {
	err := s.store.Destroy()
	s.NoError(err)
	err = s.store.Init()
	s.NoError(err)
}

// TearDownSuite runs after all tests have been run and destroys
// the queue store and closes database connection.
func (s *QueueStoreTestSuite) TearDownSuite() {
	err := s.store.Destroy()
	s.NoError(err)

	err = s.db.Close()
	s.NoError(err)
}

// TestIntegrationPersistFetchQueue first persists a queue then
// fetches it and checks it's fields.
func (s *QueueStoreTestSuite) TestIntegrationQueueCRUD() {
	queue := &model.Queue{
		ClientID:  "CLIENT-ID-TEST",
		GuildID:   "GUILD-ID-TEST",
		MessageID: "MESSAGE-ID-TEST",
		ChannelID: "CHANNEL-ID-TEST",
		Offset:    0,
		Limit:     10,
	}

	// First persist the queue
	err := s.store.PersistQueue(queue)
	s.NoError(err)

	// Should successfully fetch the persisted queue
	queue2, err := s.store.GetQueue(queue.ClientID, queue.GuildID)
	s.NoError(err)
	s.Equal(queue.GuildID, queue2.GuildID)
	s.Equal(queue.ClientID, queue2.ClientID)
	s.Equal(queue.ChannelID, queue2.ChannelID)
	s.Equal(queue.MessageID, queue2.MessageID)
	s.Equal(10, queue2.Limit)
	s.Equal(0, queue2.Offset)
	s.Equal((*model.Song)(nil), queue2.HeadSong)
	s.Equal(0, queue2.InactiveSize)
	s.Equal(0, queue2.Size)

	queue.MessageID = "MESSAGE-ID-TEST2"
	queue.Limit = 5
	// Should successfully update the queue
	err = s.store.UpdateQueue(queue)
	s.NoError(err)

	// Should successfully fetch the persisted queue
	queue2, err = s.store.GetQueue(queue.ClientID, queue.GuildID)
	s.NoError(err)
	s.Equal("MESSAGE-ID-TEST2", queue2.MessageID)
	s.Equal(5, queue.Limit)

	// Remove the queue
	err = s.store.RemoveQueue(queue.ChannelID, queue.GuildID)
	s.NoError(err)

	// Try to fetch the queue again, it should return error
	_, err = s.store.GetQueue(queue.ChannelID, queue.GuildID)
	s.Error(err)
	s.Equal("sql: no rows in result set", err.Error())
}

// TestIntegrationFindAllQueues creates queues then
// fetches all of them and checks their data.
func (s *QueueStoreTestSuite) TestIntegrationFindAllQueues() {
	// Should not fetch any queues
	queried_queues, err := s.store.FindAllQueues()
	s.NoError(err)
	s.Len(queried_queues, 0)

	queues := make([]*model.Queue, 0)
	for i := 1; i < 10; i++ {
		queue := &model.Queue{
			ClientID:  fmt.Sprintf("CLIENT-ID-TEST%d", i),
			GuildID:   fmt.Sprintf("GUILD-ID-TEST%d", i),
			MessageID: fmt.Sprintf("MESSAGE-ID-TEST%d", i),
			ChannelID: fmt.Sprintf("CHANNEL-ID-TEST%d", i),
			Limit:     10,
			Offset:    0,
		}
		queues = append(queues, queue)
		err := s.store.PersistQueue(queue)
		s.NoError(err)
	}

	// Should fetch all inserted queues
	queried_queues, err = s.store.FindAllQueues()
	s.NoError(err)
	s.Len(queried_queues, len(queues))
	for _, queue := range queues {
		found := false
		for _, queried_queue := range queried_queues {
			if queue.ClientID == queried_queue.ClientID &&
				queue.GuildID == queried_queue.GuildID &&
				queue.MessageID == queried_queue.MessageID &&
				queue.ChannelID == queried_queue.ChannelID &&
				queue.Offset == queried_queue.Offset &&
				queue.Limit == queried_queue.Limit {
				found = true
				break

			}
		}
		s.Equal(true, found)
	}
}

// TestIntegrationAddRemoveQueueOptions creates a queue then adds and
// removes options from it.
func (s *QueueStoreTestSuite) TestIntegrationAddRemoveQueueOptions() {
	queue := &model.Queue{
		ClientID:  "CLIENT-ID-TEST",
		GuildID:   "GUILD-ID-TEST",
		MessageID: "MESSAGE-ID-TEST",
		ChannelID: "CHANNEL-ID-TEST",
		Limit:     10,
		Offset:    0,
	}
	err := s.store.PersistQueue(queue)
	s.NoError(err)
	s.Len(queue.Options, 0)

	err = s.store.PersistQueueOptions(
		queue.ClientID,
		queue.GuildID,
		model.LoopOption(),
		model.PausedOption(),
	)
	s.NoError(err)

	// Make sure the added options are there
	queue, err = s.store.GetQueue(queue.ClientID, queue.GuildID)
	s.NoError(err)
	s.Len(queue.Options, 2)

	found := 0
	for _, v := range queue.Options {
		if v.Name == model.Loop {
			found++
		} else if v.Name == model.Paused {
			found++
		}
	}
	s.Equal(2, found)

	err = s.store.RemoveQueueOptions(queue.ClientID, queue.GuildID, model.Paused)
	s.NoError(err)

	options, err := s.store.GetOptionsForQueue(queue.ClientID, queue.GuildID)
	s.NoError(err)
	s.Len(options, 1)
	s.Equal(options[0].Name, model.Loop)
}

// TestQueueStorageTestSuite runs all tests under
// the QueueStoreTestSuite suite.
func TestQueueStorageTestSuite(t *testing.T) {
	suite.Run(t, new(QueueStoreTestSuite))
}
