package bot

import (
	"discord-music-bot/bot/transaction"

	"github.com/bwmarrin/discordgo"
)

type Util struct {
	*Bot
}

// checkVoice checks if the interaction user is in a voice channel and if the bot
// is either not in any channel or in the same channel as the user. If this is false,
// the bot responds to the interaction and warns the user, else the bot does not
// respond and true is returned.
func (bot *Util) checkVoice(t *transaction.Transaction) bool {
	botState, _ := bot.session.State.VoiceState(
		t.GuildID(),
		bot.session.State.User.ID,
	)
	userState, _ := bot.session.State.VoiceState(
		t.GuildID(),
		t.Interaction().Member.User.ID,
	)
	// user should always be in a voice channel
	if userState == nil {
		bot.session.InteractionRespond(t.Interaction(),
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You need to be in a voice channel!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		return false
	} else if userState.Deaf || userState.SelfDeaf {
		bot.session.InteractionRespond(t.Interaction(),
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "You need to undeafen!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		return false
	}
	if botState != nil && botState.ChannelID != userState.ChannelID {
		// if the bot is in a voice channel, the user should be in the same channel
		bot.session.InteractionRespond(t.Interaction(),
			&discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "We need to be in the same voice channel!",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
		return false
	}
	return true
}

// deleteQueue checks if any of the provided messageIDs belongs
// to a queue message. If so, it deletes it.
func (bot *Util) deleteQueue(s *discordgo.Session, guildID string, messageIDs []string) {
	clientID := s.State.User.ID

	queue, err := bot.datastore.Queue().GetQueue(
		clientID,
		guildID,
	)
	if err != nil {
		return
	}
	ok := false
	for _, v := range messageIDs {
		if queue.MessageID == v {
			ok = true
			break
		}
	}
	if !ok {
		bot.log.Trace("The queue message was not deleted")
		return
	}
	bot.log.Trace("The queue message was deleted, removing the queue")
	if ap, ok := bot.audioplayers.Get(guildID); ok {
		ap.Subscriptions().Emit("stop")
	}
	if vc, ok := s.VoiceConnections[guildID]; ok {
		vc.Disconnect()
	}

	if err := bot.datastore.Queue().RemoveQueue(
		clientID,
		queue.GuildID,
	); err != nil {
		bot.log.Errorf(
			"Error when removing queue after message delete: %v",
			err,
		)
	}
}

// cleanDiscordMusicQueues removes all queue messages from datastore,
// for which the messages not longer exist in the discord channels.
// For those that exist, it marks them as paused.
// NOTE: when updating a queue, Ii bot._start is false, queue is
// marked  offline (has only one disabled button).
// Otherwise if a queue is in a guild in which the bot is not in
// a voice channel, the queue is marked inactive (has only one Join button)
func (bot *Util) cleanDiscordMusicQueues() {
	bot.log.Debug("Cleaning up discord music queues ...")

	queues, err := bot.datastore.Queue().FindAllQueues()
	if err != nil {
		bot.log.Errorf(
			"Error when checking if all queues exist: %v", err,
		)
		return
	}
	for _, queue := range queues {
		t := bot.transactions.New("CleanQueues", queue.GuildID, nil)
		if err := t.UpdateQueue(0); err != nil {
			// NOTE: this will be called if updating the queue
			// failed... the queue was then deleted while the
			// bot had been offline
			err = bot.datastore.Queue().RemoveQueue(
				queue.ClientID,
				queue.GuildID,
			)
			if err != nil {
				bot.log.Errorf(
					"Error when cleaning up queues : %v", err,
				)
			}
		}
	}
}
