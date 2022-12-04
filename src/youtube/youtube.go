package youtube

import (
	"discord-music-bot/youtube/search"
	"discord-music-bot/youtube/stream"
)

type Youtube struct {
	search *search.Search
    stream *stream.Stream
}

// NewYoutube constructs an object that handles
// youtube integration
func NewYoutube() *Youtube {
	return &Youtube{
		search: search.NewSearch(),
        stream: stream.NewStream(),
	}
}

// Search returns an object that handles searching
// the youtube.
func (y *Youtube) Search() *search.Search {
	return y.search
}

// Stream returns an object that handles searching
// the youtube.
func (y *Youtube) Stream() *stream.Stream {
	return y.stream
}
