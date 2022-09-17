package bot

import (
	"github.com/bwmarrin/discordgo"
)

// onButtonClick is a handler function called when a user
// clicks a button on a message owned by the bot.
// This is not emitted through the discord websocket, but is rather
// called from the INTERACTION_CREATE event when the interaction type
// is button click and the message author is bot
func (bot *Bot) onButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	label := bot.builder.GetButtonLabelFromComponentData(i.MessageComponentData())
	bot.WithField("GuildID", i.GuildID).Tracef("Button clicked (%s)", label)

	switch label {
	case bot.builder.Config.Components.AddSongs:
		bot.onAddSongsCommand(s, i)
		return
	default:
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Sorry, something went wrong ...",
				Flags:   1 << 6, // Ephemeral
			},
		})
	}

}
