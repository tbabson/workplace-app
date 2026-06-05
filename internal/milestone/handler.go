package milestone

import (
	"encoding/json"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"

	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/projects/{projectID}/milestones", h.list)
	r.Get("/milestones/{id}", h.getByID)
	r.With(middleware.RequireRole("super_admin")).Post("/projects/{projectID}/milestones", h.create)
	r.With(middleware.RequireRole("super_admin")).Put("/milestones/{id}", h.update)
	r.With(middleware.RequireRole("super_admin")).Put("/milestones/{id}/achieve", h.achieve)
	r.With(middleware.RequireRole("super_admin")).Delete("/milestones/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListByProject(r.Context(), chi.URLParam(r, "projectID"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateMilestoneDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	m, err := h.svc.Create(r.Context(), chi.URLParam(r, "projectID"), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, m)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "milestone not found")
		return
	}
	response.JSON(w, http.StatusOK, m)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var dto UpdateMilestoneDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	m, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), dto)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, m)
}

func (h *Handler) achieve(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Achieve(r.Context(), chi.URLParam(r, "id")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Milestone marked as achieved")
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Milestone deleted successfully")
}
