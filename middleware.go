package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/devilcove/cookie"
)

// Logger is a logging middleware that logs useragent, RemoteAddr, Method, Host, Path and response.Status to stdlib log.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := statusRecorder{w, http.StatusOK}
		next.ServeHTTP(&rec, r)
		remote := r.RemoteAddr
		if r.Header.Get("X-Forwarded-For") != "" {
			remote = r.Header.Get("X-Forwarded-For")
		}
		log.Println(remote, r.Method, r.Host, r.URL.Path, rec.status, r.UserAgent())
	})
}

type statusRecorder struct {
	http.ResponseWriter

	status int
}

// WriteHeader overrides std WriteHeader func to save response code.
func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}

func auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := cookie.Get(r, cookieName); err != nil {
			http.Redirect(w, r, "/login", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func sessionUser(r *http.Request) (User, error) {
	user := User{}
	data, err := cookie.Get(r, cookieName)
	if err != nil {
		return user, err
	}
	err = json.Unmarshal(data, &user)
	return user, err
}

func isAdmin(r *http.Request) bool {
	user, err := sessionUser(r)
	if err != nil {
		return false
	}
	return user.Admin
}
