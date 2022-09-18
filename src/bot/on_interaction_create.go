package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// onInteractionCreate is a handler function called when discord emits
// INTERACTION_CREATE event. It determines the type of interaction
// and which handler should be called.
func (bot *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {

	if i.GuildID == "" || i.Interaction.AppID != s.State.User.ID {
		// NOTE: only listen for interactions in guilds.
		// Interaction's appID should be equal to the bot' user id, so we
		// respond only to application commands authored by the bot and
		// interactions on messages authored by the bot
		return
	}
	t := time.Now()
	bot.WithField(
		"GuildID", i.GuildID,
	).Tracef("Interaction created (%s)", i.ID)

	defer func() {
		bot.WithField(
			"Latency", time.Since(t),
		).Tracef("Interaction handled (%s)", i.ID)

	}()

	if i.Type == discordgo.InteractionApplicationCommand {
		// NOTE: an application command has been used,
		// determine which one

		switch i.ApplicationCommandData().Name {
		case bot.applicationCommandsConfig.Music.Name:
			// music slash command has been used
			if !bot.checkVoice(s, i) {
				// should check voice connection when starting
				// a music queue
				return
			}
			bot.onMusicSlashCommand(s, i)
		case bot.applicationCommandsConfig.Help.Name:
			// help slash command has been used
			bot.onHelpSlashCommand(s, i)
		}
	} else if i.Type == discordgo.InteractionModalSubmit {
		// NOTE: a user has submited a modal in the discord server
		// determine which modal has been submitted
		// NOTE: no need to check voice connection, as
		// it has already been checked in order to reach the modal
		switch bot.getModalName(i.Interaction.ModalSubmitData()) {
		case bot.applicationCommandsConfig.AddSongs.Name:
			// add songs modal has been submited
			bot.onAddSongsModalSubmit(s, i)
		}

	} else if i.Type == discordgo.InteractionMessageComponent {

		//NOTE: all message component id's authored by the bot start with the same prefix
		// that way we know bot is the author

		if !bot.checkVoice(s, i) {
			// NOTE: all message components require the user to
			// be in a voice channel and the bot to either not be
			// in any channel or be in the same channel as the user.
			return
		}

		switch i.Interaction.MessageComponentData().ComponentType {
		case discordgo.ButtonComponent:
			// a button has been clicked
			bot.onButtonClick(s, i)
		}
	}
}

// checkVoice checks if the interaction user is in a voice channel and if the bot
// is either not in any channel or in the same channel as the user. If this is false,
// the bot responds to the interaction and warns the user, else the bot does not
// respond and true is returned.
func (bot *Bot) checkVoice(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	botState, _ := s.State.VoiceState(i.GuildID, s.State.User.ID)
	userState, _ := s.State.VoiceState(i.GuildID, i.Member.User.ID)
	// user should always be in a voice channel
	if userState == nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You need to be in a voice channel!",
				Flags:   1 << 6, // Ephemeral
			},
		})
		return false
	}
	if botState != nil && botState.ChannelID != userState.ChannelID {
		// if the bot is in a voice channel, the user should be in the same channel
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "We need to be in the same voice channel!",
				Flags:   1 << 6, // Ephemeral
			},
		})
		return false
	}
	return true
}
