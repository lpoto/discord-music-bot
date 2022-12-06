package queue

import (
	"database/sql"
	"discord-music-bot/model"
	"fmt"
	"time"

	"github.com/lib/pq"

	log "github.com/sirupsen/logrus"
)

type QueueStore struct {
	log *log.Logger
	db  *sql.DB
	idx int
}

// NewQueueStore creates an object that handles
// persisting and removing Queues in postgres database.
func NewQueueStore(db *sql.DB, log *log.Logger) *QueueStore {
	return &QueueStore{
		db:  db,
		log: log,
		idx: 0,
	}
}

// Init creates the required tables for the Queue store.
func (store *QueueStore) Init() error {
	if err := store.createQueueTable(); err != nil {
		return err
	}
	return store.createQueueOptionTable()
}

// Destroy drops the created tables for the Queue store.
func (store *QueueStore) Destroy() error {
	if err := store.dropQueueTable(); err != nil {
		return err
	}
	return store.dropQueueOptionTable()
}

// PersistQueue saves the provided queue and returns the inserted queue.
// Returns error if the queue,
// identified by the same clientID and guildID, already exists.
func (store *QueueStore) PersistQueue(queue *model.Queue) error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": queue.ClientID,
		"GuildID":  queue.GuildID,
	}).Tracef("[Q%d]Start: Persist queue", i)

	newQueue := &model.Queue{}

	if err := store.db.QueryRow(
		`
        INSERT INTO "queue" (
            client_id, guild_id, message_id, channel_id, "offset", "limit"
        ) VALUES
            ($1, $2, $3, $4, $5, $6)
        RETURNING *;
        `,
		queue.ClientID,
		queue.GuildID,
		queue.MessageID,
		queue.ChannelID,
		queue.Offset,
		queue.Limit,
	).Scan(
		&newQueue.ClientID, &newQueue.GuildID,
		&newQueue.MessageID, &newQueue.ChannelID,
		&newQueue.Offset,
		&newQueue.Limit,
	); err != nil {
		store.log.Tracef(
			"[Q%d]Error: %v", i, err,
		)
		return err
	} else {
		store.log.WithField(
			"Latency", time.Since(t),
		).Tracef("[Q%d]Done : persisted the queue", i)

		return store.PersistQueueOptions(
			queue.ClientID,
			queue.GuildID,
			queue.Options...,
		)
	}
}

// UpdateQueue updates the provided queue. This does not update
// the queue's clientID or guildID.
// Returns error if the queue does not exist in the databse.
func (store *QueueStore) UpdateQueue(queue *model.Queue) error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": queue.ClientID,
		"GuildID":  queue.GuildID,
	}).Tracef("[Q%d]Start: Update queue", i)

	if _, err := store.db.Exec(
		`
        UPDATE "queue"
        SET "offset" = $3,
            "limit" = $4,
            message_id = $5,
            channel_id = $6
        WHERE "queue".client_id = $1 AND
            "queue".guild_id = $2;
        `,
		queue.ClientID,
		queue.GuildID,
		queue.Offset,
		queue.Limit,
		queue.MessageID,
		queue.ChannelID,
	); err != nil {
		store.log.Tracef(
			"[Q%d]Error: %v", i, err,
		)
		return err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[Q%d]Done : Queue updated", i)
	return nil
}

// RemoveQueue removes the queue identified by the clientID and guildID
// from the database. Returns error if no such queue exists.
func (store *QueueStore) RemoveQueue(clientID string, guildID string) error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[Q%d]Start: Remove queue", i)

	if _, err := store.db.Exec(
		`
        DELETE FROM "queue"
        WHERE "queue".guild_id = $1 AND
            "queue".client_id = $2;
        `,
		guildID,
		clientID,
	); err != nil {
		store.log.Tracef(
			"[Q%d]Error: %v", i, err,
		)
		return err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[Q%d]Done : Queue removed", i)
	return nil
}

