package youtube

import (
	"discord-music-bot/model"
	"errors"
	"strconv"
	"sync"
	"time"
)

// SearchSongs searches the provided queries on the youtube and
// recieved the found videos' information. Always returns the first
// search result. If the query is a youtube video url, the url is used
// for fetching the info.
func (client *YoutubeClient) SearchSongs(queries []string) []*model.SongInfo {
	i, t := client.GetIdx(), time.Now()

	client.Tracef("[%d]Youtube start: Search %d song/s on Youtube", i, len(queries))

	if len(queries) > client.Config.MaxParallelQueries {
		client.Tracef(
			"[%d]Youtube error: Tried to query more than %d songs",
			i,
			client.Config.MaxParallelQueries,
		)
		return []*model.SongInfo{}
	}
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
			info, err := client.searchSong(query)
			if err != nil {
				client.Tracef("[%d]Youtube error: %v", i, err)
				return
			}
			if prevWG != nil {
				prevWG.Wait()
			}
			select {
			case songBuffer <- info:
			default:
				client.Panic("Song buffer full")
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
		client.WithField(
			"Latency", time.Since(t),
		).Tracef("[%d]Youtube done : Found %d song/s", i, len(songs))
		return songs
	}
	client.WithField(
		"Latency", time.Since(t),
	).Tracef("[%d]Youtube done : Found no songs", i)
	return []*model.SongInfo{}
}

func (client *YoutubeClient) searchSong(q string) (*model.SongInfo, error) {
	endpoint := YoutubeVideoEndpoint
	videoID := ""

	if id, ok := client.extractYoutubeVideoID(q); !ok {
		if id2, err := client.getVideoIDFromQuery(q); err != nil {
			return nil, err
		} else {
			videoID = id2
		}
	} else {
		videoID = id
	}

	req := client.Get(endpoint).AddQueryParam(
		YoutubeVideoIDQueryParam,
		videoID,
	)

	info := new(model.SongInfo)
	info.Url = req.Url()

	b, err := req.DoAndRead()
	if err != nil {
		return nil, err
	}
	s := string(b)
	content := client.getRegExpGroupValues(
		`"videoDetails":{.*?`+
			`("videoId":"(?P<videoID>.*?)")|`+
			`("title":"(?P<title>.*?)")|`+
			`("lengthSeconds":"(?P<lengthSeconds>.*?)")`,
		s,
		[]string{"videoID", "title", "lengthSeconds"},
	)
	err = errors.New("Invalid query param: " + q)
	if v, ok := content["title"]; ok {
		info.Name = v
	} else {
		return nil, err
	}
	if v, ok := content["videoID"]; ok {
		info.VideoID = v
	} else {
		return nil, err
	}
	if v, ok := content["lengthSeconds"]; ok {
		if i, err := strconv.Atoi(v); err == nil {
			info.LengthSeconds = i
		} else {
			return nil, err
		}
	} else {
		return nil, err
	}
	info.Name = client.decodeJsonEncoding(info.Name)
	return info, nil
}

func (client *YoutubeClient) getVideoIDFromQuery(query string) (string, error) {

	r := client.Get("/results").AddQueryParam("search_query", query)

	body, err := r.DoAndRead()
	if err != nil {
		return query, err
	}
	s := string(body)
	content := client.getRegExpGroupValues(
		`"videoId":"(?P<videoID>.*?)"`,
		s,
		[]string{"videoID"},
	)
	if v, ok := content["videoID"]; ok {
		return v, nil
	}
	return query, errors.New("Invalid query param: " + query)
}
