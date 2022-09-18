package bot

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type ChatCommandConfig struct {
	Name        string `yaml:"Name" validate:"required"`
	Description string `yaml:"Description" validate:"required"`
}

type MessageCommandConfig struct {
	Name string `yaml:"Name" validate:"required"`
}

type ApplicationCommandsConfig struct {
	Music    *ChatCommandConfig    `yaml:"Music" validate:"required"`
	Help     *ChatCommandConfig    `yaml:"Help" validate:"required"`
	AddSongs *MessageCommandConfig `yaml:"AddSongs" validate:"required"`
}

// setSlashCommands deletes all of the bot's previously
// registers slash commands, then registers the new
// music and help slash commands
func (bot *Bot) setSlashCommands(session *discordgo.Session) error {
	bot.Debug("Registering global application commands ...")
	// NOTE: guildID  is an empty string, so the commands are
	// global
	guildID := ""

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        bot.applicationCommandsConfig.Music.Name,
			Description: bot.applicationCommandsConfig.Music.Description,
		},
		{
			Name:        bot.applicationCommandsConfig.Help.Name,
			Description: bot.applicationCommandsConfig.Help.Description,
		},
	}
	// fetch all global application commands defined by
	// the bot user
	registeredCommands, err := session.ApplicationCommands(
		session.State.User.ID,
		guildID,
	)
	if err != nil {
		e := errors.New(
			fmt.Sprintf(
				"Could not fetch global application commands: %v",
				err,
			),
		)
		return e
	}
	toDelete := make([]*discordgo.ApplicationCommand, 0)
	toAdd := make([]*discordgo.ApplicationCommand, 0)

	for _, v := range registeredCommands {
		del := true
		for _, v2 := range commands {
			if v.Name == v2.Name && v.Description == v2.Description {
				del = false
				break
			}
		}
		if del {
			toDelete = append(toDelete, v)
		}
	}
	for _, v := range commands {
		add := true
		for _, v2 := range registeredCommands {
			if v.Name == v2.Name && v.Description == v2.Description {
				add = false
				break
			}
		}
		if add {
			toAdd = append(toAdd, v)
		}
	}
	// delete the fetched global application commands
	for _, v := range toDelete {
		bot.WithField("Name", v.Name).Trace(
			"Deleting global application command",
		)
		if err := session.ApplicationCommandDelete(
			session.State.User.ID,
			guildID,
			v.ID,
		); err != nil {
			e := errors.New(
				fmt.Sprintf(
					"Could not delete global application command '%v': %v",
					v.Name,
					err,
				),
			)
			return e
		}
	}
	// register the global application commands
	for _, cmd := range toAdd {
		bot.WithField("Name", cmd.Name).Trace(
			"Registering global application command",
		)
		if _, err := session.ApplicationCommandCreate(
			session.State.User.ID,
			guildID,
			cmd,
		); err != nil {
			e := errors.New(
				fmt.Sprintf(
					"Could not create global application command '%v': %v",
					cmd.Name,
					err,
				),
			)
			return e
		}
	}
	bot.Debug("Successfully registered global application commands")
	return nil
}
