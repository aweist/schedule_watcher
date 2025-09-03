package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/aweist/schedule-watcher/client"
	"github.com/aweist/schedule-watcher/models"
	"github.com/aweist/schedule-watcher/notifier"
	"github.com/aweist/schedule-watcher/parser"
	"github.com/aweist/schedule-watcher/storage"
)

type Poller struct {
	apiClient  *client.APIClient
	storage    *storage.BoltStorage
	parser     *parser.CSVParser
	notifiers  []notifier.Notifier
	instance   string
	compID     string
	interval   time.Duration
}

type PollerConfig struct {
	APIBaseURL string
	Instance   string
	CompID     string
	TeamName   string
	Interval   time.Duration
	Storage    *storage.BoltStorage
	Notifiers  []notifier.Notifier
}

func NewPoller(config PollerConfig) *Poller {
	return &Poller{
		apiClient: client.NewAPIClient(config.APIBaseURL),
		storage:   config.Storage,
		parser:    parser.NewCSVParser(config.TeamName),
		notifiers: config.Notifiers,
		instance:  config.Instance,
		compID:    config.CompID,
		interval:  config.Interval,
	}
}

func (p *Poller) Start() {
	log.Println("Starting schedule poller...")
	
	p.poll()
	
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	
	for range ticker.C {
		p.poll()
	}
}

func (p *Poller) poll() {
	log.Println("Polling for schedule updates...")
	
	schedule, err := p.apiClient.FetchSchedule(p.instance, p.compID)
	if err != nil {
		log.Printf("Error fetching schedule: %v", err)
		return
	}
	
	games, err := p.parser.ParseSchedule(schedule.CSVData)
	if err != nil {
		log.Printf("Error parsing schedule: %v", err)
		return
	}
	
	log.Printf("Found %d games for tracked team", len(games))
	
	newGamesFound := 0
	for _, game := range games {
		if game.Date.Before(time.Now().AddDate(0, 0, -1)) {
			continue
		}
		
		isNew, err := p.processGame(game)
		if err != nil {
			log.Printf("Error processing game %s: %v", game.ID, err)
			continue
		}
		
		if isNew {
			newGamesFound++
		}
	}
	
	if newGamesFound > 0 {
		log.Printf("Found %d new games and sent notifications", newGamesFound)
	} else {
		log.Println("No new games found")
	}
	
	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	if err := p.storage.CleanupOldNotifications(oneMonthAgo); err != nil {
		log.Printf("Error cleaning up old notifications: %v", err)
	}
}

func (p *Poller) processGame(game models.Game) (bool, error) {
	notified, err := p.storage.IsGameNotified(game.ID)
	if err != nil {
		return false, fmt.Errorf("checking if game notified: %w", err)
	}
	
	if notified {
		return false, nil
	}
	
	existingGame, err := p.storage.GetGame(game.ID)
	if err != nil {
		return false, fmt.Errorf("getting existing game: %w", err)
	}
	
	if existingGame != nil {
		return false, nil
	}
	
	if err := p.storage.SaveGame(game); err != nil {
		return false, fmt.Errorf("saving game: %w", err)
	}
	
	for _, n := range p.notifiers {
		if err := n.SendNotification(game); err != nil {
			log.Printf("Error sending %s notification for game %s: %v", n.GetType(), game.ID, err)
		} else {
			log.Printf("Sent %s notification for game on %s at %s", 
				n.GetType(), game.Date.Format("Jan 2"), game.Time)
		}
	}
	
	if err := p.storage.MarkGameNotified(game); err != nil {
		return false, fmt.Errorf("marking game as notified: %w", err)
	}
	
	return true, nil
}