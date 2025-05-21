package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"go.etcd.io/bbolt"
	berrors "go.etcd.io/bbolt/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	dbFile = "uptime.db"
)

var (
	db                *bbolt.DB
	errPath           = errors.New("invalid path")
	errKeyExists      = errors.New("key exists")
	errNoKey          = errors.New("no such key")
	errUser           = errors.New("user exists")
	errNoUser         = errors.New("no such user")
	errNotImplemented = errors.New("not implemented")
)

func openDB() error {
	var err error
	xdg, ok := os.LookupEnv("XDG_DATA_HOME")
	if !ok {
		home, _ := os.UserHomeDir()
		xdg = filepath.Join(home, ".local/share/uptime")
	}
	file := filepath.Join(xdg, dbFile)
	db, err = bbolt.Open(file, 0o666, &bbolt.Options{Timeout: time.Second})
	if err != nil {
		return err
	}
	log.Println("loaded db file", file)
	return nil
}

func createBucket(path []string, tx *bbolt.Tx) *bbolt.Bucket {
	if path == nil {
		return nil
	}
	bucket, err := tx.CreateBucketIfNotExists([]byte(path[0]))
	if err != nil {
		return nil
	}
	for _, name := range path[1:] {
		bucket, err = bucket.CreateBucketIfNotExists([]byte(name))
		if err != nil {
			return nil
		}
	}
	return bucket
}

func getBucket(path []string, tx *bbolt.Tx) *bbolt.Bucket {
	bucket := tx.Bucket([]byte(path[0]))
	for _, name := range path[1:] {
		bucket = bucket.Bucket([]byte(name))
	}
	return bucket
}

func getKey(path []string, tx *bbolt.Tx) []byte {
	bucket := getBucket(path[:len(path)-1], tx)
	if bucket == nil {
		return []byte{}
	}
	key := bucket.Get([]byte(path[len(path)-1]))
	return key
}

func keyExists(path []string, tx *bbolt.Tx) bool {
	key := getKey(path, tx)
	return key != nil
}

func addKey(name string, path []string, value []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := createBucket(path, tx)
		return bucket.Put([]byte(name), value)
	})
}

func getKeys(path []string) ([]Status, error) {
	allStatus := []Status{}
	status := Status{}
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket(path, tx)
		if err := bucket.ForEach(func(k, v []byte) error {
			if err := json.Unmarshal(v, &status); err != nil {
				return err
			}
			allStatus = append(allStatus, status)
			return nil
		}); err != nil {
			return err
		}
		return nil
	})
	return allStatus, err
}

func getHistory(path []string, frame TimeFrame) ([]Status, error) {
	stats := []Status{}
	status := Status{}
	max := []byte(time.Now().Format(time.RFC3339))
	min := []byte{}
	switch frame {
	case Hour:
		min = []byte(time.Now().Add(-time.Hour).Format(time.RFC3339))
	case Day:
		min = []byte(time.Now().Add(-time.Hour * 24).Format(time.RFC3339))
	case Month:
		min = []byte(time.Now().Add(-time.Hour * 24 * 30).Format(time.RFC3339))
	case Year:
		min = []byte(time.Now().Add(-time.Hour * 24 * 365).Format(time.RFC3339))
	case Week:
		min = []byte(time.Now().Add(-time.Hour * 24 * 7).Format(time.RFC3339))
	}
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket(path, tx)
		if bucket == nil {
			return errPath
		}
		c := bucket.Cursor()
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

func getMonitors() ([]Monitor, error) {
	monitors := []Monitor{}
	monitor := Monitor{}
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket([]string{"monitors"}, tx)
		return bucket.ForEach(func(k, v []byte) error {
			if err := json.Unmarshal(v, &monitor); err != nil {
				return err
			}
			monitors = append(monitors, monitor)
			return nil
		})
	})
	return monitors, err
}

