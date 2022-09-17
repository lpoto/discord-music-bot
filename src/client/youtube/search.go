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
func (client *YoutubeClient) SearchSongs(queries []string) []*model.SongInfo {
	i := client.GetIdx()

	client.Tracef("Youtube start %d: Search %d song/s on Youtube", i, len(queries))

	if len(queries) > MaxSongQueries {
		client.Tracef("Youtube error %d: Tried to query more than %d songs", i, MaxSongQueries)
		return []*model.SongInfo{}
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
			defer func() {
				wg.Done()
			}()
			info, err := client.searchSong(query)
			if err != nil {
				client.Tracef("Youtube error %d: %v", i, err)
				return
			}
			select {
			case songBuffer <- info:
			default:
				client.Panic("Song buffer full")
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
		client.Tracef("Youtube done  %d: Found %d songs", i, len(songs))
		return songs
	}
	client.Tracef("Youtube done  %d: Found no songs", i)
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
