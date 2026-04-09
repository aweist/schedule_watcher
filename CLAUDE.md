# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

### Development
```bash
# Run the application locally
go run main.go -config config.yaml

# Install dependencies
go mod download

# Run all tests
go test ./...

# Run specific package tests
go test ./parser -v
go test ./league/pins -v

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
The application uses a YAML config file (`config.yaml`). Copy `config.yaml.example` to `config.yaml` and configure. Secrets can be injected via `${ENV_VAR}` interpolation in the YAML.

## Architecture Overview

This is a **multi-league volleyball schedule monitoring service** that polls multiple schedule sources and sends email notifications for new games.

### Supported Leagues

- **IVP**: Polls a Wix Visual Data API for CSV schedule data. Client in `client/api.go`, parser in `parser/csv.go`.
- **PINS**: Scrapes HTML schedules from `pins.killerworld.com`. Auto-discovers current season and team IDs from dropdown menus. Client/parser/discovery in `league/pins/`.

### Core Components

**Data Flow**: League.FetchAndParse() → Storage → Notification System

1. **`league/league.go`**: `League` interface that all schedule sources implement (`FetchAndParse()`, `Name()`, `Teams()`)
2. **`league/factory.go`**: Registry-based factory that builds League instances from config. Leagues self-register via `init()`.
3. **`league/ivp/ivp.go`**: IVP league implementation wrapping existing `client/` and `parser/` packages
4. **`league/pins/`**: PINS league implementation with HTML client, auto-discovery, and HTML table parser
5. **`storage/bolt.go`**: BoltDB persistence with scoped keys (`league:teamKey:id`) for multi-league isolation
6. **`notifier/`**: Interface-based notification system. Recipients are per-team per-league, stored in DB.
7. **`scheduler/poller.go`**: Iterates over all leagues, fetches/parses, processes new games, sends notifications

### Key Data Structures

- **`models.Game`**: Core game entity with League, TeamKey, ID, team info, date/time, court, opponent
- **`models.NotifiedGame`**: Tracks notification history to prevent duplicates, scoped by league/team
- **`models.EmailRecipient`**: Per-league/team email recipients stored in DB
- **`notifier.Notifier`**: Interface: `SendNotification(game, recipients) error`

### Config Structure (`config.yaml`)

```yaml
leagues:
  - type: ivp       # or "pins"
    name: IVP        # Display name
    api:             # League-specific params (map[string]string)
      base_url: ...
      instance: ...
      comp_id: ...
    teams:
      - key: weist   # Stable key for storage scoping
        name: Weist   # Name used for team matching
        day: Tue      # Required for PINS (day of week for season discovery)
```

### PINS Auto-Discovery

The PINS league automatically discovers schedule and team IDs:
1. Fetches `/schedules.cgi` to get the season dropdown
2. Matches the team's `day` config to find current season SCHEDULE_ID
3. Fetches the teams page and matches team name to find TEAM_ID
4. Fetches and parses the HTML schedule table

### Storage Scoping

All BoltDB keys use the format `{league}:{teamKey}:{id}` to isolate data per league/team. A one-time migration runs on startup to convert any existing flat keys to scoped format.

### CSV Parsing (IVP)

The CSV parser handles volleyball-specific formats:
- **Multiple games per night**: "8/9pm" → ["8", "9"] 
- **Multiple courts**: "ct 7/8" → ["7", "8"]
- **Team matching**: Case-insensitive partial matching on captain names
- **Date columns**: Finds date headers with "/" pattern, time is in previous column, court data is in date column

### Notification System

```go
type Notifier interface {
    SendNotification(game models.Game, recipients []string) error
    GetType() string
}
```

Recipients are managed per league/team via the web admin UI and stored in BoltDB. Email content includes league name, opponent info, and league-specific schedule links.

### Testing Notes

Tests exist for:
- `parser/csv.go`: IVP CSV parsing (time formats, courts, team matching, dates, edge cases)
- `league/pins/discovery.go`: Season auto-discovery, team ID matching
- `league/pins/parser.go`: HTML table parsing, division extraction, game ID generation

### Adding a New League

1. Create `league/{name}/` package
2. Implement `league.League` interface
3. Register via `init()`: `league.Register("name", builderFunc)`
4. Add config type handling in `config.Validate()`
5. Import the package in `main.go` with blank import: `_ "...league/{name}"`