func getMonitor(name string) (Monitor, error) {
	monitor := Monitor{}
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket([]string{"monitors"}, tx)
		value := bucket.Get([]byte(name))
		if err := json.Unmarshal(value, &monitor); err != nil {
			return err
		}
		return nil
	})
	return monitor, err
}

func saveMonitor(monitor Monitor, update bool) error {
	bytes, err := json.Marshal(monitor)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		keyExists := keyExists([]string{"monitors", monitor.Name}, tx)
		if keyExists && !update {
			return errKeyExists
		}
		if !keyExists && update {
			return errNoKey
		}
		bucket := tx.Bucket([]byte("monitors"))
		if err := bucket.Put([]byte(monitor.Name), bytes); err != nil {
			return err
		}
		return nil
	})
}

func removeMonitor(name string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("monitors"))
		return bucket.Delete([]byte(name))
	})
}

func deleteHistory(name string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		if err := bucket.DeleteBucket([]byte(name)); err != nil {
			return err
		}
		bucket = tx.Bucket([]byte("status"))
		return bucket.Delete([]byte(name))
	})
}

func validateUser(user User) bool {
	var truth User
	if err := db.View(func(tx *bbolt.Tx) error {
		bytes := getKey([]string{"users", user.Name}, tx)
		if err := json.Unmarshal(bytes, &truth); err != nil {
			return err
		}
		return nil
	}); err != nil {
		log.Println("validate user", err)
		return false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(truth.Pass), []byte(user.Pass)); err != nil {
		log.Println("invalid pass", err)
		return false
	}
	return true
}

func setPass(user User) (User, error) {
	pass, err := bcrypt.GenerateFromPassword([]byte(user.Pass), bcrypt.DefaultCost)
	if err != nil {
		log.Println("genertat password", err)
		return user, err
	}
	user.Pass = string(pass)
	return user, nil
}

func checkAdmin(name string) bool {
	var user User
	if err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		value := bucket.Get([]byte(name))
		if err := json.Unmarshal(value, &user); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return false
	}
	return user.Admin
}

func getUsers() []User {
	var user User
	var users []User
	if err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		return bucket.ForEach(func(k, v []byte) error {
			if err := json.Unmarshal(v, &user); err != nil {
				return err
			}
			user.Name = string(k)
			users = append(users, user)
			return nil
		})
	}); err != nil {
		return []User{}
	}
	return users
}

func getUser(name string) User {
	var user User
	if err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		v := bucket.Get([]byte(name))
		return json.Unmarshal(v, &user)
	}); err != nil {
		return User{}
	}
	return user
}

func insertUser(user User) error {
	var err error
	user, err = setPass(user)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		v := bucket.Get([]byte(user.Name))
		if v != nil {
			return errUser
		}
		bytes, err := json.Marshal(&user)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(user.Name), bytes)
	})
}

func modifyUser(user User) error {
	var err error
	user, err = setPass(user)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		v := bucket.Get([]byte(user.Name))
		if v == nil {
			return errNoUser
		}
		bytes, err := json.Marshal(&user)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(user.Name), bytes)
	})
}

func removeUser(name string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("users"))
		v := bucket.Get([]byte(name))
		if v == nil {
			return errNoUser
		}
		return bucket.Delete([]byte(name))
	})
}

func getSlack(bucket *bbolt.Bucket, name []byte) (SlackNotifier, error) {
	log.Println("get slack notification", string(name))
	var slack SlackNotifier
	notifyType := bucket.Get([]byte("type"))
	if notifyType == nil {
		log.Println("type key not found")
		return slack, errNoKey
	}
	if notifyType[0] != 0 {
		log.Println("wrong notification type")
		return slack, errors.New("wrong notification type")
	}
	key := bucket.Get([]byte("data"))
	if key == nil {
		log.Println("no data")
		return slack, errNoKey
	}
	err := json.Unmarshal(key, &slack)
	return slack, err
}

