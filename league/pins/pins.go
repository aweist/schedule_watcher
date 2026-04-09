package pins

import (
	"fmt"
	"log"

	"github.com/aweist/schedule-watcher/config"
	"github.com/aweist/schedule-watcher/league"
	"github.com/aweist/schedule-watcher/models"
)

type PINSLeague struct {
	name         string
	displayName  string
	notifyMode   string
	reminderTime string
	client       *PINSClient
	teams        []league.TeamConfig
	teamEntries  []config.TeamEntry
}

func New(name string, cfg config.LeagueConfig) (*PINSLeague, error) {
	baseURL := cfg.API["base_url"]
	if baseURL == "" {
		return nil, fmt.Errorf("api.base_url is required for PINS league")
	}

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

	return &PINSLeague{
		name:         "pins",
		displayName:  name,
		notifyMode:   notifyMode,
		reminderTime: cfg.ReminderTime,
		client:       NewClient(baseURL),
		teams:        teams,
		teamEntries:  cfg.Teams,
	}, nil
}

func (l *PINSLeague) Name() string               { return l.name }
func (l *PINSLeague) DisplayName() string        { return l.displayName }
func (l *PINSLeague) NotifyMode() string         { return l.notifyMode }
func (l *PINSLeague) ReminderTime() string       { return l.reminderTime }
func (l *PINSLeague) Teams() []league.TeamConfig { return l.teams }

func (l *PINSLeague) FetchAndParse() (map[string][]models.Game, error) {
	// Step 1: Fetch the main schedules page for season discovery
	schedulesHTML, err := l.client.FetchSchedulesPage()
	if err != nil {
		return nil, fmt.Errorf("fetching schedules page: %w", err)
	}

	result := make(map[string][]models.Game)

	for _, team := range l.teamEntries {
		games, err := l.fetchTeamGames(schedulesHTML, team)
		if err != nil {
			log.Printf("Error fetching PINS games for team %s (%s): %v", team.Key, team.Name, err)
			continue
		}
		result[team.Key] = games
	}

	return result, nil
}

func (l *PINSLeague) fetchTeamGames(schedulesHTML string, team config.TeamEntry) ([]models.Game, error) {
	// Step 2: Discover the current SCHEDULE_ID for this team's day
	scheduleID, err := DiscoverCurrentScheduleID(schedulesHTML, team.Day)
	if err != nil {
		return nil, fmt.Errorf("discovering schedule for %s: %w", team.Day, err)
	}
	log.Printf("PINS: discovered schedule ID %s for %s night", scheduleID, team.Day)

	// Step 3: Fetch the teams page and discover the TEAM_ID
	teamsHTML, err := l.client.FetchTeamsPage(scheduleID)
	if err != nil {
		return nil, fmt.Errorf("fetching teams page: %w", err)
	}

	teamID, fullTeamName, err := DiscoverTeamID(teamsHTML, team.Name)
	if err != nil {
		return nil, fmt.Errorf("discovering team ID for %q: %w", team.Name, err)
	}
	log.Printf("PINS: discovered team ID %s (%s) for %q", teamID, fullTeamName, team.Name)

	// Step 4: Fetch and parse the team schedule
	scheduleHTML, err := l.client.FetchTeamSchedule(scheduleID, teamID)
	if err != nil {
		return nil, fmt.Errorf("fetching team schedule: %w", err)
	}

	games, err := ParseSchedule(scheduleHTML, team.Key, fullTeamName)
	if err != nil {
		return nil, fmt.Errorf("parsing team schedule: %w", err)
	}

	log.Printf("PINS: found %d games for team %s (%s)", len(games), team.Key, fullTeamName)
	return games, nil
}
