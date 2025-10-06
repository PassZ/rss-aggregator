package cli

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/PassZ/rss-aggregator/internal/database"
	"github.com/PassZ/rss-aggregator/internal/rss"
)

// MiddlewareLoggedIn is a higher-order function that wraps handlers requiring authentication
func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		// Get the current user from the database
		currentUser := s.Config.CurrentUserName
		user, err := s.DB.GetUser(context.Background(), currentUser)
		if err != nil {
			if err == sql.ErrNoRows {
				return fmt.Errorf("current user '%s' not found in database", currentUser)
			}
			return fmt.Errorf("failed to get current user: %w", err)
		}

		// Call the wrapped handler with the user
		return handler(s, cmd, user)
	}
}

// HandlerLogin handles the login command
func HandlerLogin(s *State, cmd Command) error {
	// Check if the command has the required argument
	if len(cmd.Args) == 0 {
		return fmt.Errorf("username is required")
	}

	// Get the username from the first argument
	username := cmd.Args[0]

	// Check if user exists in database
	_, err := s.DB.GetUser(context.Background(), username)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("user '%s' does not exist", username)
		}
		return fmt.Errorf("failed to check user: %w", err)
	}

	// Set the user in the config
	if err := s.Config.SetUser(username); err != nil {
		return fmt.Errorf("failed to set user: %w", err)
	}

	// Print success message
	fmt.Printf("User set to: %s\n", username)
	return nil
}

// HandlerRegister handles the register command
func HandlerRegister(s *State, cmd Command) error {
	// Check if the command has the required argument
	if len(cmd.Args) == 0 {
		return fmt.Errorf("username is required")
	}

	// Get the username from the first argument
	username := cmd.Args[0]

	// Check if user already exists
	_, err := s.DB.GetUser(context.Background(), username)
	if err == nil {
		return fmt.Errorf("user '%s' already exists", username)
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("failed to check user: %w", err)
	}

	// Create new user
	now := time.Now()
	user, err := s.DB.CreateUser(context.Background(), database.CreateUserParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      username,
	})
	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	// Set the user in the config
	if err := s.Config.SetUser(username); err != nil {
		return fmt.Errorf("failed to set user: %w", err)
	}

	// Print success message and user data
	fmt.Printf("User created: %s\n", username)
	fmt.Printf("User data: ID=%s, CreatedAt=%s, UpdatedAt=%s, Name=%s\n",
		user.ID, user.CreatedAt, user.UpdatedAt, user.Name)
	return nil
}

// HandlerReset handles the reset command
func HandlerReset(s *State, cmd Command) error {
	// Delete all users from the database
	err := s.DB.DeleteAllUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to reset database: %w", err)
	}

	// Print success message
	fmt.Println("Database reset successfully - all users deleted")
	return nil
}

// HandlerUsers handles the users command
func HandlerUsers(s *State, cmd Command) error {
	// Get all users from the database
	users, err := s.DB.GetUsers(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get users: %w", err)
	}

	// Get the current user from config
	currentUser := s.Config.CurrentUserName

	// Print all users with current user marked
	for _, user := range users {
		if user.Name == currentUser {
			fmt.Printf("* %s (current)\n", user.Name)
		} else {
			fmt.Printf("* %s\n", user.Name)
		}
	}

	return nil
}

// HandlerAgg handles the agg command
func HandlerAgg(s *State, cmd Command) error {
	// Check if the command has the required argument
	if len(cmd.Args) == 0 {
		return fmt.Errorf("time_between_reqs is required (e.g., 1s, 1m, 1h)")
	}

	// Parse the duration
	timeBetweenRequests, err := time.ParseDuration(cmd.Args[0])
	if err != nil {
		return fmt.Errorf("invalid duration format: %w", err)
	}

	fmt.Printf("Collecting feeds every %v\n", timeBetweenRequests)
	fmt.Println("Press Ctrl+C to stop...")

	// Create ticker
	ticker := time.NewTicker(timeBetweenRequests)
	defer ticker.Stop()

	// Run immediately, then on every tick
	for {
		scrapeFeeds(s)
		<-ticker.C
	}
}

