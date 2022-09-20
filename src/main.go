package main

import (
	"context"
	"discord-music-bot/bot"
	"discord-music-bot/builder"
	"discord-music-bot/client/youtube"
	"discord-music-bot/config"
	"discord-music-bot/datastore"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

type MusicBot struct {
	Config *Configuration `yaml:"MusicBot" validate:"required"`
}

type Configuration struct {
	LogLevel            log.Level                      `yaml:"LogLevel" validate:"required"`
	DiscordToken        string                         `yaml:"DiscordToken" validate:"required"`
	Datastore           *datastore.Configuration       `yaml:"Datastore" validate:"required"`
	QueueBuilder        *builder.Configuration         `yaml:"QueueBuilder" validate:"required"`
	ApplicationCommands *bot.ApplicationCommandsConfig `yaml:"ApplicationCommands" validate:"required"`
	Youtube             *youtube.Configuration         `yaml:"Youtube" validate:"required"`
}

// initBot creates a new bot object with the provided config,
// initializes it and returns the bot object
func initBot(ctx context.Context, configuration *Configuration) *bot.Bot {
	bot := bot.NewBot(
		ctx,
		configuration.LogLevel,
		configuration.ApplicationCommands,
		configuration.QueueBuilder,
		configuration.Datastore,
		configuration.Youtube,
	)
	if err := bot.Init(); err != nil {
		log.Panic(err)
	}
	return bot
}

// loadConfig loads the config from the provided yaml
// files into the Configuration object, panics on error
func loadConfig(configFiles []string) *Configuration {
	var musicBot MusicBot
	err := config.LoadAndValidateConfiguration(configFiles, &musicBot)
	if err != nil {
		log.Panic(err)
	}
	return musicBot.Config

}

func main() {
	configFileParam := flag.String(
		"configFiles",
		"config.yaml",
		"File with configuration",
	)
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	shutdownSignal := make(chan os.Signal, 2)
	signal.Notify(shutdownSignal, syscall.SIGTERM, syscall.SIGINT)

	configuration := loadConfig(strings.Split(*configFileParam, ","))

	bot := initBot(ctx, configuration)

	go func() {
		// graceful shutdown
		<-shutdownSignal
		log.Println()
		log.Warn("Shutdown requested ...")
		cancel()
		select {
		case <-time.After(time.Second * 10):
		}
		log.Fatal("Forced shutdown")
	}()

	bot.Run(configuration.DiscordToken)
	log.Print("Clean Shutdown")
}
