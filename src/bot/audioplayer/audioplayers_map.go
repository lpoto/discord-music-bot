package audioplayer

import "sync"

type AudioPlayersMap struct {
	audiopayers map[string]*AudioPlayer
	mutex       sync.Mutex
}

// NewAudioPlayersMap constructs a new object
// that holds audioplayers for all the guilds
func NewAudioPlayersMap() *AudioPlayersMap {
	return &AudioPlayersMap{
		audiopayers: make(map[string]*AudioPlayer),
		mutex:       sync.Mutex{},
	}
}

// Add adds the provided audioplayer as the value
// to the map with the provided guildID as key
func (apm *AudioPlayersMap) Add(guildID string, ap *AudioPlayer) {
	apm.audiopayers[guildID] = ap
}

// Remove removes the audioplayer from the map
// for the provided guildiD
func (apm *AudioPlayersMap) Remove(guildID string) {
	apm.audiopayers[guildID] = nil
	delete(apm.audiopayers, guildID)
}

// Get returns the audioplayer for the guildID from the map
// returns nil, false if there is no such audioplayer
func (apm *AudioPlayersMap) Get(guildID string) (*AudioPlayer, bool) {
	ap, ok := apm.audiopayers[guildID]
	return ap, ok
}
