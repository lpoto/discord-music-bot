package bot

import "github.com/bwmarrin/discordgo"

type DiscordIntentsHandler struct {
	*Bot
}

// setIntents sets the intents for the session, required
// by the music bot
func (bot *DiscordIntentsHandler) setIntents() {
	//NOTE: guilds for interactions in guilds,
	// guild messages for message delete events,
	// voice states for voice state update events
	bot.session.Identify.Intents =
		discordgo.IntentsGuilds +
			discordgo.IntentsGuildVoiceStates +
			discordgo.IntentGuildMessages

}
