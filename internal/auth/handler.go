package auth

import (
	"encoding/json"
	"errors"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct {
	svc          *Service
	secureCookie bool
}

// NewHandler creates the auth handler. secureCookie should be true whenever the
// app is served over HTTPS (i.e. outside local development).
func NewHandler(svc *Service, secureCookie bool) *Handler {
	return &Handler{svc: svc, secureCookie: secureCookie}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	mux.HandleFunc("POST "+prefix+"/login", h.login)
	mux.HandleFunc("POST "+prefix+"/logout", h.logout)
}

func (h *Handler) login(w http.ResponseWriter, r *http.Request) {
	var dto LoginDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	token, err := h.svc.Login(r.Context(), dto)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, ErrTooManyAttempts) {
			status = http.StatusTooManyRequests
		}
		response.Error(w, status, err.Error())
		return
	}

	h.setCookie(w, token, int(h.svc.TokenTTL().Seconds()))
	response.JSON(w, http.StatusOK, map[string]string{"message": "Login successful", "token": token})
}

func (h *Handler) logout(w http.ResponseWriter, r *http.Request) {
	token := middleware.TokenFromRequest(r)
	if token == "" {
		response.Error(w, http.StatusBadRequest, "missing token")
		return
	}
	if err := h.svc.Logout(r.Context(), token); err != nil {
		response.ServerError(w, r, err)
		return
	}

	h.setCookie(w, "", -1)
	response.Message(w, http.StatusOK, "Logged out successfully")
}

func (h *Handler) setCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookie,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   maxAge,
	})
}
