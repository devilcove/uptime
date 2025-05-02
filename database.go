package main

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
	"golang.org/x/crypto/bcrypt"
)

var db *bbolt.DB

func OpenDB() error {
	var success bool
	var err error
	config := GetConfig()
	if config == nil {
		log.Println("no config ... bailing")
		return fmt.Errorf("no configuration")
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
		log.Println("get nested bucket", name)
		bucket = bucket.Bucket([]byte(name))
	}
	return bucket
}

func getKey(path []string, tx *bbolt.Tx) []byte {
	log.Println("get bucket", path[:len(path)-1])
	bucket := getBucket(path[:len(path)-1], tx)
	if bucket == nil {
		log.Println("nil bucket")
		return []byte{}
	}
	log.Println("checking if key exists", path, path[len(path)-1])
	key := bucket.Get([]byte(path[len(path)-1]))
	return key
}

func keyExists(path []string, tx *bbolt.Tx) bool {
	key := getKey(path, tx)
	return key != nil

}

func AddKey(name string, path []string, value []byte) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := createBucket(path, tx)
		return bucket.Put([]byte(name), value)
	})
}

func GetKeys(path []string) ([]Status, error) {
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

func GetHistory(path []string, frame TimeFrame) ([]Status, error) {
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

	log.Println("get history", path, frame.Name(),
		"\nfrom:", string(min), "\nto  :", string(max))

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
			log.Println("add to history", status.Time)
			stats = append(stats, status)
		}
		return nil
	})
	return stats, err
}

func GetMonitors() ([]Monitor, error) {
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

func GetMonitor(name string) (Monitor, error) {
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

func SaveMonitor(monitor Monitor, update bool) error {
	bytes, err := json.Marshal(monitor)
	if err != nil {
		return err
	}
	return db.Update(func(tx *bbolt.Tx) error {
		log.Println("checking if monitor", monitor.Name, "exists")
		keyExists := keyExists([]string{"monitors", monitor.Name}, tx)
		if keyExists && !update {
			return errors.New("key exists")
		}
		if !keyExists && update {
			return errors.New("no suck key")
		}
		bucket := tx.Bucket([]byte("monitors"))
		if err := bucket.Put([]byte(monitor.Name), bytes); err != nil {
			return err
		}
		return nil
	})
}

func DeleteMonitor(name string) error {
	return db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("monitors"))
		return bucket.Delete([]byte(name))
	})
}

func DeleteHistory(name string) error {
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
			return errors.New("user exists")
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
			return errors.New("no such user")
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
			return errors.New("user does not exist")
		}
		return bucket.Delete([]byte(name))
	})
}
