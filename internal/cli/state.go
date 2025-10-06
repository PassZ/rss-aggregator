package cli

import (
	"github.com/PassZ/rss-aggregator/internal/config"
	"github.com/PassZ/rss-aggregator/internal/database"
)

// State holds the application state
type State struct {
	DB     *database.Queries
	Config *config.Config
}

// Command represents a CLI command with its name and arguments
type Command struct {
	Name string
	Args []string
}