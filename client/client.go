package base

type BaseClient struct {
	baseUrl string
	headers map[string]string
}

// NewClient construct a new object that handles http requests.
func NewClient(baseUrl string) *BaseClient {
	return (&BaseClient{
		baseUrl,
		make(map[string]string),
	}).AddHeader(
		"User-Agent", "DiscordMusicBot",
	).AddHeader(
		"Content-Type", "application/json",
	)
}

// AddHeader adds a header to every http request made
// by the client.
func (client *BaseClient) AddHeader(key string, value string) *BaseClient {
	client.headers[key] = value
	return client
}
