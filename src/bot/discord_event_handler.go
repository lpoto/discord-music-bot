package bot

import "github.com/bwmarrin/discordgo"

type DiscordEventHandler struct {
	*Bot
}

// setHandlers adds handlers for discord events to the
// provided session.
// It Adds handler for ready, message (bulk) delete,
// and interaction create events, but it determines
// the type of interaction and calls the appropriate function.
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

				bot.session = s
				util := &Util{bot.Bot}

				if len(i.ChannelID) > 0 {
					if !util.ensureClientTextChannelPermissions(i.ChannelID) {
						return
					}
				}

				switch i.Interaction.Type {
				case discordgo.InteractionApplicationCommand:
					t := bot.transactions.New(
						"Interaction/ApplicationCommand",
						i.GuildID,
						i.Interaction,
					)
					bot.onApplicationCommand(t)
					return
				case discordgo.InteractionMessageComponent:
					switch i.Interaction.MessageComponentData().ComponentType {
					case discordgo.ButtonComponent:
						t := bot.transactions.New(
							"Interaction/ButtonClick",
							i.GuildID,
							i.Interaction,
						)
						bot.onButtonClick(t)
						return
					}
					return
				case discordgo.InteractionModalSubmit:
					t := bot.transactions.New(
						"Interaction/ModalSubmit",
						i.GuildID,
						i.Interaction,
					)
					bot.onModalSubmit(t)
					return

				}
			}
		},
	)
}
