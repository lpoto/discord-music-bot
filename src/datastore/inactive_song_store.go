package datastore

import (
	"discord-music-bot/model"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
)

// PersistInactiveSongs inserts all of the provided inactive songs to
// the database in a single query.
// The persisted inactive songs will be automatically deleted
// after some time.
func (datastore *Datastore) PersistInactiveSongs(clientID string, guildID string, songs ...*model.Song) error {
	if len(songs) < 1 {
		return nil
	}

	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Persist %d inactive songs", i, len(songs))

	s := `
    INSERT INTO "inactive_song" (
        name, short_name, url, duration_seconds,
        duration_string, color, queue_client_id, queue_guild_id
    ) VALUES
    `
	idx := 0
	for _, song := range songs {
		if song == nil {
			continue
		}
		idx++
		if idx > 1 {
			s += ","
		}
		s += fmt.Sprintf(
			` ('%s', '%s', '%s', %d, '%s', %d, '%s', '%s')`,
			datastore.escapeSingleQuotes(song.Name),
			datastore.escapeSingleQuotes(song.ShortName),
			datastore.escapeSingleQuotes(song.Url),
			song.DurationSeconds,
			datastore.escapeSingleQuotes(song.DurationString),
			song.Color,
			clientID,
			guildID,
		)
	}
	if _, err := datastore.Exec(s); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[%d]Done : %d inactive songs persisted", i, len(songs))
	return nil
}

// PopLatestInactiveSong deletes the inactive song, belonging to the queue
// identified with the provided clientID and guildID, that was added last
// to the database, and returns it
func (datastore *Datastore) PopLatestInactiveSong(clientID string, guildID string) (*model.Song, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Pop latest inactive song", i)

	song := &model.Song{}
	var ignore string

	if err := datastore.QueryRow(
		`
        DELETE FROM "inactive_song"
        WHERE "inactive_song".queue_client_id = $1 AND
            "inactive_song".queue_guild_id = $2 AND
            "inactive_song".id = ANY(
                array(
                    SELECT id FROM "inactive_song"
                    ORDER BY id DESC
                    LIMIT 1
                )
            )
        RETURNING *
        `,
		clientID,
		guildID,
	).Scan(
		&song.ID, &song.Name, &song.ShortName,
		&song.Url, &song.DurationSeconds, &song.DurationString,
		&song.Color, &ignore, &ignore, &ignore,
	); err != nil {
		datastore.Tracef("[%d]Error: %v", err)
		return nil, err
	}
	song.Name = datastore.unescapeSingleQuotes(song.Name)
	song.ShortName = datastore.unescapeSingleQuotes(song.ShortName)
	song.Url = datastore.unescapeSingleQuotes(song.Url)

	datastore.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[%d]Done : Popped latest inactive song", i)

	return song, nil
}

// GetInactiveSongCountForQueue returns the number of inactive
// songs that belong to the queue
// identified by the provided clientID and guildID
func (datastore *Datastore) GetInactiveSongCountForQueue(clientID string, guildID string) int {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Fetch inactive song count for queue", i)

	var count int
	if err := datastore.QueryRow(
		`
        SELECT COUNT(*) FROM "inactive_song"
        WHERE "inactive_song".queue_client_id = $1 AND
            "inactive_song".queue_guild_id = $2
        `,
		clientID,
		guildID,
	).Scan(&count); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		count = 0
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Inactive song count for queue fetched (%d)", i, count)
	return count
}

// createInactiveSongTable creates the "inactive_song" table
// with all it's constraints
// if it does  not already exist
func (datastore *Datastore) createInactiveSongTable() error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithField("TableName", "inactive_song").Debugf(
		"[%d]Start: Create psql table (if not exists)", i,
	)

	if _, err := datastore.Exec(
		`
        CREATE TABLE IF NOT EXISTS "inactive_song" (
            id SERIAL,
            name VARCHAR NOT NULL,
            short_name VARCHAR NOT NULL,
            url VARCHAR NOT NULL,
            duration_seconds INTEGER NOT NULL,
            duration_string VARCHAR NOT NULL,
            color INTEGER NOT NULL,
            queue_client_id VARCHAR,
            queue_guild_id VARCHAR,
            added timestamp DEFAULT ((CURRENT_TIMESTAMP)),
            PRIMARY KEY (id),
            FOREIGN KEY (queue_client_id, queue_guild_id)
                REFERENCES "queue" (client_id, guild_id)
                    ON DELETE CASCADE
        );
        `,
	); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return err
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : psql table created", i)
	return nil
}
