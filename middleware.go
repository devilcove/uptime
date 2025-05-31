package main

import (
	"log"
	"net/http"
)

// Middleware provides a chain of middlewares.
type Middleware struct {
	chain http.Handler
}

// DefaultMiddleware creates a default middleware using DefaultServeMux.
func DefaultMiddleware() *Middleware {
	return &Middleware{http.DefaultServeMux}
}

// NewMiddleware returns a new middleware.
func NewMiddleware(handler http.Handler) *Middleware {
	return &Middleware{handler}
}

// Router creates a severMux for given prefix and middleware chain.
func Router(prefix string, middleware ...func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	middle := NewMiddleware(mux)
	middle.UseGroup(middleware...)
	http.Handle(prefix+"/", http.StripPrefix(prefix, middle))
	return mux
}

// Use adds middleware to the chain.
func (a *Middleware) Use(handler func(http.Handler) http.Handler) {
	a.chain = handler(a.chain)
}

// UseGroup adds a chain of middlewares.
func (a *Middleware) UseGroup(handlers ...func(http.Handler) http.Handler) {
	for _, handler := range handlers {
		a.Use(handler)
	}
}

// ServeHTTP implements http.Handler interface.
func (a *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.chain.ServeHTTP(w, r)
}

// Logger is a logging middleware that logs useragent, RemoteAddr, Method, Host, Path and response.Status to stdlib log.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := statusRecorder{w, http.StatusOK}
		next.ServeHTTP(&rec, r)
		remote := r.RemoteAddr
		if r.Header.Get("X-Forwarded-For") != "" {
			remote = r.Header.Get("X-Forwared-For")
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
		log.Println("checking authorization")
		session, err := sessionData(r)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if !session.LoggedIn {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if err := session.Session.Save(r, w); err != nil {
			log.Println("save session", err)
		}
		next.ServeHTTP(w, r)
	})
}

func sessionData(r *http.Request) (Session, error) {
	s := Session{}
	session, err := store.Get(r, "devilcove-uptime")
	if err != nil {
		log.Println("session err", err)
		return Session{}, err
	}
	user := session.Values["user"]
	loggedIn := session.Values["logged in"]
	admin := session.Values["admin"]
	if x, ok := loggedIn.(bool); !ok || !x {
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
