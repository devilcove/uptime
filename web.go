package main

import (
	"context"
	"crypto/rand"
	"embed"
	"html/template"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/sessions"
)

//go:embed html/*
var f embed.FS

var (
	templates = template.Must(template.New("").ParseFS(f, "html/*"))
	store     *sessions.CookieStore
)

type Report struct {
	Site   string
	Status string
	Code   string
	Time   string
}

func web(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()
	store = sessions.NewCookieStore(randBytes(32))
	store.MaxAge(300)
	store.Options.HttpOnly = true
	store.Options.SameSite = http.SameSiteStrictMode
	log.Println("starting web server")

	http.Handle("GET /admin", logger(auth(http.HandlerFunc(admin))))
	http.Handle("GET /user", logger(auth(isAdmin(http.HandlerFunc(newUser)))))
	http.Handle("GET /user/{user}", logger(auth(http.HandlerFunc(editUser))))
	http.Handle("POST /user/delete/{user}", logger(auth(http.HandlerFunc(deleteUser))))
	http.Handle("POST /user", logger(http.HandlerFunc(addUser)))
	http.Handle("POST /user/{user}", logger(http.HandlerFunc(updateUser)))

	http.Handle("/logout", logger(http.HandlerFunc(loggout)))
	http.Handle("GET /login", logger(http.HandlerFunc(displayLogin)))
	http.Handle("POST /login", logger(http.HandlerFunc(login)))
	http.Handle("/{$}", logger(auth(http.HandlerFunc(mainPage))))

	http.Handle("/logs", logger(auth(http.HandlerFunc(logs))))

	http.Handle("GET /new", logger(auth(http.HandlerFunc(new))))
	http.Handle("POST /new", logger(auth(http.HandlerFunc(create))))
	http.Handle("GET /delete/{site}", logger(auth(http.HandlerFunc(deleteSite))))
	http.Handle("POST /delete/{site}", logger(auth(http.HandlerFunc(deleteMonitor))))
	http.Handle("GET /edit/{site}", logger(auth(http.HandlerFunc(edit))))
	http.Handle("POST /edit/{site}", logger(auth(http.HandlerFunc(editMonitor))))

	http.Handle("/history/{site}/{duration}", logger(auth(http.HandlerFunc(history))))

	http.Handle("GET /notifications", logger(auth(http.HandlerFunc(notifications))))
	http.Handle("GET /notify", logger(auth(http.HandlerFunc(notify))))
	http.Handle("POST /notify", logger(auth(http.HandlerFunc(createNewNotify))))
	http.Handle("GET /notify/delete/{notify}", logger(auth(http.HandlerFunc(displayDeleteNotify))))
	http.Handle("POST /notify/delete/{notify}", logger(auth(http.HandlerFunc(deleteNotify))))
	http.Handle("GET /notify/edit/{notify}", logger(auth(http.HandlerFunc(displayEditNotify))))
	http.Handle("POST /notify/edit/{notify}", logger(auth(http.HandlerFunc(editNotify))))

	server := http.Server{Addr: ":8090", ReadHeaderTimeout: time.Second}
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
		session, err := store.Get(r, "devilcove-uptime")
		if err != nil {
			log.Println("session err", err)
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		loggedIn := session.Values["logged in"]
		if x, ok := loggedIn.(bool); !ok || !x {
			log.Println("no session")
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		if !loggedIn.(bool) {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		if err := session.Save(r, w); err != nil {
			log.Println("save session", err)
		}
		next.ServeHTTP(w, r)
	})
}

func isAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "devilcove-uptime")
		if err != nil {
			log.Println("session err", err)
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		isAdmin := session.Values["admin"]
		if x, ok := isAdmin.(bool); !ok || !x {
			log.Println("not admin")
			http.Error(w, "admin privileges are required", http.StatusUnauthorized)
			return
		}
		if err := session.Save(r, w); err != nil {
			log.Println("save session", err)
		}
		next.ServeHTTP(w, r)
	})
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		log.Println(r.UserAgent(), r.RemoteAddr, r.Method, r.Host, r.URL.Path)
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
