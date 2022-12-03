package song

import (
	"discord-music-bot/model"
	"math/rand"
)

type SongService struct{}

// NewSongService constructs an object that holds some
// logic for manipulating songs.
func NewSongService() *SongService {
	return &SongService{}
}

// ShuffleSongs shuffles the positions of the provided songs without
// adding any new positions or chaning any of the other fields. The position
// of the song with the minimum position is not changed.
// NOTE: this is performed in place.
// NOTE: this assumes the provided songs are ordered ascendingly by their positions.
func (service *SongService) ShuffleSongs(songs []*model.Song) []*model.Song {
	// NOTE: no point  in shuffling less than 3 songs, as they
	// won't be shuffled
	shuffled := make(map[uint]int)

	for idx, song := range songs {
		if _, ok := shuffled[song.ID]; ok || idx == 0 {
			continue
		}

		var song2 *model.Song
		for i := 0; i < 100; i++ {
			song2 = songs[rand.Intn(len(songs)-1)+1]
			if v, ok := shuffled[song2.ID]; (!ok || v != song.Position) &&
				song2.ID != song.ID {
				break
			}
		}
		shuffled[song.ID] = song.Position
		if _, ok := shuffled[song2.ID]; !ok {
			shuffled[song2.ID] = song2.Position
		}
		song.Position, song2.Position = song2.Position, song.Position
	}
	return songs
}
