package store

import (
	"encoding/json"
	"errors"
	"os"
	"path"
	"time"

	bolt "go.etcd.io/bbolt"
)

const (
	historyBucket = "history"
	metaBucket    = "meta"
)

type Store struct {
	db *bolt.DB
}

type HistoryEntry struct {
	Timestamp time.Time `json:"ts"`
	Cmd       string    `json:"cmd"`
}

func NewStore(pathStr string) (*Store, error) {
	//ensure directory exists
	dir := path.Dir(pathStr)
	if dir != "" && dir != "." {
		_ = os.MkdirAll(dir, 0755)
	}

	db, err := bolt.Open(pathStr, 0600, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		return nil, err
	}
	//create buckets
	err = db.Update(func(tx *bolt.Tx) error {
		if _, e := tx.CreateBucketIfNotExists([]byte(historyBucket)); e != nil {
			return e
		}
		if _, e := tx.CreateBucketIfNotExists([]byte(metaBucket)); e != nil {
			return e
		}
		return nil
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	if s.db == nil {
		return nil
	}
	return s.db.Close()
}

// appends a history entry (stores by timestamp key)
func (s *Store) SaveHistory(cmd string) error {
	if s.db == nil {
		return errors.New("db not opened")
	}
	entry := HistoryEntry{Timestamp: time.Now().UTC(), Cmd: cmd}
	b, _ := json.Marshal(entry)
	return s.db.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(historyBucket))
		if bk == nil {
			return errors.New("history bucket missing")
		}
		key := []byte(entry.Timestamp.Format(time.RFC3339Nano))
		return bk.Put(key, b)
	})
}

// returns up to 'limit' most recent commands (newest first)
func (s *Store) ListHistory(limit int) ([]string, error) {
	if s.db == nil {
		return nil, errors.New("db not opened")
	}
	out := []string{}
	err := s.db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(historyBucket))
		if bk == nil {
			return nil
		}
		c := bk.Cursor()
		k, v := c.Last()
		count := 0
		for ; k != nil && count < limit; k, v = c.Prev() {
			var en HistoryEntry
			if err := json.Unmarshal(v, &en); err == nil {
				out = append(out, en.Timestamp.Format("2006-01-02 15:04:05")+"  "+en.Cmd)
			}
			count++
		}
		return nil
	})
	return out, err
}
