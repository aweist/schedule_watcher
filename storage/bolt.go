package storage

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/aweist/schedule-watcher/models"
	bolt "go.etcd.io/bbolt"
)

const (
	bucketGames = "games"
	bucketNotified = "notified"
	bucketRecipients = "recipients"
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
		_, err := tx.CreateBucketIfNotExists([]byte(bucketGames))
		if err != nil {
			return fmt.Errorf("creating games bucket: %w", err)
		}
		
		_, err = tx.CreateBucketIfNotExists([]byte(bucketNotified))
		if err != nil {
			return fmt.Errorf("creating notified bucket: %w", err)
		}
		
		_, err = tx.CreateBucketIfNotExists([]byte(bucketRecipients))
		if err != nil {
			return fmt.Errorf("creating recipients bucket: %w", err)
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

func (s *BoltStorage) SaveGame(game models.Game) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketGames))
		
		data, err := json.Marshal(game)
		if err != nil {
			return fmt.Errorf("marshaling game: %w", err)
		}
		
		return b.Put([]byte(game.ID), data)
	})
}

func (s *BoltStorage) GetGame(gameID string) (*models.Game, error) {
	var game models.Game
	
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketGames))
		data := b.Get([]byte(gameID))
		
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

func (s *BoltStorage) DeleteGame(gameID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketGames))
		return b.Delete([]byte(gameID))
	})
}

func (s *BoltStorage) DeleteNotifiedGame(gameID string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketNotified))
		return b.Delete([]byte(gameID))
	})
}

func (s *BoltStorage) IsGameNotified(gameID string) (bool, error) {
	var exists bool
	
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketNotified))
		data := b.Get([]byte(gameID))
		exists = data != nil
		return nil
	})
	
	return exists, err
}

func (s *BoltStorage) MarkGameNotified(game models.Game) error {
	notified := models.NotifiedGame{
		GameID:      game.ID,
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
		
		return b.Put([]byte(game.ID), data)
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

func (s *BoltStorage) AddEmailRecipient(recipient models.EmailRecipient) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		
		data, err := json.Marshal(recipient)
		if err != nil {
			return fmt.Errorf("marshaling recipient: %w", err)
		}
		
		return b.Put([]byte(recipient.ID), data)
	})
}

func (s *BoltStorage) GetEmailRecipient(id string) (*models.EmailRecipient, error) {
	var recipient models.EmailRecipient
	
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		data := b.Get([]byte(id))
		
		if data == nil {
			return nil
		}
		
		return json.Unmarshal(data, &recipient)
	})
	
	if err != nil {
		return nil, err
	}
	
	if recipient.ID == "" {
		return nil, nil
	}
	
	return &recipient, nil
}

func (s *BoltStorage) GetAllEmailRecipients() ([]models.EmailRecipient, error) {
	var recipients []models.EmailRecipient
	
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		
		return b.ForEach(func(k, v []byte) error {
			var recipient models.EmailRecipient
			if err := json.Unmarshal(v, &recipient); err != nil {
				return err
			}
			recipients = append(recipients, recipient)
			return nil
		})
	})
	
	return recipients, err
}

func (s *BoltStorage) GetActiveEmailRecipients() ([]models.EmailRecipient, error) {
	all, err := s.GetAllEmailRecipients()
	if err != nil {
		return nil, err
	}
	
	var active []models.EmailRecipient
	for _, r := range all {
		if r.IsActive {
			active = append(active, r)
		}
	}
	
	return active, nil
}

func (s *BoltStorage) DeleteEmailRecipient(id string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bucketRecipients))
		return b.Delete([]byte(id))
	})
}

func (s *BoltStorage) UpdateEmailRecipient(recipient models.EmailRecipient) error {
	return s.AddEmailRecipient(recipient)
}