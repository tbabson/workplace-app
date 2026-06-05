package middleware

import (
	"net/http"

	"workplace/pkg/response"
	"workplace/pkg/utils"

	"github.com/go-chi/chi/v5"
)

// ValidateUUIDParams rejects requests where any of the listed URL params is not a valid UUID.
func ValidateUUIDParams(params ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, p := range params {
				val := chi.URLParam(r, p)
				if val != "" && !utils.IsValidUUID(val) {
					response.Error(w, http.StatusBadRequest, "invalid UUID for parameter: "+p)
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}
