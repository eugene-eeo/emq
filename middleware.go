package main

import "net/http"
import "regexp"

type Middleware func(http.HandlerFunc) http.HandlerFunc

func EnforceMethodMiddleware(method string) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				http.Error(w, http.StatusText(400), 400)
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

var ContentTypeRegex *regexp.Regexp = regexp.MustCompile("^application/json(;.+)?$")

func EnforceJSONMiddleware() Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if !ContentTypeRegex.MatchString(r.Header.Get("Content-Type")) {
				http.Error(w, http.StatusText(400), 400)
				return
			}
			w.Header().Set(
				"Content-Type",
				"application/json; charset=UTF-8",
			)
			next.ServeHTTP(w, r)
		}
	}
}

func Chain(h http.HandlerFunc, m ...Middleware) http.Handler {
	for i := len(m) - 1; i >= 0; i-- {
		h = m[i](h)
	}
	return h
}
