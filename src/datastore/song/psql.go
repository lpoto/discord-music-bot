package song

import (
	"context"
	"database/sql"
	"discord-music-bot/model"
	"fmt"
	"time"

	"github.com/lib/pq"

	log "github.com/sirupsen/logrus"
)

type SongStore struct {
	log             *log.Logger
	db              *sql.DB
	inactiveSongTTL time.Duration
	idx             int
}

// NewSongStore creates an object that handles
// persisting and removing Songs in postgres database.
func NewSongStore(db *sql.DB, log *log.Logger, inactiveSongTTL time.Duration) *SongStore {
	return &SongStore{
		log:             log,
		db:              db,
		idx:             0,
		inactiveSongTTL: inactiveSongTTL,
	}
}

// Init creates the required tables for the Song store.
func (store *SongStore) Init() error {
	if err := store.createSongTable(); err != nil {
		return err
	}
	return store.createInactiveSongTable()
}

// Destroy drops the created tables for the Song store.
func (store *SongStore) Destroy() error {
	if err := store.dropSongTable(); err != nil {
		return err
	}
	return store.dropInactiveSongTable()
}

// UpdateQueueWithSongs fetches the queue's songs,
// limited by the queue's offset and limit, and the total
// size of the queue.
func (store *SongStore) UpdateQueueWithSongs(queue *model.Queue) (*model.Queue, error) {
	if headSongs, err := store.GetSongsForQueue(
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
	if songs, err := store.GetSongsForQueue(
		queue.ClientID,
		queue.GuildID,
		queue.Offset+1,
		queue.Limit,
	); err == nil {
		queue.Songs = songs
		queue.Size = store.GetSongCountForQueue(
			queue.ClientID,
			queue.GuildID,
		)
		queue.InactiveSize = store.GetInactiveSongCountForQueue(
			queue.ClientID,
			queue.GuildID,
		)
	} else {
		return nil, err
	}
	return queue, nil
}

// PersistSongs inserts all of the provided songs to the database
// in a single query.
// The saved songs belong to the queue identified by the provided
// clientID and guildID
func (store *SongStore) PersistSongs(clientID string, guildID string, songs ...*model.Song) error {
	if len(songs) < 1 {
		return nil
	}

	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Persist %d songs", i, len(songs))

	maxPosition, err := store.getMaxSongPosition(
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
	if _, err := store.db.Exec(s, params...); err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	store.log.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[S%d]Done : %d songs persisted", i, len(songs))
	return nil
}

// PersistSongToFront saves the provided song to the database.
// The song's position is set to 1 less than the minimum position of the
// queue identified with the provided clientID and guildID
func (store *SongStore) PersistSongToFront(clientID string, guildID string, song *model.Song) error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Persist song to front", i)

	minPosition, err := store.getMinSongPosition(
		clientID,
		guildID,
	)
	if err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}

	if _, err := store.db.Exec(
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
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}

	store.log.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[S%d]Done : song persisted to front", i)
	return nil
}

// GetSongsForQueue fetches the songs that belong to the queue identified
// by the provided clientID and guilID,
// limited by the provided offset and limit.
func (store *SongStore) GetSongsForQueue(clientID string, guildID string, offset int, limit int) ([]*model.Song, error) {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
		"Offset":   offset,
	}).Tracef("[S%d]Start: Fetch %d songs for queue", i, limit)

	if rows, err := store.db.Query(
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
		store.log.Tracef("[S%d]Error: %v", i, err)
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
				store.log.Tracef(
					"[S%d]Error: %v", i, err,
				)
				return nil, err
			} else {
				songs = append(songs, song)
			}
		}
		store.log.WithField(
			"Latency", time.Since(t),
		).Tracef("[S%d]Done : %d songs for queue fetched", i, len(songs))
		return songs, nil
	}
}

// GetSongsForQueue fetches all the songs that belong to the queue identified
// by the provided clientID and guilID.
func (store *SongStore) GetAllSongsForQueue(clientID string, guildID string) ([]*model.Song, error) {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Fetch all songs for queue", i)

	if rows, err := store.db.Query(
		`
        SELECT * FROM "song"
        WHERE "song".queue_client_id = $1 AND
            "song".queue_guild_id = $2
        ORDER BY position;
        `,
		clientID,
		guildID,
	); err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
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
				store.log.Tracef(
					"[S%d]Error: %v", i, err,
				)
				return nil, err
			} else {
				songs = append(songs, song)
			}
		}
		store.log.WithField("Latency", time.Since(t)).Tracef(
			"[S%d]Done : %d songs fetched for queue", i, len(songs),
		)
		return songs, nil
	}
}

// GetSongCountForQueue returns the number of songs that belong
// to the queue identified by the provided clientID and guildID
func (store *SongStore) GetSongCountForQueue(clientID string, guildID string) int {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Fetch song count for queue", i)

	var count int
	if err := store.db.QueryRow(
		`
        SELECT COUNT(*) FROM "song"
        WHERE "song".queue_client_id = $1 AND
            "song".queue_guild_id = $2
        `,
		clientID,
		guildID,
	).Scan(&count); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		count = 0
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : Song count for queue fetched (%d)", i, count)
	return count
}

