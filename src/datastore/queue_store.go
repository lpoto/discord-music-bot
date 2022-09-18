package datastore

import (
	"discord-music-bot/model"
	"time"

	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// PersistQueue saves the provided queue and returns the inserted queue.
// Returns error if the queue,
// identified by the same clientID and guildID, already exists.
func (datastore *Datastore) PersistQueue(queue *model.Queue) (*model.Queue, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": queue.ClientID,
		"GuildID":  queue.GuildID,
	}).Tracef("[%d]Start: Persist queue", i)

	newQueue := &model.Queue{}
	opts := make([]string, 0)

	if err := datastore.QueryRow(
		`
        INSERT INTO "queue" (
            client_id, guild_id, message_id, channel_id, "offset", "limit", options
        ) VALUES
            ($1, $2, $3, $4, $5, $6, $7)
        RETURNING *;
        `,
		queue.ClientID,
		queue.GuildID,
		queue.MessageID,
		queue.ChannelID,
		queue.Offset,
		queue.Limit,
		pq.Array(model.QueueOptionsToStringSlice(queue.Options)),
	).Scan(
		&newQueue.ClientID, &newQueue.GuildID,
		&newQueue.MessageID, &newQueue.ChannelID,
		&newQueue.Offset,
		&newQueue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return nil, err
	} else {
		newQueue.Options = model.StringSliceToQueueOptions(opts)
		datastore.WithField(
			"Latency", time.Since(t),
		).Tracef("[%d]Done : persisted the queue", i)

		return newQueue, nil
	}
}

// UpdateQueue updates the provided queue. This does not update
// the queue's clientID or guildID.
// Returns error if the queue does not exist in the databse.
func (datastore *Datastore) UpdateQueue(queue *model.Queue) (*model.Queue, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": queue.ClientID,
		"GuildID":  queue.GuildID,
	}).Tracef("[%d]Start: Update queue", i)

	newQueue := &model.Queue{}
	opts := make([]string, 0)

	if err := datastore.QueryRow(
		`
        UPATE "queue" 
        SET offset = $3,
            limit = $4,
            options = $5
            message_id = $6
            channel_id = $7
        WHERE "queue".client_id = $1 AND
            "queue".guild_id = $2
        RETURNING *;
        `,
		queue.ClientID,
		queue.GuildID,
		queue.Offset,
		queue.Limit,
		pq.Array(model.QueueOptionsToStringSlice(queue.Options)),
		queue.MessageID,
		queue.ChannelID,
	).Scan(
		&newQueue.ClientID, &newQueue.GuildID,
		&newQueue.MessageID, &newQueue.ChannelID,
		&newQueue.Offset,
		&newQueue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return nil, err
	} else {
		newQueue.Options = model.StringSliceToQueueOptions(opts)

		datastore.WithField(
			"Latency", time.Since(t),
		).Tracef("[%d]Done : Queeu updated", i)

		return datastore.GetQueueData(newQueue)
	}
}

// RemoveQueue removes the queue identified by the clientID and guildID
// from the database. Returns error if no such queue exists.
func (datastore *Datastore) RemoveQueue(clientID string, guildID string) error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Remove queue", i)

	if _, err := datastore.Exec(
		`
        DELETE FROM "queue"
        WHERE "queue".guild_id = $1 AND
            "queue".client_id = $2;
        `,
		guildID,
		clientID,
	); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return err
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Queue removed", i)
	return nil
}

// GetQueue fetches the queue identified by the provided clientID and guildID.
// Fetches all the required song data for the queue.
// Returns error if no such queue exists.
func (datastore *Datastore) GetQueue(clientID string, guildID string) (*model.Queue, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Fetch queue", i)

	queue, err := datastore.FindQueue(clientID, guildID)
	if err != nil {
		return nil, err
	}

	queue, err = datastore.GetQueueData(queue)

	if err != nil {
		return nil, err
	}

	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Queue fetched", i)

	return queue, nil
}

// FindQueue searches for a queue with the provided clientID and guildID.
// Returns error if no such queue exists.
// WARNING: This does not fetch any song data for the found queues.
func (datastore *Datastore) FindQueue(clientID string, guildID string) (*model.Queue, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Find queue", i)

	queue := &model.Queue{}
	opts := make([]string, 0)

	if err := datastore.QueryRow(
		`
        SELECT * FROM "queue"
        WHERE "queue".guild_id = $1 AND
            "queue".client_id = $2;
        `,
		guildID,
		clientID,
	).Scan(
		&queue.ClientID, &queue.GuildID,
		&queue.MessageID, &queue.ChannelID,
		&queue.Offset, &queue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return nil, err
	}
	queue.Options = model.StringSliceToQueueOptions(opts)

	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Queue found", i)
	return queue, nil
}

// FindAllQueue returns all queues in the datastore.
// WARNING: This does not fetch any song data for the found queues.
func (datastore *Datastore) FindAllQueues() ([]*model.Queue, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.Tracef("[%d]Start: Find all queues", i)

	queues := make([]*model.Queue, 0)

	if rows, err := datastore.Query(
		`SELECT * FROM "queue"`,
	); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return nil, err
	} else {
		for rows.Next() {
			queue := &model.Queue{}
			opts := make([]string, 0)
			if err := rows.Scan(
				&queue.ClientID, &queue.GuildID,
				&queue.MessageID, &queue.ChannelID,
				&queue.Offset, &queue.Limit, pq.Array(&opts),
			); err != nil {
				datastore.Tracef(
					"[%d]Error: %v", i, err,
				)
			}
			queue.Options = model.StringSliceToQueueOptions(opts)
			queues = append(queues, queue)
		}
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Queues found", i)
	return queues, nil
}

// GetQueueData fetches the queue's songs,
// limited by the queue's offset and limit, and the total
// size of the queue.
func (datastore *Datastore) GetQueueData(queue *model.Queue) (*model.Queue, error) {
	if headSongs, err := datastore.GetSongsForQueue(
		queue.ClientID,
		queue.GuildID,
		0, 1,
	); err == nil {
		if len(headSongs) > 0 {
			queue.HeadSong = headSongs[0]
		}
	} else {
		return nil, err
	}
	if songs, err := datastore.GetSongsForQueue(
		queue.ClientID,
		queue.GuildID,
		queue.Offset+1,
		queue.Limit,
	); err == nil {

		queue.Songs = songs
		queue.Size = datastore.GetSongCountForQueue(
			queue.ClientID,
			queue.GuildID,
		)
	} else {
		return nil, err
	}
	return queue, nil
}

func (datastore *Datastore) createQueueTable() error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithField("TableName", "queue").Debugf(
		"[%d]Start: Create psql table (if not exists)",
		i,
	)

	if _, err := datastore.Exec(
		`
        CREATE TABLE IF NOT EXISTS "queue" (
            client_id VARCHAR,
            guild_id VARCHAR,
            message_id VARCHAR NOT NULL,
            channel_id VARCHAR NOT NULL,
            "offset" INTEGER NOT NULL DEFAULT '0',
            "limit" INTEGER NOT NULL DEFAULT '10',
            options TEXT[] DEFAULT ARRAY[]::TEXT[],
            UNIQUE (client_id, guild_id),
            PRIMARY KEY (client_id, guild_id)
        );
        `,
	); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField("Latency", time.Since(t)).Tracef(
		"[%d]Done : psql table created", i,
	)
	return nil
}
