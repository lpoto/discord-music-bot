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
	apm.mutex.Lock()
	defer apm.mutex.Unlock()
	apm.audiopayers[guildID] = ap
}

// Remove removes the audioplayer from the map
// for the provided guildiD
func (apm *AudioPlayersMap) Remove(guildID string) {
	apm.mutex.Lock()
	defer apm.mutex.Unlock()
	apm.audiopayers[guildID] = nil
	delete(apm.audiopayers, guildID)
}

// Get returns the audioplayer for the guildID from the map
// returns nil, false if there is no such audioplayer
func (apm *AudioPlayersMap) Get(guildID string) (*AudioPlayer, bool) {
	apm.mutex.Lock()
	defer apm.mutex.Unlock()

	ap, ok := apm.audiopayers[guildID]
	return ap, ok
}

// Keys returns a list of slices containing all the guildIDs
// in the map
func (apm *AudioPlayersMap) Keys() []string {
	apm.mutex.Lock()
	defer apm.mutex.Unlock()
	keys := make([]string, 0, len(apm.audiopayers))
	for k := range apm.audiopayers {
		keys = append(keys, k)
	}
	return keys
}
