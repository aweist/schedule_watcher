package scheduler

import (
	"log"
	"time"

	"github.com/aweist/schedule-watcher/league"
	"github.com/aweist/schedule-watcher/notifier"
	"github.com/aweist/schedule-watcher/storage"
)

// DailyReminder sends game-day reminders for leagues with notify_mode "daily_reminder".
// It checks once per minute and fires at the configured reminder_time each day.
type DailyReminder struct {
	leagues  []league.League
	storage  *storage.BoltStorage
	notifier notifier.Notifier
	lastSent map[string]string // leagueName -> last date sent (YYYY-MM-DD)
}

func NewDailyReminder(leagues []league.League, store *storage.BoltStorage, n notifier.Notifier) *DailyReminder {
	return &DailyReminder{
		leagues:  leagues,
		storage:  store,
		notifier: n,
		lastSent: make(map[string]string),
	}
}

func (d *DailyReminder) Start() {
	// Filter to only daily_reminder leagues
	var reminderLeagues []league.League
	for _, lg := range d.leagues {
		if lg.NotifyMode() == league.NotifyDailyReminder {
			reminderLeagues = append(reminderLeagues, lg)
		}
	}

	if len(reminderLeagues) == 0 {
		log.Println("No daily_reminder leagues configured, skipping reminder scheduler")
		return
	}

	for _, lg := range reminderLeagues {
		log.Printf("Daily reminders enabled for %s at %s", lg.DisplayName(), lg.ReminderTime())
	}

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		d.check(reminderLeagues)
	}
}

func (d *DailyReminder) check(leagues []league.League) {
	now := time.Now()
	currentTime := now.Format("15:04")
	today := now.Format("2006-01-02")

	for _, lg := range leagues {
		// Already sent reminders for this league today
		if d.lastSent[lg.Name()] == today {
			continue
		}

		// Check if it's time (compare HH:MM)
		if currentTime < lg.ReminderTime() {
			continue
		}

		log.Printf("Sending daily reminders for %s", lg.DisplayName())
		d.sendRemindersForToday(lg, today)
		d.lastSent[lg.Name()] = today
	}
}

func (d *DailyReminder) sendRemindersForToday(lg league.League, today string) {
	if d.notifier == nil {
		return
	}

	for _, team := range lg.Teams() {
		games, err := d.storage.GetGamesByLeagueTeam(lg.Name(), team.Key)
		if err != nil {
			log.Printf("Error getting games for %s/%s: %v", lg.Name(), team.Key, err)
			continue
		}

		for _, game := range games {
			gameDate := game.Date.Format("2006-01-02")
			if gameDate != today {
				continue
			}

			// Check if already notified (don't remind twice)
			notified, err := d.storage.IsGameNotified(game.League, game.TeamKey, game.ID)
			if err != nil {
				log.Printf("Error checking notification status for %s: %v", game.ID, err)
				continue
			}
			if notified {
				continue
			}

			// Get recipients and send
			recipients, err := d.storage.GetActiveRecipientsForTeam(game.League, game.TeamKey)
			if err != nil || len(recipients) == 0 {
				continue
			}

			var emails []string
			for _, r := range recipients {
				emails = append(emails, r.Email)
			}

			if err := d.notifier.SendNotification(game, emails); err != nil {
				log.Printf("Error sending daily reminder for %s game %s: %v", lg.DisplayName(), game.ID, err)
			} else {
				log.Printf("Sent daily reminder for %s: %s at %s on Court %s",
					lg.DisplayName(), game.Date.Format("Jan 2"), game.Time, game.Court)
			}

			// Mark as notified so we don't remind again
			if err := d.storage.MarkGameNotified(game); err != nil {
				log.Printf("Error marking game as notified: %v", err)
			}
		}
	}
}
