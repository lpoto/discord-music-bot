package blocked_command

import "sync"

type BlockedCommands struct {
	commands map[string]map[string]struct{}
	mutex    sync.Mutex
}

// NewBlockedCommands constructs a new object that holds
// information about the currently blocked bot commands
func NewBlockedCommands() *BlockedCommands {
	return &BlockedCommands{
		commands: make(map[string]map[string]struct{}),
		mutex:    sync.Mutex{},
	}
}

// IsBlocked returns true if the command with the provided
// name is blocked in the guild identified by the provided guildID,
// false otherwise.
func (bc *BlockedCommands) IsBlocked(guildID string, name string) bool {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	_, ok := bc.commands[guildID][name]
	return ok
}

// Block blocks the command with the provided name in the guild
// identified with the provided guildID
func (bc *BlockedCommands) Block(guildID string, name string) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if _, ok := bc.commands[guildID]; !ok {
		bc.commands[guildID] = make(map[string]struct{})
	}
	bc.commands[guildID][name] = struct{}{}
}

// Unblock unblocks the command with the provided name in the guild
// identified with the provided guildID, if it is blocked.
func (bc *BlockedCommands) Unblock(guildID string, name string) {
	bc.mutex.Lock()
	defer bc.mutex.Unlock()

	if _, ok := bc.commands[guildID]; ok {
		delete(bc.commands[guildID], name)
	}
}
