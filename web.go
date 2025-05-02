package main

import (
	"context"
	"embed"
	"html/template"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/sessions"
)

//go:embed html/*
var f embed.FS

var (
	templates = template.Must(template.New("").ParseFS(f, "html/*"))
	store     = sessions.NewCookieStore([]byte("secret")) // TODO change for production
)

type Report struct {
	Site   string
	Status string
	Code   string
	Time   string
}

func web(ctx context.Context, wg *sync.WaitGroup, restart func()) {
	defer wg.Done()
	log.Println("starting web server")
	log.SetFlags(log.Ltime | log.Ldate | log.Lshortfile)

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
	http.Handle("GET /delete/{site}", logger(auth(http.HandlerFunc(delete))))
	http.Handle("POST /delete/{site}", logger(auth(http.HandlerFunc(deleteMonitor))))
	http.Handle("GET /edit/{site}", logger(auth(http.HandlerFunc(edit))))
	http.Handle("POST /edit/{site}", logger(auth(http.HandlerFunc(editMonitor))))
	http.Handle("/history/{site}/{duration}", logger(auth(http.HandlerFunc(history))))

	server := http.Server{Addr: ":8090"}
	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Println("web server", err)
		}
	}()
	log.Println("web server running :8090")
	<-ctx.Done()
	log.Println("shutdown web ...")
	if err := server.Shutdown(context.Background()); err != nil {
		log.Println("web server shutdown", err)
	}
}

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := store.Get(r, "helloworld")
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
		session, err := store.Get(r, "helloworld")
		if err != nil {
			log.Println("session err", err)
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		isAdmin := session.Values["admin"]
		if x, ok := isAdmin.(bool); !ok || !x {
			log.Println("not admin")
			http.Error(w, "admin privilages are required", http.StatusUnauthorized)
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
