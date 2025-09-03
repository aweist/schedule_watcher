package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	API      APIConfig
	Team     TeamConfig
	Email    EmailConfig
	Storage  StorageConfig
	Schedule ScheduleConfig
	Web      WebConfig
}

type APIConfig struct {
	BaseURL  string
	Instance string
	CompID   string
}

type TeamConfig struct {
	Name string
}

type EmailConfig struct {
	Enabled  bool
	SMTPHost string
	SMTPPort string
	Username string
	Password string
	From     string
	To       []string
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

func LoadFromEnv() *Config {
	return &Config{
		API: APIConfig{
			BaseURL:  getEnv("API_BASE_URL", "https://wix-visual-data.appspot.com"),
			Instance: os.Getenv("API_INSTANCE"),
			CompID:   os.Getenv("API_COMP_ID"),
		},
		Team: TeamConfig{
			Name: getEnv("TEAM_NAME", ""),
		},
		Email: EmailConfig{
			Enabled:  getEnvBool("EMAIL_ENABLED", true),
			SMTPHost: getEnv("SMTP_HOST", "smtp.gmail.com"),
			SMTPPort: getEnv("SMTP_PORT", "587"),
			Username: os.Getenv("SMTP_USERNAME"),
			Password: os.Getenv("SMTP_PASSWORD"),
			From:     os.Getenv("EMAIL_FROM"),
			To:       []string{os.Getenv("EMAIL_TO")},
		},
		Storage: StorageConfig{
			DatabasePath: getEnv("DB_PATH", "./schedule.db"),
		},
		Schedule: ScheduleConfig{
			PollInterval: getEnv("POLL_INTERVAL", "5m"),
		},
		Web: WebConfig{
			Enabled: getEnvBool("WEB_ENABLED", true),
			Port:    getEnv("WEB_PORT", "8080"),
		},
	}
}

func (c *Config) Validate() error {
	if c.API.BaseURL == "" {
		return fmt.Errorf("api.base_url is required")
	}
	
	if c.API.Instance == "" {
		return fmt.Errorf("api.instance is required")
	}
	
	if c.API.CompID == "" {
		return fmt.Errorf("api.comp_id is required")
	}
	
	if c.Team.Name == "" {
		return fmt.Errorf("team.name is required")
	}
	
	if c.Email.Enabled {
		if c.Email.SMTPHost == "" {
			return fmt.Errorf("email.smtp_host is required when email is enabled")
		}
		
		if c.Email.From == "" {
			return fmt.Errorf("email.from is required when email is enabled")
		}
		
		if len(c.Email.To) == 0 || c.Email.To[0] == "" {
			return fmt.Errorf("email.to is required when email is enabled")
		}
	}
	
	if _, err := time.ParseDuration(c.Schedule.PollInterval); err != nil {
		return fmt.Errorf("invalid poll_interval: %w", err)
	}
	
	return nil
}

func (c *Config) GetPollInterval() time.Duration {
	d, _ := time.ParseDuration(c.Schedule.PollInterval)
	return d
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1"
	}
	return defaultValue
}