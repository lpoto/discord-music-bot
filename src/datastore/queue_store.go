package datastore

import (
	"discord-music-bot/model"

	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// PersistQueue saves the provided queue and returns the inserted queue.
// Returns error if the queue,
// identified by the same clientID and guildID, already exists.
func (datastore *Datastore) PersistQueue(queue *model.Queue) (*model.Queue, error) {
	datastore.WithFields(log.Fields{
		"ClientID": queue.ClientID,
		"GuildID":  queue.GuildID,
	}).Trace("Persisting queue")

	newQueue := &model.Queue{}
	opts := make([]string, 0)

	if err := datastore.QueryRow(
		`
        INSERT INTO "queue" (
            client_id, guild_id, message_id, "offset", "limit", options
        ) VALUES
            ($1, $2, $3, $4, $5, $6)
        RETURNING *;
        `,
		queue.ClientID,
		queue.GuildID,
		queue.MessageID,
		queue.Offset,
		queue.Limit,
		pq.Array(model.QueueOptionsToStringSlice(queue.Options)),
	).Scan(
		&newQueue.ClientID, &newQueue.GuildID, &newQueue.MessageID,
		&newQueue.Offset,
		&newQueue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Tracef(
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
	datastore.WithFields(log.Fields{
		"ClientID": queue.ClientID,
		"GuildID":  queue.GuildID,
	}).Trace("Updating queue")

	newQueue := &model.Queue{}
	opts := make([]string, 0)

	if err := datastore.QueryRow(
		`
        UPATE "queue" 
        SET offset = $3,
            limit = $4,
            options = $5
            message_id = $6
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
	).Scan(
		&newQueue.ClientID, &newQueue.GuildID, &newQueue.MessageID,
		&newQueue.Offset,
		&newQueue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Tracef(
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
	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Trace("Removing queue")

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
			"Error when removing the queue: %v", err,
		)
	}
	datastore.Trace("Successfully removed the queue")
	return nil
}

// GetQueue fetches the queue identified by the provided clientID and guildID.
// Fetches all the required song data for the queue.
// Returns error if no such queue exists.
func (datastore *Datastore) GetQueue(clientID string, guildID string) (*model.Queue, error) {
	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Trace("Fetching queue")

	queue, err := datastore.FindQueue(clientID, guildID)
	if err != nil {
		return nil, err
	}
	datastore.Trace("Successfully fetched the queue")

	return datastore.GetQueueData(queue)
}

// FindQueue searches for a queue with the provided clientID and guildID.
// Returns error if no such queue exists.
// WARNING: This does not fetch any song data for the found queues.
func (datastore *Datastore) FindQueue(clientID string, guildID string) (*model.Queue, error) {
	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Trace("Finding a queue")

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
		&queue.ClientID, &queue.GuildID, &queue.MessageID,
		&queue.Offset, &queue.Limit, pq.Array(&opts),
	); err != nil {
		datastore.Tracef(
			"Error when finding queues: %v", err,
		)
		return nil, err
	}
	queue.Options = model.StringSliceToQueueOptions(opts)

	datastore.Trace("Successfully found the queue")
	return queue, nil
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
	datastore.WithField("TableName", "queue").Debug(
		"Creating psql table (if not exists)",
	)

	if _, err := datastore.Exec(
		`
        CREATE TABLE IF NOT EXISTS "queue" (
            client_id VARCHAR,
            guild_id VARCHAR,
            message_id VARCHAR NOT NULL,
            "offset" INTEGER NOT NULL DEFAULT '0',
            "limit" INTEGER NOT NULL DEFAULT '10',
            options TEXT[] DEFAULT ARRAY[]::TEXT[],
            UNIQUE (client_id, guild_id),
            PRIMARY KEY (client_id, guild_id)
        );
        `,
	); err != nil {
		datastore.Tracef("Error when creating table 'queue': %v", err)
		return err
	}
	datastore.WithField("TableName", "queue").Trace(
		"Successfully created psql table",
	)
	return nil
}
