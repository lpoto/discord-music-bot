package youtube

import (
	base "discord-music-bot/client"

	log "github.com/sirupsen/logrus"
)

const (
	BaseYoutubeUrl string = "https://www.youtube.com"

	YoutubeVideoEndpoint     string = "/watch"
	YoutubeVideoIDQueryParam string = "v"
)

type YoutubeClient struct {
	*log.Logger
	*base.BaseClient
    idx int
}

// NewYoutubeClient constructs a new object that handles
// the requests send to the youtube.
func NewYoutubeClient(logLevel log.Level) *YoutubeClient {
	l := log.New()
	l.SetLevel(logLevel)
	l.Debug("Youtube client created")

	return &YoutubeClient{
		l,
		base.NewClient(BaseYoutubeUrl),
        0,
	}
}

// Get constructs a new Get http request for with the
// youtube + provided endpoint as url
func (client *YoutubeClient) Get(endpoint string) *base.Request {
	req, _ := client.NewRequest("GET", endpoint)
	return req
}

func (client *YoutubeClient) GetIdx() int {
    idx := client.idx
    client.idx = (client.idx + 1) % 100
    return idx
}
