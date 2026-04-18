package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aweist/schedule-watcher/config"
	"github.com/aweist/schedule-watcher/league"
	"github.com/aweist/schedule-watcher/league/ivp"
	"github.com/aweist/schedule-watcher/league/pins"
	"github.com/aweist/schedule-watcher/notifier"
	"github.com/aweist/schedule-watcher/scheduler"
	"github.com/aweist/schedule-watcher/storage"
	"github.com/aweist/schedule-watcher/web"
)

func main() {
	cfg := config.Load()
	log.Println("Configuration loaded")

	db, err := storage.NewBoltStorage(cfg.Storage.DatabasePath)
	if err != nil {
		log.Fatalf("Error initializing storage: %v", err)
	}
	defer db.Close()

	// Migrate existing data to scoped keys if needed
	for name, lg := range cfg.Leagues {
		if len(lg.Teams) > 0 {
			if err := db.MigrateToScoped(name, lg.Teams[0].Key); err != nil {
				log.Printf("Warning: data migration failed: %v", err)
			}
			break // only migrate once using the first league
		}
	}

	// Clean up DB data for league/team combos no longer in the config
	// Storage keys use the league type (e.g., "pins", "ivp") as the first segment
	validTeams := make(map[string]bool)
	for _, lgCfg := range cfg.Leagues {
		for _, t := range lgCfg.Teams {
			validTeams[lgCfg.Type+":"+t.Key] = true
		}
	}
	if err := db.CleanupStaleData(validTeams); err != nil {
		log.Printf("Warning: stale data cleanup failed: %v", err)
	}

	// Build leagues
	var leagues []league.League
	for name, leagueConfig := range cfg.Leagues {
		var lg league.League
		switch leagueConfig.Type {
		case "ivp":
			lg, err = ivp.New(name, leagueConfig)
		case "pins":
			lg, err = pins.New(name, leagueConfig)
		default:
			log.Fatalf("Unknown league type %q for %q", leagueConfig.Type, name)
		}
		if err != nil {
			log.Fatalf("Error building league %q: %v", name, err)
		}
		leagues = append(leagues, lg)
	}

	// Set up notifier
	var emailNotifier notifier.Notifier
	if cfg.Email.Enabled {
		emailNotifier = notifier.NewEmailNotifier(notifier.EmailConfig{
			SMTPHost: cfg.Email.SMTPHost,
			SMTPPort: cfg.Email.SMTPPort,
			Username: cfg.Email.Username,
			Password: cfg.Email.Password,
			From:     cfg.Email.From,
		})
		log.Printf("Email notifications enabled: host=%s:%s from=%s", cfg.Email.SMTPHost, cfg.Email.SMTPPort, cfg.Email.From)
	} else {
		log.Println("WARNING: Email notifications disabled. Games will be tracked but no notifications will be sent.")
	}

	// Create poller
	poller := scheduler.NewPoller(scheduler.PollerConfig{
		Leagues:  leagues,
		Storage:  db,
		Notifier: emailNotifier,
		Interval: cfg.GetPollInterval(),
	})

	log.Printf("Starting Schedule Watcher with %d league(s)", len(leagues))
	for _, lg := range leagues {
		log.Printf("  League: %s (%d teams)", lg.DisplayName(), len(lg.Teams()))
		for _, t := range lg.Teams() {
			log.Printf("    Team: %s (%s)", t.Name, t.Key)
		}
	}
	log.Printf("Polling interval: %s", cfg.Schedule.PollInterval)
	log.Printf("Database: %s", cfg.Storage.DatabasePath)

	// Start web server
	if cfg.Web.Enabled {
		webServer := web.NewServer(db, cfg.Web.Port, leagues)
		if emailNotifier != nil {
			webServer.SetNotifier(emailNotifier)
		}
		go webServer.Start()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go poller.Start()

	// Start daily reminder for leagues with notify_mode: daily_reminder
	reminder := scheduler.NewDailyReminder(leagues, db, emailNotifier)
	go reminder.Start()

	<-sigChan
	log.Println("Shutting down Schedule Watcher...")
}
