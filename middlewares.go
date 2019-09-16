package main

import (
	"log"
	"net/http"
)

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			log.Printf("%s %s", r.Method, r.URL.String())
		}()
		next.ServeHTTP(w, r)
	})
}

func enforceJSONHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "POST" && r.ContentLength == 0 {
			http.Error(w, http.StatusText(400), 400)
			return
		}
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func enforceMethod(method string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				http.Error(w, http.StatusText(405), 405)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type Middleware func(http.Handler) http.Handler

func Chain(h http.HandlerFunc, m ...Middleware) http.Handler {
	if len(m) < 1 {
		return h
	}
	wrapped := http.Handler(h)
	// loop in reverse to preserve middleware order
	for i := len(m) - 1; i >= 0; i-- {
		wrapped = m[i](wrapped)
	}
	return wrapped
}
