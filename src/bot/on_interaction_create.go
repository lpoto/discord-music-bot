package bot

import (
	"discord-music-bot/bot/modal"
	"discord-music-bot/bot/transaction"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

// onInteractionCreate is a handler function called when discord emits
// INTERACTION_CREATE event. It determines the type of interaction
// and which handler should be called.
func (bot *DiscordEventHandler) onInteractionCreate(t *transaction.Transaction) {
	cur_time := time.Now()
	bot.log.WithField(
		"GuildID", t.GuildID(),
	).Tracef("Interaction created (%s)", t.Interaction().ID)

	// NOTE: check permissions for the client in the channel
	// ... it should always have the send messages permission
	per, err := bot.session.State.UserChannelPermissions(
		bot.session.State.User.ID,
		t.Interaction().ChannelID,
	)
	if err != nil {
		bot.log.Trace("Client missing all permissions in text channel")
		return
	}
	if per&discordgo.PermissionSendMessages !=
		discordgo.PermissionSendMessages {
		bot.log.Trace("Client missing send messages permission")
		return
	}

	channelID := ""
	if userState, _ := bot.session.State.VoiceState(
		t.GuildID(),
		t.Interaction().Member.User.ID,
	); userState != nil {
		channelID = userState.ChannelID
	}

	defer func() {
		bot.log.WithField(
			"Latency", time.Since(cur_time),
		).Tracef(
			"Interaction handled (%s)",
			t.Interaction().ID,
		)

	}()

	util := &Util{bot.Bot}

	if t.Interaction().Type == discordgo.InteractionApplicationCommand {
		// NOTE: an application command has been used,
		// determine which one

		name := strings.TrimSpace(
			t.Interaction().ApplicationCommandData().Name,
		)
		switch name {
		case strings.TrimSpace(bot.config.SlashCommands.Music.Name):
			// music slash command has been used
			if !util.checkVoice(t) {
				// should check voice connection when starting
				// a music queue
				return
			}
			bot.onMusicSlashCommand(t)
			return
		case strings.TrimSpace(bot.config.SlashCommands.Stop.Name):
			bot.onStopSlashCommand(t)
			return
		case strings.TrimSpace(bot.config.SlashCommands.Help.Name):
			// help slash command has been used
			bot.onHelpSlashCommand(t)
			return
		}
	} else if t.Interaction().Type == discordgo.InteractionModalSubmit {
		// NOTE: a user has submited a modal in the discord server
		// determine which modal has been submitted
		// NOTE: no need to check voice connection, as
		// it has already been checked in order to reach the modal
		name := strings.TrimSpace(
			modal.GetModalName(
				t.Interaction().ModalSubmitData(),
			),
		)
		switch name {
		// add songs modal has been submited
		case strings.TrimSpace(bot.config.Modals.AddSongs.Name):
			bot.onAddSongsModalSubmit(t)
			bot.play(t, channelID)
			return
		}

	} else if t.Interaction().Type == discordgo.InteractionMessageComponent {

		//NOTE: all message component id's authored by the bot start with the same prefix
		// that way we know bot is the author

		if !util.checkVoice(t) {
			// NOTE: all message components require the user to
			// be in a voice channel and the bot to either not be
			// in any channel or be in the same channel as the user.
			return
		}

		switch t.Interaction().MessageComponentData().ComponentType {
		case discordgo.ButtonComponent:
			// a button has been clicked
			bot.onButtonClick(t)
			return
		}
	}
}
