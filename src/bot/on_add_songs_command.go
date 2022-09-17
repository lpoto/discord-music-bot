package bot

import (
	"github.com/bwmarrin/discordgo"
	"github.com/google/uuid"
)

// onAddSongsComamnd is a handler function called when the bot's
// add songs command is called from queue message's context menu.
// This is called from INTERACTION_CREATE event when
// the interaction's command data name matches the add songs
// message command's name.
func (bot *Bot) onAddSongsComamnd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Add songs message command")
	m := bot.getModal(bot.addSongsComponents())
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
	textInput := discordgo.TextInput{
		CustomID: uuid.NewString(),
		Label:    "Enter names or urls to youtube songs",
		Placeholder: `song name or url #1
song name or url  #2
        ...`,
		Style:     discordgo.TextInputParagraph,
		MinLength: 1,
		MaxLength: 4000,
		Required:  true,
	}
	return []discordgo.MessageComponent{textInput}
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
