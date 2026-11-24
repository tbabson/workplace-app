package project

import (
	"encoding/json"
	"fmt"
	"net/http"

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
	validate := middleware.ValidateUUIDParams("id", "userID")
	// Read-only — anyone who can see the project (see Service.CanAccess)
	anyRole := middleware.With(validate)
	readable := middleware.With(validate, h.RequireAccess("id"))
	// Project management — super_admin only
	admin := middleware.With(validate, middleware.RequireRole(middleware.RoleSuperAdmin))
	// Budget & expenses — procurement or super_admin
	finance := middleware.With(validate, middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleProcurement))

	mux.Handle("GET "+prefix, anyRole(h.list))
	mux.Handle("GET "+prefix+"/{id}", readable(h.getByID))
	mux.Handle("GET "+prefix+"/{id}/assignees", readable(h.assignees))
	mux.Handle("GET "+prefix+"/{id}/room", readable(h.projectRoom))
	mux.Handle("GET "+prefix+"/{id}/history", readable(h.history))
	mux.Handle("GET "+prefix+"/{id}/budget", readable(h.getBudget))
	mux.Handle("GET "+prefix+"/{id}/files", readable(h.listFiles))

	mux.Handle("POST "+prefix, admin(h.create))
	mux.Handle("PUT "+prefix+"/{id}", admin(h.update))
	mux.Handle("DELETE "+prefix+"/{id}", admin(h.delete))
	mux.Handle("PUT "+prefix+"/{id}/progress", admin(h.updateProgress))
	mux.Handle("POST "+prefix+"/{id}/assign", admin(h.assign))
	mux.Handle("DELETE "+prefix+"/{id}/assign/{userID}", admin(h.unassign))
	mux.Handle("POST "+prefix+"/{id}/files", admin(h.addFile))
	mux.Handle("DELETE "+prefix+"/{id}/files/{fileID}", admin(h.deleteFile))

	mux.Handle("PUT "+prefix+"/{id}/budget", finance(h.setBudget))
	mux.Handle("POST "+prefix+"/{id}/expenses", finance(h.addExpense))
	mux.Handle("DELETE "+prefix+"/{id}/expenses/{expenseID}", finance(h.deleteExpense))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	scopeUserID := ""
	if !middleware.HasRole(r, middleware.RoleSuperAdmin, middleware.RoleProcurement) {
		scopeUserID = middleware.GetUserID(r)
	}

	projects, err := h.svc.List(r.Context(), middleware.GetDeptID(r), scopeUserID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, projects, len(projects))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateProjectDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.svc.Create(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, p)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	p, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "project not found")
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var dto UpdateProjectDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	p, err := h.svc.Update(r.Context(), r.PathValue("id"), middleware.GetUserID(r), dto)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, p)
}

func (h *Handler) updateProgress(w http.ResponseWriter, r *http.Request) {
	var dto UpdateProgressDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if err := h.svc.UpdateProgress(r.Context(), r.PathValue("id"), middleware.GetUserID(r), dto); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Project progress updated successfully")
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Project deleted successfully")
}

func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	var dto AssignDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	added, err := h.svc.Assign(r.Context(), r.PathValue("id"), dto.UserID)
	if err != nil {
		if err.Error() == "user not found" {
			response.Error(w, http.StatusBadRequest, "user not found")
			return
		}
		response.ServerError(w, r, err)
		return
	}
	msg := "User assigned to project successfully and added to project chat"
	if n := len(added) - 1; n > 0 {
		msg = fmt.Sprintf("Department head assigned; %d staff from their department were added too and are awaiting duties", n)
	}
	response.JSON(w, http.StatusOK, map[string]any{"message": msg, "added_user_ids": added})
}

func (h *Handler) unassign(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Unassign(r.Context(), r.PathValue("id"), r.PathValue("userID")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "User unassigned from project successfully and removed from project chat")
}

func (h *Handler) assignees(w http.ResponseWriter, r *http.Request) {
	ids, err := h.svc.GetAssignees(r.Context(), r.PathValue("id"))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, ids, len(ids))
}

func (h *Handler) projectRoom(w http.ResponseWriter, r *http.Request) {
	room, err := h.svc.GetProjectRoom(r.Context(), r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "project chat room not found")
		return
	}
	response.JSON(w, http.StatusOK, room)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GetHistory(r.Context(), r.PathValue("id"))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) getBudget(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.GetBudget(r.Context(), r.PathValue("id"))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, summary)
}

func (h *Handler) setBudget(w http.ResponseWriter, r *http.Request) {
	var dto SetBudgetDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.svc.SetBudget(r.Context(), r.PathValue("id"), middleware.GetUserID(r), dto); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Budget updated successfully")
}

func (h *Handler) addExpense(w http.ResponseWriter, r *http.Request) {
	var dto AddExpenseDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	e, err := h.svc.AddExpense(r.Context(), r.PathValue("id"), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, e)
}

func (h *Handler) deleteExpense(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteExpense(r.Context(), r.PathValue("id"), r.PathValue("expenseID")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Expense deleted successfully")
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListFiles(r.Context(), r.PathValue("id"))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) addFile(w http.ResponseWriter, r *http.Request) {
	var dto AddFileDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	f, err := h.svc.AddFile(r.Context(), r.PathValue("id"), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, f)
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteFile(r.Context(), r.PathValue("id"), r.PathValue("fileID")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "File deleted successfully")
}

// RequireAccess returns middleware that responds 404 unless the caller may read
// the project whose ID is in the given path parameter. Other packages use it to
// protect routes nested under a project.
func (h *Handler) RequireAccess(param string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p, err := h.svc.GetByID(r.Context(), r.PathValue(param))
			if err != nil {
				response.Error(w, http.StatusNotFound, "project not found")
				return
			}
			ok, err := h.svc.CanAccess(r.Context(), p,
				middleware.GetUserID(r), middleware.GetRole(r), middleware.GetDeptID(r))
			if err != nil {
				response.ServerError(w, r, err)
				return
			}
			if !ok {
				response.Error(w, http.StatusNotFound, "project not found")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