// GetQueue fetches the queue identified by the provided clientID and guildID.
// Returns error if no such queue exists.
func (store *QueueStore) GetQueue(clientID string, guildID string) (*model.Queue, error) {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[Q%d]Start: Find queue", i)

	queue := &model.Queue{}

	if err := store.db.QueryRow(
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
		&queue.Offset, &queue.Limit,
	); err != nil {
		store.log.Tracef(
			"[Q%d]Error: %v", i, err,
		)
		return nil, err
	}

	opts, err := store.GetOptionsForQueue(clientID, guildID)
	if err != nil {
		return nil, err
	}
	queue.Options = opts

	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[Q%d]Done : Queue found", i)
	return queue, nil
}

// FindAllQueue returns all queues in the store.
func (store *QueueStore) FindAllQueues() ([]*model.Queue, error) {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.Tracef("[Q%d]Start: Find all queues", i)

	queues := make([]*model.Queue, 0)

	if rows, err := store.db.Query(
		`SELECT * FROM "queue"`,
	); err != nil {
		store.log.Tracef(
			"[Q%d]Error: %v", i, err,
		)
		return nil, err
	} else {
		for rows.Next() {
			queue := &model.Queue{}
			if err := rows.Scan(
				&queue.ClientID, &queue.GuildID,
				&queue.MessageID, &queue.ChannelID,
				&queue.Offset, &queue.Limit,
			); err != nil {
				store.log.Tracef(
					"[Q%d]Error: %v", i, err,
				)
			}
			queues = append(queues, queue)
		}
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[Q%d]Done : Queues found", i)
	return queues, nil
}

// PersistQueueOptions inserts all of the provided queue options to the
// database in a single query. Options with name equal to some other
// already persisted option (for the same queue) are not persisted.
func (store *QueueStore) PersistQueueOptions(clientID string, guildID string, options ...*model.QueueOption) error {
	if options == nil || len(options) < 1 {
		return nil
	}
	if _, err := store.GetQueue(clientID, guildID); err != nil {
		return nil
	}
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[Q%d]Start: Persist %d queue options", i, len(options))

	s := `
    INSERT INTO "queue_option" (
        name, queue_client_id, queue_guild_id
    ) VALUES
    `
	idx := 0
	for _, o := range options {
		if o == nil {
			continue
		}
		idx++
		if idx > 1 {
			s += ","
		}
		s += fmt.Sprintf(
			` ('%s', '%s', '%s')`,
			o.Name, clientID, guildID,
		)
	}
	// NOTE: do not insert duplicated options
	s += `
     ON CONFLICT DO NOTHING;
    `
	if _, err := store.db.Exec(s); err != nil {
		store.log.Tracef("[Q%d]Error: %v", i, err)
		return err
	}

	store.log.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[Q%d]Done : %d queue_options persisted", i, len(options))
	return nil
}

// RemoveQueueOptions removes all the provided options from
// the queue identified by the provided clientID and guildID.
// This does not throw error if no such option exists in the database.
func (store *QueueStore) RemoveQueueOptions(clientID string, guildID string, options ...model.QueueOptionName) error {
	if options == nil || len(options) < 1 {
		return nil
	}
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[Q%d]Start: Remove %d queue options", i, len(options))

	l := make([]string, 0)
	for _, o := range options {
		l = append(l, string(o))
	}

	if _, err := store.db.Exec(
		`
        DELETE FROM "queue_option"
        WHERE "queue_option".name = ANY($1) AND
            "queue_option".queue_client_id = $2 AND
            "queue_option".queue_guild_id = $3;
        `,
		pq.Array(l),
		clientID,
		guildID,
	); err != nil {
		store.log.Tracef("[Q%d]Error: %v", i, err)
		return err
	}

	store.log.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[Q%d]Done : queue_options removed", i)
	return nil
}

// QueueHasOption checks whether the queue identified by the
// provided clientID and guildID has the option with the provided name.
func (store *QueueStore) QueueHasOption(clientID string, guildID string, name model.QueueOptionName) bool {
	i, t := store.idx, time.Now()
	store.idx++

	opt := &model.QueueOption{}
	var ignore interface{}
	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[Q%d]Start: Check if queue has option", i)
	err := store.db.QueryRow(
		`
        SELECT * FROM "queue_option"
        WHERE "queue_option".queue_client_id = $1 AND
            "queue_option".queue_guild_id = $2 AND
            "queue_option".name = $3;
        `,
		clientID,
		guildID,
		name,
	).Scan(&opt.Name, &ignore, &ignore)

	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[Q%d]Done : checked if queue has option", i)

	if err != nil {
		return false
	}
	return true
}

// GetOptionsForQueue returns all queue options that belong to
// the queue identified by the provided clientID and guildID
func (store *QueueStore) GetOptionsForQueue(clientID string, guildID string) ([]*model.QueueOption, error) {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[Q%d]Start: Fetch options for queue", i)

	rows, err := store.db.Query(
		`
        SELECT * FROM "queue_option"
        WHERE "queue_option".queue_client_id = $1 AND
            "queue_option".queue_guild_id = $2;
        `,
		clientID,
		guildID,
	)
	if err != nil {
		store.log.Tracef("[Q%d]Error: %v", i, err)
		return nil, err
	}
	options := make([]*model.QueueOption, 0)
	for rows.Next() {
		opt := &model.QueueOption{}
		var ignore interface{}
		if err := rows.Scan(
			&opt.Name, &ignore, &ignore,
		); err != nil {
			store.log.Tracef(
				"[Q%d]Error: %v", i, err,
			)
			return nil, err
		}
		options = append(options, opt)
	}

	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[Q%d]Done : %d songs for queue fetched", i, len(options))

	return options, nil
}

// createQueueTable creates the "queue" table
// with all it's constraints
// if it does  not already exist
func (store *QueueStore) createQueueTable() error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithField("TableName", "queue").Tracef(
		"[Q%d]Start: Create psql table (if not exists)",
		i,
	)

	if _, err := store.db.Exec(
		`
        CREATE TABLE IF NOT EXISTS "queue" (
            client_id VARCHAR,
            guild_id VARCHAR,
            message_id VARCHAR NOT NULL,
            channel_id VARCHAR NOT NULL,
            "offset" INTEGER NOT NULL DEFAULT '0',
            "limit" INTEGER NOT NULL DEFAULT '10',
            UNIQUE (client_id, guild_id),
            PRIMARY KEY (client_id, guild_id)
        );
        `,
	); err != nil {
		store.log.Tracef("[Q%d]Error: %v", i, err)
		return err
	}
	store.log.WithField("Latency", time.Since(t)).Tracef(
		"[Q%d]Done : psql table created", i,
	)
	return nil
}

