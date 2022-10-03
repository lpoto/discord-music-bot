package modal

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

type ModalConfig struct {
	Name        string `yaml:"Name" validate:"required"`
	Label       string `yaml:"Label" validate:"required"`
	Placeholder string `yaml:"Placeholder" validate:"required"`
}

type ModalsConfig struct {
	AddSongs *ModalConfig `yaml:"AddSongs" validate:"required"`
}

// GetModal constructs a modal submit interaction data
// with the provided components
func GetModal(name string, components []discordgo.MessageComponent) *discordgo.ModalSubmitInteractionData {
	return &discordgo.ModalSubmitInteractionData{
		CustomID: name + "<split>" + uuid.NewString(),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: components,
			},
		},
	}
}

// GetModalName retrieves the name of the modal from
// it's customID
func GetModalName(data discordgo.ModalSubmitInteractionData) string {
	return strings.Split(data.CustomID, "<split>")[0]
}
