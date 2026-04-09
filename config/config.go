package config

import (
	"os"
	"time"
)

type Config struct {
	Email    EmailConfig
	Storage  StorageConfig
	Schedule ScheduleConfig
	Web      WebConfig
	Leagues  map[string]LeagueConfig
}

type LeagueConfig struct {
	Type         string
	NotifyMode   string
	ReminderTime string
	API          map[string]string
	Teams        []TeamEntry
}

type TeamEntry struct {
	Key  string
	Name string
	Day  string
}

type EmailConfig struct {
	Enabled  bool
	SMTPHost string
	SMTPPort string
	Username string
	Password string
	From     string
}

type StorageConfig struct {
	DatabasePath string
}

type ScheduleConfig struct {
	PollInterval string
}

type WebConfig struct {
	Enabled bool
	Port    string
}

func Load() *Config {
	return &Config{
		Email: EmailConfig{
			Enabled:  true,
			SMTPHost: "smtp.gmail.com",
			SMTPPort: "587",
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("EMAIL_FROM"),
		},
		Storage: StorageConfig{
			DatabasePath: "./schedule.db",
		},
		Schedule: ScheduleConfig{
			PollInterval: "5m",
		},
		Web: WebConfig{
			Enabled: true,
			Port:    "8080",
		},
		Leagues: map[string]LeagueConfig{
			"IVP": {
				Type: "ivp",
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
				ReminderTime: "08:00",
				API: map[string]string{
					"base_url": "https://pins.killerworld.com",
				},
				Teams: []TeamEntry{
					{Key: "French Toast Mafia", Name: "French Toast Mafia", Day: "Wed"},
				},
			},
		},
	}
}

func (c *Config) GetPollInterval() time.Duration {
	d, _ := time.ParseDuration(c.Schedule.PollInterval)
	return d
}
