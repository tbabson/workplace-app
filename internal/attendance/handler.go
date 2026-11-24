package attendance

import (
	"encoding/json"
	"net/http"
	"time"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	manager := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleDeptHead))

	mux.HandleFunc("POST "+prefix+"/sign-in", h.signIn)
	mux.HandleFunc("POST "+prefix+"/sign-out", h.signOut)
	mux.Handle("PUT "+prefix+"/{id}/sign-out", manager(h.adminSignOut))
	mux.HandleFunc("GET "+prefix+"/me", h.myHistory)
	mux.Handle("GET "+prefix+"/user/{userID}", manager(h.userHistory))
	mux.Handle("GET "+prefix, manager(h.allHistory))
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
	var body struct {
		SignOutAt *time.Time `json:"sign_out_at"`
	}
	// ignore decode error — body is optional
	_ = json.NewDecoder(r.Body).Decode(&body)

	// Dept heads may only close records of staff in their own department.
	rec, err := h.svc.Get(r.Context(), r.PathValue("id"))
	if err != nil || !middleware.CanAccessUser(r, rec.UserID) {
		response.Error(w, http.StatusNotFound, "attendance record not found")
		return
	}

	a, err := h.svc.AdminSignOut(r.Context(), r.PathValue("id"), body.SignOutAt)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, a)
}

func (h *Handler) myHistory(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.MyHistory(r.Context(), middleware.GetUserID(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) userHistory(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if !middleware.CanAccessUser(r, userID) {
		response.Error(w, http.StatusForbidden, "you can only view attendance of staff in your department")
		return
	}
	list, err := h.svc.UserHistory(r.Context(), userID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) allHistory(w http.ResponseWriter, r *http.Request) {
	// Dept heads see only their own department's attendance.
	list, err := h.svc.AllHistory(r.Context(), middleware.ScopeDept(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}
