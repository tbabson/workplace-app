package middleware

import (
	"net/http"
	"strings"
)

// StripSlashes removes a trailing slash from the request path before routing,
// so "/api/v1/users/" is served by the "/api/v1/users" route.
func StripSlashes(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if p := r.URL.Path; len(p) > 1 && strings.HasSuffix(p, "/") {
			r2 := r.Clone(r.Context())
			r2.URL.Path = strings.TrimRight(p, "/")
			if r2.URL.Path == "" {
				r2.URL.Path = "/"
			}
			if rp := r.URL.RawPath; rp != "" {
				r2.URL.RawPath = strings.TrimRight(rp, "/")
			}
			r = r2
		}
		next.ServeHTTP(w, r)
	})
}
