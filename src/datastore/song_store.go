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
func (datastore *Datastore) PersistSongs(clientID string, guildID string, songs ...*model.Song) error {
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
	params := make([]interface{}, 0)
	p := 1
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
		params = append(params, maxPosition)
		params = append(params, song.Name)
		params = append(params, song.ShortName)
		params = append(params, song.Url)
		params = append(params, song.DurationSeconds)
		params = append(params, song.DurationString)
		params = append(params, song.Color)
		params = append(params, clientID)
		params = append(params, guildID)
		s += fmt.Sprintf(
			` ($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)`,
			p, p+1, p+2, p+3, p+4, p+5, p+6, p+7, p+8,
		)
		p += 9
	}
	s += ";"
	if _, err := datastore.Exec(s, params...); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[%d]Done : %d songs persisted", i, len(songs))
	return nil
}

// PersistSongToFront saves the provided song to the database.
// The song's position is set to 1 less than the minimum position of the
// queue identified with the provided clientID and guildID
func (datastore *Datastore) PersistSongToFront(clientID string, guildID string, song *model.Song) error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Persist song to front", i)

	minPosition, err := datastore.getMinSongPosition(
		clientID,
		guildID,
	)
	if err != nil {
		datastore.Errorf("[%d]Error: %v", i, err)
		return err
	}

	if _, err := datastore.Exec(
		`
    INSERT INTO "song" (
        position, name, short_name, url, duration_seconds,
        duration_string, color, queue_client_id, queue_guild_id
    ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
    `,
		minPosition-1,
		song.Name,
		song.ShortName,
		song.Url,
		song.DurationSeconds,
		song.DurationString,
		song.Color,
		clientID,
		guildID,
	); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}

	datastore.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[%d]Done : song persisted to front", i)
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
	params := make([]interface{}, 0)
	p := 1
	for _, song := range songs {
		if _, ok := used[song.ID]; ok {
			continue
		}
		used[song.ID] = struct{}{}
		idx++
		if idx > 1 {
			s += ","
		}
		params = append(params, song.ID)
		params = append(params, song.Position)
		params = append(params, song.Name)
		params = append(params, song.ShortName)
		params = append(params, song.Url)
		params = append(params, song.DurationSeconds)
		params = append(params, song.DurationString)
		params = append(params, song.Color)

		s += fmt.Sprintf(
			` ($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)`,
			p, p+1, p+2, p+3, p+4, p+5, p+6, p+7,
		)
		p += 8
	}
	s += `
        ) as s2(
            id, position, name, short_name, url, duration_seconds,
            duration_string, color
        )
    WHERE s.id = s2.id;
    `
	if _, err := datastore.Exec(s, params...); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
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
		datastore.Tracef("[%d]Error: %v", i, err)
		return nil, err
	} else {
		songs := make([]*model.Song, 0)
		for rows.Next() {
			song := &model.Song{}
			var ignore string
			if err := rows.Scan(
				&song.ID, &song.Position,
				&song.Name, &song.ShortName, &song.Url,
				&song.DurationSeconds,
				&song.DurationString,
				&song.Color, &ignore, &ignore,
			); err != nil {
				datastore.Tracef(
					"[%d]Error: %v", i, err,
				)
				return nil, err
			} else {
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
		datastore.Tracef("[%d]Error: %v", i, err)
		return nil, err
	} else {
		songs := make([]*model.Song, 0)
		for rows.Next() {
			song := &model.Song{}
			var ignore string
			if err := rows.Scan(
				&song.ID, &song.Position,
				&song.Name, &song.ShortName, &song.Url,
				&song.DurationSeconds,
				&song.DurationString,
				&song.Color, &ignore, &ignore,
			); err != nil {
				datastore.Tracef(
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
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
		count = 0
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Song count for queue fetched (%d)", i, count)
	return count
}

// RemoveHeadSong removes song with the minimum position belonging to the
// queue identified with the provided clientID and guildID
func (datastore *Datastore) RemoveHeadSong(clientID string, guildID string) error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Remove head song", i)

	minPosition, err := datastore.getMinSongPosition(clientID, guildID)
	if err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	if _, err := datastore.Exec(
		`

        DELETE FROM "song"
        WHERE "song".position = $1 AND
        "song".queue_client_id = $2 AND
        "song".queue_guild_id = $3;
        `,
		minPosition,
		clientID,
		guildID,
	); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField("Latency", time.Since(t)).Tracef(
		"[%d]Done : Removed head song", i,
	)
	return nil
}

// PushHeadSongToBack places the song with the min song position to the back
// of the queue, by setting it's position 1 more than the song with max position
func (datastore *Datastore) PushHeadSongToBack(clientID string, guildID string) error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[%d]Start: Push head song to back", i)

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
        "song".queue_guild_id = $4;
        `,
		maxPosition+1,
		minPosition,
		clientID,
		guildID,
	); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField("Latency", time.Since(t)).Tracef(
		"[%d]Done : Pushed head song to back", i,
	)
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
        "song".queue_guild_id = $4;
        `,
		minPosition-1,
		maxPosition,
		clientID,
		guildID,
	); err != nil {
		datastore.Tracef("[%d]Error: %v", i, err)
		return err
	}
	datastore.WithField("Latency", time.Since(t)).Tracef(
		"[%d]Done : Pushed last song to front", i,
	)
	return nil
}

// RemoveSongs removes songs with ID in the provided ids that belong to the
// queue, identified by the provided clientID and guildID.
// If force is true, the songs are deleted, else they are moved
// to the 'inactive_song' table.
func (datastore *Datastore) RemoveSongs(clientID string, guildID string, ids ...uint) error {
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
		datastore.Tracef(
			"[%d]Error: %v", i, err,
		)
	}
	datastore.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Done : Removed songs from queeu", i)
	return nil
}

// createSongTable creates the "song" table
// with all it's constraints
// if it does  not already exist
func (datastore *Datastore) createSongTable() error {
	i, t := datastore.getIdx(), time.Now()

	datastore.WithField("TableName", "song").Tracef(
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

// getMaxSongPosition returns the maximum position of a song
// that belongs to the queue identified with the provided clientID and guildID
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

// getMaxSongPosition returns the minimum position of a song
// that belongs to the queue identified with the provided clientID and guildID
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
