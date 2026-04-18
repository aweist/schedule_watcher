# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common Commands

### Development
```bash
# Run the application locally (loads .env via dotenv)
go run main.go

# Live-reload dev loop (recompiles + restarts on file change)
air

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

# Run with Docker Compose (local)
docker compose up -d

# View container logs
docker compose logs -f
```

Air is configured via [.air.toml](.air.toml) and runs the binary through `dotenv --` so `.env` values are loaded automatically. Watches `.go`, `.html`, `.css` files; excludes `tmp/`, `data/`, `docs/`, `scripts/`.

### Configuration
Config lives in Go code at [config/config.go](config/config.go) — there is **no** YAML config file. `config.Load()` returns a hardcoded struct; only secrets and the SMTP host/port come from environment variables (loaded from `.env` locally, passed through `docker-compose*.yml` in containers).

Required env vars:
- `SMTP_HOST`, `SMTP_PORT` — SMTP relay (prod uses `smtp.mailgun.org:587`)
- `SMTP_USERNAME`, `SMTP_PASSWORD`, `EMAIL_FROM` — SMTP auth + sender
- `API_INSTANCE`, `API_COMP_ID` — IVP Wix API credentials
- `TZ` — timezone (defaults to `America/Denver`)

To change polling interval, reminder time, tracked teams, etc., edit the struct literal in `config.Load()` and redeploy. The startup log line `Email notifications enabled: host=... from=...` confirms SMTP config is wired correctly (see [main.go:79](main.go#L79)).

### Deployment
Prod runs on a Digital Ocean droplet. Pushing to `main` triggers [.github/workflows/deploy.yml](.github/workflows/deploy.yml):
1. Builds and pushes image to `ghcr.io/aweist/schedule_watcher`
2. SSHes to the droplet, `cd /opt/schedule-watcher`, `git pull`, `docker compose -f docker-compose.prod.yml pull && down && up -d`

The droplet has a `.env` file at `/opt/schedule-watcher/.env` with production secrets. Container logs are capped at 30MB (3×10MB rotated json-file) and **are wiped on every deploy** (because `docker compose down` removes the container). For logs spanning deploys, grab them before redeploying:
```bash
ssh <droplet> 'docker logs -t volleyball-schedule-watcher' > logs.txt
```

`scripts/build-and-push.sh` is a manual build/push helper; it no longer drives deployment (GitHub Actions does).

## Architecture Overview

This is a **multi-league volleyball schedule monitoring service** that polls multiple schedule sources and sends email notifications for new games.

### Supported Leagues

- **IVP**: Polls a Wix Visual Data API for CSV schedule data. Client in `client/api.go`, parser in `parser/csv.go`.
- **PINS**: Scrapes HTML schedules from `pins.killerworld.com`. Auto-discovers current season and team IDs from dropdown menus. Client/parser/discovery in `league/pins/`.

### Core Components

**Data Flow**: League.FetchAndParse() → Storage → Notification System

1. **`league/league.go`**: `League` interface (`Name`, `DisplayName`, `NotifyMode`, `ReminderTime`, `FetchAndParse`, `Teams`) plus the `NotifyImmediate` / `NotifyDailyReminder` mode constants
2. **`main.go`**: Constructs leagues via a `switch leagueConfig.Type` dispatch (no factory registry — just `case "ivp"` / `case "pins"`)
3. **`league/ivp/ivp.go`**: IVP league implementation wrapping existing `client/` and `parser/` packages
4. **`league/pins/`**: PINS league implementation with HTML client, auto-discovery, and HTML table parser
5. **`storage/bolt.go`**: BoltDB persistence with scoped keys (`league:teamKey:id`) for multi-league isolation
6. **`notifier/`**: Interface-based notification system. Recipients are per-team per-league, stored in DB.
7. **`scheduler/poller.go`**: Iterates over all leagues, fetches/parses, processes new games, sends immediate-mode notifications
8. **`scheduler/reminder.go`**: Separate per-minute loop that fires game-day reminders for `daily_reminder` leagues at each league's `ReminderTime`

### Key Data Structures

- **`models.Game`**: Core game entity with League, TeamKey, ID, team info, date/time, court, opponent
- **`models.NotifiedGame`**: Tracks notification history to prevent duplicates, scoped by league/team
- **`models.EmailRecipient`**: Per-league/team email recipients stored in DB
- **`notifier.Notifier`**: Interface: `SendNotification(game, recipients) error`

### Config Structure (in Go)

Defined in [config/config.go](config/config.go). `Config.Leagues` is a `map[string]LeagueConfig` keyed by display name:

```go
Leagues: map[string]LeagueConfig{
    "IVP": {
        Type: "ivp",                 // dispatched in main.go switch
        // NotifyMode defaults to "immediate" if unset
        API: map[string]string{
            "base_url": "https://wix-visual-data.appspot.com",
            "instance": os.Getenv("API_INSTANCE"),
            "comp_id":  os.Getenv("API_COMP_ID"),
        },
        Teams: []TeamEntry{
            {Key: "Taylor Sisneros", Name: "Taylor Sisneros"},
        },
    },
    "PINS": {
        Type:         "pins",
        NotifyMode:   "daily_reminder",
        ReminderTime: "08:00",        // HH:MM, local TZ
        API: map[string]string{"base_url": "https://pins.killerworld.com"},
        Teams: []TeamEntry{
            {Key: "French Toast Mafia", Name: "French Toast Mafia", Day: "Wed"},
        },
    },
},
```

`TeamEntry.Day` (3-letter weekday) is required for PINS — used to match the current season in the schedule dropdown.

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

**Two delivery modes** (`league.NotifyMode()`):
- **`immediate`** (IVP): poller sends as soon as a new game is discovered.
- **`daily_reminder`** (PINS): poller saves the game silently; `scheduler/reminder.go` sends the email on the morning of the game at `ReminderTime`.

**Retry behavior**: both paths gate on `storage.IsGameNotified` (not on "newly discovered") and only call `MarkGameNotified` when `SendNotification` returns nil. So a transient SMTP failure (timeout, 5xx) leaves the game un-notified and it retries on the next poll / next reminder tick. If SMTP stays broken indefinitely, we'll keep retrying — acceptable given the volume.

**Test email** ([web/server.go](web/server.go) `/api/test-email`): takes an `email` form param and sends a synthetic game to just that address. The debug page has a modal that collects it — do **not** send to "all recipients".

### Testing Notes

Tests exist for:
- `parser/csv.go`: IVP CSV parsing (time formats, courts, team matching, dates, edge cases)
- `league/pins/discovery.go`: Season auto-discovery, team ID matching
- `league/pins/parser.go`: HTML table parsing, division extraction, game ID generation

### Adding a New League

1. Create `league/{name}/` package
2. Implement the `league.League` interface (see `league/ivp/ivp.go` for a minimal reference)
3. Add a `case "{name}":` branch to the switch in [main.go:55](main.go#L55) that calls your constructor
4. Add the league entry to `Leagues` in [config/config.go](config/config.go) `Load()` — including any required env-var-backed API params
5. If the league uses `daily_reminder` mode, set `NotifyMode` and `ReminderTime` in the config entry
