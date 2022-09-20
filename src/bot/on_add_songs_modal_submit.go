package bot

import (
	"discord-music-bot/model"
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

// onAddSongsModalSubmit is a handler function called when a user
// submits the add songs modal in a discord servier. This
// is called when the type of interaction is determined to be
// add songs modal submit, in the onInteractionCreate function.
func (bot *Bot) onAddSongsModalSubmit(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Add songs modal submit")

	actionsRow := (i.ModalSubmitData().Components[0]).(*discordgo.ActionsRow)
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
	if len(queries) > bot.youtubeClient.Config.MaxParallelQueries {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf(
					"Cannot query more than %d songs at once",
					bot.youtubeClient.Config.MaxParallelQueries,
				),
				Flags: 1 << 6, // Ephemeral
			},
		})
		return
	}

	bot.interactionToQueueUpdateBuffer(s, i.Interaction)

	songInfos := bot.youtubeClient.SearchSongs(queries)
	if len(songInfos) == 0 {
		return
	}

	// a positive number of songs has been found, save them to the queue
	// and update it
	songs := make([]*model.Song, len(songInfos))
	for i, info := range songInfos {
		songs[i] = bot.builder.NewSong(info)
	}

	if err := bot.datastore.PersistSongs(
		s.State.User.ID,
		i.GuildID,
		songs,
	); err != nil {
		bot.Errorf("Error when submitting add songs modal: %v", err)
		return
	}

	bot.updateQueue(s, i.GuildID)
}
