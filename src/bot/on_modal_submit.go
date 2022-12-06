package bot

import (
	"discord-music-bot/bot/modal"
	"discord-music-bot/bot/transaction"
	"strings"
)

// onModalSubmit is a handler function called when discord emits
// INTERACTION_CREATE event and the interaction's type is modalSubmit.
func (bot *DiscordEventHandler) onModalSubmit(t *transaction.Transaction) {
	// NOTE: a user has submited a modal in the discord server
	// determine which modal has been submitted
	// NOTE: no need to check voice connection, as
	// it has already been checked in order to reach the modal
	channelID := ""
	if userState, _ := bot.session.State.VoiceState(
		t.GuildID(),
		t.Interaction().Member.User.ID,
	); userState != nil {
		channelID = userState.ChannelID
	}

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
}
