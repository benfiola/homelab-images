package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
)

type Entry struct {
	Processed   bool      `json:"processed"`
	SettingsHash string    `json:"settings_hash"`
	ProcessedAt time.Time `json:"processed_at"`
}

type DB struct {
	db *bbolt.DB
}

func New(dataDir string) (*DB, error) {
	dbPath := filepath.Join(dataDir, "loudnorm.db")

	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, err
	}

	bdb, err := bbolt.Open(dbPath, 0600, nil)
	if err != nil {
		return nil, err
	}

	if err := bdb.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("files"))
		return err
	}); err != nil {
		bdb.Close()
		return nil, err
	}

	return &DB{db: bdb}, nil
}

func (d *DB) Close() error {
	return d.db.Close()
}

func (d *DB) MarkProcessed(path string, hash string) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("files"))
		entry := Entry{
			Processed:   true,
			SettingsHash: hash,
			ProcessedAt: time.Now(),
		}
		data, err := json.Marshal(entry)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(path), data)
	})
}

func (d *DB) IsProcessed(path string) bool {
	var processed bool
	d.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("files"))
		data := bucket.Get([]byte(path))
		if data != nil {
			var entry Entry
			if err := json.Unmarshal(data, &entry); err == nil {
				processed = entry.Processed
			}
		}
		return nil
	})
	return processed
}

func (d *DB) DeleteEntry(path string) error {
	return d.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("files"))
		return bucket.Delete([]byte(path))
	})
}

func (d *DB) GetAll() ([]string, error) {
	var paths []string
	err := d.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("files"))
		return bucket.ForEach(func(k, v []byte) error {
			paths = append(paths, string(k))
			return nil
		})
	})
	return paths, err
}

var ErrNotFound = errors.New("entry not found")

func (d *DB) GetSettingsHash(path string) (string, error) {
	var hash string
	err := d.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("files"))
		data := bucket.Get([]byte(path))
		if data == nil {
			return ErrNotFound
		}
		var entry Entry
		if err := json.Unmarshal(data, &entry); err != nil {
			return err
		}
		hash = entry.SettingsHash
		return nil
	})
	return hash, err
}
