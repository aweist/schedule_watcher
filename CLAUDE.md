# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

### Development
```bash
# Run the application locally
go run main.go

# Install dependencies
go mod download

# Run all tests
go test ./...

# Run specific package tests
go test ./parser -v

# Run tests with coverage
go test -cover ./...

# Build binary
go build -o schedule-watcher

# Run with Docker Compose
docker-compose up -d

# View container logs
docker-compose logs -f
```

### Configuration
The application only uses environment variables for configuration (no JSON files). Create `.env` from `.env.example` and configure:
- `API_INSTANCE`: Wix API instance ID
- `API_COMP_ID`: Component ID 
- `TEAM_NAME`: Team captain name for filtering
- Email SMTP settings

## Architecture Overview

This is a **volleyball schedule monitoring service** that polls a Wix Visual Data API for CSV schedule data and sends email notifications for new games.

### Core Components

**Data Flow**: API → CSV Parser → Storage → Notification System

1. **`client/api.go`**: HTTP client that fetches CSV schedule data from Wix Visual Data API
2. **`parser/csv.go`**: Parses complex volleyball schedule CSV format with multiple games per team and handles time/court formats like "8/9pm" and "ct 7/8"
3. **`storage/bolt.go`**: BoltDB persistence layer tracking games and notification history to prevent duplicate alerts
4. **`notifier/`**: Interface-based notification system (currently email only, extensible for SMS/Slack)
5. **`scheduler/poller.go`**: Main polling loop that orchestrates the pipeline every 5 minutes

### Key Data Structures

- **`models.Game`**: Core game entity with ID, team info, date/time, court
- **`models.NotifiedGame`**: Tracks notification history to prevent duplicates
- **`notifier.Notifier`**: Interface for extensible notification channels

### CSV Parsing Complexity

The CSV parser handles volleyball-specific formats:
- **Multiple games per night**: "8/9pm" → ["8", "9"] 
- **Multiple courts**: "ct 7/8" → ["7", "8"]
- **Team matching**: Case-insensitive partial matching on captain names
- **Date columns**: Finds date headers with "/" pattern, time is in previous column, court data is in date column

### Persistence Strategy

Uses BoltDB with two buckets:
- `games`: Stores all parsed games
- `notified`: Tracks which games have been notified to prevent duplicates

Game IDs are generated using MD5 hash of captain+date+time+court for consistency across polls.

### Notification System

Interface-based design allows adding new notification types:
```go
type Notifier interface {
    SendNotification(game models.Game) error
    GetType() string
}
```

Current implementation is HTML email via SMTP. To add new types, implement the interface and initialize in `main.go`.

### Testing Notes

Comprehensive tests exist for `parser/csv.go` covering:
- Multiple game time formats ("8/9pm", "7:00 PM")
- Court parsing ("ct 7/8", "CT 4") 
- Team name matching (case-insensitive, partial)
- Date parsing (various formats, year handling)
- Edge cases (malformed CSV, empty data)

The CSV format is complex with specific column patterns that tests validate extensively.