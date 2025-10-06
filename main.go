package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/PassZ/rss-aggregator/internal/cli"
	"github.com/PassZ/rss-aggregator/internal/config"
	"github.com/PassZ/rss-aggregator/internal/database"
)

func main() {
	// Read the config file
	cfg, err := config.Read()
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Open database connection
	db, err := sql.Open("postgres", cfg.DbURL)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Create database queries instance
	dbQueries := database.New(db)

	// Create state with database and config
	state := &cli.State{
		DB:     dbQueries,
		Config: cfg,
	}

	// Create commands instance and register handlers
	commands := cli.NewCommands()
	commands.Register("login", cli.HandlerLogin)
	commands.Register("register", cli.HandlerRegister)
	commands.Register("reset", cli.HandlerReset)
	commands.Register("users", cli.HandlerUsers)
	commands.Register("agg", cli.HandlerAgg)
	commands.Register("addfeed", cli.MiddlewareLoggedIn(cli.HandlerAddFeed))
	commands.Register("feeds", cli.HandlerFeeds)
	commands.Register("follow", cli.MiddlewareLoggedIn(cli.HandlerFollow))
	commands.Register("following", cli.MiddlewareLoggedIn(cli.HandlerFollowing))
	commands.Register("unfollow", cli.MiddlewareLoggedIn(cli.HandlerUnfollow))
	commands.Register("browse", cli.MiddlewareLoggedIn(cli.HandlerBrowse))

	// Check if enough arguments were provided
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Error: not enough arguments provided\n")
		os.Exit(1)
	}

	// Parse command line arguments
	cmdName := os.Args[1]
	cmdArgs := os.Args[2:]

	// Create command instance
	cmd := cli.Command{
		Name: cmdName,
		Args: cmdArgs,
	}

	// Run the command
	if err := commands.Run(state, cmd); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}