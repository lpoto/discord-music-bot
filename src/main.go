package main

import (
	"context"
	"discord-music-bot/bot"
	"discord-music-bot/config"
	"flag"
	"io/ioutil"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	defaultLog "log"

	log "github.com/sirupsen/logrus"
)

type Configuration struct {
	MusicBot *bot.Configuration `yaml:"MusicBot" validate:"required"`
}

// initBot creates a new bot object with the provided config,
// initializes it and returns the bot object
func initBot(ctx context.Context, configuration *Configuration, help string) *bot.Bot {
	bot := bot.NewBot(
		ctx,
		configuration.MusicBot,
		help,
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

// loadHelp loads the content from the provided help
// files, that holds the information about using the bot
func loadHelp(helpFiles []string) string {
	help := make([]string, 0)
	for _, helpFilePath := range helpFiles {
		content, err := ioutil.ReadFile(helpFilePath)
		if err != nil {
			log.Panic(err)
		}
		help = append(help, string(content))
	}
	return strings.Join(help, "\n")
}

func main() {
	// NOTE: allow only logging with logrus
	defaultLog.SetOutput(ioutil.Discard)

	configFileParam := flag.String(
		"configFiles",
		"../conf/config.yaml",
		"File with configuration",
	)
	helpFileParam := flag.String(
		"helpFiles",
		"../conf/help.txt",
		"File with information about the commands",
	)
	flag.Parse()
	ctx, cancel := context.WithCancel(context.Background())
	shutdownSignal := make(chan os.Signal, 2)
	signal.Notify(shutdownSignal, syscall.SIGTERM, syscall.SIGINT)

	configuration := loadConfig(strings.Split(*configFileParam, ","))
	help := loadHelp(strings.Split(*helpFileParam, ","))

	bot := initBot(ctx, configuration, help)

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
