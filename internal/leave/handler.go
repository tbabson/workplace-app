package leave

import (
	"encoding/json"
	"net/http"
	"strconv"

	"workplace/pkg/middleware"
	"workplace/pkg/response"

	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Get("/{id}", h.getByID)
	r.Put("/{id}/approve", h.approve)
	r.Put("/{id}/reject", h.reject)
	r.Put("/{id}/cancel", h.cancel)
	r.Get("/balance/{userID}", h.balance)
	r.Post("/balance/{userID}", h.setBalance)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), middleware.GetRole(r), middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
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
	req, err := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "leave request not found")
		return
	}
	response.JSON(w, http.StatusOK, req)
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	var dto ReviewDTO
	json.NewDecoder(r.Body).Decode(&dto)
	if err := h.svc.Approve(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r), dto); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Leave request approved")
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	var dto ReviewDTO
	json.NewDecoder(r.Body).Decode(&dto)
	if err := h.svc.Reject(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r), dto); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Leave request rejected")
}

func (h *Handler) cancel(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Cancel(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r)); err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Leave request cancelled")
}

func (h *Handler) balance(w http.ResponseWriter, r *http.Request) {
	year, _ := strconv.Atoi(r.URL.Query().Get("year"))
	list, err := h.svc.GetBalance(r.Context(), chi.URLParam(r, "userID"), year)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
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
	if err := h.svc.SetBalance(r.Context(), dto, chi.URLParam(r, "userID")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Leave balance updated")
}
