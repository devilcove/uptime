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

// openDB Opens, creates if non-existent, db file in XDG_DATA_HOME/uptime.db.
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

// createBucket creates and return bucket at given path, intermediate buckets along path are also created.
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

// getBucket returns bucket at given path.
func getBucket(path []string, tx *bbolt.Tx) *bbolt.Bucket {
	bucket := tx.Bucket([]byte(path[0]))
	for _, name := range path[1:] {
		bucket = bucket.Bucket([]byte(name))
	}
	return bucket
}

// getKey return value of key at given path.
func getKey(path []string, tx *bbolt.Tx) []byte {
	bucket := getBucket(path[:len(path)-1], tx)
	if bucket == nil {
		return []byte{}
	}
	value := bucket.Get([]byte(path[len(path)-1]))
	return value
}

// keyExists returns true if key exists at given path.
func keyExists(path []string, tx *bbolt.Tx) bool {
	key := getKey(path, tx)
	return key != nil
}

// addKey creates a new key with name, value in bucket at path, bucket and intermediate buckets
// will be created if not existing.
func addKey(name string, path []string, value []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := createBucket(path, tx)
		return bucket.Put([]byte(name), value)
	})
}

// getStatus return the name status.
func getStatus(name string) (Status, error) {
	status := Status{}
	path := []string{"status", name}
	err := db.View(func(tx *bbolt.Tx) error {
		key := getKey(path, tx)
		if err := json.Unmarshal(key, &status); err != nil {
			return err
		}
		return nil
	})
	return status, err
}

func purgeHistData(site, date string) error {
	log.Println("purge data from", site, "before", date)
	dateTime, err := time.Parse(time.DateOnly, date)
	if err != nil {
		return err
	}
	stop := []byte(dateTime.Format(time.RFC3339))
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		history := bucket.Bucket([]byte(site))
		c := history.Cursor()
		for k, _ := c.First(); k != nil && bytes.Compare(k, stop) <= 0; k, _ = c.Next() {
			if err := history.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

// getTime returns RFC3339 time for given timeframe.
func getTime(frame TimeFrame) []byte {
	var duration []byte
	switch frame {
	case hour:
		duration = []byte(time.Now().Add(-time.Hour).Format(time.RFC3339))
	case day:
		duration = []byte(time.Now().Add(-time.Hour * 24).Format(time.RFC3339))
	case month:
		duration = []byte(time.Now().Add(-time.Hour * 24 * 30).Format(time.RFC3339))
	case year:
		duration = []byte(time.Now().Add(-time.Hour * 24 * 365).Format(time.RFC3339))
	case week:
		duration = []byte(time.Now().Add(-time.Hour * 24 * 7).Format(time.RFC3339))
	case all:
		duration = []byte(time.Time{}.Format(time.RFC3339))
	}
	return duration
}

// getHistory returns array of status values from the given path for the given timeframe.
func getHistory(path []string, frame TimeFrame) ([]Status, error) {
	stats := []Status{}
	status := Status{}
	end := []byte(time.Now().Format(time.RFC3339))
	start := getTime(frame)
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket(path, tx)
		if bucket == nil {
			return errPath
		}
		c := bucket.Cursor()
		for k, v := c.Seek(start); k != nil && bytes.Compare(k, end) <= 0; k, v = c.Next() {
			if err := json.Unmarshal(v, &status); err != nil {
				return err
			}
			stats = append(stats, status)
		}
		return nil
	})
	slices.Reverse(stats)
	return stats, err
}

func getHistoryDetails(monitor string, ok int) (Details, error) {
	var details Details
	var err error
	details.Status, err = getHistory([]string{"history", monitor}, all)
	if err != nil {
		return details, err
	}
	details.Response24, details.Uptime24, err = getStats(monitor, day, ok)
	if err != nil {
		return details, err
	}
	details.Response30, details.Uptime30, err = getStats(monitor, month, ok)
	return details, err
}

func getStats(monitor string, timeFrame TimeFrame, ok int) (int, float64, error) {
	var good, total float64
	var status Status
	var responseTime int
	first := getTime(timeFrame)
	now := []byte(time.Now().Format(time.RFC3339))
	if err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("history"))
		history := bucket.Bucket([]byte(monitor))
		c := history.Cursor()
		for k, v := c.Seek(first); k != nil && bytes.Compare(k, now) <= 0; k, v = c.Next() {
			if err := json.Unmarshal(v, &status); err != nil {
				return err
			}
			total++
			if status.StatusCode == ok {
				good++
			}
			responseTime += int(status.ResponseTime.Milliseconds())
		}
		return nil
	}); err != nil {
		return 0, 0, err
	}
	if total == 0 {
		return 0, 0, nil
	}
	return responseTime / int(total), good / total * 100, nil
}

