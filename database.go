package uptime

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"go.etcd.io/bbolt"
	berrors "go.etcd.io/bbolt/errors"
)

var (
	//DB     *bbolt.DB
	old    string
	Tables = []Table{
		{Name: "status", Path: []string{"status"}},
	}
)

type Table struct {
	Name string
	Path []string
}

func OpenDB() (*bbolt.DB, error) {
	var db *bbolt.DB
	var success bool
	var err error
	config := GetConfig()
	if config == nil {
		log.Println("no config ... bailing")
		return nil, fmt.Errorf("no configuration")
	}
	xdg, ok := os.LookupEnv("XDG_DATA_HOME")
	if !ok {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".local/share/uptime")
	}
	file := filepath.Join(xdg, config.DBFile)
	for range 5 {
		db, err = bbolt.Open(file, 0666, &bbolt.Options{Timeout: time.Second})
		if err != nil {
			if errors.Is(err, berrors.ErrTimeout) {
				time.Sleep(time.Second)
				log.Println("database timeout...trying again")
				continue
			} else {
				break
			}
		} else {
			success = true
			break
		}
	}
	if !success {
		return nil, err
	}

	//old = file
	log.Println("loaded db file", file)
	//if err := createTables(Tables); err != nil {
	//return fmt.Errorf("create tables %w", err)
	//}
	return db, nil
}

func createBucket(path []string, tx *bbolt.Tx) *bbolt.Bucket {
	if path == nil {
		return nil
	}
	bucket, err := tx.CreateBucketIfNotExists([]byte(path[0]))
	if err != nil {
		log.Println("create root bucket", path[0], err)
		return nil
	}
	for _, name := range path[1:] {
		bucket, err = bucket.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			log.Println("create nested bucket", name, err)
			return nil
		}
	}
	return bucket
}

func getBucket(path []string, tx *bbolt.Tx) *bbolt.Bucket {
	log.Println("get root bucket", path[0])
	bucket := tx.Bucket([]byte(path[0]))
	for _, name := range path[1:] {
		bucket = bucket.Bucket([]byte(name))
	}
	return bucket
}

func AddKey(db *bbolt.DB, name string, path []string, value []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := createBucket(path, tx)
		return bucket.Put([]byte(name), value)
	})
}

func GetKeys(db *bbolt.DB, path []string) ([]Status, error) {
	allStatus := []Status{}
	status := Status{}
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket(path, tx)
		bucket.ForEach(func(k, v []byte) error {
			if err := json.Unmarshal(v, &status); err != nil {
				return err
			}
			allStatus = append(allStatus, status)
			return nil
		})
		return nil
	})
	return allStatus, err
}

func GetHistory(db *bbolt.DB, path []string, frame TimeFrame) ([]Status, error) {
	stats := []Status{}
	status := Status{}
	max := []byte(time.Now().Format(time.RFC3339))
	min := []byte{}
	switch frame {
	case Hour:
		min = []byte(time.Now().Add(-time.Hour).Format(time.RFC3339))
	case Day:
		min = []byte(time.Now().Add(-time.Hour * 25).Format(time.RFC3339))
	case Month:
		min = []byte(time.Now().Add(-time.Hour * 24 * 30).Format(time.RFC3339))
	case Year:
		min = []byte(time.Now().Add(-time.Hour * 24 * 365).Format(time.RFC3339))
	}

	log.Println("get history", path, frame.Name(), string(max), string(min))

	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket(path, tx)
		if bucket == nil {
			return errors.New("invalid path")
		}
		c := bucket.Cursor()
		k, _ := c.Seek(min)
		log.Println("first key", string(k))
		for k, v := c.Seek(min); k != nil && bytes.Compare(k, max) <= 0; k, v = c.Next() {
			if err := json.Unmarshal(v, &status); err != nil {
				return err
			}
			stats = append(stats, status)
		}
		return nil
	})
	return stats, err
}

func GetMonitors(db *bbolt.DB) ([]Monitor, error) {
	monitors := []Monitor{}
	monitor := Monitor{}
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket([]string{"monitors"}, tx)
		bucket.ForEach(func(k, v []byte) error {
			if err := json.Unmarshal(v, &monitor); err != nil {
				return err
			}
			monitors = append(monitors, monitor)
			return nil
		})
		return nil
	})
	return monitors, err
}
