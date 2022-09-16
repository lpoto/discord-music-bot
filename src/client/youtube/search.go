package youtube

import (
	"discord-music-bot/model"
	"errors"
	"strconv"
	"sync"
)

const (
	MaxSongQueries int = 100
)

// SearchSongs searches the provided queries on the youtube and
// recieved the found videos' information. Always returns the first
// search result. If the query is a youtube video url, the url is used
// for fetching the info.
func (client *YoutubeClient) SearchSongs(queries []string) ([]*model.SongInfo, error) {
	client.Trace("Searching for %d songs on Youtube", len(queries))

	if len(queries) == 0 {
		return nil, errors.New("No queries provided")
	}
	if len(queries) > MaxSongQueries {
		return nil, errors.New("Cannot query more than 100 songs at once")
	}
	added := make(map[string]struct{})
	songBuffer := make(chan *model.SongInfo, len(queries))
	var wg sync.WaitGroup

	for _, query := range queries {
		if _, ok := added[query]; ok {
			continue
		}
		wg.Add(1)
		added[query] = struct{}{}
		go func(query string) {
			if info, err := client.searchSong(query); err != nil {
				return
			} else {
				select {
				case songBuffer <- info:
				default:
					client.Panic("Song buffer full")
				}
			}

		}(query)
	}

	wg.Wait()

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
		client.Trace("Successfuly found %d songs", len(songs))
		return songs, nil
	}
	return []*model.SongInfo{}, nil
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
		info.TrimmedName = client.trimYoutubeSongName(v)
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
			info.DurationSeconds = i
			info.DurationString = client.secondsToTimeString(i)
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
