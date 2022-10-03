package bot

import "github.com/bwmarrin/discordgo"

// onJumpMessageCommand is a handler function called when the name of interaction's
// application command data matches the registered Jump global message command.
func (bot *Bot) onJumpMessageCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "This is currently not yet implemented",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
