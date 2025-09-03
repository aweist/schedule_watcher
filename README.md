# Volleyball Schedule Watcher

A Go service that monitors volleyball league schedules and sends email notifications when new games are posted for your team.

## Features

- Polls volleyball schedule API at configurable intervals
- Tracks which games have already been notified
- Sends HTML email notifications for new games
- Persists data using BoltDB
- Dockerized for easy deployment
- Extensible notification system (ready for SMS, Slack, etc.)
- Web debug interface to view games and notification history

## Architecture

- **API Client**: Fetches schedule data from Wix Visual Data API
- **CSV Parser**: Parses schedule data and extracts team games
- **BoltDB Storage**: Persists game data and notification history
- **Notification System**: Interface-based design for multiple notification channels
- **Scheduler**: Polls API and manages change detection

## Setup

### Configuration

The service is configured using environment variables via a `.env` file:

1. Copy `.env.example` to `.env`
2. Update with your values:
   - `API_INSTANCE`: Your API instance ID (from the log file)
   - `API_COMP_ID`: Component ID (from the log file)  
   - `TEAM_NAME`: Your team captain's name (e.g., "Jeff", "Rachel Wise")
   - Email settings for notifications

### Email Setup (Gmail)

For Gmail, you'll need an app-specific password:
1. Enable 2-factor authentication on your Google account
2. Go to Google Account settings → Security → 2-Step Verification → App passwords
3. Generate a new app password for "Mail"
4. Use this password in the configuration

## Running Locally

### Direct Go Execution

```bash
# Install dependencies
go mod download

# Create .env file
cp .env.example .env
# Edit .env with your values

# Run the application
go run main.go
```

### Using Docker Compose

```bash
# Create .env file from example
cp .env.example .env
# Edit .env with your values

# Build and run
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

## Data Persistence

The service uses BoltDB to store:
- Game information (ID, team, date, time, court)
- Notification history (which games have been notified)

The database is stored in:
- Local: `./schedule.db` (configurable via `DB_PATH`)
- Docker: `/data/schedule.db` (mounted volume)

## Adding New Notification Types

To add SMS or other notification types:

1. Create a new file in `notifier/` (e.g., `sms.go`)
2. Implement the `Notifier` interface:
```go
type SMSNotifier struct {
    // your fields
}

func (s *SMSNotifier) SendNotification(game models.Game) error {
    // implementation
}

func (s *SMSNotifier) GetType() string {
    return "sms"
}
```
3. Add environment variables for configuration in `config/config.go`
4. Initialize in `main.go`

## API Details

The service polls the Wix Visual Data API endpoint:
```
GET https://wix-visual-data.appspot.com/api/file?instance={instance}&compId={compId}
```

The response contains CSV data with team schedules including:
- Team captain names
- Game dates and times
- Court assignments
- Division information

## Monitoring

The service logs:
- Polling activities
- New games found
- Notifications sent
- Errors encountered

### Web Debug Interface

Access the debug interface at `http://localhost:8080` (or configured port) to view:
- All parsed games from the schedule
- Notification history with timestamps
- Visual indicators for past, present, and future games
- Auto-refresh every 30 seconds

Configuration:
- `WEB_ENABLED`: Enable/disable web interface (default: true)
- `WEB_PORT`: Port for web server (default: 8080)

## Troubleshooting

- **No notifications**: Check team name matches exactly (case-sensitive)
- **Email failures**: Verify SMTP settings and app password
- **No games found**: Ensure API credentials are correct
- **Database errors**: Check file permissions for database path

## License

MIT