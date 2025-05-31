package main

import (
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/sessions"
)

const (
	sessionBytes = 32
	cookieAge    = 300
)

var store *sessions.CookieStore //nolint:gochecknoglobals

func web(ctx context.Context, wg *sync.WaitGroup) { //nolint:funlen
	defer wg.Done()
	store = sessions.NewCookieStore(randBytes(sessionBytes))
	store.MaxAge(cookieAge)
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteStrictMode
	log.Println("starting web server")

	logger := NewMiddleware(http.DefaultServeMux)

	logger.Use(Logger)
	http.HandleFunc("/favicon.ico", favicon)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("GET /login", displayLogin)
	http.HandleFunc("POST /login", login)
	http.HandleFunc("/styles.css", styles)

	plain := Router("", auth)
	plain.HandleFunc("/{$}", mainPage)
	plain.HandleFunc("/logs", logs)

	user := Router("/user", auth)
	user.HandleFunc("GET /{$}", admin)
	user.HandleFunc("GET /{user}", editUser)
	user.HandleFunc("POST /delete/{user}", deleteUser)
	user.HandleFunc("POST /add", addUser)
	user.HandleFunc("POST /{user}", updateUser)

	monitor := Router("/monitor", auth)
	monitor.HandleFunc("GET /new", newMonitor)
	monitor.HandleFunc("POST /new", createMonitor)
	monitor.HandleFunc("GET /delete/{site}", deleteSite)
	monitor.HandleFunc("POST /delete/{site}", deleteMonitor)
	monitor.HandleFunc("GET /edit/{site}", editMonitor)
	monitor.HandleFunc("POST /edit/{site}", updateMonitor)
	monitor.HandleFunc("GET /history/{site}/{duration}", history)

	notification := Router("/notifications", auth)
	notification.HandleFunc("GET /", notifications)
	notification.HandleFunc("GET /new", newNotification)
	notification.HandleFunc("POST /new", createNotification)
	notification.HandleFunc("POST /delete/{notify}", deleletNotification)
	notification.HandleFunc("GET /edit/{notify}", displayEditnotification)
	notification.HandleFunc("POST /edit/{notify}", editNotification)
	notification.HandleFunc("GET /test/{notify}", testNotification)

	server := http.Server{Addr: httpAddr, ReadHeaderTimeout: time.Second, Handler: logger} //nolint:exhaustruct
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println("web server", err)
		}
	}()
	log.Println("web server running", httpAddr)
	<-ctx.Done()
	log.Println("shutdown web ...")
	if err := server.Shutdown(context.Background()); err != nil { //nolint:contextcheck
		log.Println("web server shutdown", err)
	}
}

func randBytes(l int) []byte {
	bytes := make([]byte, l)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return bytes
}