// scrapeFeeds fetches the next feed and processes its posts
func scrapeFeeds(s *State) {
	// Get the next feed to fetch
	feed, err := s.DB.GetNextFeedToFetch(context.Background())
	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("No feeds to fetch")
			return
		}
		fmt.Printf("Error getting next feed: %v\n", err)
		return
	}

	// Mark feed as fetched
	err = s.DB.MarkFeedFetched(context.Background(), feed.ID)
	if err != nil {
		fmt.Printf("Error marking feed as fetched: %v\n", err)
		return
	}

	fmt.Printf("Fetching feed: %s (%s)\n", feed.Name, feed.Url)

	// Fetch the RSS feed
	rssFeed, err := rss.FetchFeed(context.Background(), feed.Url)
	if err != nil {
		fmt.Printf("Error fetching RSS feed %s: %v\n", feed.Url, err)
		return
	}

	// Process each item in the feed
	for _, item := range rssFeed.Channel.Item {
		err := processPost(s, item, feed.ID)
		if err != nil {
			// Log error but continue processing other posts
			fmt.Printf("Error processing post '%s': %v\n", item.Title, err)
		}
	}

	fmt.Printf("Processed %d posts from %s\n", len(rssFeed.Channel.Item), feed.Name)
}

// processPost saves a single post to the database
func processPost(s *State, item rss.RSSItem, feedID uuid.UUID) error {
	// Parse published date
	var publishedAt sql.NullTime
	if item.PubDate != "" {
		// Try different date formats
		formats := []string{
			time.RFC1123Z,     // Mon, 02 Jan 2006 15:04:05 -0700
			time.RFC1123,      // Mon, 02 Jan 2006 15:04:05 MST
			time.RFC3339,      // 2006-01-02T15:04:05Z07:00
			"Mon, 2 Jan 2006 15:04:05 MST", // Alternative format
		}

		for _, format := range formats {
			if parsed, err := time.Parse(format, item.PubDate); err == nil {
				publishedAt = sql.NullTime{Time: parsed, Valid: true}
				break
			}
		}
	}

	// Create post
	now := time.Now()
	_, err := s.DB.CreatePost(context.Background(), database.CreatePostParams{
		ID:          uuid.New(),
		CreatedAt:   now,
		UpdatedAt:   now,
		Title:       item.Title,
		Url:         item.Link,
		Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
		PublishedAt: publishedAt,
		FeedID:      feedID,
	})

	if err != nil {
		// Check if it's a duplicate URL error
		if strings.Contains(err.Error(), "duplicate key value violates unique constraint") {
			// Ignore duplicate posts
			return nil
		}
		return err
	}

	fmt.Printf("  Saved: %s\n", item.Title)
	return nil
}

// HandlerAddFeed handles the addfeed command
func HandlerAddFeed(s *State, cmd Command, user database.User) error {
	// Check if the command has the required arguments
	if len(cmd.Args) < 2 {
		return fmt.Errorf("feed name and URL are required")
	}

	// Get the feed name and URL from arguments
	feedName := cmd.Args[0]
	feedURL := cmd.Args[1]

	// Create new feed
	now := time.Now()
	feed, err := s.DB.CreateFeed(context.Background(), database.CreateFeedParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		Name:      feedName,
		Url:       feedURL,
		UserID:    user.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed: %w", err)
	}

	// Automatically create a feed follow record for the current user
	follow, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %w", err)
	}

	// Print the new feed record
	fmt.Printf("Feed created successfully:\n")
	fmt.Printf("  ID: %s\n", feed.ID)
	fmt.Printf("  Name: %s\n", feed.Name)
	fmt.Printf("  URL: %s\n", feed.Url)
	fmt.Printf("  User ID: %s\n", feed.UserID)
	fmt.Printf("  Created At: %s\n", feed.CreatedAt)
	fmt.Printf("  Updated At: %s\n", feed.UpdatedAt)
	fmt.Printf("\nYou are now following %s\n", follow.FeedName)

	return nil
}