// createQueueOptionTable creates the "queue_option" table
// with all it's constraints if it does  not already exist
func (store *QueueStore) createQueueOptionTable() error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithField("TableName", "queue_option").Tracef(
		"[Q%d]Start: Create psql table (if not exists)",
		i,
	)

	if _, err := store.db.Exec(
		`
        CREATE TABLE IF NOT EXISTS "queue_option" (
            name VARCHAR,
            queue_client_id VARCHAR,
            queue_guild_id VARCHAR,
            PRIMARY KEY (name, queue_client_id, queue_guild_id),
            FOREIGN KEY (queue_client_id, queue_guild_id)
                REFERENCES "queue" (client_id, guild_id)
                    ON DELETE CASCADE
        );
        `,
	); err != nil {
		store.log.Tracef("[Q%d]Error: %v", i, err)
		return err
	}
	store.log.WithField("Latency", time.Since(t)).Tracef(
		"[Q%d]Done : psql table created", i,
	)
	return nil
}

// dropQueueTable() drops the "queue" table.
func (store *QueueStore) dropQueueTable() error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithField("TableName", "queue").Tracef(
		"[Q%d]Start: Drop psql table (if exists)", i,
	)

	if _, err := store.db.Exec(
		`DROP TABLE IF EXISTS "queue" CASCADE`,
	); err != nil {
		store.log.Tracef(
			"[Q%d]Error: %v", i, err,
		)
		return err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[Q%d]Done : psql table dropped", i)
	return nil
}

// dropQueueOptionTable drops the "queue_option" table.
func (store *QueueStore) dropQueueOptionTable() error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithField("TableName", "queue_option").Tracef(
		"[Q%d]Start: Drop psql table (if exists)", i,
	)

	if _, err := store.db.Exec(
		`DROP TABLE IF EXISTS "queue_option" CASCADE`,
	); err != nil {
		store.log.Tracef(
			"[Q%d]Error: %v", i, err,
		)
		return err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[Q%d]Done : psql table dropped", i)
	return nil
}
