package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/devilcove/cookie"
)

func web(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	if err := cookie.New(cookieName, cookieAge); err != nil {
		log.Println("new cookie", err)
	}
	log.Println("starting web server")

	router := NewRouter(Logger)

	router.Get("/favicon.ico", favicon)
	router.Get("/logout", logout)
	router.Get("/login", displayLogin)
	router.Post("/login", login)
	router.Get("/styles.css", styles)
	router.Get("/{$}", mainPage)

	plain := router.Group("", auth)
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