// RemoveHeadSong removes song with the minimum position belonging to the
// queue identified with the provided clientID and guildID
func (store *SongStore) RemoveHeadSong(clientID string, guildID string) error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Remove head song", i)

	minPosition, err := store.getMinSongPosition(clientID, guildID)
	if err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	if _, err := store.db.Exec(
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
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	store.log.WithField("Latency", time.Since(t)).Tracef(
		"[S%d]Done : Removed head song", i,
	)
	return nil
}

// PushHeadSongToBack places the song with the min song position to the back
// of the queue, by setting it's position 1 more than the song with max position
func (store *SongStore) PushHeadSongToBack(clientID string, guildID string) error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Push head song to back", i)

	minPosition, err := store.getMinSongPosition(clientID, guildID)
	if err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	maxPosition, err := store.getMaxSongPosition(clientID, guildID)
	if err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	if _, err := store.db.Exec(
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
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	store.log.WithField("Latency", time.Since(t)).Tracef(
		"[S%d]Done : Pushed head song to back", i,
	)
	return nil
}

// PushLastSongToFront places the song with the max song position to the front
// of the queue, by setting it's position 1 less than the song with min position
func (store *SongStore) PushLastSongToFront(clientID string, guildID string) error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Push last song to front", i)

	minPosition, err := store.getMinSongPosition(clientID, guildID)
	if err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	maxPosition, err := store.getMaxSongPosition(clientID, guildID)
	if err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	if _, err := store.db.Exec(
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
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	store.log.WithField("Latency", time.Since(t)).Tracef(
		"[S%d]Done : Pushed last song to front", i,
	)
	return nil
}

// RemoveSongs removes songs with ID in the provided ids that belong to the
// queue, identified by the provided clientID and guildID.
// If force is true, the songs are deleted, else they are moved
// to the 'inactive_song' table.
func (store *SongStore) RemoveSongs(clientID string, guildID string, ids ...uint) error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Remove %d songs from queue", i, len(ids))

	if _, err := store.db.Exec(
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
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : Removed songs from queeu", i)
	return nil
}

// getMaxSongPosition returns the maximum position of a song
// that belongs to the queue identified with the provided clientID and guildID
func (store *SongStore) getMaxSongPosition(clientID string, guildID string) (int, error) {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Fetch max song position for queue", i)

	var position int = 0
	if err := store.db.QueryRow(
		`
        SELECT COALESCE(MAX(s.position), 0)
        FROM "song" s
        WHERE s.queue_guild_id = $1 AND
            s.queue_client_id = $2
    `,
		guildID,
		clientID,
	).Scan(&position); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		return 0, err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : Fetched max song position (%d)", i, position)
	return position, nil
}

// getMaxSongPosition returns the minimum position of a song
// that belongs to the queue identified with the provided clientID and guildID
func (store *SongStore) getMinSongPosition(clientID string, guildID string) (int, error) {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Fetch min song position for queue", i)

	var position int = 0
	if err := store.db.QueryRow(
		`
        SELECT COALESCE(MIN(s.position), 0)
        FROM "song" s
        WHERE s.queue_guild_id = $1 AND
            s.queue_client_id = $2
    `,
		guildID,
		clientID,
	).Scan(&position); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		return 0, err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : Fetched min song position (%d)", i, position)
	return position, nil
}

// PersistInactiveSongs inserts all of the provided inactive songs to
// the database in a single query.
// The persisted inactive songs will be automatically deleted
// after some time.
func (store *SongStore) PersistInactiveSongs(clientID string, guildID string, songs ...*model.Song) error {
	if len(songs) < 1 {
		return nil
	}

	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Persist %d inactive songs", i, len(songs))

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
		params = append(params, song.Name)
		params = append(params, song.ShortName)
		params = append(params, song.Url)
		params = append(params, song.DurationSeconds)
		params = append(params, song.DurationString)
		params = append(params, song.Color)
		params = append(params, clientID)
		params = append(params, guildID)
		s += fmt.Sprintf(
			` ($%d, $%d, $%d, $%d, $%d, $%d, $%d, $%d)`,
			p, p+1, p+2, p+3, p+4, p+5, p+6, p+7,
		)
		p += 8
	}
	if _, err := store.db.Exec(s, params...); err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return err
	}
	store.log.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[S%d]Done : %d inactive songs persisted", i, len(songs))
	return nil
}

// PopLatestInactiveSong deletes the inactive song, belonging to the queue
// identified with the provided clientID and guildID, that was added last
// to the database, and returns it
func (store *SongStore) PopLatestInactiveSong(clientID string, guildID string) (*model.Song, error) {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Pop latest inactive song", i)

	song := &model.Song{}
	var ignore string

	if err := store.db.QueryRow(
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
		&song.Name, &song.ShortName, &song.Url,
		&song.DurationSeconds, &song.DurationString,
		&song.Color, &ignore, &ignore, &ignore,
	); err != nil {
		store.log.Tracef("[S%d]Error: %v", i, err)
		return nil, err
	}
	store.log.WithField(
		"Latency",
		time.Since(t),
	).Tracef("[S%d]Done : Popped latest inactive song", i)

	return song, nil
}

