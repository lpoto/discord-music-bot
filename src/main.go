package main

import (
	"context"
	"discord-music-bot/bot"
	"discord-music-bot/config"
	"flag"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	MusicBot *bot.Configuration `yaml:"MusicBot" validate:"required"`
}

// initBot creates a new bot object with the provided config,
// initializes it and returns the bot object
func initBot(ctx context.Context, configuration *Configuration) *bot.Bot {
	bot := bot.NewBot(
		ctx,
		configuration.MusicBot,
	)
	if err := bot.Init(); err != nil {
		log.Panic(err)
	}
	return bot
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

	bot.Run()
	log.Print("Clean Shutdown")
}
