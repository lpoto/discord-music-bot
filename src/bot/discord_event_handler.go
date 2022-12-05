package bot

import "github.com/bwmarrin/discordgo"

type DiscordEventHandler struct {
	*Bot
}

// setHandlers adds handlers for discord events to the
// provided session
func (bot *DiscordEventHandler) setHandlers() {
	bot.session.AddHandler(
		func(s *discordgo.Session, r *discordgo.Ready) {
			bot.session = s
			bot.onReady(r)
		},
	)
	bot.session.AddHandler(
		func(s *discordgo.Session, m *discordgo.MessageDelete) {
			if len(m.GuildID) > 0 && bot.ready &&
				(m.Author == nil || len(m.Author.ID) == 0 ||
					m.Author.ID == bot.session.State.User.ID) {

				bot.session = s
				bot.onMessageDelete(m)
			}
		},
	)
	bot.session.AddHandler(
		func(s *discordgo.Session, m *discordgo.MessageDeleteBulk) {
			if len(m.GuildID) > 0 && bot.ready {
				bot.session = s
				bot.onBulkMessageDelete(m)
			}
		},
	)
	bot.session.AddHandler(
		func(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {
			if len(v.GuildID) > 0 && bot.ready &&
				v.UserID == bot.session.State.User.ID {

				t := bot.transactions.New("VoiceStateUpdate", v.GuildID, nil)
				bot.session = s
				bot.onVoiceStateUpdate(t, v)
			}
		},
	)
	bot.session.AddHandler(
		func(s *discordgo.Session, i *discordgo.InteractionCreate) {
			if len(i.GuildID) > 0 && bot.ready &&
				i.Interaction.AppID == bot.session.State.User.ID &&
				(i.Interaction.Type == discordgo.InteractionApplicationCommand ||
					i.Interaction.Type == discordgo.InteractionModalSubmit ||
					i.Interaction.Type == discordgo.InteractionMessageComponent) {

				t := bot.transactions.New("Interaction", i.GuildID, i.Interaction)
				bot.session = s
				bot.onInteractionCreate(t)
			}
		},
	)
}
