package models

import (
	"time"
)

type Schedule struct {
	CSVData string `json:"csvData"`
}

type Game struct {
	ID          string    `json:"id"`
	TeamCaptain string    `json:"team_captain"`
	TeamNumber  int       `json:"team_number"`
	Division    string    `json:"division"`
	Date        time.Time `json:"date"`
	Time        string    `json:"time"`
	Court       string    `json:"court"`
	Raw         string    `json:"raw"`
}

type NotifiedGame struct {
	GameID      string    `json:"game_id"`
	NotifiedAt  time.Time `json:"notified_at"`
	TeamCaptain string    `json:"team_captain"`
	GameDate    time.Time `json:"game_date"`
	GameTime    string    `json:"game_time"`
	Court       string    `json:"court"`
}
