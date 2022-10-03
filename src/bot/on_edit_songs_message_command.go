package bot

import "github.com/bwmarrin/discordgo"

// onEditSongsMessageCommand is a handler function called when the name of interaction's
// application command data matches the registered EditSongs global message command.
func (bot *Bot) onEditSongsMessageCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "This is currently not yet implemented",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
