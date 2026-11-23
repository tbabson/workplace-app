package middleware

import (
	"net/http"
	"strings"
)

// OriginChecker returns a function reporting whether a request's Origin is in
// the allowlist. Requests without an Origin header (non-browser clients) pass.
func OriginChecker(allowed []string) func(r *http.Request) bool {
	set := make(map[string]bool, len(allowed))
	for _, o := range allowed {
		if o = strings.TrimRight(strings.TrimSpace(o), "/"); o != "" {
			set[o] = true
		}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		return origin == "" || set[origin]
	}
}

// CORS allows credentialed cross-origin requests only from allowlisted origins.
func CORS(allowed []string) func(http.Handler) http.Handler {
	ok := OriginChecker(allowed)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			w.Header().Add("Vary", "Origin")

			if origin != "" && ok(r) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				if origin != "" && !ok(r) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
