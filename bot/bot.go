package bot

import (
	"context"
	"discord-music-bot/client/youtube"
	"discord-music-bot/datastore"
	"discord-music-bot/service"

	log "github.com/sirupsen/logrus"
)

type Bot struct {
	*log.Logger
	service       *service.Service
	datastore     *datastore.Datastore
	youtubeClient *youtube.YoutubeClient
}

// NewBot constructs an object that connects the logic in the
// service module with the discord api and the datastore.
func NewBot(logLevel log.Level) *Bot {
	l := log.New()
	l.SetLevel(logLevel)
	l.Debug("Creating Discord music bot ...")

	bot := &Bot{
		Logger:        l,
		service:       service.NewService(logLevel),
		datastore:     datastore.NewDatastore(logLevel),
		youtubeClient: youtube.NewYoutubeClient(logLevel),
	}
	l.Info("Discord music bot created")
	return bot
}

// Init authorized the provided token through the discord client,
// connects to a postgres database based on the provided config,
// initialized the tables and other initial data in the datastore and
// registers all the required commands through the discord api.
func (bot *Bot) Init(ctx context.Context, token string, datastoreConfig *datastore.Configuration) error {
	bot.Debug("Initializing the bot ...")

	if err := bot.datastore.Connect(datastoreConfig); err != nil {
		return err
	}
	if err := bot.datastore.Init(ctx); err != nil {
		return err
	}
	bot.Info("Bot initialized")
	return nil
}

// Run logs in the bot by subscribing to the discord api websocket,
// and listens to the relevant events comming through the websocket.
func (bot *Bot) Run(ctx context.Context) {
	bot.Info("Starting the bot ...")
}
