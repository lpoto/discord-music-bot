package bot

import "github.com/bwmarrin/discordgo"

// onMusicSlashCommand is a handler function called when the bot's music slash
// command is called in the discord channel, this is not emmited through the
// discord's websocket, but is rather called from INTERACTION_CREATE event when
// the interaction's command data name matches the music slash command's name.
func (bot *Bot) onMusicSlashCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	bot.WithField("GuildID", i.GuildID).Trace("Music slash command")
}
