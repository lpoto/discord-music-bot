package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// addSongs responds to the provided interaction with the
// add songs modal.
func (bot *Bot) addSongs(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	bot.WithField("GuildID", i.GuildID).Trace("Send add songs modal")

	m := bot.getModal(
		bot.config.Modals.AddSongs.Name,
		bot.addSongsComponents(),
	)
	if err := s.InteractionRespond(
		i.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Components: m.Components,
				CustomID:   m.CustomID,
				Title:      bot.config.Modals.AddSongs.Name,
			},
		},
	); err != nil {
		bot.Errorf(
			"Error when responding with add songs modal: %v",
			err,
		)
		return err
	}
	return nil
}

func (bot *Bot) addSongsComponents() []discordgo.MessageComponent {
	textInput := discordgo.TextInput{
		CustomID:    uuid.NewString(),
		Label:       bot.config.Modals.AddSongs.Label,
		Placeholder: bot.config.Modals.AddSongs.Placeholder,
		Style:       discordgo.TextInputParagraph,
		MinLength:   1,
		MaxLength:   4000,
		Required:    true,
	}
	return []discordgo.MessageComponent{textInput}
}

// getModal constructs a modal submit interaction data
// with the provided components
func (bot *Bot) getModal(name string, components []discordgo.MessageComponent) *discordgo.ModalSubmitInteractionData {
	return &discordgo.ModalSubmitInteractionData{
		CustomID: name + "<split>" + uuid.NewString(),
		Components: []discordgo.MessageComponent{
			discordgo.ActionsRow{
				Components: components,
			},
		},
	}
}

// getModalName retrieves the name of the modal from
// it's customID
func (bot *Bot) getModalName(data discordgo.ModalSubmitInteractionData) string {
	return strings.Split(data.CustomID, "<split>")[0]
}
