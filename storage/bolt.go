package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/aweist/schedule-watcher/models"
	bolt "go.etcd.io/bbolt"
)

const (
	bucketGames      = "games"
	bucketNotified   = "notified"
	bucketRecipients = "recipients"
	bucketSnapshots  = "snapshots"
	bucketMeta       = "_meta"
)

type BoltStorage struct {
	db *bolt.DB
}

func NewBoltStorage(dbPath string) (*BoltStorage, error) {
	db, err := bolt.Open(dbPath, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		for _, bucket := range []string{bucketGames, bucketNotified, bucketRecipients, bucketSnapshots, bucketMeta} {
			if _, err := tx.CreateBucketIfNotExists([]byte(bucket)); err != nil {
				return fmt.Errorf("creating %s bucket: %w", bucket, err)
			}
		}
		return nil
	})

	if err != nil {
		db.Close()
		return nil, err
	}

	return &BoltStorage{db: db}, nil
}

func (s *BoltStorage) Close() error {
	return s.db.Close()
}

// scopedKey creates a key in the format "league:teamKey:id"
func scopedKey(league, teamKey, id string) string {
	return fmt.Sprintf("%s:%s:%s", league, teamKey, id)
}

// snapshotKey creates a key in the format "league:id"
func snapshotKey(league, id string) string {
	return fmt.Sprintf("%s:%s", league, id)
}

// --- Games ---

func (s *BoltStorage) SaveGame(game models.Game) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketGames))
		data, err := json.Marshal(game)
		if err != nil {
			return fmt.Errorf("marshaling game: %w", err)
		}
		key := scopedKey(game.League, game.TeamKey, game.ID)
		return b.Put([]byte(key), data)
	})
}

func (s *BoltStorage) GetGame(league, teamKey, gameID string) (*models.Game, error) {
	var game models.Game

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketGames))
		data := b.Get([]byte(scopedKey(league, teamKey, gameID)))
		if data == nil {
			return nil
		}
		return json.Unmarshal(data, &game)
	})

	if err != nil {
		return nil, err
	}
	if game.ID == "" {
		return nil, nil
	}
	return &game, nil
}

func (s *BoltStorage) GetAllGames() ([]models.Game, error) {
	var games []models.Game

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketGames))
		return b.ForEach(func(k, v []byte) error {
			var game models.Game
			if err := json.Unmarshal(v, &game); err != nil {
				return err
			}
			games = append(games, game)
			return nil
		})
	})

	return games, err
}

func (s *BoltStorage) GetGamesByLeagueTeam(league, teamKey string) ([]models.Game, error) {
	prefix := fmt.Sprintf("%s:%s:", league, teamKey)
	var games []models.Game

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketGames))
		c := b.Cursor()
		for k, v := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, v = c.Next() {
			var game models.Game
			if err := json.Unmarshal(v, &game); err != nil {
				return err
			}
			games = append(games, game)
		}
		return nil
	})

	return games, err
}

func (s *BoltStorage) DeleteGame(league, teamKey, gameID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketGames))
		return b.Delete([]byte(scopedKey(league, teamKey, gameID)))
	})
}

// --- Notified Games ---

func (s *BoltStorage) IsGameNotified(league, teamKey, gameID string) (bool, error) {
	var exists bool

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketNotified))
		data := b.Get([]byte(scopedKey(league, teamKey, gameID)))
		exists = data != nil
		return nil
	})

	return exists, err
}

func (s *BoltStorage) MarkGameNotified(game models.Game) error {
	notified := models.NotifiedGame{
		GameID:      game.ID,
		League:      game.League,
		TeamKey:     game.TeamKey,
		NotifiedAt:  time.Now(),
		TeamCaptain: game.TeamCaptain,
		GameDate:    game.Date,
		GameTime:    game.Time,
		Court:       game.Court,
	}

	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketNotified))
		data, err := json.Marshal(notified)
		if err != nil {
			return fmt.Errorf("marshaling notified game: %w", err)
		}
		key := scopedKey(game.League, game.TeamKey, game.ID)
		return b.Put([]byte(key), data)
	})
}

func (s *BoltStorage) GetAllNotifiedGames() ([]models.NotifiedGame, error) {
	var games []models.NotifiedGame

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketNotified))
		return b.ForEach(func(k, v []byte) error {
			var game models.NotifiedGame
			if err := json.Unmarshal(v, &game); err != nil {
				return err
			}
			games = append(games, game)
			return nil
		})
	})

	return games, err
}

