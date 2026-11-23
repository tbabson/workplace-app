package response

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/jackc/pgx/v5"
)

type envelope struct {
	Success bool        `json:"success"`
	Count   *int        `json:"count,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	Message string      `json:"message,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data interface{}, count ...int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	env := envelope{Success: status < 400, Data: data}
	if len(count) > 0 {
		env.Count = &count[0]
	}
	json.NewEncoder(w).Encode(env)
}

func Message(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{Success: status < 400, Message: msg})
}

// ServerError logs the underlying error and returns a generic 500 so that
// database and internal details are never exposed to clients.
func ServerError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%s %s: %v", r.Method, r.URL.Path, err)
	Error(w, http.StatusInternalServerError, "internal server error")
}

// NotFound reports whether err means "no such row".
func NotFound(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}

func Error(w http.ResponseWriter, status int, err string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(envelope{Success: false, Error: err})
}
