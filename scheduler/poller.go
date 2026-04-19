package scheduler

import (
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/aweist/schedule-watcher/league"
	"github.com/aweist/schedule-watcher/models"
	"github.com/aweist/schedule-watcher/notifier"
	"github.com/aweist/schedule-watcher/storage"
)

type Poller struct {
	leagues  []league.League
	storage  *storage.BoltStorage
	notifier notifier.Notifier
	interval time.Duration
}

type PollerConfig struct {
	Leagues  []league.League
	Storage  *storage.BoltStorage
	Notifier notifier.Notifier
	Interval time.Duration
}

func NewPoller(config PollerConfig) *Poller {
	return &Poller{
		leagues:  config.Leagues,
		storage:  config.Storage,
		notifier: config.Notifier,
		interval: config.Interval,
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
	for _, lg := range p.leagues {
		log.Printf("Polling league: %s", lg.DisplayName())

		teamGames, err := lg.FetchAndParse()
		if err != nil {
			log.Printf("Error fetching %s: %v", lg.DisplayName(), err)
			continue
		}

		p.maybeSaveSnapshot(lg, teamGames)

		for _, team := range lg.Teams() {
			games := teamGames[team.Key]
			if len(games) == 0 {
				continue
			}

			newGamesFound := 0
			for _, game := range games {
				if game.Date.Before(time.Now().AddDate(0, 0, -1)) {
					continue
				}

				isNew, err := p.saveNewGame(game)
				if err != nil {
					log.Printf("Error saving game %s: %v", game.ID, err)
					continue
				}

				if isNew {
					newGamesFound++
				}

				// Immediate mode: notify any game not yet marked notified.
				// Gating on notification state (rather than isNew) lets transient
				// SMTP failures retry on subsequent polls.
				if lg.NotifyMode() == league.NotifyImmediate {
					notified, err := p.storage.IsGameNotified(game.League, game.TeamKey, game.ID)
					if err != nil {
						log.Printf("Error checking notification status for %s: %v", game.ID, err)
						continue
					}
					if notified {
						continue
					}
					if err := p.sendNotification(game); err != nil {
						continue
					}
					if err := p.storage.MarkGameNotified(game); err != nil {
						log.Printf("Error marking game as notified: %v", err)
					}
				}
			}

			if newGamesFound > 0 {
				if lg.NotifyMode() == league.NotifyImmediate {
					log.Printf("%s/%s: found %d new games and sent notifications", lg.DisplayName(), team.Key, newGamesFound)
				} else {
					log.Printf("%s/%s: saved %d new games (reminders will be sent on game day)", lg.DisplayName(), team.Key, newGamesFound)
				}
			}
		}
	}

	oneMonthAgo := time.Now().AddDate(0, -1, 0)
	if err := p.storage.CleanupOldNotifications(oneMonthAgo); err != nil {
		log.Printf("Error cleaning up old notifications: %v", err)
	}
}

// saveNewGame saves a game if it doesn't already exist. Returns true if the game is new.
func (p *Poller) saveNewGame(game models.Game) (bool, error) {
	existingGame, err := p.storage.GetGame(game.League, game.TeamKey, game.ID)
	if err != nil {
		return false, fmt.Errorf("getting existing game: %w", err)
	}

	if existingGame != nil {
		return false, nil
	}

	if err := p.storage.SaveGame(game); err != nil {
		return false, fmt.Errorf("saving game: %w", err)
	}

	return true, nil
}

// sendNotification sends an email notification for a game to the team's recipients.
// Returns nil on success (or when there are no recipients to notify), or an error if
// the send failed. Callers should only mark the game notified when this returns nil.
func (p *Poller) sendNotification(game models.Game) error {
	if p.notifier == nil {
		return nil
	}

	recipients, err := p.storage.GetActiveRecipientsForTeam(game.League, game.TeamKey)
	if err != nil {
		log.Printf("Error getting recipients for %s/%s: %v", game.League, game.TeamKey, err)
		return err
	}

	if len(recipients) == 0 {
		log.Printf("No active recipients for %s/%s, skipping notification", game.League, game.TeamKey)
		return nil
	}

	var emails []string
	for _, r := range recipients {
		emails = append(emails, r.Email)
	}

	if err := p.notifier.SendNotification(game, emails); err != nil {
		log.Printf("Error sending %s notification for game %s: %v", p.notifier.GetType(), game.ID, err)
		return err
	}
	log.Printf("Sent %s notification for %s game on %s at %s",
		p.notifier.GetType(), game.League, game.Date.Format("Jan 2"), game.Time)
	return nil
}

// maybeSaveSnapshot hashes the league's upstream payload, compares to the
// latest stored snapshot, and saves a new one only when the schedule has
// actually changed. Prefers raw upstream data (via league.RawDataProvider)
// because hashing parsed games can yield spurious differences from parse
// ordering.
func (p *Poller) maybeSaveSnapshot(lg league.League, teamGames map[string][]models.Game) {
	var rawData string
	if provider, ok := lg.(league.RawDataProvider); ok {
		rawData = provider.LastRawData()
	}

	var dataHash string
	if rawData != "" {
		dataHash = fmt.Sprintf("%x", sha256.Sum256([]byte(rawData)))
	} else {
		var allGames []models.Game
		for _, games := range teamGames {
			allGames = append(allGames, games...)
		}
		if len(allGames) == 0 {
			return
		}
		dataHash = hashGames(allGames)
	}

	lastHash, _ := p.storage.GetLatestSnapshotHash(lg.Name())
	if dataHash == lastHash {
		return
	}

	snapshot := models.Snapshot{
		ID:        fmt.Sprintf("snap-%s", dataHash[:12]),
		League:    lg.Name(),
		Hash:      dataHash,
		CSVData:   rawData,
		FetchedAt: time.Now(),
	}
	if err := p.storage.SaveSnapshot(snapshot); err != nil {
		log.Printf("Error saving snapshot: %v", err)
		return
	}
	log.Printf("%s: schedule changed, saved new snapshot %s", lg.DisplayName(), snapshot.ID)
}

func hashGames(games []models.Game) string {
	var data string
	for _, g := range games {
		data += fmt.Sprintf("%s|%s|%s|%s|", g.ID, g.Date.Format("2006-01-02"), g.Time, g.Court)
	}
	return fmt.Sprintf("%x", sha256.Sum256([]byte(data)))
}
