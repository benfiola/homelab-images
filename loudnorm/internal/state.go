package internal

import (
	"strings"

	"go.etcd.io/bbolt"
)

var markersBucket = []byte("markers")

// Store is a durable path -> marker map, backed by a single bbolt file.
type Store struct {
	db *bbolt.DB
}

func OpenStore(path string) (*Store, error) {
	db, err := bbolt.Open(path, 0o600, nil)
	if err != nil {
		return nil, err
	}
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(markersBucket)
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) GetMarker(file string) (string, error) {
	var marker string
	err := s.db.View(func(tx *bbolt.Tx) error {
		if v := tx.Bucket(markersBucket).Get([]byte(file)); v != nil {
			marker = string(v)
		}
		return nil
	})
	return marker, err
}

func (s *Store) SetMarker(file, marker string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(markersBucket).Put([]byte(file), []byte(marker))
	})
}

func (s *Store) DeleteMarker(file string) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return tx.Bucket(markersBucket).Delete([]byte(file))
	})
}

// DeletePrefix removes every marker whose path falls under dir, returning
// the removed paths.
func (s *Store) DeletePrefix(dir string) ([]string, error) {
	prefix := strings.TrimSuffix(dir, "/") + "/"
	var matched []string

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(markersBucket)
		c := b.Cursor()
		for k, _ := c.Seek([]byte(prefix)); k != nil && strings.HasPrefix(string(k), prefix); k, _ = c.Next() {
			matched = append(matched, string(k))
		}
		for _, k := range matched {
			if err := b.Delete([]byte(k)); err != nil {
				return err
			}
		}
		return nil
	})
	return matched, err
}

// PruneOrphans removes every marker whose path isn't in known, returning
// the removed paths.
func (s *Store) PruneOrphans(known map[string]bool) ([]string, error) {
	var stale []string

	err := s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(markersBucket)
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if !known[string(k)] {
				stale = append(stale, string(k))
			}
		}
		for _, k := range stale {
			if err := b.Delete([]byte(k)); err != nil {
				return err
			}
		}
		return nil
	})
	return stale, err
}
