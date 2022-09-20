package datastore

import (
	"discord-music-bot/model"
	"fmt"
	"time"

	"github.com/lib/pq"
	log "github.com/sirupsen/logrus"
)

// PersistSongs inserts all of the provided songs to the database
// in a single query.
// The saved songs belong to the queue identified by the provided
// clientID and guildID
func (datastore *Datastore) PersistSongs(clientID string, guildID string, songs []*model.Song) error {
	if len(songs) < 1 {
		return nil
	}

	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Persist %d songs", i, len(songs))

	maxPosition, err := datastore.getMaxSongPosition(
		clientID,
		guildID,
	)
	if err != nil {
		return err
	}
	s := `
    INSERT INTO "song" (
        position, name, short_name, url, duration_seconds,
        duration_string, color, queue_client_id, queue_guild_id
    ) VALUES
    `
	used := make(map[string]struct{})
	idx := 0
	for _, song := range songs {
		if song == nil {
			continue
		}
		if _, ok := used[song.Name]; ok {
			continue
		}
		idx++
		maxPosition++
		used[song.Name] = struct{}{}
		if idx > 1 {
			s += ","
		}
		s += fmt.Sprintf(
			` (%d, '%s', '%s', '%s', %d, '%s', %d, '%s', '%s')`,
			maxPosition,
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
		datastore.Errorf("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[%d]Done : %d songs persisted", i, len(songs))
	return nil
}

// UpdateSongs updates all of the provided songs in the database
// in a single query.
// This does not update their ID's or their foreign keys that
// reference a Queue.
func (datastore *Datastore) UpdateSongs(songs []*model.Song) error {
	if len(songs) < 1 {
		return nil
	}

	i, t := datastore.getIdx(), time.Now()

	datastore.Tracef("[%d]Start: Update %d songs", i, len(songs))

	s := `
    UPDATE "song" as s set
        position = s2.position,
        name = s2.name,
        short_name = s2.short_name,
        url = s2.url,
        duration_seconds = s2.duration_seconds,
        duration_string = s2.duration_string,
        color = s2.color,
    FROM (
        VALUES
    `
	used := make(map[uint]struct{})
	idx := 0
	for _, song := range songs {
		if _, ok := used[song.ID]; ok {
			continue
		}
		used[song.ID] = struct{}{}
		idx++
		if idx > 1 {
			s += ","
		}
		s += fmt.Sprintf(
			` (%d, %d, '%s', '%s', '%s', %d, '%s', %d)`,
			song.ID,
			song.Position,
			datastore.escapeSingleQuotes(song.Name),
			datastore.escapeSingleQuotes(song.ShortName),
			datastore.escapeSingleQuotes(song.Url),
			song.DurationSeconds,
			datastore.escapeSingleQuotes(song.DurationString),
			song.Color,
		)
	}
	s += `
        ) as s2(
            id, position, name, short_name, url, duration_seconds,
            duration_string, color
        )
    WHERE s.id = s2.id;
    `
	if _, err := datastore.Exec(s); err != nil {
		datastore.Errorf("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Songs updated", i)
	return nil
}

// GetSongsForQueue fetches the songs that belong to the queue identified
// by the provided clientID and guilID,
// limited by the provided offset and limit.
func (datastore *Datastore) GetSongsForQueue(clientID string, guildID string, offset int, limit int) ([]*model.Song, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
		"Offset":   offset,
	}).Tracef("[%d]Start: Fetch %d songs for queue", i, limit)

	if rows, err := datastore.Query(
		`
        SELECT * FROM "song"
        WHERE "song".queue_client_id = $1 AND
            "song".queue_guild_id = $2
        ORDER BY position ASC
        OFFSET $3
        LIMIT $4;
        `,
		clientID,
		guildID,
		offset,
		limit,
	); err != nil {
		datastore.Errorf("[%d]Error: %v", i, err)
		return nil, err
	} else {
		songs := make([]*model.Song, 0)
		for rows.Next() {
			song := &model.Song{}
			var ignore string
			if err := rows.Scan(
				&song.ID, &song.Position,
				&song.Name, &song.ShortName, &song.Url,
				&song.DurationSeconds, &song.DurationString,
				&song.Color, &ignore, &ignore,
			); err != nil {
				datastore.Errorf(
					"[%d]Error: %v", i, err,
				)
				return nil, err
			} else {
				song.Name = datastore.unescapeSingleQuotes(song.Name)
				song.ShortName = datastore.unescapeSingleQuotes(song.ShortName)
				song.Url = datastore.unescapeSingleQuotes(song.Url)
				songs = append(songs, song)
			}
		}
		datastore.WithField(
			"Latency", time.Since(t),
		).Tracef("[%d]Done : %d songs for queue fetched", i, len(songs))
		return songs, nil
	}
}

// GetSongsForQueue fetches all the songs that belong to the queue identified
// by the provided clientID and guilID.
func (datastore *Datastore) GetAllSongsForQueue(clientID string, guildID string) ([]*model.Song, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Fetch all songs for queue", i)

	if rows, err := datastore.Query(
		`
        SELECT * FROM "song"
        WHERE "song".queue_client_id = $1 AND
            "song".queue_guild_id = $2
        ORDER BY position;
        `,
		clientID,
		guildID,
	); err != nil {
		datastore.Errorf("[%d]Error: %v", i, err)
		return nil, err
	} else {
		songs := make([]*model.Song, 0)
		for rows.Next() {
			song := &model.Song{}
			var ignore string
			if err := rows.Scan(
				&song.ID, &song.Position,
				&song.Name, &song.ShortName, &song.Url,
				&song.DurationSeconds, &song.DurationString,
				&song.Color, &ignore, &ignore,
			); err != nil {
				datastore.Errorf(
					"[%d]Error: %v", i, err,
				)
				return nil, err
			} else {
				songs = append(songs, song)
			}
		}
		datastore.WithField("Latency", time.Since(t)).Tracef(
			"[%d]Done : %d songs fetched for queue", i, len(songs),
		)
		return songs, nil
	}
}

// GetSongCountForQueue returns the number of songs that belong
// to the queue identified by the provided clientID and guildID
func (datastore *Datastore) GetSongCountForQueue(clientID string, guildID string) int {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Fetch song count for queue", i)

	var count int
	if err := datastore.QueryRow(
		`
        SELECT COUNT(*) FROM "song"
        WHERE "song".queue_client_id = $1 AND
            "song".queue_guild_id = $2
        `,
		clientID,
		guildID,
	).Scan(&count); err != nil {
		datastore.Errorf(
			"[%d]Error: %v", i, err,
		)
		count = 0
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Song count for queue fetched (%d)", i, count)
	return count
}

// RemoveSongs removes songs with ID in the provided ids that belong to the
// queue, identified by the provided clientID and guildID.
// If force is true, the songs are deleted, else they are moved
// to the 'inactive_song' table.
func (datastore *Datastore) RemoveSongs(clientID string, guildID string, ids []uint) error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Remove %d songs from queue", i, len(ids))

	if _, err := datastore.Exec(
		`
        DELETE FROM "song"
        WHERE "song".id = ANY($1) AND
            "song".queue_client_id = $2 AND
            "song".queue_guild_id = $3
        `,
		pq.Array(ids),
		clientID,
		guildID,
	); err != nil {
		datastore.Errorf(
			"[%d]Error: %v", i, err,
		)
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Removed songs from queeu", i)
	return nil
}

// PushLastSongToFront places the song with the max song position to the front
// of the queue, by setting it's position 1 less than the song with min position
func (datastore *Datastore) PushLastSongToFront(clientID string, guildID string) error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Push last song to front", i)

	minPosition, err := datastore.getMinSongPosition(clientID, guildID)
	if err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	maxPosition, err := datastore.getMaxSongPosition(clientID, guildID)
	if err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	if _, err := datastore.Exec(
		`
        UPDATE "song" SET
        position = $1
        WHERE "song".position = $2 AND
        "song".queue_client_id = $3 AND
        "song".queue_guild_id = $4 AND
        `,
		minPosition-1,
		maxPosition,
		clientID,
		guildID,
	); err != nil {
		datastore.Errorf("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField("Latency", time.Since(t)).Tracef(
		"[%d]Done : Pushed last song to front", i,
	)
	return nil
}

func (datastore *Datastore) createSongTable() error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithField("TableName", "song").Debugf(
		"[%d]Start: Create psql table (if not exists)", i,
	)

	if _, err := datastore.Exec(
		`
        CREATE TABLE IF NOT EXISTS "song" (
            id SERIAL,
            position INTEGER NOT NULL DEFAULT '0',
            name VARCHAR NOT NULL,
            short_name VARCHAR NOT NULL,
            url VARCHAR NOT NULL,
            duration_seconds INTEGER NOT NULL,
            duration_string VARCHAR NOT NULL,
            color INTEGER NOT NULL,
            queue_client_id VARCHAR,
            queue_guild_id VARCHAR,
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

func (datastore *Datastore) getMaxSongPosition(clientID string, guildID string) (int, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Fetch max song position for queue", i)

	var position int = 0
	if err := datastore.QueryRow(
		`
        SELECT COALESCE(MAX(s.position), 0)
        FROM "song" s
        WHERE s.queue_guild_id = $1 AND
            s.queue_client_id = $2
    `,
		guildID,
		clientID,
	).Scan(&position); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return 0, err
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Fetched max song position (%d)", i, position)
	return position, nil
}

func (datastore *Datastore) getMinSongPosition(clientID string, guildID string) (int, error) {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Fetch min song position for queue", i)

	var position int = 0
	if err := datastore.QueryRow(
		`
        SELECT COALESCE(MIN(s.position), 0)
        FROM "song" s
        WHERE s.queue_guild_id = $1 AND
            s.queue_client_id = $2
    `,
		guildID,
		clientID,
	).Scan(&position); err != nil {
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		return 0, err
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Fetched min song position (%d)", i, position)
	return position, nil
}
