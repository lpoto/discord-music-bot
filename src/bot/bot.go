package bot

import (
	"context"
	"discord-music-bot/client/youtube"
	"discord-music-bot/datastore"
	"discord-music-bot/service"
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type Bot struct {
	*log.Logger
	service             *service.Service
	datastore           *datastore.Datastore
	youtubeClient       *youtube.YoutubeClient
	slashCommandsConfig *SlashCommandsConfig
	datastoreConfig     *datastore.Configuration
}

type SlashCommandConfig struct {
	Name        string `yaml:"Name" validate:"required"`
	Description string `yaml:"Description" validate:"required"`
}

type SlashCommandsConfig struct {
	Music *SlashCommandConfig `yaml:"Music" validate:"required"`
	Help  *SlashCommandConfig `yaml:"Help" validate:"required"`
}

// NewBot constructs an object that connects the logic in the
// service module with the discord api and the datastore.
func NewBot(logLevel log.Level, slashSlashCommandsConfig *SlashCommandsConfig, datastoreConfig *datastore.Configuration) *Bot {
	l := log.New()
	l.SetLevel(logLevel)
	l.Debug("Creating Discord music bot ...")

	bot := &Bot{
		Logger:              l,
		service:             service.NewService(logLevel),
		datastore:           datastore.NewDatastore(logLevel),
		youtubeClient:       youtube.NewYoutubeClient(logLevel),
		slashCommandsConfig: slashSlashCommandsConfig,
		datastoreConfig:     datastoreConfig,
	}
	l.Info("Discord music bot created")
	return bot
}

// Init authorized the provided token through the discord client,
// connects to a postgres database based on the provided config,
// initialized the tables and other initial data in the datastore and
// registers all the required commands through the discord api.
func (bot *Bot) Init(ctx context.Context) error {
	bot.Debug("Initializing the bot ...")

	if err := bot.datastore.Connect(bot.datastoreConfig); err != nil {
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
	session, err := discordgo.New("Bot " + token)
	if err != nil {
		bot.Panic(err)
	}
	// Set intents required by the bot
	bot.setIntents(session)
	// Set handlers for events emitted by the discord
	bot.setHandlers(session)

	if err := session.Open(); err != nil {
		bot.Panic(err)
	}

	// Register slash commands required by the bot
	if err := bot.setSlashCommands(session); err != nil {
		bot.Panic(err)
	}

	defer func() {
		bot.Info("Closing discord session ... ")
		session.Close()
	}()

	// Run loop until the context is done
	// All logic is performed by the handlers
	for {
		select {
		case <-done:
			return
		}
	}
}

// setIntents sets the intents for the session, required
// by the music bot
func (bot *Bot) setIntents(session *discordgo.Session) {
	//NOTE: guilds for interactions in guilds,
	// guild messages for message delete events,
	// voice states for voice state update events
	session.Identify.Intents =
		discordgo.IntentsGuilds +
			discordgo.IntentsGuildVoiceStates +
			discordgo.IntentGuildMessages

}

// setHandlers adds handlers for discord events to the
// provided session
func (bot *Bot) setHandlers(session *discordgo.Session) {
	session.AddHandler(bot.onReady)
	session.AddHandler(bot.onMessageDelete)
	session.AddHandler(bot.onBulkMessageDelete)
	session.AddHandler(bot.onVoiceStateUpdate)
	session.AddHandler(bot.onInteractionCreate)
}

// setSlashCommands deletes all of the bot's previously
// registers slash commands, then registers the new
// music and help slash commands
func (bot *Bot) setSlashCommands(session *discordgo.Session) error {
	bot.Debug("Registering global application commands ...")
	// NOTE: guildID  is an empty string, so the commands are
	// global
	guildID := ""
	// fetch all global application commands defined by
	// the bot user
	registeredCommands, err := session.ApplicationCommands(
		session.State.User.ID,
		guildID,
	)
	if err != nil {
		e := errors.New(
			fmt.Sprintf(
				"Could not fetch global application commands: %v",
				err,
			),
		)
		return e
	}
	// delete the fetched global application commands
	for _, v := range registeredCommands {
		bot.WithField("Name", v.Name).Trace(
			"Deleting global application command",
		)
		if err := session.ApplicationCommandDelete(
			session.State.User.ID,
			guildID,
			v.ID,
		); err != nil {
			e := errors.New(
				fmt.Sprintf(
					"Could not delete global application command '%v': %v",
					v.Name,
					err,
				),
			)
			return e
		}
	}
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        bot.slashCommandsConfig.Music.Name,
			Description: bot.slashCommandsConfig.Music.Description,
		},
		{
			Name:        bot.slashCommandsConfig.Help.Name,
			Description: bot.slashCommandsConfig.Help.Description,
		},
	}
	// register the global application commands
	for _, cmd := range commands {
		bot.WithField("Name", cmd.Name).Trace(
			"Registering global application command",
		)
		if _, err := session.ApplicationCommandCreate(
			session.State.User.ID,
			guildID,
			cmd,
		); err != nil {
			e := errors.New(
				fmt.Sprintf(
					"Could not create global application command '%v': %v",
					cmd.Name,
					err,
				),
			)
			return e
		}
	}
	bot.Debug("Successfully registered global application commands")
	return nil
}
