package bot

import (
	"context"
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/bot/blocked_command"
	"discord-music-bot/bot/modal"
	"discord-music-bot/bot/slash_command"
	"discord-music-bot/bot/transaction"
	"discord-music-bot/builder"
	"discord-music-bot/datastore"
	"discord-music-bot/service"
	"discord-music-bot/youtube"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type Bot struct {
	log             *log.Logger
	ctx             context.Context
	ready           bool
	_ready          bool
	service         *service.Service
	builder         *builder.Builder
	datastore       *datastore.Datastore
	youtube         *youtube.Youtube
	audioplayers    *audioplayer.AudioPlayersMap
	transactions    *transaction.Transactions
	blockedCommands *blocked_command.BlockedCommands
	session         *discordgo.Session
	config          *Configuration
	helpContent     string
}

type Configuration struct {
	LogLevel      log.Level                          `yaml:"LogLevel" validate:"required"`
	DiscordToken  string                             `yaml:"DiscordToken" validate:"required"`
	Datastore     *datastore.Configuration           `yaml:"Datastore" validate:"required"`
	Builder       *builder.Configuration             `yaml:"Builder" validate:"required"`
	SlashCommands *slash_command.SlashCommandsConfig `yaml:"SlashCommands" validate:"required"`
	Modals        *modal.ModalsConfig                `yaml:"Modals"`
	MaxAloneTime  time.Duration                      `yaml:"MaxAloneTime" validate:"required"`
}

// NewBot constructs an object that connects the logic in the
// service module with the discord api and the datastore.
func NewBot(ctx context.Context, config *Configuration, help string) *Bot {
	l := log.New()
	l.SetLevel(config.LogLevel)
	l.Debug("Creating Discord music bot ...")

	bot := &Bot{
		ctx:             ctx,
		log:             l,
		ready:           false,
		_ready:          false,
		service:         service.NewService(),
		builder:         builder.NewBuilder(config.Builder),
		datastore:       datastore.NewDatastore(config.Datastore),
		youtube:         youtube.NewYoutube(),
		config:          config,
		audioplayers:    audioplayer.NewAudioPlayersMap(),
		blockedCommands: blocked_command.NewBlockedCommands(),
		session:         nil,
		helpContent:     help,
	}
	bot.transactions = transaction.NewTransactions(
		func() *discordgo.Session { return bot.session },
		bot.log,
		bot.datastore,
		bot.builder,
		func() bool { return bot._ready },
	)
	l.Info("Discord music bot created")
	return bot
}

// Init authorized the provided token through the discord client,
// connects to a postgres database based on the provided config,
// initialized the tables and other initial data in the datastore and
// registers all the required commands through the discord api.
func (bot *Bot) Init() error {
	bot.log.Debug("Initializing the bot ...")

	if err := bot.datastore.Connect(); err != nil {
		return err
	}
	if err := bot.datastore.Init(bot.ctx, true); err != nil {
		return err
	}
	bot.log.Info("Bot initialized")
	return nil
}

// Run is a long lived worker that creates a new discord session,
// verifies it, adds required intents and discord event handlers,
// then runs while the context is alive.
func (bot *Bot) Run() {
	done := bot.ctx.Done()

	bot.log.Info("Creating new Discord session...")
	session, err := discordgo.New("Bot " + bot.config.DiscordToken)
	if err != nil {
		bot.log.Panic(err)
	}
	bot.session = session

	// Set intents required by the bot
	intentsHandler := &DiscordIntentsHandler{bot}
	intentsHandler.setIntents()

	// Set handlers for events emitted by the discord
	eventHandler := &DiscordEventHandler{bot}
	eventHandler.setHandlers()

	if err := session.Open(); err != nil {
		bot.log.Panic(err)
	}

	// Register slash commands required by the bot
	bot.log.Debug("Registering global slash commands ...")
	if err := slash_command.Register(
		bot.session,
		bot.config.SlashCommands,
	); err != nil {
		bot.log.Warn(err)
	}

	util := &Util{bot}

	defer func() {
		bot.ready = false
		bot._ready = false
		util.cleanDiscordMusicQueues()
		bot.log.Info("Closing discord session ... ")
		bot.session.Close()
	}()

	for {
		select {
		case <-done:
			return
		}
	}
}
