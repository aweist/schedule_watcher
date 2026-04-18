package ivp

import (
	"fmt"
	"log"

	"github.com/aweist/schedule-watcher/client"
	"github.com/aweist/schedule-watcher/config"
	"github.com/aweist/schedule-watcher/league"
	"github.com/aweist/schedule-watcher/models"
	"github.com/aweist/schedule-watcher/parser"
)

type IVPLeague struct {
	name         string
	displayName  string
	notifyMode   string
	reminderTime string
	apiClient    *client.APIClient
	instance     string
	compID       string
	teams        []league.TeamConfig
	teamEntries  []config.TeamEntry
	lastRawCSV   string
}

func New(name string, cfg config.LeagueConfig) (*IVPLeague, error) {
	baseURL := cfg.API["base_url"]
	if baseURL == "" {
		baseURL = "https://wix-visual-data.appspot.com"
	}

	instance := cfg.API["instance"]
	compID := cfg.API["comp_id"]

	var teams []league.TeamConfig
	for _, t := range cfg.Teams {
		teams = append(teams, league.TeamConfig{
			Key:  t.Key,
			Name: t.Name,
		})
	}

	notifyMode := cfg.NotifyMode
	if notifyMode == "" {
		notifyMode = league.NotifyImmediate
	}

	return &IVPLeague{
		name:         "ivp",
		displayName:  name,
		notifyMode:   notifyMode,
		reminderTime: cfg.ReminderTime,
		apiClient:    client.NewAPIClient(baseURL),
		instance:     instance,
		compID:       compID,
		teams:        teams,
		teamEntries:  cfg.Teams,
	}, nil
}

func (l *IVPLeague) Name() string               { return l.name }
func (l *IVPLeague) DisplayName() string        { return l.displayName }
func (l *IVPLeague) NotifyMode() string         { return l.notifyMode }
func (l *IVPLeague) ReminderTime() string       { return l.reminderTime }
func (l *IVPLeague) Teams() []league.TeamConfig { return l.teams }
func (l *IVPLeague) LastRawData() string        { return l.lastRawCSV }

func (l *IVPLeague) FetchAndParse() (map[string][]models.Game, error) {
	schedule, err := l.apiClient.FetchSchedule(l.instance, l.compID)
	if err != nil {
		return nil, fmt.Errorf("fetching IVP schedule: %w", err)
	}
	l.lastRawCSV = schedule.CSVData

	result := make(map[string][]models.Game)

	for _, team := range l.teamEntries {
		csvParser := parser.NewCSVParser(team.Name)
		games, err := csvParser.ParseSchedule(schedule.CSVData)
		if err != nil {
			log.Printf("Error parsing IVP schedule for team %s: %v", team.Key, err)
			continue
		}

		// Tag each game with league and team info, prefix IDs
		for i := range games {
			games[i].League = l.name
			games[i].TeamKey = team.Key
			games[i].ID = fmt.Sprintf("ivp-%s", games[i].ID)
		}

		result[team.Key] = games
	}

	return result, nil
}