// getMonitors returns array of all Monitor structs.
func getMonitors() ([]Monitor, error) {
	monitors := []Monitor{}
	monitor := Monitor{}
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := getBucket([]string{"monitors"}, tx)
		return bucket.ForEach(func(_, v []byte) error {
			if err := json.Unmarshal(v, &monitor); err != nil {
				return err
			}
			monitors = append(monitors, monitor)
			return nil
		})
	})
	return monitors, err
}

// getMonitor returns monitor struct of given name.
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

// saveMonitor save Monitor in database.
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

// removeMonitor deletes the named monitor from database.
func removeMonitor(name string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("monitors"))
		return bucket.Delete([]byte(name))
	})
}

// deleteHistory deletes named history bucket.
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

// validateUser confirms provide username/password matches username/password in db.
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

// setPass encrypts password.
func setPass(user User) (User, error) {
	pass, err := bcrypt.GenerateFromPassword([]byte(user.Pass), bcrypt.DefaultCost)
	if err != nil {
		log.Println("genertat password", err)
		return user, err
	}
	user.Pass = string(pass)
	return user, nil
}

// checkAdmin checks if given username is an admin.
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

// getUsers returns array of all users in db; password is nulled.
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
			user.Pass = ""
			users = append(users, user)
			return nil
		})
	}); err != nil {
		return []User{}
	}
	return users
}

// getUers returns user struct for named user.
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

// insertUser creates a new user in db.
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

// modifyUser updates a user in db.
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

// removeUser deletes a user from db.
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

// removeNotify deletes the name notification bucket.
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
		err := bucket.ForEach(func(key, v []byte) error {
			if err := json.Unmarshal(v, &monitor); err != nil {
				return fmt.Errorf("unmarshal monitor %s %w", string(key), err)
			}
			monitor.Notifiers = slices.DeleteFunc(monitor.Notifiers, func(n string) bool {
				return n == name
			})
			bytes, err := json.Marshal(monitor)
			if err != nil {
				return fmt.Errorf("marshal monitor %s %w", monitor.Name, err)
			}
			return bucket.Put(key, bytes)
		})
		return err
	})
}

// createNofify inserts a new nofification bucket into database.
func createNotify(name string, notifyType NotifyType, data any) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("notify"))
		if bucket == nil {
			return berrors.ErrBucketNotFound
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		bucket, err = bucket.CreateBucket([]byte(name))
		if err != nil {
			return err
		}
		if err := bucket.Put([]byte("type"), []byte(notifyType)); err != nil {
			return err
		}
		return bucket.Put([]byte("data"), bytes)
	})
}

// updateNotify updates an existing notification bucket.
func updateNotify(name string, notifyType NotifyType, data any) error {
	log.Println("update notification", name, notifyType, data)
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("notify"))
		if bucket == nil {
			return fmt.Errorf("%w notify", berrors.ErrBucketNotFound)
		}
		bytes, err := json.Marshal(data)
		if err != nil {
			return err
		}
		bucket = bucket.Bucket([]byte(name))
		if bucket == nil {
			return fmt.Errorf("%w %s", berrors.ErrBucketNotFound, name)
		}
		return bucket.Put([]byte("data"), bytes)
	})
}

// getNotify retrieves named notification data.
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
		notifyType = NotifyType(notify.Get([]byte("type")))
		data = notify.Get([]byte("data"))
		return nil
	})
	return notifyType, data, err
}

// getAllNotifications retrieves array of Notification data from db.
func getAllNotifications() []Notification {
	var notifications []Notification
	var notification Notification
	err := db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("notify"))
		if bucket == nil {
			return berrors.ErrBucketNotFound
		}
		return bucket.ForEach(func(k, _ []byte) error {
			notification.Name = string(k)
			notifyBucket := bucket.Bucket(k)
			if notifyBucket == nil {
				return berrors.ErrBucketNotFound
			}
			notification.Type = NotifyType(notifyBucket.Get([]byte("type")))
			dataValue := notifyBucket.Get([]byte("data"))
			if dataValue == nil {
				log.Println("no data for notification", notification.Name)
				return errNoKey
			}
			if err := json.Unmarshal(dataValue, &notification.Notification); err != nil {
				log.Println("unmarshal notification data", err)
				return err
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

func getAllMonitorsForDisplay() []MonitorDisplay {
	display := []MonitorDisplay{}
	monitors, _ := getMonitors()
	for _, monitor := range monitors {
		disp := MonitorDisplay{
			Name:   monitor.Name,
			Active: monitor.Active,
		}
		disp.Status, _ = getStatus(monitor.Name)
		history, _ := getHistory([]string{"history", monitor.Name}, day)
		var total, good float64
		for i, hist := range history {
			if i == 0 {
				if history[i].StatusCode == monitor.StatusOK {
					disp.DisplayStatus = true
				}
			}
			total++
			if hist.StatusCode == monitor.StatusOK {
				good++
			}
			disp.PerCent = good / total * 100
		}
		display = append(display, disp)
	}
	return display
}