// HandlerFeeds handles the feeds command
func HandlerFeeds(s *State, cmd Command) error {
	// Get all feeds with user names from the database
	feeds, err := s.DB.GetFeeds(context.Background())
	if err != nil {
		return fmt.Errorf("failed to get feeds: %w", err)
	}

	// Check if there are any feeds
	if len(feeds) == 0 {
		fmt.Println("No feeds found.")
		return nil
	}

	// Print all feeds
	fmt.Printf("Found %d feed(s):\n\n", len(feeds))
	for i, feed := range feeds {
		fmt.Printf("%d. %s\n", i+1, feed.Name)
		fmt.Printf("   URL: %s\n", feed.Url)
		fmt.Printf("   Created by: %s\n", feed.UserName)
		fmt.Printf("   Created at: %s\n", feed.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

// HandlerFollow handles the follow command
func HandlerFollow(s *State, cmd Command, user database.User) error {
	// Check if the command has the required argument
	if len(cmd.Args) == 0 {
		return fmt.Errorf("feed URL is required")
	}

	// Get the feed URL from arguments
	feedURL := cmd.Args[0]

	// Look up the feed by URL
	feed, err := s.DB.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("feed with URL '%s' not found", feedURL)
		}
		return fmt.Errorf("failed to get feed: %w", err)
	}

	// Create feed follow record
	now := time.Now()
	follow, err := s.DB.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{
		ID:        uuid.New(),
		CreatedAt: now,
		UpdatedAt: now,
		UserID:    user.ID,
		FeedID:    feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to create feed follow: %w", err)
	}

	// Print success message
	fmt.Printf("You are now following %s (created by %s)\n", follow.FeedName, follow.UserName)
	return nil
}

// HandlerFollowing handles the following command
func HandlerFollowing(s *State, cmd Command, user database.User) error {
	// Get all feed follows for the current user
	follows, err := s.DB.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return fmt.Errorf("failed to get feed follows: %w", err)
	}

	// Check if user is following any feeds
	if len(follows) == 0 {
		fmt.Println("You are not following any feeds.")
		return nil
	}

	// Print all followed feeds
	fmt.Printf("You are following %d feed(s):\n\n", len(follows))
	for i, follow := range follows {
		fmt.Printf("%d. %s\n", i+1, follow.FeedName)
		fmt.Printf("   Followed at: %s\n", follow.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Println()
	}

	return nil
}

// HandlerUnfollow handles the unfollow command
func HandlerUnfollow(s *State, cmd Command, user database.User) error {
	// Check if the command has the required argument
	if len(cmd.Args) == 0 {
		return fmt.Errorf("feed URL is required")
	}

	// Get the feed URL from arguments
	feedURL := cmd.Args[0]

	// Look up the feed by URL
	feed, err := s.DB.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("feed with URL '%s' not found", feedURL)
		}
		return fmt.Errorf("failed to get feed: %w", err)
	}

	// Delete the feed follow record
	err = s.DB.DeleteFeedFollow(context.Background(), database.DeleteFeedFollowParams{
		UserID: user.ID,
		FeedID: feed.ID,
	})
	if err != nil {
		return fmt.Errorf("failed to unfollow feed: %w", err)
	}

	// Print success message
	fmt.Printf("You have unfollowed %s\n", feed.Name)
	return nil
}

// HandlerBrowse handles the browse command
func HandlerBrowse(s *State, cmd Command, user database.User) error {
	// Default limit is 2
	limit := int32(2)
	
	// Check if limit argument is provided
	if len(cmd.Args) > 0 {
		// Parse limit argument
		if parsedLimit, err := fmt.Sscanf(cmd.Args[0], "%d", &limit); err != nil || parsedLimit != 1 {
			return fmt.Errorf("invalid limit format: %s", cmd.Args[0])
		}
	}

	// Get posts for the user
	posts, err := s.DB.GetPostsForUser(context.Background(), database.GetPostsForUserParams{
		UserID: user.ID,
		Limit:  limit,
	})
	if err != nil {
		return fmt.Errorf("failed to get posts: %w", err)
	}

	// Check if there are any posts
	if len(posts) == 0 {
		fmt.Println("No posts found. Make sure you're following some feeds and the aggregator is running.")
		return nil
	}

	// Print posts
	fmt.Printf("Found %d post(s):\n\n", len(posts))
	for i, post := range posts {
		fmt.Printf("%d. %s\n", i+1, post.Title)
		fmt.Printf("   Feed: %s\n", post.FeedName)
		fmt.Printf("   URL: %s\n", post.Url)
		if post.Description.Valid && post.Description.String != "" {
			// Truncate description if too long
			desc := post.Description.String
			if len(desc) > 200 {
				desc = desc[:200] + "..."
			}
			fmt.Printf("   Description: %s\n", desc)
		}
		if post.PublishedAt.Valid {
			fmt.Printf("   Published: %s\n", post.PublishedAt.Time.Format("2006-01-02 15:04:05"))
		}
		fmt.Println()
	}

	return nil
}