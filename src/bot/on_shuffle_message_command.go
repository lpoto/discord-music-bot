package bot

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// onShuffleMessageCommand is a handler function called when the name of interaction's
// application command data matches the registered Shuffle global message command.
func (bot *Bot) onShuffleMessageCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if bot.blockedCommands.IsBlocked(i.GuildID, "SHUFFLE") {
		return
	}
	bot.blockedCommands.Block(i.GuildID, "SHUFFLE")
	defer bot.blockedCommands.Unblock(i.GuildID, "SHUFFLE")

	songs, err := bot.datastore.GetAllSongsForQueue(s.State.User.ID, i.GuildID)
	if err != nil {
		bot.Errorf("Error when fetching all songs for queue: %v", err)
		return
	}
	t := time.Now()
	songs = bot.service.ShuffleSongs(songs)
	if err := bot.datastore.UpdateSongs(songs); err != nil {
		bot.Errorf("Error when updating songs for queue: %v", err)
		return
	}
	// NOTE: do not allow spamming shuffle
	d := 500 - time.Since(t)
	if d > 0 {
		time.Sleep(d)
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
	})
	s.InteractionResponseDelete(i.Interaction)
	bot.queueUpdater.NeedsUpdate(i.GuildID)
	bot.queueUpdater.Update(s, i.GuildID)
}
