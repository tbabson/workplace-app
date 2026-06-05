package project

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
	r.Use(middleware.ValidateUUIDParams("id", "userID"))
	// Read-only — all authenticated roles
	r.Get("/", h.list)
	r.Get("/{id}", h.getByID)
	r.Get("/{id}/assignees", h.assignees)
	r.Get("/{id}/room", h.projectRoom)
	r.Get("/{id}/history", h.history)
	r.Get("/{id}/budget", h.getBudget)
	// Project management — super_admin only
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("super_admin"))
		r.Post("/", h.create)
		r.Put("/{id}", h.update)
		r.Delete("/{id}", h.delete)
		r.Put("/{id}/progress", h.updateProgress)
		r.Post("/{id}/assign", h.assign)
		r.Delete("/{id}/assign/{userID}", h.unassign)
		r.Post("/{id}/files", h.addFile)
		r.Delete("/{id}/files/{fileID}", h.deleteFile)
	})
	// Budget & expenses — procurement or super_admin
	r.Group(func(r chi.Router) {
		r.Use(middleware.RequireRole("super_admin", "procurement"))
		r.Put("/{id}/budget", h.setBudget)
		r.Post("/{id}/expenses", h.addExpense)
		r.Delete("/{id}/expenses/{expenseID}", h.deleteExpense)
	})
	r.Get("/{id}/files", h.listFiles)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetRole(r)
	deptID := ""
	if role == "staff" || role == "dept_head" {
		deptID = middleware.GetDeptID(r)
	}

	projects, err := h.svc.List(r.Context(), deptID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
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
	p, err := h.svc.GetByID(r.Context(), chi.URLParam(r, "id"))
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

	p, err := h.svc.Update(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r), dto)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
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

	if err := h.svc.UpdateProgress(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r), dto); err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Project progress updated successfully")
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), chi.URLParam(r, "id")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
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

	if err := h.svc.Assign(r.Context(), chi.URLParam(r, "id"), dto.UserID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "User assigned to project successfully and added to project chat")
}

func (h *Handler) unassign(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Unassign(r.Context(), chi.URLParam(r, "id"), chi.URLParam(r, "userID")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "User unassigned from project successfully and removed from project chat")
}

func (h *Handler) assignees(w http.ResponseWriter, r *http.Request) {
	ids, err := h.svc.GetAssignees(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, ids, len(ids))
}

func (h *Handler) projectRoom(w http.ResponseWriter, r *http.Request) {
	room, err := h.svc.GetProjectRoom(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "project chat room not found")
		return
	}
	response.JSON(w, http.StatusOK, room)
}

func (h *Handler) history(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GetHistory(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) getBudget(w http.ResponseWriter, r *http.Request) {
	summary, err := h.svc.GetBudget(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
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
	if err := h.svc.SetBudget(r.Context(), chi.URLParam(r, "id"), middleware.GetUserID(r), dto); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
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
	e, err := h.svc.AddExpense(r.Context(), chi.URLParam(r, "id"), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, e)
}

func (h *Handler) deleteExpense(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteExpense(r.Context(), chi.URLParam(r, "expenseID")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Expense deleted successfully")
}

func (h *Handler) listFiles(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListFiles(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
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
	f, err := h.svc.AddFile(r.Context(), chi.URLParam(r, "id"), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, f)
}

func (h *Handler) deleteFile(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteFile(r.Context(), chi.URLParam(r, "fileID")); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "File deleted successfully")
}
