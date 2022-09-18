package bot

import (
	"discord-music-bot/model"

	"github.com/bwmarrin/discordgo"
)

// onButtonClick is a handler function called when a user
// clicks a button on a message owned by the bot.
// This is not emitted through the discord websocket, but is rather
// called from the INTERACTION_CREATE event when the interaction type
// is button click and the message author is bot
func (bot *Bot) onButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	label := bot.builder.GetButtonLabelFromComponentData(i.MessageComponentData())
	bot.WithField("GuildID", i.GuildID).Tracef("Button clicked (%s)", label)

	switch label {
	case bot.builder.Config.Components.AddSongs:
		bot.onAddSongsCommand(s, i)
		return
	case bot.builder.Config.Components.Backward:
		bot.backwardButtonClick(s, i)
		return
	case bot.builder.Config.Components.Forward:
		bot.forwardButtonClick(s, i)
		return
	case bot.builder.Config.Components.Loop:
		bot.loopButtonClick(s, i)
		return
	case bot.builder.Config.Components.Pause:
		bot.pauseButtonClick(s, i)
		return
	default:
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "Sorry, something went wrong ...",
				Flags:   1 << 6, // Ephemeral
			},
		})
	}
}

// forwardButtonClick increments the queue's offset, updates it
// and then updates the queue message
func (bot *Bot) forwardButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	queue, _ := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	bot.service.IncrementQueueOffset(queue)
	if err := bot.datastore.UpdateQueue(queue); err != nil {
		bot.Errorf("Error on forward button click: %v", err)
		return
	}
	bot.onUpdateQueueFromInteraction(s, i)
}

// backwardButtonClick decrements the queue's offset, updates it
// and then updates the queue message
func (bot *Bot) backwardButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	queue, _ := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	bot.service.DecrementQueueOffset(queue)
	if err := bot.datastore.UpdateQueue(queue); err != nil {
		bot.Errorf("Error on backward button click: %v", err)
		return
	}
	bot.onUpdateQueueFromInteraction(s, i)
}

// pauseButtonClick adds or removes the queue's Paused option, updates it
// and then updates the queue message
func (bot *Bot) pauseButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	queue, _ := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	bot.service.AddOrRemoveQueueOption(queue, model.Paused)
	if err := bot.datastore.UpdateQueue(queue); err != nil {
		bot.Errorf("Error on pause button click: %v", err)
		return
	}
	bot.onUpdateQueueFromInteraction(s, i)
}

// loopButtonClick adds or removes the queue's Loop option, updates it
// and then updates the queue message
func (bot *Bot) loopButtonClick(s *discordgo.Session, i *discordgo.InteractionCreate) {
	queue, _ := bot.datastore.GetQueue(s.State.User.ID, i.GuildID)
	bot.service.AddOrRemoveQueueOption(queue, model.Loop)
	if err := bot.datastore.UpdateQueue(queue); err != nil {
		bot.Errorf("Error on loop button click: %v", err)
		return
	}
	bot.onUpdateQueueFromInteraction(s, i)
}
