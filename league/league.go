package league

import "github.com/aweist/schedule-watcher/models"

const (
	NotifyImmediate     = "immediate"
	NotifyDailyReminder = "daily_reminder"
)

// League defines the contract that any schedule source must fulfill.
type League interface {
	// Name returns a stable identifier for this league (e.g., "ivp", "pins").
	Name() string

	// DisplayName returns a human-readable name (e.g., "IVP", "PINS").
	DisplayName() string

	// NotifyMode returns how this league should send notifications:
	// "immediate" (on discovery) or "daily_reminder" (morning of game day).
	NotifyMode() string

	// ReminderTime returns the HH:MM time for daily reminders (only used when NotifyMode is "daily_reminder").
	ReminderTime() string

	// FetchAndParse fetches schedule data and parses it into games
	// for all configured teams in this league. Returns a map of team key -> games.
	FetchAndParse() (map[string][]models.Game, error)

	// Teams returns the list of teams this league is tracking.
	Teams() []TeamConfig
}

// TeamConfig holds per-team configuration within a league.
type TeamConfig struct {
	Key  string // Stable identifier
	Name string // Display name
}

// RawDataProvider is optionally implemented by leagues whose raw upstream
// payload (e.g., CSV, HTML) is worth archiving in snapshots for debugging
// historical schedule changes. Leagues that don't implement it produce
// snapshots without stored raw data.
type RawDataProvider interface {
	LastRawData() string
}
