package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
)

// Recoverer recovers from panics in downstream handlers, logs the stack trace
// and responds with 500 Internal Server Error.
func Recoverer(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				// http.ErrAbortHandler is used to deliberately abort a response;
				// let net/http handle it as intended.
				if rec == http.ErrAbortHandler {
					panic(rec)
				}
				log.Printf("panic: %v\n%s", rec, debug.Stack())
				// Upgraded (WebSocket) connections can't receive a status code.
				if r.Header.Get("Connection") != "Upgrade" {
					w.WriteHeader(http.StatusInternalServerError)
				}
			}
		}()
		next.ServeHTTP(w, r)
	})
}
