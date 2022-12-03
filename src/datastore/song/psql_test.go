package song_test

import (
	"database/sql"
	"discord-music-bot/datastore/song"
	"discord-music-bot/model"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/suite"
)

type SongStoreTestSuite struct {
	db              *sql.DB
	store           *song.SongStore
	inactiveSongTTL time.Duration
	suite.Suite
}

// SetupSuite runs when the suite is initialized and
// connects to the database and initialized the song store.
func (s *SongStoreTestSuite) SetupSuite() {
	db, err := sql.Open(
		"postgres",
		"host=postgres port=5432 user=postgres password=postgres "+
			"dbname=discord_bot_test sslmode=disable",
	)
	s.NoError(err)

	s.db = db
	s.inactiveSongTTL = 2 * time.Second
	s.store = song.NewSongStore(db, logrus.StandardLogger(), s.inactiveSongTTL)

	err = s.store.Destroy()
	s.NoError(err)
	err = s.store.Init()
	s.NoError(err)
}

// TearDownSuite runs after all tests have been run and destroys
// the song store and closes database connection.
func (s *SongStoreTestSuite) TearDownSuite() {
	err := s.store.Destroy()
	s.NoError(err)

	err = s.db.Close()
	s.NoError(err)
}

// TestIntegrationSongsCRUD first persists songs then
// fetches them, checks their fields and removes them.
// Checks also pushing head songs to back, last to front,
// removing head song...
func (s *SongStoreTestSuite) TestIntegrationSongsCRUD() {
	// First insert 3 songs normally into
	// the store
	err := s.store.PersistSongs(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
		&model.Song{
			ID:              1,
			Name:            "Song1",
			ShortName:       "Song1",
			Url:             "SongUrl1",
			DurationSeconds: 10,
			DurationString:  "00:10",
			Color:           0,
		},
		&model.Song{
			ID:              2,
			Name:            "Song2",
			ShortName:       "Song2",
			Url:             "SongUrl2",
			DurationSeconds: 10,
			DurationString:  "00:10",
			Color:           0,
		},
		&model.Song{
			ID:              3,
			Name:            "Song3",
			ShortName:       "Song3",
			Url:             "SongUrl3",
			DurationSeconds: 10,
			DurationString:  "00:10",
			Color:           0,
		},
	)
	s.NoError(err)

	// Add a song to the front.
	// It's Position should then be the
	// smallest of the added songs
	err = s.store.PersistSongToFront(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
		&model.Song{
			ID:              4,
			Name:            "Song4",
			ShortName:       "Song4",
			Url:             "SongUrl4",
			DurationSeconds: 10,
			DurationString:  "00:10",
			Color:           0,
		},
	)
	s.NoError(err)

	// Should get that there are 4 songs
	count := s.store.GetSongCountForQueue(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.Equal(count, 4)

	// should get songs with id 2 and 3 as they
	// are ordered by position and song with ID 4 has
	// been added to front
	songs, err := s.store.GetSongsForQueue(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
		2,
		2,
	)
	s.NoError(err)
	s.Len(songs, 2)
	s.Equal(uint(2), songs[0].ID)
	s.Equal(uint(3), songs[1].ID)

	// Should get all 4 songs. The first one
	// should be the one with id=4 and position=0
	// as it was added to front.
	// Others should have id and position equal
	// to their index in the slice.
	songs, err = s.store.GetAllSongsForQueue(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.NoError(err)
	s.Len(songs, 4)
	s.Equal(uint(1), songs[1].ID)
	s.Equal(1, songs[1].Position)
	s.Equal(uint(2), songs[2].ID)
	s.Equal(2, songs[2].Position)
	s.Equal(uint(3), songs[3].ID)
	s.Equal(3, songs[3].Position)

	s.Equal(uint(4), songs[0].ID)
	s.Equal(0, songs[0].Position)

	// Push the song with the smallest position back
	// and then fetch them again.
	// In this context song pushed back should be the one
	// with id=4. It should then have the largest position.
	s.store.PushHeadSongToBack(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.NoError(err)
	songs, err = s.store.GetSongsForQueue(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
		0,
		4,
	)
	s.NoError(err)
	s.Len(songs, 4)
	s.Equal(uint(1), songs[0].ID)
	s.Equal(1, songs[0].Position)
	s.Equal(uint(4), songs[3].ID)
	s.Equal(4, songs[3].Position)

	// Not the song with ID=4 should be in front again
	err = s.store.PushLastSongToFront(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.NoError(err)
	songs, err = s.store.GetSongsForQueue(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
		0,
		2,
	)
	s.NoError(err)
	s.Len(songs, 2)
	s.Equal(uint(4), songs[0].ID)
	s.Equal(0, songs[0].Position)
	s.Equal(uint(1), songs[1].ID)
	s.Equal(1, songs[1].Position)

	// Not the song with ID=1 should be in front again
	err = s.store.PushHeadSongToBack(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.NoError(err)
	songs, err = s.store.GetAllSongsForQueue(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.NoError(err)
	s.Len(songs, 4)
	s.Equal(uint(1), songs[0].ID)
	s.Equal(1, songs[0].Position)
	s.Equal(uint(4), songs[3].ID)
	s.Equal(4, songs[3].Position)

	err = s.store.RemoveHeadSong(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.NoError(err)
	// Now the song with position 1 and id=1
	// should be removed
	songs, err = s.store.GetAllSongsForQueue(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.NoError(err)
	s.Len(songs, 3)
	s.Equal(2, songs[0].Position)
	s.Equal(uint(2), songs[0].ID)

	err = s.store.RemoveSongs(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
		2, 3,
	)
	s.NoError(err)
	// Now only the song with id=4 should remain
	songs, err = s.store.GetAllSongsForQueue(
		"CLIENT-ID-TEST",
		"GUILD-ID-TEST",
	)
	s.NoError(err)
	s.Len(songs, 1)
	s.Equal(uint(4), songs[0].ID)
	s.Equal(4, songs[0].Position)
}

// TestIntegrationSongsForQueue creates songs then
// updates a queue with songs and checks whether the queue
// not has the correct songs fields.
// TODO
func (s *SongStoreTestSuite) TestIntegrationSongsForQueue() {}

// TestIntegrationInactiveSongsCRUD first persists songs then
// fetches them and checks their fields.
// TODO
func (s *SongStoreTestSuite) TestIntegrationInactiveSongsCRUD() {}

// TestSongStorageTestSuite runs all tests under
// the SongStoreTestSuite suite.
func TestSongStorageTestSuite(t *testing.T) {
	suite.Run(t, new(SongStoreTestSuite))
}
