package bot

import (
	"context"
	"discord-music-bot/bot/audioplayer"
	"discord-music-bot/builder"
	"discord-music-bot/client/youtube"
	"discord-music-bot/datastore"
	"discord-music-bot/model"
	"discord-music-bot/service"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type Bot struct {
	*log.Logger
	ctx                           context.Context
	ready                         bool
	service                       *service.Service
	builder                       *builder.Builder
	datastore                     *datastore.Datastore
	youtubeClient                 *youtube.YoutubeClient
	audioplayers                  map[string]*audioplayer.AudioPlayer
	queueUpdateInteractionsBuffer map[string]chan *discordgo.Interaction
	blockedButtons                map[string]map[string]struct{}
	config                        *Configuration
}

type Configuration struct {
	LogLevel      log.Level                `yaml:"LogLevel" validate:"required"`
	DiscordToken  string                   `yaml:"DiscordToken" validate:"required"`
	Datastore     *datastore.Configuration `yaml:"Datastore" validate:"required"`
	QueueBuilder  *builder.Configuration   `yaml:"QueueBuilder" validate:"required"`
	SlashCommands *SlashCommandsConfig     `yaml:"SlashCommands" validate:"required"`
	Modals        *ModalsConfig            `yaml:"Modals"`
	Youtube       *youtube.Configuration   `yaml:"Youtube" validate:"required"`
}

// NewBot constructs an object that connects the logic in the
// service module with the discord api and the datastore.
func NewBot(ctx context.Context, config *Configuration) *Bot {
	l := log.New()
	l.SetLevel(config.LogLevel)
	l.Debug("Creating Discord music bot ...")

	bot := &Bot{
		ctx:                           ctx,
		Logger:                        l,
		ready:                         false,
		service:                       service.NewService(),
		builder:                       builder.NewBuilder(config.QueueBuilder),
		datastore:                     datastore.NewDatastore(config.Datastore),
		youtubeClient:                 youtube.NewYoutubeClient(config.Youtube),
		config:                        config,
		audioplayers:                  make(map[string]*audioplayer.AudioPlayer),
		queueUpdateInteractionsBuffer: make(map[string]chan *discordgo.Interaction),
		blockedButtons:                make(map[string]map[string]struct{}),
	}
	l.Info("Discord music bot created")
	return bot
}

// Init authorized the provided token through the discord client,
// connects to a postgres database based on the provided config,
// initialized the tables and other initial data in the datastore and
// registers all the required commands through the discord api.
func (bot *Bot) Init() error {
	bot.Debug("Initializing the bot ...")

	if err := bot.datastore.Connect(); err != nil {
		return err
	}
	if err := bot.datastore.Init(bot.ctx); err != nil {
		return err
	}
	bot.Info("Bot initialized")
	return nil
}

// Run is a long lived worker that creates a new discord session,
// verifies it, adds required intents and discord event handlers,
// then runs while the context is alive.
func (bot *Bot) Run() {
	done := bot.ctx.Done()

	bot.Info("Creating new Discord session...")
	session, err := discordgo.New("Bot " + bot.config.DiscordToken)
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
		bot.Warn(err)
	}

	// check if any queues should be removed from datastore
	bot.cleanDiscordMusicQueues(session)

	defer func() {
		bot.cleanDiscordMusicQueues(session)
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

// cleanDiscordMusicQueues removes all queue messages from datastore,
// for which the messages not longer exist in the discord channels.
// For those that exist, it marks them as paused
func (bot *Bot) cleanDiscordMusicQueues(session *discordgo.Session) {
	bot.Debug("Cleaning up discord music queues ...")

	queues, err := bot.datastore.FindAllQueues()
	if err != nil {
		bot.Errorf(
			"Error when checking if all queues exist: %v", err,
		)
		return
	}
	for _, queue := range queues {
		err := bot.datastore.RemoveQueueOptions(
			queue.ClientID,
			queue.GuildID,
			model.Paused,
		)
		if err == nil {
			err = bot.datastore.PersistQueueOptions(
				queue.ClientID,
				queue.GuildID,
				model.InactiveOption(),
			)
		}
		if err == nil {
			err = bot.updateQueue(session, queue.GuildID)
		}
		if err != nil {
			err = bot.datastore.RemoveQueue(
				queue.ClientID,
				queue.GuildID,
			)
			if err != nil {
				bot.Errorf(
					"Error when cleaning up queues : %v", err,
				)
			}

		}
	}
}
