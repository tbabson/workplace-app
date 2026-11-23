package leave

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	reviewer := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleDeptHead))
	admin := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin))

	mux.HandleFunc("GET "+prefix, h.list)
	mux.HandleFunc("POST "+prefix, h.create)
	mux.HandleFunc("GET "+prefix+"/{id}", h.getByID)
	mux.Handle("PUT "+prefix+"/{id}/approve", reviewer(h.approve))
	mux.Handle("PUT "+prefix+"/{id}/reject", reviewer(h.reject))
	mux.HandleFunc("PUT "+prefix+"/{id}/cancel", h.cancel)
	mux.HandleFunc("GET "+prefix+"/balance/{userID}", h.balance)
	mux.Handle("POST "+prefix+"/balance/{userID}", admin(h.setBalance))
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrOwnRequest), errors.Is(err, ErrForbidden):
		response.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrNotPending):
		response.Error(w, http.StatusConflict, err.Error())
	default:
		response.ServerError(w, r, err)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), middleware.IsManager(r), middleware.GetUserID(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req, err := h.svc.Create(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, req)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	req, err := h.svc.Get(r.Context(), r.PathValue("id"), middleware.GetUserID(r), middleware.IsManager(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, req)
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	var dto ReviewDTO
	_ = json.NewDecoder(r.Body).Decode(&dto) // note is optional
	if err := h.svc.Approve(r.Context(), r.PathValue("id"), middleware.GetUserID(r), dto); err != nil {
		writeErr(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Leave request approved")
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	var dto ReviewDTO
	_ = json.NewDecoder(r.Body).Decode(&dto) // note is optional
	if err := h.svc.Reject(r.Context(), r.PathValue("id"), middleware.GetUserID(r), dto); err != nil {
		writeErr(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Leave request rejected")
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Cancel(r.Context(), r.PathValue("id"), middleware.GetUserID(r)); err != nil {
		writeErr(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Leave request cancelled")
}

func (h *Handler) balance(w http.ResponseWriter, r *http.Request) {
	userID := r.PathValue("userID")
	if userID != middleware.GetUserID(r) && !middleware.IsManager(r) {
		response.Error(w, http.StatusForbidden, "cannot view another user's leave balance")
		return
	}
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	list, err := h.svc.GetBalance(r.Context(), userID, year)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) setBalance(w http.ResponseWriter, r *http.Request) {
	var dto SetBalanceDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.SetBalance(r.Context(), dto, r.PathValue("userID")); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Leave balance updated")
}
