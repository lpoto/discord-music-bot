package datastore

import (
	"discord-music-bot/model"
	"fmt"
	"time"

	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// PersistQueueOptions inserts all of the provided queue options to the
// database in a single query. Options with name equal to some other
// already persisted option (for the same queue) are not persisted.
func (datastore *Datastore) PersistQueueOptions(clientID string, guildID string, options ...*model.QueueOption) error {
	if options == nil || len(options) < 1 {
		return nil
	}
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Persist %d queue options", i, len(options))

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
	if _, err := datastore.Exec(s); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}

	datastore.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[%d]Done : %d queue_options persisted", i, len(options))
	return nil
}

// RemoveQueueOptions removes all the provided options from
// the queue identified by the provided clientID and guildID.
// This does not throw error if no such option exists in the database.
func (datastore *Datastore) RemoveQueueOptions(clientID string, guildID string, options ...model.QueueOptionName) error {
	if options == nil || len(options) < 1 {
		return nil
	}
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Remove %d queue options", i, len(options))

	l := make([]string, 0)
	for _, o := range options {
		l = append(l, string(o))
	}

	if _, err := datastore.Exec(
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
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}

	datastore.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[%d]Done : queue_options removed", i)
	return nil
}

// GetOptionsForQueue returns all queue options that belong to
// the queue identified by the provided clientID and guildID
func (datastore *Datastore) GetOptionsForQueue(clientID string, guildID string) ([]*model.QueueOption, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Fetch options for queue", i)

	rows, err := datastore.Query(
		`
        SELECT * FROM "queue_option"
        WHERE "queue_option".queue_client_id = $1 AND
            "queue_option".queue_guild_id = $2;
        `,
		clientID,
		guildID,
	)
	if err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return nil, err
	}
	options := make([]*model.QueueOption, 0)
	for rows.Next() {
		opt := &model.QueueOption{}
		var ignore uint
		if err := rows.Scan(
			&opt.Name, &ignore, &ignore,
		); err != nil {
			datastore.Tracef(
				"[%d]Error: %v", i, err,
			)
			return nil, err
		}
		options = append(options, opt)
	}

	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : %d songs for queue fetched", i, len(options))

	return options, nil
}

// createQueueOptionTable creates the "queue_option" table
// with all it's constraints if it does  not already exist
func (datastore *Datastore) createQueueOptionTable() error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithField("TableName", "queue_option").Tracef(
		"[%d]Start: Create psql table (if not exists)",
		i,
	)

	if _, err := datastore.Exec(
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
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField("Latency", time.Since(t)).Tracef(
		"[%d]Done : psql table created", i,
	)
	return nil
}
