package slash_command

import (
	"errors"
	"fmt"

	"github.com/bwmarrin/discordgo"
)

type ChatCommandConfig struct {
	Name        string `yaml:"Name" validate:"required"`
	Description string `yaml:"Description" validate:"required"`
}

type SlashCommandsConfig struct {
	Music *ChatCommandConfig `yaml:"Music" validate:"required"`
	Help  *ChatCommandConfig `yaml:"Help" validate:"required"`
}

// Register deletes all of the bot's previously
// registered gloabal slash commands, then registers the new
// music and help global slash commands.
func Register(session *discordgo.Session, config *SlashCommandsConfig) error {
	// NOTE: guildID  is an empty string, so the commands are
	// global
	guildID := ""

	commands := []*discordgo.ApplicationCommand{
		{
			Name:        config.Music.Name,
			Description: config.Music.Description,
		},
		{
			Name:        config.Help.Name,
			Description: config.Help.Description,
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
	return nil
}
