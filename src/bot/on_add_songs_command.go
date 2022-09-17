package bot

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// onAddSongsComamnd is a handler function called when the bot's
// add songs command is called from queue message's context menu.
// This is called from AddSongs button click or music slash command
// if the queue already exists in the server
func (bot *Bot) onAddSongsCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Add songs message command")
	m := bot.getModal(
		bot.applicationCommandsConfig.AddSongs.Name,
		bot.addSongsComponents(),
	)
	if err := s.InteractionRespond(
		i.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Components: m.Components,
				CustomID:   m.CustomID,
				Title:      bot.applicationCommandsConfig.AddSongs.Name,
			},
		},
	); err != nil {
		bot.Errorf(
			"Error when responding with add songs modal: %v",
			err,
		)
	}
}

func (bot *Bot) addSongsComponents() []discordgo.MessageComponent {
	placeholder := `song name or url #1
song name or url  #2
...`
	textInput := discordgo.TextInput{
		CustomID:    uuid.NewString(),
		Label:       "Enter names or urls of youtube songs",
		Placeholder: placeholder,
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
