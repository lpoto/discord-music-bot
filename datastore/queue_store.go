package datastore

import (
	"discord-music-bot/model"

	"github.com/lib/pq"
)

// PersistQueue saves the provided queue and returns the inserted queue.
// Returns error if the queue,
// identified by the same clientID and guildID, already exists.
func (datastore *Datastore) PersistQueue(queue *model.Queue) (*model.Queue, error) {
	datastore.Trace(
		"Persisting queue: clientID=%s, guildID=%s, messageID=%s",
		queue.ClientID, queue.GuildID, queue.MessageID,
	)

	newQueue := &model.Queue{}
	opts := make([]string, 0)

	if err := datastore.QueryRow(
		`
        INSERT INTO "queue" (
            client_id, guild_id, channel_id, message_id,
            "offset", "limit", options
        ) VALUES
            ($1, $2, $3, $4, $5, $6, $7)
        RETURNING *;
        `,
		queue.ClientID,
		queue.GuildID,
		queue.ChannelID,
		queue.MessageID,
		queue.Offset,
		queue.Limit,
		pq.Array(model.QueueOptionsToStringSlice(queue.Options)),
	).Scan(
		&newQueue.ClientID, &newQueue.GuildID,
		&newQueue.ChannelID, &newQueue.MessageID, &newQueue.Offset,
		&newQueue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Error(
			"Error when persisting queue: %v", err,
		)
		return nil, err
	} else {
		datastore.Trace("Successfully persisted the queue")

		newQueue.Options = model.StringSliceToQueueOptions(opts)
		return newQueue, nil
	}
}

// UpdateQueue updates the provided queue. This does not update
// the queue's clientID or guildID.
// Returns error if the queue does not exist in the databse.
func (datastore *Datastore) UpdateQueue(queue *model.Queue) (*model.Queue, error) {
	datastore.Trace(
		"Updatind queue: clientID=%s, guildID=%s, messageID=%s",
		queue.ClientID, queue.GuildID, queue.MessageID,
	)

	newQueue := &model.Queue{}
	opts := make([]string, 0)

	if err := datastore.QueryRow(
		`
        UPATE "queue" 
        SET message_id = $3,
            offset = $4,
            limit = $5,
            options = $6
        WHERE "queue".client_id = $1 AND
            "queue".guild_id = $2
        RETURNING *;
        `,
		queue.ClientID,
		queue.GuildID,
		queue.ChannelID,
		queue.MessageID,
		queue.Offset,
		queue.Limit,
		pq.Array(model.QueueOptionsToStringSlice(queue.Options)),
	).Scan(
		&newQueue.ClientID, &newQueue.GuildID,
		&newQueue.ChannelID, &newQueue.MessageID, &newQueue.Offset,
		&newQueue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Error(
			"Error when updating queue: %v", err,
		)
		return nil, err
	} else {
		datastore.Trace("Successfully updated the queue")
		newQueue.Options = model.StringSliceToQueueOptions(opts)
		return datastore.GetQueueData(newQueue)
	}
}

// RemoveQueue removes the queue identified by the clientID and guildID
// from the database. Returns error if no such queue exists.
func (datastore *Datastore) RemoveQueue(clientID string, guildID string) error {
	datastore.Trace(
		"Removing queue: clientID=%s, guildID=%s",
		clientID, guildID,
	)
	if _, err := datastore.Exec(
		`
        DELETE FROM "queue"
        WHERE "queue".guild_id = $1 AND
            "queue".client_id = $2;
        `,
		guildID,
		clientID,
	); err != nil {
		datastore.Error(
			"Error when removing the queue: %v", err,
		)

	}
	return nil
}

// GetQueue fetches the queue identified by the provided clientID and guildID.
// Returns error if no such queue exists.
func (datastore *Datastore) GetQueue(clientID string, guildID string) (*model.Queue, error) {
	datastore.Trace(
		"Fetching queue: clientID=%s, guildID=%s",
		clientID, guildID,
	)
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
		&queue.ChannelID, &queue.MessageID, &queue.Offset,
		&queue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Trace(
			"Error when fetching the queue: %v", err,
		)
		return nil, err
	}
	datastore.Trace("Successfully fetched the queue")

	queue.Options = model.StringSliceToQueueOptions(opts)

	return datastore.GetQueueData(queue)
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
		queue.Offset,
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
	datastore.WithField("TableName", "queue").Debug("Creating psql table")

	if _, err := datastore.Exec(
		`
        CREATE TABLE IF NOT EXISTS "queue" (
            client_id VARCHAR,
            guild_id VARCHAR,
            channel_id VARCHAR NOT NULL,
            message_id VARCHAR NOT NULL,
            "offset" INTEGER NOT NULL DEFAULT '0',
            "limit" INTEGER NOT NULL DEFAULT '10',
            options TEXT[] DEFAULT ARRAY[]::TEXT[],
            UNIQUE (client_id, guild_id),
            PRIMARY KEY (client_id, guild_id)
        );
        `,
	); err != nil {
		datastore.Error("Error when creating table 'queue': %v", err)
		return err
	}
	datastore.WithField("TableName", "queue").Trace(
		"Successfully created psql table",
	)
	return nil
}
