package asset

import (
	"encoding/json"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	manage := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleProcurement))

	mux.HandleFunc("GET "+prefix, h.list)
	mux.Handle("POST "+prefix, manage(h.create))
	mux.HandleFunc("GET "+prefix+"/{id}", h.getByID)
	mux.Handle("PUT "+prefix+"/{id}", manage(h.update))
	mux.Handle("POST "+prefix+"/{id}/assign", manage(h.assign))
	mux.Handle("POST "+prefix+"/{id}/unassign", manage(h.unassign))
	mux.Handle("DELETE "+prefix+"/{id}", manage(h.delete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), middleware.GetRole(r) == middleware.RoleStaff, middleware.GetUserID(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateAssetDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	a, err := h.svc.Create(r.Context(), dto)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, a)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || (middleware.GetRole(r) == middleware.RoleStaff && (a.AssignedTo == nil || *a.AssignedTo != middleware.GetUserID(r))) {
		response.Error(w, http.StatusNotFound, "asset not found")
		return
	}
	response.JSON(w, http.StatusOK, a)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var dto UpdateAssetDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	a, err := h.svc.Update(r.Context(), r.PathValue("id"), dto)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, a)
}

func (h *Handler) assign(w http.ResponseWriter, r *http.Request) {
	var dto AssignAssetDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if dto.UserID == "" {
		response.Error(w, http.StatusBadRequest, "user_id is required")
		return
	}
	a, err := h.svc.Assign(r.Context(), r.PathValue("id"), dto.UserID)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, a)
}

func (h *Handler) unassign(w http.ResponseWriter, r *http.Request) {
	a, err := h.svc.Unassign(r.Context(), r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, a)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Asset deleted successfully")
}
