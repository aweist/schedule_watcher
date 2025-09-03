package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aweist/schedule-watcher/config"
	"github.com/aweist/schedule-watcher/notifier"
	"github.com/aweist/schedule-watcher/scheduler"
	"github.com/aweist/schedule-watcher/storage"
	"github.com/aweist/schedule-watcher/web"
)

func main() {
	cfg := config.LoadFromEnv()
	log.Println("Loaded configuration from environment variables")
	
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}
	
	db, err := storage.NewBoltStorage(cfg.Storage.DatabasePath)
	if err != nil {
		log.Fatalf("Error initializing storage: %v", err)
	}
	defer db.Close()
	
	var notifiers []notifier.Notifier
	
	if cfg.Email.Enabled {
		emailNotifier := notifier.NewEmailNotifier(notifier.EmailConfig{
			SMTPHost: cfg.Email.SMTPHost,
			SMTPPort: cfg.Email.SMTPPort,
			Username: cfg.Email.Username,
			Password: cfg.Email.Password,
			From:     cfg.Email.From,
			TeamName: cfg.Team.Name,
			Storage:  db,
		})
		notifiers = append(notifiers, emailNotifier)
		log.Println("Email notifications enabled")
	}
	
	if len(notifiers) == 0 {
		log.Println("WARNING: No notifiers configured. Games will be tracked but no notifications will be sent.")
	}
	
	poller := scheduler.NewPoller(scheduler.PollerConfig{
		APIBaseURL: cfg.API.BaseURL,
		Instance:   cfg.API.Instance,
		CompID:     cfg.API.CompID,
		TeamName:   cfg.Team.Name,
		Interval:   cfg.GetPollInterval(),
		Storage:    db,
		Notifiers:  notifiers,
	})
	
	log.Printf("Starting Schedule Watcher for team: %s", cfg.Team.Name)
	log.Printf("Polling interval: %s", cfg.Schedule.PollInterval)
	log.Printf("Database: %s", cfg.Storage.DatabasePath)
	
	// Start web server for debug interface
	if cfg.Web.Enabled {
		webServer := web.NewServer(db, cfg.Web.Port)
		webServer.SetNotifiers(notifiers)
		go webServer.Start(cfg.Team.Name)
	}
	
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	go func() {
		poller.Start()
	}()
	
	<-sigChan
	log.Println("Shutting down Schedule Watcher...")
}