package datastore

import (
	"discord-music-bot/model"
	"fmt"

	"github.com/lib/pq"
)

// PersistSongs inserts all of the provided songs to the database
// in a single query.
// The saved songs belong to the queue identified by the provided
// clientID and guildID
func (datastore *Datastore) PersistSongs(clientID string, guildID string, songs []*model.Song) error {
	if len(songs) < 1 {
		return nil
	}
	datastore.Trace(
		"Persisting %d songs: clientID=%s, guildID=%s",
		len(songs), clientID, guildID,
	)

	maxPosition, err := datastore.getMaxSongPosition(
		clientID,
		guildID,
	)
	if err != nil {
		return err
	}
	s := `
    INSERT INTO "song" (
        position, name, trimmed_name, url, duration_seconds, video_id,
        duration_string, color, queue_client_id, queue_guild_id
    ) VALUES
    `
	used := make(map[string]struct{})
	idx := 0
	for _, song := range songs {
		if _, ok := used[song.Info.VideoID]; ok {
			continue
		}
		idx++
		maxPosition++
		used[song.Info.VideoID] = struct{}{}
		if idx > 1 {
			s += ","
		}
		s += fmt.Sprintf(
			` (%d, '%s', '%s', '%s', %d, '%s', '%s', %d, '%s', '%s')`,
			maxPosition,
			datastore.escapeSingleQuotes(song.Info.Name),
			datastore.escapeSingleQuotes(song.Info.TrimmedName),
			datastore.escapeSingleQuotes(song.Info.Url),
			song.Info.DurationSeconds,
			datastore.escapeSingleQuotes(song.Info.VideoID),
			datastore.escapeSingleQuotes(song.Info.DurationString),
			song.Color,
			clientID,
			guildID,
		)
	}
	if _, err := datastore.Exec(s); err != nil {
		datastore.Error("Error when persisting songs: %v", err)
		return err
	}
	datastore.Trace("Successfully persisted %d songs", len(songs))
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

	datastore.Trace("Updating %d songs", len(songs))

	s := `
    UPDATE "song" as s set
        position = s2.position,
        name = s2.name,
        trimmed_name = s2.trimmed_name,
        url = s2.url,
        duration_seconds = s2.duration_seconds,
        video_id = s2.video_id,
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
			` (%d, %d, '%s', '%s', '%s', %d, '%s', '%s', %d)`,
			song.ID,
			song.Position,
			datastore.escapeSingleQuotes(song.Info.Name),
			datastore.escapeSingleQuotes(song.Info.TrimmedName),
			datastore.escapeSingleQuotes(song.Info.Url),
			song.Info.DurationSeconds,
			datastore.escapeSingleQuotes(song.Info.VideoID),
			datastore.escapeSingleQuotes(song.Info.DurationString),
			song.Color,
		)
	}
	s += `
        ) as s2(
            id, position, name, trimmed_name, url, duration_seconds,
            video_id, duration_string, color
        )
    WHERE s.id = s2.id;
    `
	if _, err := datastore.Exec(s); err != nil {
		datastore.Error("Error when updating songs: %v", err)
		return err
	}
	datastore.Trace("Successfully updated the songs")
	return nil
}

// GetSongsForQueue fetches the songs that belong to the queue identified
// by the provided clientID and guilID,
// limited by the provided offset and limit.
func (datastore *Datastore) GetSongsForQueue(clientID string, guildID string, offset int, limit int) ([]*model.Song, error) {
	datastore.Trace(
		"Fetching %d songs for queue: clientID=%s, guildID=%s, offset=%d",
		limit, clientID, guildID, offset,
	)

	if rows, err := datastore.Query(
		`
        SELECT * FROM "song"
        WHERE "song".queue_client_id = $1 AND
            "song".queue_guild_id = $2
        ORDER BY position
        OFFSET $3
        LIMIT $4;
        `,
		clientID,
		guildID,
		offset,
		limit,
	); err != nil {
		datastore.Error("Error when fetchings songs for queue: %v", err)
		return nil, err
	} else {
		songs := make([]*model.Song, 0)
		for rows.Next() {
			song := &model.Song{}
			song.Info = &model.SongInfo{}
			var ignore string
			if err := rows.Scan(
				&song.ID, &song.Position,
				&song.Info.Name, &song.Info.TrimmedName,
				&song.Info.Url, &song.Info.VideoID,
				&song.Info.DurationSeconds, &song.Info.DurationString,
				&song.Color, &ignore, &ignore, &song.Timestamp,
			); err != nil {
				datastore.Error(
					"Error when fetchings songs for queue: %v", err,
				)
				return nil, err
			} else {
				songs = append(songs, song)
			}
		}
		datastore.Trace(
			"Successfully fetched %d songs for the queue", len(songs),
		)
		return songs, nil
	}
}

// GetSongsForQueue fetches all the songs that belong to the queue identified
// by the provided clientID and guilID.
func (datastore *Datastore) GetAllSongsForQueue(clientID string, guildID string) ([]*model.Song, error) {
	datastore.Trace(
		"Fetching all songs for queue: clientID=%s, guildID=%s",
		clientID, guildID,
	)

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
		datastore.Error("Error when fetchings songs for queue: %v", err)
		return nil, err
	} else {
		songs := make([]*model.Song, 0)
		for rows.Next() {
			song := &model.Song{}
			song.Info = &model.SongInfo{}
			var ignore string
			if err := rows.Scan(
				&song.ID, &song.Position,
				&song.Info.Name, &song.Info.TrimmedName,
				&song.Info.Url, &song.Info.VideoID,
				&song.Info.DurationSeconds, &song.Info.DurationString,
				&song.Color, &ignore, &ignore, &song.Timestamp,
			); err != nil {
				datastore.Error(
					"Error when fetchings songs for queue: %v", err,
				)
				return nil, err
			} else {
				songs = append(songs, song)
			}
		}
		datastore.Trace(
			"Successfully fetched %d songs for the queue", len(songs),
		)
		return songs, nil
	}
}

// GetSongCountForQueue returns the number of songs that belong
// to the queue identified by the provided clientID and guildID
func (datastore *Datastore) GetSongCountForQueue(clientID string, guildID string) int {
	datastore.Trace(
		"Fetching song count for queue: clientID=%s, guildID=%s",
		clientID, guildID,
	)
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
		datastore.Error(
			"Error when fetching song count for queue: %v", err,
		)
		count = 0
	}
	datastore.Trace("Got song count for the queue: %d", count)
	return count
}

// RemoveSongs removes songs with ID in the provided ids that belong to the
// queue, identified by the provided clientID and guildID.
// If force is true, the songs are deleted, else they are moved
// to the 'inactive_song' table.
func (datastore *Datastore) RemoveSongs(clientID string, guildID string, ids []string) error {
	datastore.Trace(
		"Removing %d songs: clientID=%s, guildID=%s",
		len(ids), clientID, guildID,
	)
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
		datastore.Error(
			"Error when deleting songs: %v", err,
		)
	}
	return nil
}

func (datastore *Datastore) createSongTable() error {
	datastore.WithField("TableName", "song").Debug("Creating psql table")

	if _, err := datastore.Exec(
		`
        CREATE TABLE IF NOT EXISTS "song" (
            id SERIAL,
            position INTEGER NOT NULL DEFAULT '0',
            name VARCHAR NOT NULL,
            trimmed_name VARCHAR NOT NULL,
            url VARCHAR NOT NULL,
            video_id VARCHAR NOT NULL,
            duration_seconds INTEGER NOT NULL,
            duration_string VARCHAR NOT NULL,
            color INTEGER NOT NULL,
            queue_client_id VARCHAR,
            queue_guild_id VARCHAR,
            timestamp TIMESTAMP DEFAULT ((CURRENT_TIMESTAMP)),
            PRIMARY KEY (id),
            FOREIGN KEY (queue_client_id, queue_guild_id)
                REFERENCES "queue" (client_id, guild_id)
                    ON DELETE CASCADE
        );
        `,
	); err != nil {
		datastore.Tracef(
			"Error when creating table 'song'",
			err,
		)
		return err
	}
	datastore.WithField("TableName", "song").Trace(
		"Successfully created psql table",
	)
	return nil
}

func (datastore *Datastore) getMaxSongPosition(clientID string, guildID string) (int, error) {
	datastore.Trace(
		"Fetching max song position: clientID=%s, guildID=%s",
		clientID,
		guildID,
	)
	var position int = 0
	if err := datastore.QueryRow(
		`
        SELECT COALESCE(MAX(s.position), 0)
        FROM "song" s
        WHERE s.queue_guild_id = $1 AND
            s.queue_client_id = $2 AND
            s.position <> $3
    `,
		guildID,
		clientID,
	).Scan(&position); err != nil {
		datastore.Trace(
			"Error when fetching max song position: %v", err,
		)
		return 0, err
	}
	datastore.Trace("Successfully fetched max song position: %d", position)
	return position, nil
}
