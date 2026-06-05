package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"workplace/pkg/response"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var dto LoginDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.svc.Login(r.Context(), dto)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, err.Error())
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   86400 * 7, // 7 days
	})

	response.JSON(w, http.StatusOK, map[string]string{"message": "Login successful", "token": token})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	// Read token from cookie
	cookie, err := r.Cookie("auth_token")
	if err != nil || cookie.Value == "" {
		// Also check Authorization header as fallback
		header := r.Header.Get("Authorization")
		token := strings.TrimPrefix(header, "Bearer ")
		if token == "" {
			response.Error(w, http.StatusBadRequest, "missing token")
			return
		}
		_ = h.svc.Logout(r.Context(), token)
	} else {
		_ = h.svc.Logout(r.Context(), cookie.Value)
	}

	// Clear the cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})

	response.Message(w, http.StatusOK, "Logged out successfully")
}
