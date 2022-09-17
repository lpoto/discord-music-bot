package bot

import (
	"discord-music-bot/model"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

type ComponentsConfig struct {
	Backward string `yaml:"Forward" validate:"required"`
	Forward  string `yaml:"Backward" validate:"required"`
	Pause    string `yaml:"Pause" validate:"required"`
	Skip     string `yaml:"Skip" validate:"required"`
	Previous string `yaml:"Previous" validate:"required"`
	Replay   string `yaml:"Replay" validate:"required"`
	AddSongs string `yaml:"AddSongs" validate:"required"`
	Loop     string `yaml:"Loop" validate:"required"`
}

// getMusicQueueComponents constructs a list od message components
// that belong to the provided queue, they may vary based on
// the queue's options
func (bot *Bot) getMusicQueueComponents(queue *model.Queue) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				bot.newButton(bot.componentsConfig.Forward, discordgo.SecondaryButton, false),
				bot.newButton(bot.componentsConfig.Backward, discordgo.SecondaryButton, false),
				bot.newButton(bot.componentsConfig.Previous, discordgo.SecondaryButton, false),
				bot.newButton(bot.componentsConfig.Skip, discordgo.SecondaryButton, false),
			},
		},
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				bot.newButton(bot.componentsConfig.AddSongs, discordgo.SecondaryButton, false),
				bot.newButton(bot.componentsConfig.Loop, discordgo.SecondaryButton, false),
				bot.newButton(bot.componentsConfig.Pause, discordgo.SecondaryButton, false),
				bot.newButton(bot.componentsConfig.Replay, discordgo.SecondaryButton, false),
			},
		},
	}
}

// newButton constructs a new button with
// the provided label and style
func (bot *Bot) newButton(label string, style discordgo.ButtonStyle, disabled bool) discordgo.Button {
	return discordgo.Button{
		CustomID: uuid.NewString(),
		Label:    label,
		Style:    style,
		Disabled: disabled,
	}
}

// getModal constructs a modal submit interaction data
// with the provided components
func (bot *Bot) getModal(components []discordgo.MessageComponent) *discordgo.ModalSubmitInteractionData {
	return &discordgo.ModalSubmitInteractionData{
		CustomID: uuid.NewString(),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: components,
			},
		},
	}
}
