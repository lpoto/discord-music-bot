package service

import (
	"discord-music-bot/model"
	"math/rand"
)

// ShuffleSongs shuffles the positions of the provided songs without
// adding any new positions or chaning any of the other fields. The position
// of the song with the minimum position is not changed.
func (service *Service) ShuffleSongs(songs []*model.Song) []*model.Song {
	// NOTE: no point  in shuffling less than 3 songs, as they
	// won't be shuffled
	if len(songs) < 3 {
		return songs
	}
	// NOTE: find the id of the song with minum position
	minPos, minID := -1, -1
	for _, song := range songs {
		if minID == -1 || song.Position < minPos {

			minPos, minID = song.Position, int(song.ID)
		}
	}
	for _, song := range songs {
		if song.ID == uint(minID) {
			continue
		}
		song2 := songs[rand.Intn(len(songs))]
		for song2.ID == uint(minID) {
			song2 = songs[rand.Intn(len(songs))]
		}
		temp := song.Position
		song.Position = song2.Position
		song2.Position = temp
	}
	return songs
}
