package bot

import (
	"discord-music-bot/bot/modal"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// onInteractionCreate is a handler function called when discord emits
// INTERACTION_CREATE event. It determines the type of interaction
// and which handler should be called.
func (bot *Bot) onInteractionCreate(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !bot.ready || i.GuildID == "" || i.Interaction.AppID != s.State.User.ID {
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

	// NOTE: check permissions for the client in the channel
	// ... it should always have the send messages permission
	per, err := s.State.UserChannelPermissions(s.State.User.ID, i.ChannelID)
	if err != nil {
		bot.Trace("Client missing all permissions in text channel")
		return
	}
	if per&discordgo.PermissionSendMessages != discordgo.PermissionSendMessages {
		bot.Trace("Client missing send messages permission")
		return
	}

	channelID := ""
	if userState, _ := s.State.VoiceState(
		i.GuildID, i.Member.User.ID,
	); userState != nil {
		channelID = userState.ChannelID
	}

	defer func() {
		bot.WithField(
			"Latency", time.Since(t),
		).Tracef("Interaction handled (%s)", i.ID)

	}()

	if i.Type == discordgo.InteractionApplicationCommand {
		// NOTE: an application command has been used,
		// determine which one

		name := strings.TrimSpace(i.ApplicationCommandData().Name)
		switch name {
		case strings.TrimSpace(bot.config.SlashCommands.Music.Name):
			// music slash command has been used
			if !bot.checkVoice(s, i) {
				// should check voice connection when starting
				// a music queue
				return
			}
			bot.onMusicSlashCommand(s, i)
			bot.play(s, i.GuildID, channelID)
		case strings.TrimSpace(bot.config.SlashCommands.Help.Name):
			// help slash command has been used
			bot.onHelpSlashCommand(s, i)
		case strings.TrimSpace(bot.config.MessageCommands.Resend):
			// Resend message command has been used
			bot.onResendMessageCommand(s, i)
		case strings.TrimSpace(bot.config.MessageCommands.Stop):
			// Stop message command has been used
			bot.onStopMessageCommand(s, i)
		case strings.TrimSpace(bot.config.MessageCommands.EditSongs):
			// EditSongs message command has been used
			bot.onEditSongsMessageCommand(s, i)
		}
	} else if i.Type == discordgo.InteractionModalSubmit {
		// NOTE: a user has submited a modal in the discord server
		// determine which modal has been submitted
		// NOTE: no need to check voice connection, as
		// it has already been checked in order to reach the modal
		name := strings.TrimSpace(modal.GetModalName(i.Interaction.ModalSubmitData()))
		switch name {
		// add songs modal has been submited
		case strings.TrimSpace(bot.config.Modals.AddSongs.Name):
			bot.onAddSongsModalSubmit(s, i)
			bot.play(s, i.GuildID, channelID)
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
			bot.play(s, i.GuildID, channelID)
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
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return false
	} else if userState.Deaf || userState.SelfDeaf {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "You need to undeafen!",
				Flags:   discordgo.MessageFlagsEphemeral,
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
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return false
	}
	return true
}
