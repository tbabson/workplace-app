package department

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

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
	admin := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin))

	mux.HandleFunc("GET "+prefix, h.list)
	mux.Handle("POST "+prefix, admin(h.create))
	mux.HandleFunc("GET "+prefix+"/{id}", h.getByID)
	mux.Handle("PUT "+prefix+"/{id}", admin(h.update))
	mux.Handle("DELETE "+prefix+"/{id}", admin(h.delete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	depts, err := h.svc.List(r.Context())
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, depts, len(depts))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateDepartmentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	d, err := h.svc.Create(r.Context(), dto)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusCreated, d)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	d, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "department not found")
		return
	}
	response.JSON(w, http.StatusOK, d)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var dto UpdateDepartmentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	d, err := h.svc.Update(r.Context(), r.PathValue("id"), dto)
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, d)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Department deleted successfully")
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrValidation):
		response.Error(w, http.StatusBadRequest, strings.TrimPrefix(err.Error(), ErrValidation.Error()+": "))
	case response.NotFound(err):
		response.Error(w, http.StatusNotFound, "department not found")
	default:
		response.ServerError(w, r, err)
	}
}
