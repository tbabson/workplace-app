package task

import (
	"encoding/json"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// Who may manage a project's tasks (its duties):
//   - super_admin, on any project;
//   - a dept_head who is assigned to the project, for staff in their department.
//
// The assignee of a task may change only its status.

// RegisterRoutes registers task routes. projectAccess guards routes nested under
// /projects/{projectID} so callers only see tasks of projects they can access.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string, projectAccess func(http.Handler) http.Handler) {
	managers := middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleDeptHead)
	inProject := middleware.With(projectAccess)

	mux.Handle("GET "+prefix+"/projects/{projectID}/tasks", inProject(h.list))
	mux.HandleFunc("GET "+prefix+"/tasks/{id}", h.getByID)
	mux.HandleFunc("PUT "+prefix+"/tasks/{id}", h.update)
	mux.Handle("POST "+prefix+"/projects/{projectID}/tasks", middleware.With(managers, projectAccess)(h.create))
	mux.Handle("DELETE "+prefix+"/tasks/{id}", middleware.With(managers)(h.delete))
}

// canManage reports whether the caller may create, reassign, edit or delete
// tasks in projectID.
func (h *Handler) canManage(r *http.Request, projectID string) (bool, error) {
	switch middleware.GetRole(r) {
	case middleware.RoleSuperAdmin:
		return true, nil
	case middleware.RoleDeptHead:
		return h.svc.IsProjectMember(r.Context(), projectID, middleware.GetUserID(r))
	default:
		return false, nil
	}
}

// checkAssignee verifies a task can be given to assigneeID: they must be on the
// project, and a dept head may only give duties to staff in their department.
func (h *Handler) checkAssignee(w http.ResponseWriter, r *http.Request, projectID string, assigneeID *string) bool {
	if assigneeID == nil || *assigneeID == "" {
		return true
	}
	member, err := h.svc.IsProjectMember(r.Context(), projectID, *assigneeID)
	if err != nil {
		response.ServerError(w, r, err)
		return false
	}
	if !member {
		response.Error(w, http.StatusBadRequest, "the assignee must be a member of this project")
		return false
	}
	if !middleware.CanAccessUser(r, *assigneeID) {
		response.Error(w, http.StatusForbidden, "you can only assign duties to staff in your department")
		return false
	}
	return true
}

// visible reports whether the caller may see a task:
//   - super admins see every task;
//   - dept heads see tasks assigned to them or to staff they oversee (their
//     department and its sub-departments), plus tasks they created — not other
//     departments' duties on a shared project;
//   - everyone else (staff, procurement) sees only tasks assigned to them, plus
//     any they created.
func visible(r *http.Request, t *Task) bool {
	me := middleware.GetUserID(r)
	switch {
	case middleware.HasRole(r, middleware.RoleSuperAdmin):
		return true
	case t.CreatedBy == me:
		return true
	case middleware.HasRole(r, middleware.RoleDeptHead):
		return t.AssignedTo != nil && middleware.CanAccessUser(r, *t.AssignedTo)
	default:
		return t.AssignedTo != nil && *t.AssignedTo == me
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.svc.ListByProject(r.Context(), r.PathValue("projectID"))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	shown := make([]*Task, 0, len(tasks))
	for _, t := range tasks {
		if visible(r, t) {
			shown = append(shown, t)
		}
	}
	response.JSON(w, http.StatusOK, shown, len(shown))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	projectID := r.PathValue("projectID")
	var dto CreateTaskDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ok, err := h.canManage(r, projectID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	if !ok {
		response.Error(w, http.StatusForbidden, "only a super admin or a department head on this project can assign duties")
		return
	}
	if !h.checkAssignee(w, r, projectID, dto.AssignedTo) {
		return
	}
	t, err := h.svc.Create(r.Context(), projectID, dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, t)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !visible(r, t) {
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
	t, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !visible(r, t) {
		response.Error(w, http.StatusNotFound, "task not found")
		return
	}

	manager, err := h.canManage(r, t.ProjectID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	if manager {
		if !h.checkAssignee(w, r, t.ProjectID, dto.AssignedTo) {
			return
		}
	} else {
		// Otherwise only the assignee may update the task, and only its status;
		// reassigning would let a user add themselves to any project's chat.
		if t.AssignedTo == nil || *t.AssignedTo != middleware.GetUserID(r) {
			response.Error(w, http.StatusForbidden, "only the assignee can update this task")
			return
		}
		if dto.Title != nil || dto.Description != nil || dto.AssignedTo != nil || dto.DueDate != nil {
			response.Error(w, http.StatusForbidden, "you can only change the status of your task")
			return
		}
	}

	updated, err := h.svc.Update(r.Context(), t.ID, dto)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, updated)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	t, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !visible(r, t) {
		response.Error(w, http.StatusNotFound, "task not found")
		return
	}
	ok, err := h.canManage(r, t.ProjectID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	if !ok {
		response.Error(w, http.StatusForbidden, "only a super admin or a department head on this project can delete duties")
		return
	}
	if err := h.svc.Delete(r.Context(), t.ID); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Task deleted successfully")
}
