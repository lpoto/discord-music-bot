package permissions

import "github.com/bwmarrin/discordgo"

type PermissionsChecker struct {
	session *discordgo.Session
}

// NewPermissionsChecker constructs a new object that
// handles checking permissions for users and the client
func NewPermissionsChecker(s *discordgo.Session) *PermissionsChecker {
	return &PermissionsChecker{
		session: s,
	}
}
