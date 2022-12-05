package bot

import (
	"discord-music-bot/bot/transaction"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

// onMusicSlashCommand is a handler function called when the bot's music slash
// command is called in the discord channel, this is not emmited through the
// discord's websocket, but is rather called from INTERACTION_CREATE event when
// the interaction's command data name matches the music slash command's name.
func (bot *DiscordEventHandler) onMusicSlashCommand(t *transaction.Transaction) {
	bot.log.WithField("GuildID", t.GuildID()).Trace("Music slash command")
	defer t.Defer()

	// NOTE: only a single queue may be active in a guild at once
	if queue, err := bot.datastore.Queue().GetQueue(
		bot.session.State.User.ID,
		t.GuildID(),
	); err == nil {
		bot.session.InteractionRespond(t.Interaction(),
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(
						"A music queue is already active in this server: "+
							"https://discord.com/channels/%s/%s/%s",
						queue.GuildID,
						queue.ChannelID,
						queue.MessageID,
					),
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
		return
	}

	// Construct a new queue, send it to the channel
	// and persist it in the datastore
	queue := bot.builder.Queue().NewQueue(
		bot.session.State.User.ID,
		t.GuildID(),
		"", "",
	)
	embed := bot.builder.Queue().MapQueueToEmbed(queue)
	components := bot.builder.Queue().GetMusicQueueComponents(queue)

	err := bot.session.InteractionRespond(
		t.Interaction(),
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{embed},
				Components: components,
			},
		})
	if err != nil {
		bot.log.WithField("GuildID", t.GuildID()).Errorf(
			"Error when sending a new queue: %v",
			err,
		)
		return
	}
	msg, err := bot.session.InteractionResponse(t.Interaction())
	if err != nil {
		bot.log.Errorf(
			"Error when fetching interaction response message: %v",
			err,
		)
		return
	}
	queue.MessageID = msg.ID
	queue.ChannelID = msg.ChannelID
	if err := bot.datastore.Queue().PersistQueue(queue); err != nil {
		bot.log.Errorf("Error when persisting a new queue: %v", err)
		return
	}
}
