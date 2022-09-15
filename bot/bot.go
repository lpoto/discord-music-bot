package bot

import (
	"context"
	"discord-music-bot/client/youtube"
	"discord-music-bot/datastore"
	"discord-music-bot/service"
	"errors"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type Bot struct {
	*log.Logger
	discord       *discordgo.Session
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
func (bot *Bot) Init(ctx context.Context, datastoreConfig *datastore.Configuration) error {
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

// Run is a long lived worker that creates a new discord session,
// verifies it, adds required intents and discord event handlers,
// then runs while the context is alive.
func (bot *Bot) Run(ctx context.Context, token string) {
	done := ctx.Done()

	bot.Info("Creating new Discord session...")
	if discord, err := discordgo.New("Bot " + token); err != nil {
		bot.Panic(err)
	} else {
		// NOTE: fetch bot user to verify the provided
		// token was valid
		user, err := discord.User("@me")
		if err != nil {
			bot.Panic(errors.New("Invalid token: " + token))
		}
		bot.Infof(
			"Logged in with bot %s #%s",
			user.Username, user.Discriminator,
		)
		bot.discord = discord
		defer bot.discord.Close()
	}
	// Set intents required by the bot
	bot.setIntents()
	// Set handlers for events emitted by the discord
	bot.setHandlers()

	// Run loop until the context is done
	// All logic is performed by the handlers
	for {
		select {
		case <-done:
			return
		}
	}
}

func (bot *Bot) setIntents() {

}

func (bot *Bot) setHandlers() {

}
