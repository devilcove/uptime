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

func web(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	store = sessions.NewCookieStore(randBytes(sessionBytes))
	store.MaxAge(cookieAge)
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteStrictMode
	log.Println("starting web server")

	router := NewRouter(Logger)

	router.Get("/favicon.ico", favicon)
	router.Get("/logout", logout)
	router.Get("/login", displayLogin)
	router.Post("/login", login)
	router.Get("/styles.css", styles)

	plain := router.Group("", auth)
	plain.Get("/{$}", mainPage)
	plain.Get("/logs", logs)

	user := router.Group("/user", auth)
	user.Get("/{$}", admin)
	user.Get("/{user}", editUser)
	user.Post("/delete/{user}", deleteUser)
	user.Post("/add", addUser)
	user.Post("/{user}", updateUser)

	monitor := router.Group("/monitor", auth)
	monitor.Get("/details/{site}", details)
	monitor.Get("/pause/{site}", pauseMonitor)
	monitor.Get("/resume/{site}", resumeMonitor)
	monitor.Get("/new", newMonitor)
	monitor.Post("/new", createMonitor)
	monitor.Get("/delete/{site}", deleteSite)
	monitor.Post("/delete/{site}", deleteMonitor)
	monitor.Get("/edit/{site}", editMonitor)
	monitor.Post("/edit/{site}", updateMonitor)
	monitor.Get("/history/{site}/{duration}", history)
	monitor.Post("/history/purge/{site}", purgeHistory)

	notification := router.Group("/notifications", auth)
	notification.Get("/", notifications)
	notification.Get("/new", newNotification)
	notification.Post("/new", createNotification)
	notification.Post("/delete/{notify}", deleletNotification)
	notification.Get("/edit/{notify}", displayEditnotification)
	notification.Post("/edit/{notify}", editNotification)
	notification.Get("/test/{notify}", testNotification)

	server := http.Server{Addr: httpAddr, ReadHeaderTimeout: time.Second, Handler: router} //nolint:exhaustruct
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
