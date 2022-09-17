package service

type Service struct{}

// NewService constructs an object that holds the logic
// behind the bot's commands.
func NewService() *Service {
	return &Service{}
}
