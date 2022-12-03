package song_test

import (
	"discord-music-bot/model"
	"discord-music-bot/service/song"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
)

type SongServiceTestSuite struct {
	suite.Suite
	service *song.SongService
}

// SetupSuite runs on suit init and creates
// the song service.
func (s *SongServiceTestSuite) SetupSuite() {
	s.service = song.NewSongService()
}

// TestUnitShuffleSongs shuffles from 3 to 1000 songs
// and makes sure they were at least 75% shuffled every time.
// It also asserts that the head song is never shuffled.
func (s *SongServiceTestSuite) TestUnitShuffleSongs() {
	for i := 3; i < 1000; i++ {
		songs := make([]*model.Song, 0)
		songs_copy := make([]*model.Song, 0)
		for j := 0; j < i; j++ {
			songs = append(songs, &model.Song{
				ID:       uint(j),
				Position: j,
				Name:     fmt.Sprintf("Song%d", j),
			})
			songs_copy = append(songs_copy, &model.Song{
				ID:       uint(j),
				Position: j,
				Name:     fmt.Sprintf("Song%d", j),
			})
		}
		diff := float32(0)
		songs = s.service.ShuffleSongs(songs)
		for idx, song := range songs {
			song_copy := songs_copy[idx]
			s.Equal(song_copy.Name, song.Name)
			s.Equal(song_copy.ID, song.ID)

			if idx == 0 {
				// NOTE: head song should never be shuffled
				s.Equal(song_copy.Position, song.Position)
				diff += 1
			} else if song_copy.Position != song.Position {
				diff += 1
			}
		}
		diff /= float32(i)
		s.GreaterOrEqual(diff, float32(0.7), fmt.Sprintf(
			"Shuffling %d songs only %v successful",
			i,
			diff,
		))
	}
}

// TestSongServiceTestSuite runs all tests under
// the SongServiceTestSuite
func TestSongServiceTestSuite(t *testing.T) {
	suite.Run(t, new(SongServiceTestSuite))
}
