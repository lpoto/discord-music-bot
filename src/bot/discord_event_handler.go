package bot
import (
	"discord-music-bot/bot/modal"
	"strings"

	"github.com/bwmarrin/discordgo"
)

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

			// NOTE: handle on message delete event only
			// when it happens in a guild, the bot is ready
			// and the bot is author of the deleted message

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

			// NOTE: handle bulk message deletes only in guilds
			// this only contains a slice of messageID's so we
			// cannot check if bot authored them

			if len(m.GuildID) > 0 && bot.ready {
				bot.session = s
				bot.onBulkMessageDelete(m)
			}
		},
	)
	bot.session.AddHandler(
		func(s *discordgo.Session, v *discordgo.VoiceStateUpdate) {

			// NOTE: handle voice state update events only
			// in guilds and only updates for the client

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

			// NOTE: handle only interactions authored by the client
			// created in a guild

			if len(i.GuildID) > 0 && bot.ready &&
				i.Interaction.AppID == bot.session.State.User.ID {

				bot.session = s
				util := &Util{bot.Bot}

				if len(i.ChannelID) > 0 {
					if !util.ensureClientTextChannelPermissions(i.ChannelID) {
						return
					}
				}

				switch i.Interaction.Type {
				case discordgo.InteractionApplicationCommand:
					name := strings.TrimSpace(
						i.Interaction.ApplicationCommandData().Name,
					)
					t := bot.transactions.New(
						"Interaction/ApplicationCommand/"+name,
						i.GuildID,
						i.Interaction,
					)
					bot.onApplicationCommand(t)
					return
				case discordgo.InteractionMessageComponent:
					switch i.Interaction.MessageComponentData().ComponentType {
					case discordgo.ButtonComponent:
						label := bot.builder.Queue().GetButtonLabelFromComponentData(
							i.Interaction.MessageComponentData(),
						)
						t := bot.transactions.New(
							"Interaction/ButtonClick/"+label,
							i.GuildID,
							i.Interaction,
						)
						bot.onButtonClick(t)
						return
					}
					return
				case discordgo.InteractionModalSubmit:
					name := strings.TrimSpace(
						modal.GetModalName(
							i.Interaction.ModalSubmitData(),
						),
					)
					t := bot.transactions.New(
						"Interaction/ModalSubmit/"+name,
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