func (s *BoltStorage) DeleteNotifiedGame(league, teamKey, gameID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketNotified))
		return b.Delete([]byte(scopedKey(league, teamKey, gameID)))
	})
}

func (s *BoltStorage) CleanupOldNotifications(before time.Time) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketNotified))
		var keysToDelete [][]byte

		err := b.ForEach(func(k, v []byte) error {
			var game models.NotifiedGame
			if err := json.Unmarshal(v, &game); err != nil {
				return err
			}
			if game.GameDate.Before(before) {
				keysToDelete = append(keysToDelete, k)
			}
			return nil
		})

		if err != nil {
			return err
		}

		for _, key := range keysToDelete {
			if err := b.Delete(key); err != nil {
				return err
			}
		}

		return nil
	})
}

// --- Email Recipients (scoped per league/team) ---

func (s *BoltStorage) AddRecipientForTeam(league, teamKey string, recipient models.EmailRecipient) error {
	recipient.League = league
	recipient.TeamKey = teamKey
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		data, err := json.Marshal(recipient)
		if err != nil {
			return fmt.Errorf("marshaling recipient: %w", err)
		}
		key := scopedKey(league, teamKey, recipient.ID)
		return b.Put([]byte(key), data)
	})
}

func (s *BoltStorage) GetActiveRecipientsForTeam(league, teamKey string) ([]models.EmailRecipient, error) {
	prefix := fmt.Sprintf("%s:%s:", league, teamKey)
	var active []models.EmailRecipient

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		c := b.Cursor()
		for k, v := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, v = c.Next() {
			var r models.EmailRecipient
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			if r.IsActive {
				active = append(active, r)
			}
		}
		return nil
	})

	return active, err
}

func (s *BoltStorage) GetAllEmailRecipients() ([]models.EmailRecipient, error) {
	var recipients []models.EmailRecipient

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		return b.ForEach(func(k, v []byte) error {
			var r models.EmailRecipient
			if err := json.Unmarshal(v, &r); err != nil {
				return err
			}
			recipients = append(recipients, r)
			return nil
		})
	})

	return recipients, err
}

func (s *BoltStorage) GetEmailRecipient(id string) (*models.EmailRecipient, error) {
	var found *models.EmailRecipient

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		// Search all keys for this recipient ID
		return b.ForEach(func(k, v []byte) error {
			if strings.HasSuffix(string(k), ":"+id) {
				var r models.EmailRecipient
				if err := json.Unmarshal(v, &r); err != nil {
					return err
				}
				found = &r
			}
			return nil
		})
	})

	return found, err
}

func (s *BoltStorage) UpdateEmailRecipient(recipient models.EmailRecipient) error {
	return s.AddRecipientForTeam(recipient.League, recipient.TeamKey, recipient)
}

func (s *BoltStorage) DeleteEmailRecipient(league, teamKey, id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		return b.Delete([]byte(scopedKey(league, teamKey, id)))
	})
}

// --- Snapshots ---

func (s *BoltStorage) SaveSnapshot(snapshot models.Snapshot) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSnapshots))
		data, err := json.Marshal(snapshot)
		if err != nil {
			return fmt.Errorf("marshaling snapshot: %w", err)
		}
		key := snapshotKey(snapshot.League, snapshot.ID)
		return b.Put([]byte(key), data)
	})
}

func (s *BoltStorage) GetLatestSnapshotHash(league string) (string, error) {
	prefix := league + ":"
	var latestHash string
	var latestTime time.Time

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSnapshots))
		c := b.Cursor()
		for k, v := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, v = c.Next() {
			var snap models.Snapshot
			if err := json.Unmarshal(v, &snap); err != nil {
				return err
			}
			if snap.FetchedAt.After(latestTime) {
				latestTime = snap.FetchedAt
				latestHash = snap.Hash
			}
		}
		return nil
	})

	return latestHash, err
}

func (s *BoltStorage) GetAllSnapshots() ([]models.Snapshot, error) {
	var snapshots []models.Snapshot

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketSnapshots))
		return b.ForEach(func(k, v []byte) error {
			var snap models.Snapshot
			if err := json.Unmarshal(v, &snap); err != nil {
				return err
			}
			snapshots = append(snapshots, snap)
			return nil
		})
	})

	return snapshots, err
}

// --- Cleanup ---

