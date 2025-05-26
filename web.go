package main

import (
	"context"
	"crypto/rand"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/devilcove/uptime/middleware"
	"github.com/gorilla/sessions"
)

var store *sessions.CookieStore

func web(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	store = sessions.NewCookieStore(randBytes(32))
	store.MaxAge(300)
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteStrictMode
	log.Println("starting web server")

	logger := middleware.New(http.DefaultServeMux)
	logger.Use(middleware.Logger)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("GET /login", displayLogin)
	http.HandleFunc("POST /login", login)
	http.HandleFunc("/favicon.ico", favicon)
	http.HandleFunc("/styles.css", styles)

	plain := middleware.Router("", auth)
	plain.HandleFunc("/{$}", mainPage)
	plain.HandleFunc("/logs", logs)

	user := middleware.Router("/user", auth)
	user.HandleFunc("GET /{$}", admin)
	user.HandleFunc("GET /{user}", editUser)
	user.HandleFunc("POST /delete/{user}", deleteUser)
	user.HandleFunc("POST /add", addUser)
	user.HandleFunc("POST /{user}", updateUser)

	monitor := middleware.Router("/monitor", auth)
	monitor.HandleFunc("GET /new", newMonitor)
	monitor.HandleFunc("POST /new", create)
	monitor.HandleFunc("GET /delete/{site}", deleteSite)
	monitor.HandleFunc("POST /delete/{site}", deleteMonitor)
	monitor.HandleFunc("GET /edit/{site}", edit)
	monitor.HandleFunc("POST /edit/{site}", editMonitor)
	monitor.HandleFunc("GET /history/{site}/{duration}", history)

	notification := middleware.Router("/notifications", auth)
	notification.HandleFunc("GET /", notifications)
	notification.HandleFunc("GET /new", newNotification)
	notification.HandleFunc("POST /new", createNewNotify)
	notification.HandleFunc("GET /delete/{notify}", displayDeleteNotify)
	notification.HandleFunc("POST /delete/{notify}", deleteNotify)
	notification.HandleFunc("GET /edit/{notify}", displayEditNotify)
	notification.HandleFunc("POST /edit/{notify}", editNotify)
	notification.HandleFunc("GET /test/{notify}", testNotification)

	server := http.Server{Addr: ":8090", ReadHeaderTimeout: time.Second, Handler: logger}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println("web server", err)
		}
	}()
	log.Println("web server running :8090")
	<-ctx.Done()
	log.Println("shutdown web ...")
	if err := server.Shutdown(context.Background()); err != nil { //nolint:contextcheck
		log.Println("web server shutdown", err)
	}
}

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println("checking authorization")
		session, err := sessionData(w, r)
		if err != nil {
			log.Println("session err", err)
			return
		}
		if !session.LoggedIn {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := session.Session.Save(r, w); err != nil {
			log.Println("save session", err)
		}
		next.ServeHTTP(w, r)
	})
}

func randBytes(l int) []byte {
	bytes := make([]byte, l)
	_, err := rand.Read(bytes)
	if err != nil {
		panic(err)
	}
	return bytes
}

func sessionData(w http.ResponseWriter, r *http.Request) (Session, error) {
	s := Session{}
	session, err := store.Get(r, "devilcove-uptime")
	if err != nil {
		log.Println("session err", err)
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return Session{}, err
	}
	user := session.Values["user"]
	loggedIn := session.Values["logged in"]
	admin := session.Values["admin"]
	if x, ok := loggedIn.(bool); !ok || !x {
		http.Redirect(w, r, "/login", http.StatusUnauthorized)
		return Session{}, err
	} else {
		s.LoggedIn = x
	}
	if u, ok := user.(string); ok {
		s.User = u
	}
	if a, ok := admin.(bool); ok {
		s.Admin = a
	}
	s.Session = session
	return s, nil
}
