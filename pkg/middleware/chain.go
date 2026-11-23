package middleware

import "net/http"

// With composes middlewares into a single wrapper for a handler. The first
// middleware listed is the outermost, so it runs first.
//
//	admin := middleware.With(middleware.RequireRole("super_admin"))
//	mux.Handle("POST /api/v1/tasks", admin(h.create))
func With(mws ...func(http.Handler) http.Handler) func(http.HandlerFunc) http.Handler {
	return func(h http.HandlerFunc) http.Handler {
		var handler http.Handler = h
		for i := len(mws) - 1; i >= 0; i-- {
			handler = mws[i](handler)
		}
		return handler
	}
}
