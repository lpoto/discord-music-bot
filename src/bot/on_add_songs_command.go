package bot

import "github.com/bwmarrin/discordgo"

// onAddSongsComamnd is a handler function called when the bot's
// add songs command is called from queue message's context menu.
// This is called from INTERACTION_CREATE event when
// the interaction's command data name matches the add songs
// message command's name.
func (bot *Bot) onAddSongsComamnd(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Add songs message command")
	q, _ := bot.datastore.FindQueue(
		s.State.User.ID,
		i.GuildID,
	)
	m := bot.service.GetAddSongsModal(q)
	if err := s.InteractionRespond(
		i.Interaction,
		&discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseModal,
			Data: &discordgo.InteractionResponseData{
				Components: m.Components,
				CustomID:   m.CustomID,
				Title:      bot.applicationCommandsConfig.AddSongs.Name,
			},
		},
	); err != nil {
		bot.Errorf(
			"Error when responding with add songs modal: %v",
			err,
		)
	}
}