// GetInactiveSongCountForQueue returns the number of inactive
// songs that belong to the queue
// identified by the provided clientID and guildID
func (store *SongStore) GetInactiveSongCountForQueue(clientID string, guildID string) int {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithFields(log.Fields{
		"ClientID": clientID,
		"GuildID":  guildID,
	}).Tracef("[S%d]Start: Fetch inactive song count for queue", i)

	var count int
	if err := store.db.QueryRow(
		`
        SELECT COUNT(*) FROM "inactive_song"
        WHERE "inactive_song".queue_client_id = $1 AND
            "inactive_song".queue_guild_id = $2
        `,
		clientID,
		guildID,
	).Scan(&count); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		count = 0
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : Inactive song count for queue fetched (%d)", i, count)
	return count
}

// runInactiveSongsCleanup is a long lived worker, that cleans up
// outdated inactive songs from the store at interval.
func (store *SongStore) RunInactiveSongsCleanup(ctx context.Context) {
	interval := store.inactiveSongTTL / 2
	if interval < time.Second {
		interval = time.Second
	}
	store.log.WithFields(log.Fields{
		"TTL":      store.inactiveSongTTL,
		"Interval": interval,
	}).Debug(
		"Running inactive songs cleanup",
	)
	done := ctx.Done()
	ticker := time.NewTicker(store.inactiveSongTTL)

	store.removeOutdatedInactiveSongs()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			store.removeOutdatedInactiveSongs()
		}
	}
}

// removeOutdatedInactiveSongs removes all the inactive songs
// with "added" column older than the InactiveSongTTL cofnig option.
func (store *SongStore) removeOutdatedInactiveSongs() {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.Tracef(
		"[S%d]Start: Remove outdated inactive songs", i,
	)

	if _, err := store.db.Exec(
		`
        DELETE FROM "inactive_song"
        WHERE "inactive_song".added <= $1;
        `,
		time.Now().Add(store.inactiveSongTTL*(-1)),
	); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		return
	}

	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : Outdated inactive songs removed", i)

}

// createSongTable creates the "song" table
// with all it's constraints
// if it does  not already exist
func (store *SongStore) createSongTable() error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithField("TableName", "song").Tracef(
		"[S%d]Start: Create psql table (if not exists)", i,
	)

	if _, err := store.db.Exec(
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
            PRIMARY KEY (id)
        );

        DO $$
        DECLARE X BOOL := (
            SELECT EXISTS (
                SELECT FROM information_schema.tables
                WHERE table_schema = 'schema_name'
                    AND table_name = 'queue'
                )
            );
        BEGIN
        IF X THEN
            ALTER TABLE "song"
            ADD CONSTRAINT "queue_fk"
            FOREIGN KEY (queue_client_id, queue_guild_id)
                REFERENCES "queue" (client_id, guild_id)
                    ON DELETE CASCADE;
        END IF;
        END $$;
        `,
	); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		return err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : psql table created", i)
	return nil
}

// createInactiveSongTable creates the "inactive_song" table
// with all it's constraints
// if it does  not already exist
func (store *SongStore) createInactiveSongTable() error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithField("TableName", "inactive_song").Tracef(
		"[S%d]Start: Create psql table (if not exists)", i,
	)

	if _, err := store.db.Exec(
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
            PRIMARY KEY (id)
        );

        DO $$
        DECLARE X BOOL := (
            SELECT EXISTS (
                SELECT FROM information_schema.tables
                WHERE table_schema = 'schema_name'
                    AND table_name = 'queue'
                )
            );
        BEGIN
        IF X THEN
            ALTER TABLE "inactive_song"
            ADD CONSTRAINT "queue_fk"
            FOREIGN KEY (queue_client_id, queue_guild_id)
                REFERENCES "queue" (client_id, guild_id)
                    ON DELETE CASCADE;
        END IF;
        END $$;
        `,
	); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		return err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : psql table created", i)
	return nil
}

// dropSongTable drops the "song" table.
func (store *SongStore) dropSongTable() error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithField("TableName", "song").Tracef(
		"[S%d]Start: Drop psql table (if exists)", i,
	)

	if _, err := store.db.Exec(
		`DROP TABLE IF EXISTS "song" CASCADE`,
	); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		return err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : psql table dropped", i)
	return nil
}

// dropInactiveSongTable drops the "inactive_song" table.
func (store *SongStore) dropInactiveSongTable() error {
	i, t := store.idx, time.Now()
	store.idx++

	store.log.WithField("TableName", "inactive_song").Tracef(
		"[S%d]Start: Drop psql table (if exists)", i,
	)

	if _, err := store.db.Exec(
		`DROP TABLE IF EXISTS "inactive_song" CASCADE`,
	); err != nil {
		store.log.Tracef(
			"[S%d]Error: %v", i, err,
		)
		return err
	}
	store.log.WithField(
		"Latency", time.Since(t),
	).Tracef("[S%d]Done : psql table dropped", i)
	return nil
}
