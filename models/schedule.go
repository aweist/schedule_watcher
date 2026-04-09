package models

import (
	"time"
)

type Game struct {
	ID          string    `json:"id"`
	League      string    `json:"league"`
	TeamKey     string    `json:"team_key"`
	TeamCaptain string    `json:"team_captain"`
	TeamNumber  int       `json:"team_number"`
	Division    string    `json:"division"`
	Date        time.Time `json:"date"`
	Time        string    `json:"time"`
	Court       string    `json:"court"`
	Opponent    string    `json:"opponent"`
	Raw         string    `json:"raw"`
}

type NotifiedGame struct {
	GameID      string    `json:"game_id"`
	League      string    `json:"league"`
	TeamKey     string    `json:"team_key"`
	NotifiedAt  time.Time `json:"notified_at"`
	TeamCaptain string    `json:"team_captain"`
	GameDate    time.Time `json:"game_date"`
	GameTime    string    `json:"game_time"`
	Court       string    `json:"court"`
}

type EmailRecipient struct {
	ID       string    `json:"id"`
	League   string    `json:"league"`
	TeamKey  string    `json:"team_key"`
	Email    string    `json:"email"`
	Name     string    `json:"name"`
	AddedAt  time.Time `json:"added_at"`
	IsActive bool      `json:"is_active"`
}

// Schedule represents raw schedule data fetched from an API (used by IVP client).
type Schedule struct {
	CSVData string `json:"csvData"`
}

type Snapshot struct {
	ID        string    `json:"id"`
	League    string    `json:"league"`
	CSVData   string    `json:"csv_data"`
	Hash      string    `json:"hash"`
	FetchedAt time.Time `json:"fetched_at"`
}
