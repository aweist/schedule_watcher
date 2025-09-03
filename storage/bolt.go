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