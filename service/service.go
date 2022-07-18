package service

import log "github.com/sirupsen/logrus"

type Service struct {
	*log.Logger
}

// NewService constructs an object that holds the logic
// behind the bot's commands.
func NewService(logLevel log.Level) *Service {
	l := log.New()
	l.SetLevel(logLevel)
	l.Debug("Created a new service")
	return &Service{l}
}
