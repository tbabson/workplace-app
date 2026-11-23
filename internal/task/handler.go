package task

import (
	"encoding/json"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes registers task routes. projectAccess guards routes nested under
// /projects/{projectID} so callers only see tasks of projects they can access.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string, projectAccess func(http.Handler) http.Handler) {
	admin := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin), projectAccess)
	inProject := middleware.With(projectAccess)

	mux.Handle("GET "+prefix+"/projects/{projectID}/tasks", inProject(h.list))
	mux.HandleFunc("GET "+prefix+"/tasks/{id}", h.getByID)
	mux.HandleFunc("PUT "+prefix+"/tasks/{id}", h.update)
	mux.Handle("POST "+prefix+"/projects/{projectID}/tasks", admin(h.create))
	mux.Handle("DELETE "+prefix+"/tasks/{id}", middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin))(h.delete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListByProject(r.Context(), r.PathValue("projectID"))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, tasks, len(tasks))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateTaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	t, err := h.svc.Create(r.Context(), r.PathValue("projectID"), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, t)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "task not found")
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var dto UpdateTaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// super_admin may change anything. Otherwise only the assignee may update
	// the task, and only its status — reassigning would let a user add
	// themselves to any project's chat.
	if !middleware.HasRole(r, middleware.RoleSuperAdmin) {
		t, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
		if err != nil {
			response.Error(w, http.StatusNotFound, "task not found")
			return
		}
		if t.AssignedTo == nil || *t.AssignedTo != middleware.GetUserID(r) {
			response.Error(w, http.StatusForbidden, "only the assignee can update this task")
			return
		}
		if dto.Title != nil || dto.Description != nil || dto.AssignedTo != nil || dto.DueDate != nil {
			response.Error(w, http.StatusForbidden, "you can only change the status of your task")
			return
		}
	}
	t, err := h.svc.Update(r.Context(), r.PathValue("id"), dto)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, t)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Task deleted successfully")
}
