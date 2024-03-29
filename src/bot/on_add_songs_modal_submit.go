package bot

import (
	"discord-music-bot/bot/transaction"
	"discord-music-bot/model"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// onAddSongsModalSubmit is a handler function called when a user
// submits the add songs modal in a discord servier. This
// is called when the type of interaction is determined to be
// add songs modal submit, in the onInteractionCreate function.
func (bot *DiscordEventHandler) onAddSongsModalSubmit(t *transaction.Transaction) {
	actionsRow := (t.Interaction().ModalSubmitData().Components[0]).(*discordgo.ActionsRow)
	textInput := (actionsRow.Components[0]).(*discordgo.TextInput)

	songString := textInput.Value
	queries := make([]string, 0)
	for _, s := range strings.Split(songString, "\n") {
		s := strings.TrimSpace(s)
		if len(s) > 0 {
			queries = append(queries, s)
		}
	}

	// There is a limit for a number of songs that may be queried at once
	if len(queries) > 100 {
		bot.session.InteractionRespond(t.Interaction(),
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: fmt.Sprintf(
						"Cannot query more than %d songs at once",
						100,
					),
					Flags: discordgo.MessageFlagsEphemeral,
				},
			})
		return
	}

	songInfos := bot.youtube.Search().GetSongs(queries)
	if len(songInfos) == 0 {
		return
	}

	// a positive number of songs has been found, save them to the queue
	// and update it
	songs := make([]*model.Song, len(songInfos))
	for i, info := range songInfos {
		songs[i] = bot.builder.Song().NewSong(info)
	}

	if err := bot.datastore.Song().PersistSongs(
		bot.session.State.User.ID,
		t.GuildID(),
		songs...,
	); err != nil {
		bot.log.Errorf("Error when submitting add songs modal: %v", err)
		return
	}
	bot.play(t, t.Interaction().ChannelID)
}
