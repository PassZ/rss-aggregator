# Gator - RSS Aggregator CLI

A powerful command-line RSS aggregator built in Go that allows you to follow RSS feeds, aggregate posts, and browse content from your terminal.

## Features

- **User Management**: Register and login with user accounts
- **Feed Management**: Add, follow, and unfollow RSS feeds
- **Real-time Aggregation**: Continuously fetch and store posts from RSS feeds
- **Content Browsing**: View posts from feeds you follow
- **Database Persistence**: All data stored in PostgreSQL
- **Rate Limiting**: Respectful fetching to avoid overwhelming servers

## Prerequisites

Before running Gator, make sure you have the following installed:

- **Go 1.21+**: [Download and install Go](https://golang.org/dl/)
- **PostgreSQL**: [Download and install PostgreSQL](https://www.postgresql.org/download/)

## Installation

1. **Clone the repository**:
   ```bash
   git clone https://github.com/PassZ/rss-aggregator.git
   cd rss-aggregator
   ```

2. **Install the CLI**:
   ```bash
   go install
   ```

3. **Set up PostgreSQL**:
   - Create a database named `gator`
   - Note your connection details (host, port, username, password)

4. **Configure the application**:
   Create a config file at `~/.gatorconfig.json`:
   ```json
   {
     "db_url": "postgres://username:password@localhost:5432/gator?sslmode=disable"
   }
   ```

## Quick Start

1. **Register a user**:
   ```bash
   gator register myusername
   ```

2. **Add some RSS feeds**:
   ```bash
   gator addfeed "TechCrunch" "https://techcrunch.com/feed/"
   gator addfeed "Hacker News" "https://news.ycombinator.com/rss"
   ```

3. **Start the aggregator** (in a separate terminal):
   ```bash
   gator agg 1m
   ```

4. **Browse posts**:
   ```bash
   gator browse
   gator browse 10  # Show 10 posts instead of default 2
   ```

## Commands

### User Management
- `gator register <username>` - Create a new user account
- `gator login <username>` - Login as a user
- `gator users` - List all users

### Feed Management
- `gator addfeed <name> <url>` - Add a new RSS feed
- `gator feeds` - List all available feeds
- `gator follow <url>` - Follow an existing feed
- `gator following` - List feeds you're following
- `gator unfollow <url>` - Unfollow a feed

### Content Aggregation
- `gator agg <duration>` - Start the aggregator (e.g., `gator agg 1m`)
- `gator browse [limit]` - Browse posts from followed feeds

### System
- `gator reset` - Reset the database (⚠️ deletes all data)

## Examples

### Basic Workflow

```bash
# 1. Register and login
gator register alice
gator login alice

# 2. Add some feeds
gator addfeed "TechCrunch" "https://techcrunch.com/feed/"
gator addfeed "Hacker News" "https://news.ycombinator.com/rss"

# 3. Start aggregator (in background terminal)
gator agg 30s

# 4. Browse posts
gator browse
gator browse 5
```

### Following Existing Feeds

```bash
# If someone else already added a feed, you can follow it
gator follow "https://techcrunch.com/feed/"
```

### Aggregator Configuration

The aggregator runs continuously and fetches feeds at regular intervals:

```bash
# Fetch every 30 seconds
gator agg 30s

# Fetch every 5 minutes
gator agg 5m

# Fetch every hour
gator agg 1h
```

## Architecture

### Database Schema

- **users**: User accounts
- **feeds**: RSS feed definitions
- **feed_follows**: Many-to-many relationship between users and feeds
- **posts**: Individual posts from RSS feeds

### Key Components

- **RSS Parser**: Fetches and parses RSS feeds
- **Database Layer**: SQLC-generated type-safe database operations
- **CLI Framework**: Command-based interface with middleware
- **Aggregation Engine**: Continuous feed fetching and post storage

## Development

### Running from Source

```bash
# Clone and build
git clone https://github.com/PassZ/rss-aggregator.git
cd rss-aggregator

# Run migrations
cd sql/schema
goose postgres "postgres://username:password@localhost:5432/gator" up

# Run the application
go run main.go <command>
```

### Database Migrations

Migrations are managed with [Goose](https://github.com/pressly/goose):

```bash
# Run migrations
goose postgres "postgres://username:password@localhost:5432/gator" up

# Rollback migrations
goose postgres "postgres://username:password@localhost:5432/gator" down
```

## Performance Notes

- The aggregator respects rate limits to avoid overwhelming servers
- Duplicate posts are automatically ignored
- The system is designed to handle large numbers of feeds and posts efficiently
- Use appropriate intervals for the aggregator (not too frequent)
