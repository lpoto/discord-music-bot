package client

type YoutubeClient struct {
	baseUrl string
	headers map[string]string
}

// NewYoutubeClient construct a new object that handles
// youtube http requests.
func NewYoutubeClient() *YoutubeClient {
	c := &YoutubeClient{
		baseUrl: "https://www.youtube.com",
		headers: make(map[string]string),
	}
	c.headers["Content-Type"] = "application/json"
	return c
}
