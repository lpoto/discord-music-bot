package search

import (
	"bytes"
	"discord-music-bot/model"
	"discord-music-bot/youtube/client"
	"errors"
	"regexp"
	"strconv"
	"sync"
)

type Search struct {
	client *client.YoutubeClient
}

// NewSearch constructs an object that handles
// searching songs on youtube either by url or query
func NewSearch() *Search {
	return &Search{
		client: client.NewYoutubeClient(),
	}
}

// GetSongs searches the provided queries on the youtube and
// recieved the found videos' information. Always returns the first
// search result. If the query is a youtube video url, the url is used
// for fetching the info.
func (s *Search) GetSongs(queries []string) []*model.SongInfo {
	added := make(map[string]struct{})
	songBuffer := make(chan *model.SongInfo, len(queries))
	var wg sync.WaitGroup

	var prevWG *sync.WaitGroup = nil

	// NOTE: run all queries in parallel, as each query may take
	// more than a second to complete
	for _, query := range queries {
		if _, ok := added[query]; ok {
			continue
		}
		// NOTE: all queries except the first one
		// need to wait for the previous query to complete
		// the search, so they are added to the channel in order
		wg2 := &sync.WaitGroup{}
		wg.Add(1)
		wg2.Add(1)
		added[query] = struct{}{}
		go func(query string, prevWG *sync.WaitGroup) {
			defer func() {
				wg.Done()
				wg2.Done()
			}()
			info, err := s.getSong(query)
			if err != nil {
				return
			}
			if prevWG != nil {
				prevWG.Wait()
			}
			select {
			case songBuffer <- info:
			default:
			}
		}(query, prevWG)

		prevWG = wg2
	}

	// NOTE: wait for all the queries to complete
	wg.Wait()

	// NOTE: build a slice from the songs
	// in the channel
	cnt := len(songBuffer)
	if cnt > 0 {
		songs := make([]*model.SongInfo, 0, len(songBuffer))
	songLoop:
		for i := 0; i < cnt; i++ {
			select {
			case song := <-songBuffer:
				songs = append(songs, song)
			default:
				break songLoop
			}
		}
		return songs
	}
	return []*model.SongInfo{}
}

func (s *Search) getSong(q string) (*model.SongInfo, error) {
	videoID := ""

	if id, ok := s.extractYoutubeVideoID(q); !ok {
		if id2, err := s.getVideoIDFromQuery(q); err != nil {
			return nil, err
		} else {
			videoID = id2
		}
	} else {
		videoID = id
	}

	b, url, err := s.client.NewWatchEndpointRequest(videoID)
	if err != nil {
		return nil, err
	}

	info := new(model.SongInfo)
	info.Url = url

	str := string(b)
	content := s.getRegExpGroupValues(
		`"videoDetails":{.*?`+
			`("videoId":"(?P<videoID>.*?)")|`+
			`("title":"(?P<title>.*?)")|`+
			`("lengthSeconds":"(?P<lengthSeconds>.*?)")`,
		str,
		[]string{"videoID", "title", "lengthSeconds"},
	)
	if v, ok := content["title"]; ok && len(v) > 0 {
		info.Name = v
	} else {
		return nil, errors.New("Failed to extract title for song query: " + q)
	}
	if v, ok := content["videoID"]; ok && len(v) > 0 {
		info.VideoID = v
	} else {
		return nil, errors.New("Failed to extract videoID for song query: " + q)
	}
	if v, ok := content["lengthSeconds"]; ok && len(v) > 0 {
		if i, err := strconv.Atoi(v); err == nil {
			info.LengthSeconds = i
		} else {
			return nil, errors.New("Failed to extract duration for song query: " + q)
		}
	} else {
		return nil, errors.New("Failed to extract duration for song query: " + q)
	}
	info.Name = s.unescapeHTML(info.Name)
	return info, nil
}

func (s *Search) getVideoIDFromQuery(query string) (string, error) {
	body, _, err := s.client.NewSearchRequest(query)
	if err != nil {
		return query, err
	}

	str := string(body)
	content := s.getRegExpGroupValues(
		`"videoId":"(?P<videoID>.*?)"`,
		str,
		[]string{"videoID"},
	)
	if v, ok := content["videoID"]; ok {
		return v, nil
	}
	return query, errors.New("Invalid query param: " + query)
}

func (s *Search) extractYoutubeVideoID(url string) (string, bool) {
	content := s.getRegExpGroupValues(
		`youtube.*watch\?v=(?P<videoID>[^&\/]*)`,
		url,
		[]string{"videoID"},
	)
	v, ok := content["videoID"]
	return v, ok
}

func (s *Search) getRegExpGroupValues(reString string, str string, groups []string) map[string]string {
	re := regexp.MustCompile(reString)
	matches := re.FindAllStringSubmatch(string(str), len(groups))
	values := make(map[string]string)
	for k, g := range groups {
		for i, v := range re.SubexpNames() {
			if k < len(matches) &&
				i < len(matches[k]) &&
				v == g &&
				len(matches[k][i]) > 0 {

				values[g] = matches[k][i]
			}
		}
	}
	return values
}

// unescapeHTML replaces \u0026 with &, \u003e with > and \u003c with <
func (s *Search) unescapeHTML(str string) string {
	b := []byte(str)
	b = bytes.Replace(b, []byte("\\u003c"), []byte("<"), -1)
	b = bytes.Replace(b, []byte("\\u003e"), []byte(">"), -1)
	b = bytes.Replace(b, []byte("\\u0026"), []byte("&"), -1)
	return string(b)
}
