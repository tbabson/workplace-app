package milestone

import (
	"encoding/json"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes registers milestone routes. projectAccess guards routes nested
// under /projects/{projectID}.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string, projectAccess func(http.Handler) http.Handler) {
	admin := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin))

	mux.Handle("GET "+prefix+"/projects/{projectID}/milestones", middleware.With(projectAccess)(h.list))
	mux.HandleFunc("GET "+prefix+"/milestones/{id}", h.getByID)
	mux.Handle("POST "+prefix+"/projects/{projectID}/milestones", middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin), projectAccess)(h.create))
	mux.Handle("PUT "+prefix+"/milestones/{id}", admin(h.update))
	mux.Handle("PUT "+prefix+"/milestones/{id}/achieve", admin(h.achieve))
	mux.Handle("DELETE "+prefix+"/milestones/{id}", admin(h.delete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListByProject(r.Context(), r.PathValue("projectID"))
	if err != nil {
		response.ServerError(w, r, err)
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
	m, err := h.svc.Create(r.Context(), r.PathValue("projectID"), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, m)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	m, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
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
	m, err := h.svc.Update(r.Context(), r.PathValue("id"), dto)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, m)
}

func (h *Handler) achieve(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Achieve(r.Context(), r.PathValue("id")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Milestone marked as achieved")
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Milestone deleted successfully")
}
