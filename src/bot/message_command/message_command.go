package message_command

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/bwmarrin/discordgo"
)

type MessageCommandsConfig struct {
	Resend    string `yaml:"Resend" validate:"required"`
	Stop      string `yaml:"Stop" validate:"required"`
	Shuffle   string `yaml:"Shuffle" validate:"required"`
	Jump      string `yaml:"Jump" validate:"required"`
	EditSongs string `yaml:"EditSongs" validate:"required"`
}

// Register deletes all of the bot's previously registered global message commands,
// then registers the new commands from the provided config.
func Register(session *discordgo.Session, config *MessageCommandsConfig) error {
	v := reflect.ValueOf(*config)

	commands := make([]string, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		if i >= 5 {
			break
		}
		commands[i] = v.Field(i).Interface().(string)
	}
	// NOTE: empty guildID string for global commands
	guildID := ""

	// fetch all global application commands defined by
	// the bot user
	registeredCommands, err := session.ApplicationCommands(
		session.State.User.ID,
		guildID,
	)
	if err != nil {
		e := errors.New(
			fmt.Sprintf(
				"Could not fetch global message commands: %v",
				err,
			),
		)
		return e
	}
	toDelete := make([]*discordgo.ApplicationCommand, 0)
	toAdd := make([]*discordgo.ApplicationCommand, 0)

	for _, v := range registeredCommands {
		if v.Type != discordgo.MessageApplicationCommand {
			continue
		}
		del := true
		for _, name := range commands {
			if v.Name == name {
				del = false
				break
			}
		}
		if del {
			toDelete = append(toDelete, v)
		}
	}
	for _, name := range commands {
		add := true
		for _, v2 := range registeredCommands {
			if name == v2.Name {
				add = false
				break
			}
		}
		if add {
			toAdd = append(toAdd, &discordgo.ApplicationCommand{
				Name: name,
				Type: discordgo.MessageApplicationCommand,
			})
		}
	}

	// delete the fetched global application message commands
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
	// register the global application message commands
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
