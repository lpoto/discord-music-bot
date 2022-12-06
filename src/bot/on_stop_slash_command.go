package bot

import (
	"discord-music-bot/bot/transaction"

	"github.com/bwmarrin/discordgo"
)

// onStopSlashCommand is a handler function called when the bot's stop slash
// command is called in the discord channel, this is not emmited through the
// discord's websocket, but is rather called from INTERACTION_CREATE event when
// the interaction's command data name matches the stop slash command's name.
func (bot *DiscordEventHandler) onStopSlashCommand(t *transaction.Transaction) {
	defer t.Defer()

	queue, err := bot.datastore.Queue().GetQueue(
		bot.session.State.User.ID,
		t.GuildID(),
	)
	if err != nil {
		if err := bot.session.InteractionRespond(t.Interaction(),
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "There is no active music queue!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}); err != nil {
			bot.log.WithField("GuildID", t.GuildID()).Errorf(
				"Error when responding to help command: %v",
				err,
			)
		}
		return
	}
	if err := bot.session.ChannelMessageDelete(queue.ChannelID, queue.MessageID); err != nil {
		if err := bot.session.InteractionRespond(t.Interaction(),
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "Something went wrong!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			}); err != nil {
			bot.log.WithField("GuildID", t.GuildID()).Errorf(
				"Error when responding to help command: %v",
				err,
			)
		}
	}
	util := &Util{bot.Bot}
	util.deleteQueue(t.GuildID(), []string{queue.MessageID})

	if err := bot.session.InteractionRespond(t.Interaction(),
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Music has been stopped!",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		}); err != nil {
		bot.log.WithField("GuildID", t.GuildID()).Errorf(
			"Error when responding to help command: %v",
			err,
		)
	}
}
