package main

import (
	"embed"
	"html/template"
	"log"
	"net/http"

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

func main() {
	log.SetFlags(log.Ltime | log.Ldate | log.Lshortfile)
	http.Handle("/logout", logger(http.HandlerFunc(loggout)))
	http.Handle("/login", logger(http.HandlerFunc(login)))
	http.Handle("/{$}", logger(auth(http.HandlerFunc(mainPage))))
	http.Handle("/logs", logger(auth(http.HandlerFunc(logs))))
	http.Handle("GET /new", logger(auth(http.HandlerFunc(new))))
	http.Handle("POST /new", logger(auth(http.HandlerFunc(create))))
	http.Handle("GET /delete/{site}", logger(auth(http.HandlerFunc(delete))))
	http.Handle("POST /delete/{site}", logger(auth(http.HandlerFunc(deleteMonitor))))
	http.Handle("GET /edit/{site}", logger(auth(http.HandlerFunc(edit))))
	http.Handle("POST /edit/{site}", logger(auth(http.HandlerFunc(editMonitor))))
	http.Handle("/history/{site}/{duration}", logger(auth(http.HandlerFunc(history))))

	log.Println("web server running :8090")
	log.Fatal(http.ListenAndServe(":8090", nil))
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
		next.ServeHTTP(w, r)
	})
}

func logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
		log.Println(r.UserAgent(), r.Host, r.URL.Path, r.RemoteAddr)
	})
}
