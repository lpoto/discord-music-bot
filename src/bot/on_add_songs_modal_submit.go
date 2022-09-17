package bot

import (
	"discord-music-bot/client/youtube"
	"fmt"
	"log"
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
	if len(queries) > youtube.MaxSongQueries {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf(
					"Cannot query more than %d songs at once",
					youtube.MaxSongQueries,
				),
				Flags: 1 << 6, // Ephemeral
			},
		})
		return
	}

	// Deffer the interaction, as it may be outdated before
	// the songs are found, and we never reply to this action
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredMessageUpdate,
	})

	songs := bot.youtubeClient.SearchSongs(queries)
	if len(songs) == 0 {
		return
	}

	// a positive number of songs has been found, save them to the queue
	// and update it
	for _, s := range songs {
		log.Println(s)
	}
}
