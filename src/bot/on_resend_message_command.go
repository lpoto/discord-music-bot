package bot

import "github.com/bwmarrin/discordgo"

// onResendMessageCommand is a handler function called when the name of interaction's
// application command data matches the registered Resend global message command.
func (bot *Bot) onResendMessageCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "This is currently not yet implemented",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
