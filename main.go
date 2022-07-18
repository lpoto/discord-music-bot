package main

import (
	"context"
	"discord-music-bot/bot"
	"discord-music-bot/config"
	"discord-music-bot/datastore"
	"flag"
	"os"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	LogLevel     log.Level                `yaml:"LogLevel" validate:"required"`
	DiscordToken string                   `yaml:"DiscordToken" validate:"required"`
	Datastore    *datastore.Configuration `yaml:"Datastore" validate:"required"`
}

// initBot creates a new bot object with the provided config,
// initializes it and returns the bot object
func initBot(ctx context.Context, configuration *Configuration) (*bot.Bot, error) {
	bot := bot.NewBot(configuration.LogLevel)
	token := configuration.DiscordToken
	if err := bot.Init(ctx, token, configuration.Datastore); err != nil {
		return nil, err
	}
	return bot, nil
}

// loadConfig loads the config from the provided yaml
// files into the Configuration object, panics on error
func loadConfig(configFiles []string) *Configuration {
	var configuration Configuration
	err := config.LoadAndValidateConfiguration(configFiles, &configuration)
	if err != nil {
		log.Panic(err)
	}
	return &configuration

}

func main() {
	configFileParam := flag.String(
		"configFiles",
		"config.yaml",
		"File with configuration",
	)
	flag.Parse()
	shutdownSignal := make(chan os.Signal, 2)
	ctx, cancel := context.WithCancel(context.Background())
	configuration := loadConfig(strings.Split(*configFileParam, ","))

	bot, err := initBot(ctx, configuration)
	if err != nil {
		log.Panic(err.Error())
	}

	go func() {
		<-shutdownSignal
		cancel()
		log.Fatal("Shutdown requested")
	}()

	bot.Run(ctx)
}
