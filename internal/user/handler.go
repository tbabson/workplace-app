package user

import (
	"encoding/json"
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
	admin := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin))

	mux.HandleFunc("GET "+prefix, h.list)
	mux.Handle("POST "+prefix, admin(h.create))
	mux.HandleFunc("GET "+prefix+"/me", h.me)
	mux.HandleFunc("GET "+prefix+"/{id}", h.getByID)
	mux.HandleFunc("PUT "+prefix+"/{id}", h.update)
	mux.Handle("DELETE "+prefix+"/{id}", admin(h.delete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.List(r.Context())
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, users, len(users))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateUserDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	u, err := h.svc.Create(r.Context(), dto)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, u)
}

func (h *Handler) me(w http.ResponseWriter, r *http.Request) {
	id := middleware.GetUserID(r)
	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}
	response.JSON(w, http.StatusOK, u)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	u, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}
	response.JSON(w, http.StatusOK, u)
}

// update: super_admin may change anything. Everyone else may only edit their own
// name and position — not role, department or active status.
func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	isAdmin := middleware.HasRole(r, middleware.RoleSuperAdmin)

	if !isAdmin && middleware.GetUserID(r) != id {
		response.Error(w, http.StatusForbidden, "cannot update another user")
		return
	}

	var dto UpdateUserDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if !isAdmin {
		dto.Role = nil
		dto.DepartmentID = nil
		dto.DepartmentName = nil
		dto.IsActive = nil
	} else if id == middleware.GetUserID(r) {
		// Stop an admin from locking themselves out.
		if (dto.IsActive != nil && !*dto.IsActive) || (dto.Role != nil && *dto.Role != RoleSuperAdmin) {
			response.Error(w, http.StatusBadRequest, "you cannot deactivate or demote your own account")
			return
		}
	}

	u, err := h.svc.Update(r.Context(), id, dto)
	if err != nil {
		if response.NotFound(err) {
			response.Error(w, http.StatusNotFound, "user not found")
			return
		}
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, u)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == middleware.GetUserID(r) {
		response.Error(w, http.StatusBadRequest, "you cannot delete your own account")
		return
	}
	if err := h.svc.Delete(r.Context(), id); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "User deleted successfully")
}
