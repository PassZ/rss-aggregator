package cli

import "fmt"

// Commands holds all the commands the CLI can handle
type Commands struct {
	handlers map[string]func(*State, Command) error
}

// NewCommands creates a new commands instance with an initialized map
func NewCommands() *Commands {
	return &Commands{
		handlers: make(map[string]func(*State, Command) error),
	}
}

// Run executes a given command with the provided state if it exists
func (c *Commands) Run(s *State, cmd Command) error {
	handler, exists := c.handlers[cmd.Name]
	if !exists {
		return fmt.Errorf("unknown command: %s", cmd.Name)
	}
	return handler(s, cmd)
}

// Register registers a new handler function for a command name
func (c *Commands) Register(name string, f func(*State, Command) error) {
	c.handlers[name] = f
}