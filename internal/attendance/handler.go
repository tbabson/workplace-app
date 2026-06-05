package attendance

import (
	"encoding/json"
	"net/http"
	"time"

	"workplace/pkg/middleware"
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
	r.Post("/sign-in", h.signIn)
	r.Post("/sign-out", h.signOut)
	r.Put("/{id}/sign-out", h.adminSignOut)
	r.Get("/me", h.myHistory)
	r.Get("/user/{userID}", h.userHistory)
	r.Get("/", h.allHistory)
}

func (h *Handler) signIn(w http.ResponseWriter, r *http.Request) {
	var dto SignInDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	a, err := h.svc.SignIn(r.Context(), middleware.GetUserID(r), dto)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, a)
}

func (h *Handler) signOut(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.SignOut(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, a)
}

func (h *Handler) adminSignOut(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r)
	if role != "super_admin" && role != "dept_head" {
		response.Error(w, http.StatusForbidden, "forbidden")
		return
	}

	var body struct {
		SignOutAt *time.Time `json:"sign_out_at"`
	}
	// ignore decode error — body is optional
	_ = json.NewDecoder(r.Body).Decode(&body)

	a, err := h.svc.AdminSignOut(r.Context(), chi.URLParam(r, "id"), body.SignOutAt)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, a)
}

func (h *Handler) myHistory(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.MyHistory(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) userHistory(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.UserHistory(r.Context(), chi.URLParam(r, "userID"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) allHistory(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.AllHistory(r.Context())
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}
