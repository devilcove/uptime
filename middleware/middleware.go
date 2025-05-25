package middleware

import (
	"log"
	"net/http"
)

type Middleware struct {
	chain http.Handler
}

func Default() *Middleware {
	return &Middleware{http.DefaultServeMux}
}

func New(handler http.Handler) *Middleware {
	return &Middleware{handler}
}

func Router(prefix string, middleware ...func(http.Handler) http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	middle := New(mux)
	middle.UseGroup(middleware...)
	http.Handle(prefix+"/", http.StripPrefix(prefix, middle))
	return mux
}

func (a *Middleware) Use(handler func(http.Handler) http.Handler) {
	a.chain = handler(a.chain)
}

func (a *Middleware) UseGroup(handlers ...func(http.Handler) http.Handler) {
	for _, handler := range handlers {
		a.Use(handler)
	}
}

func (a *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	a.chain.ServeHTTP(w, r)
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec := statusRecorder{w, http.StatusOK}
		next.ServeHTTP(&rec, r)
		log.Println(r.UserAgent(), r.RemoteAddr, r.Method, r.Host, r.URL.Path, rec.status)
	})
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(code int) {
	rec.status = code
	rec.ResponseWriter.WriteHeader(code)
}
