package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.etcd.io/bbolt"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/term"
)

type User struct {
	Name  string
	Pass  string
	Admin bool
}

var (
	db      *bbolt.DB
	errUser = errors.New("user exists")
)

const dbFile = "uptime.db"

func main() {
	if err := openDB(); err != nil {
		panic(err)
	}
	user := User{Admin: true}
	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter username")
	name, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}
	user.Name = strings.TrimSuffix(name, "\n")
	fmt.Println("Enter password")
	passBytes, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(err)
	}
	user.Pass = string(passBytes)
	if err := insertUser(user); err != nil {
		panic(err)
	}
	fmt.Println("new user created")
}

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
