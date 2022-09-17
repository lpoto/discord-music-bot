package bot

import (
	"log"

	"github.com/bwmarrin/discordgo"
)

// onMusicSlashCommand is a handler function called when the bot's music slash
// command is called in the discord channel, this is not emmited through the
// discord's websocket, but is rather called from INTERACTION_CREATE event when
// the interaction's command data name matches the music slash command's name.
func (bot *Bot) onMusicSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Music slash command")

	// NOTE: only a single queue may be active in a guild at once
	if _, err := bot.datastore.FindQueue(
		s.State.User.ID,
		i.GuildID,
	); err == nil {
		if err := s.InteractionRespond(
			i.Interaction,
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "A music queue already exists in this server!",
					Flags:   1 << 6, // this flag marks msg ephemeral
				},
			}); err != nil {
			bot.WithField("GuildID", i.GuildID).Errorf(
				"Error when responding to music command: %v",
				err,
			)
		}
		return
	} else {
		log.Println(err)
	}

	// Construct a new queue, send it to the channel
	// and persist it in the datastore
	queue := bot.service.NewQueue(
		s.State.User.ID,
		i.GuildID,
		"", "",
	)
	embed := bot.service.MapQueueToEmbed(
		queue,
		bot.slashCommandsConfig.Music.Description,
	)
	err := s.InteractionRespond(
		i.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds: []*discordgo.MessageEmbed{embed},
			},
		})
	if err != nil {
		bot.WithField("GuildID", i.GuildID).Errorf(
			"Error when sending a new queue: %v",
			err,
		)
		return
	}
	msg, err := s.InteractionResponse(i.Interaction)
	if err != nil {
		bot.Errorf(
			"Error when fetching interaction response message: %v",
			err,
		)
		return
	}
	queue.MessageID = msg.ID
	queue.ChannelID = msg.ChannelID
	if _, err := bot.datastore.PersistQueue(queue); err != nil {
		bot.Errorf("Error when persisting a new queue: %v", err)
	}
}
