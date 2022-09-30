package datastore

import (
	"context"
	"discord-music-bot/model"
	"fmt"
	"time"

	"github.com/lib/pq"
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
	params := make([]interface{}, 0)
	p := 1
	for _, song := range songs {
		if song == nil {
			continue
		}
		idx++
		if idx > 1 {
			s += ","
		}
		params = append(params, datastore.toPSQLArray(song.Name))
		params = append(params, datastore.toPSQLArray(song.ShortName))
		params = append(params, datastore.toPSQLArray(song.Url))
		params = append(params, song.DurationSeconds)
		params = append(params, datastore.toPSQLArray(song.DurationString))
		params = append(params, song.Color)
		params = append(params, clientID)
		params = append(params, guildID)
		s += fmt.Sprintf(
			` ($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)`,
			p, p+1, p+2, p+3, p+4, p+5, p+6, p+7,
		)
		p += 8
	}
	if _, err := datastore.Exec(s, params...); err != nil {
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
	name, shortName, url, durationString := []int64{}, []int64{}, []int64{}, []int64{}
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
		&song.ID,
		pq.Array(&name), pq.Array(&shortName),
		pq.Array(&url), &song.DurationSeconds,
		pq.Array(&durationString),
		&song.Color, &ignore, &ignore, &ignore,
	); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return nil, err
	}
	song.Name = datastore.toString(name)
	song.ShortName = datastore.toString(shortName)
	song.Url = datastore.toString(url)
	song.DurationString = datastore.toString(durationString)

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

	datastore.WithField("TableName", "inactive_song").Tracef(
		"[%d]Start: Create psql table (if not exists)", i,
	)

	if _, err := datastore.Exec(
		`
        CREATE TABLE IF NOT EXISTS "inactive_song" (
            id SERIAL,
            name int[] NOT NULL,
            short_name int[] NOT NULL,
            url int[] NOT NULL,
            duration_seconds INTEGER NOT NULL,
            duration_string int[] NOT NULL,
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

// runInactiveSongsCleanup is a long lived worker, that cleans up
// outdated inactive songs from the datastore at interval.
func (datastore *Datastore) runInactiveSongsCleanup(ctx context.Context) {
	interval := datastore.config.InactiveSongTTL / 2
	if interval < time.Second {
		interval = time.Second
	}
	datastore.WithFields(log.Fields{
		"TTL":      datastore.config.InactiveSongTTL,
		"Interval": interval,
	}).Debug(
		"Running inactive songs cleanup",
	)
	done := ctx.Done()
	ticker := time.NewTicker(datastore.config.InactiveSongTTL)

	datastore.removeOutdatedInactiveSongs()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			datastore.removeOutdatedInactiveSongs()
		}
	}
}

// removeOutdatedInactiveSongs removes all the inactive songs
// with "added" column older than the InactiveSongTTL cofnig option.
func (datastore *Datastore) removeOutdatedInactiveSongs() {
	i, t := datastore.getIdx(), time.Now()

	datastore.Tracef(
		"[%d]Start: Remove outdated inactive songs", i,
	)

	if _, err := datastore.Exec(
		`
        DELETE FROM "inactive_song"
        WHERE "inactive_song".added <= $1;
        `,
		time.Now().Add(datastore.config.InactiveSongTTL*(-1)),
	); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return
	}

	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Outdated inactive songs removed", i)

}