// CleanupStaleData removes games, notifications, and snapshots for league/team
// combos that are no longer in the config. validTeams is a set of "league:teamKey" strings.
func (s *BoltStorage) CleanupStaleData(validTeams map[string]bool) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		deleted := 0

		for _, bucketName := range []string{bucketGames, bucketNotified, bucketRecipients} {
			b := tx.Bucket([]byte(bucketName))
			var staleKeys [][]byte

			b.ForEach(func(k, v []byte) error {
				key := string(k)
				// Keys are "league:teamKey:id" — extract first two segments
				parts := strings.SplitN(key, ":", 3)
				if len(parts) < 3 {
					return nil
				}
				scope := parts[0] + ":" + parts[1]
				if !validTeams[scope] {
					staleKeys = append(staleKeys, append([]byte(nil), k...))
				}
				return nil
			})

			for _, k := range staleKeys {
				if err := b.Delete(k); err != nil {
					return err
				}
				deleted++
			}
		}

		// Snapshots use "league:id" format — check league prefix
		validLeagues := make(map[string]bool)
		for scope := range validTeams {
			parts := strings.SplitN(scope, ":", 2)
			validLeagues[parts[0]] = true
		}

		snapBucket := tx.Bucket([]byte(bucketSnapshots))
		var staleSnapKeys [][]byte
		snapBucket.ForEach(func(k, v []byte) error {
			key := string(k)
			parts := strings.SplitN(key, ":", 2)
			if len(parts) < 2 {
				return nil
			}
			if !validLeagues[parts[0]] {
				staleSnapKeys = append(staleSnapKeys, append([]byte(nil), k...))
			}
			return nil
		})
		for _, k := range staleSnapKeys {
			if err := snapBucket.Delete(k); err != nil {
				return err
			}
			deleted++
		}

		if deleted > 0 {
			log.Printf("Cleaned up %d stale records from DB", deleted)
		}

		return nil
	})
}

// --- Data Migration ---

// MigrateToScoped migrates existing flat-keyed data to scoped keys.
// Call on startup. It's idempotent - skips if already migrated.
func (s *BoltStorage) MigrateToScoped(defaultLeague, defaultTeamKey string) error {
	migrated := false

	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketMeta))
		if b.Get([]byte("migrated_to_scoped")) != nil {
			migrated = true
		}
		return nil
	})
	if err != nil {
		return err
	}
	if migrated {
		return nil
	}

	log.Println("Migrating existing data to scoped keys...")

	err = s.db.Update(func(tx *bolt.Tx) error {
		// Migrate games
		if err := migrateBucket(tx, bucketGames, defaultLeague, defaultTeamKey); err != nil {
			return fmt.Errorf("migrating games: %w", err)
		}

		// Migrate notified
		if err := migrateBucket(tx, bucketNotified, defaultLeague, defaultTeamKey); err != nil {
			return fmt.Errorf("migrating notified: %w", err)
		}

		// Migrate recipients
		if err := migrateBucket(tx, bucketRecipients, defaultLeague, defaultTeamKey); err != nil {
			return fmt.Errorf("migrating recipients: %w", err)
		}

		// Migrate snapshots (use league:id format)
		snapBucket := tx.Bucket([]byte(bucketSnapshots))
		var snapKeys [][]byte
		var snapValues [][]byte
		snapBucket.ForEach(func(k, v []byte) error {
			key := string(k)
			if !strings.Contains(key, ":") {
				snapKeys = append(snapKeys, append([]byte(nil), k...))
				snapValues = append(snapValues, append([]byte(nil), v...))
			}
			return nil
		})
		for i, k := range snapKeys {
			newKey := fmt.Sprintf("%s:%s", defaultLeague, string(k))
			if err := snapBucket.Put([]byte(newKey), snapValues[i]); err != nil {
				return err
			}
			if err := snapBucket.Delete(k); err != nil {
				return err
			}
		}

		// Mark migration complete
		meta := tx.Bucket([]byte(bucketMeta))
		return meta.Put([]byte("migrated_to_scoped"), []byte(time.Now().Format(time.RFC3339)))
	})

	if err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("Migration to scoped keys complete")
	return nil
}

func migrateBucket(tx *bolt.Tx, bucketName, league, teamKey string) error {
	b := tx.Bucket([]byte(bucketName))

	var oldKeys [][]byte
	var oldValues [][]byte

	b.ForEach(func(k, v []byte) error {
		key := string(k)
		// Only migrate keys that don't already have scoped format
		if !strings.Contains(key, ":") {
			oldKeys = append(oldKeys, append([]byte(nil), k...))
			oldValues = append(oldValues, append([]byte(nil), v...))
		}
		return nil
	})

	for i, k := range oldKeys {
		newKey := scopedKey(league, teamKey, string(k))
		if err := b.Put([]byte(newKey), oldValues[i]); err != nil {
			return err
		}
		if err := b.Delete(k); err != nil {
			return err
		}
	}

	return nil
}
