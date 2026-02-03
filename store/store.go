package store

import (
	"fmt"
	"os"
	"path/filepath"

	bolt "go.etcd.io/bbolt"
)

var bucketName = []byte("kv")

type Store struct {
	db *bolt.DB
}

func New(dataDir string) (*Store, error) {
	// Create data directory if it doesn't exist
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(dataDir, "kvstore.db")
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Create bucket if it doesn't exist
	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucketName)
		return err
	})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create bucket: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) Get(key string) (string, bool) {
	var value []byte
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		value = b.Get([]byte(key))
		return nil
	})
	if value == nil {
		return "", false
	}
	return string(value), true
}

func (s *Store) Set(key, value string) error {
	return s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		return b.Put([]byte(key), []byte(value))
	})
}

func (s *Store) Delete(key string) bool {
	var existed bool
	s.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		existing := b.Get([]byte(key))
		existed = existing != nil
		if existed {
			b.Delete([]byte(key))
		}
		return nil
	})
	return existed
}

func (s *Store) Keys() []string {
	var keys []string
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		b.ForEach(func(k, v []byte) error {
			keys = append(keys, string(k))
			return nil
		})
		return nil
	})
	return keys
}

func (s *Store) Len() int {
	var count int
	s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketName)
		count = b.Stats().KeyN
		return nil
	})
	return count
}