func createSlack(notifyBucket *bbolt.Bucket, slack SlackNotifier) error {
	bytes, err := json.Marshal(slack)
	if err != nil {
		return err
	}
	bucket, err := notifyBucket.CreateBucket([]byte(slack.Name))
	if err != nil {
		return err
	}
	if err := bucket.Put([]byte("type"), NotifyTypeBytes[Slack]); err != nil {
		return err
	}
	return bucket.Put([]byte("data"), bytes)
}

func updateSlack(notifyBucket *bbolt.Bucket, slack SlackNotifier) error {
	log.Println("update slack notification", slack)
	bytes, err := json.Marshal(slack)
	if err != nil {
		return err
	}
	bucket := notifyBucket.Bucket([]byte(slack.Name))
	if bucket == nil {
		return fmt.Errorf("%w %s", berrors.ErrBucketNotFound, slack.Name)
	}
	return bucket.Put([]byte("data"), bytes)
}

func removeNotify(name string) error {
	monitor := Monitor{}
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("notify"))
		if bucket == nil {
			return fmt.Errorf("%w nofify", berrors.ErrBucketNotFound)
		}
		if err := bucket.DeleteBucket([]byte(name)); err != nil {
			return fmt.Errorf("delete bucket %s %w", name, err)
		}
		bucket = tx.Bucket([]byte("monitors"))
		err := bucket.ForEach(func(k, v []byte) error {
			if err := json.Unmarshal(v, &monitor); err != nil {
				return fmt.Errorf("unmarshal monitor %s %w", string(k), err)
			}
			monitor.Notifiers = slices.DeleteFunc(monitor.Notifiers, func(n string) bool {
				return n == name
			})
			bytes, err := json.Marshal(monitor)
			if err != nil {
				return fmt.Errorf("marshal monitor %s %w", monitor.Name, err)
			}
			return bucket.Put(k, bytes)
		})
		return err
	})
}

func createNotify(notifyType NotifyType, data any) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("notify"))
		if bucket == nil {
			return berrors.ErrBucketNotFound
		}
		switch notifyType {
		case Slack:
			return createSlack(bucket, data.(SlackNotifier))
		default:
			return errNotImplemented
		}
	})
}

func updateNotify(notifyType NotifyType, data any) error {
	log.Println("update notification", NotifyTypeNames[notifyType], data)
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("notify"))
		if bucket == nil {
			return fmt.Errorf("%w notify", berrors.ErrBucketNotFound)
		}
		switch notifyType {
		case Slack:
			return updateSlack(bucket, data.(SlackNotifier))
		default:
			return errNotImplemented
		}
	})
}

func getNotify(name string) (NotifyType, []byte, error) {
	var notifyType NotifyType
	var data []byte
	err := db.View(func(tx *bbolt.Tx) error {
		notifications := tx.Bucket([]byte("notify"))
		if notifications == nil {
			return berrors.ErrBucketNotFound
		}
		notify := notifications.Bucket([]byte(name))
		if notify == nil {
			return berrors.ErrBucketNotFound
		}
		notifyType = NotifyType(notify.Get([]byte("type"))[0])
		data = notify.Get([]byte("data"))
		return nil
	})
	return notifyType, data, err
}

func getAllNotifications() []Notification {
	var notifications []Notification
	var notification Notification
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("notify"))
		if bucket == nil {
			return berrors.ErrBucketNotFound
		}
		return bucket.ForEach(func(k, v []byte) error {
			var err error
			notification.Name = string(k)
			notifyBucket := bucket.Bucket(k)
			if notifyBucket == nil {
				return berrors.ErrBucketNotFound
			}
			notification.Type = NotifyType(notifyBucket.Get([]byte("type"))[0])
			switch notification.Type {
			case Slack:
				notification.Notification, err = getSlack(notifyBucket, k)
				if err != nil {
					return err
				}
			default:
				return errNotImplemented
			}
			notifications = append(notifications, notification)
			return nil
		})
	})
	if err != nil {
		log.Println("get notifications", err)
		return []Notification{}
	}
	return notifications
}
