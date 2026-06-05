package overtime

import (
	"encoding/json"
	"net/http"

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
	r.Get("/", h.list)
	r.Post("/", h.create)
	r.Group(func(r chi.Router) {
		r.Use(middleware.ValidateUUIDParams("id"))
		r.Get("/{id}", h.getByID)
		r.Put("/{id}", h.update)
		r.Put("/{id}/mark-paid", h.markPaid)
		r.Delete("/{id}", h.delete)
	})
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r)
	userID := ""
	if role == "staff" {
		userID = middleware.GetUserID(r)
	}

	list, err := h.svc.List(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r)
	if role != "super_admin" && role != "dept_head" {
		response.Error(w, http.StatusForbidden, "only managers can log overtime")
		return
	}

	var dto CreateOvertimeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	o, err := h.svc.Create(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, o)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "overtime record not found")
		return
	}
	response.JSON(w, http.StatusOK, o)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var dto UpdateOvertimeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	o, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), dto)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, o)
}

func (h *Handler) markPaid(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r)
	if role != "super_admin" && role != "dept_head" {
		response.Error(w, http.StatusForbidden, "only managers can mark overtime as paid")
		return
	}
	if err := h.svc.MarkPaid(r.Context(), chi.URLParam(r, "id")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Overtime marked as paid successfully")
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r)
	if role != "super_admin" && role != "dept_head" {
		response.Error(w, http.StatusForbidden, "only managers can delete overtime records")
		return
	}
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Overtime record deleted successfully")
}
