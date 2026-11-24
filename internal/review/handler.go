package review

import (
	"encoding/json"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	manager := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleDeptHead))

	mux.HandleFunc("GET "+prefix, h.list)
	mux.Handle("POST "+prefix, manager(h.create))
	mux.HandleFunc("GET "+prefix+"/{id}", h.getByID)
	mux.Handle("PUT "+prefix+"/{id}", manager(h.update))
	mux.Handle("DELETE "+prefix+"/{id}", manager(h.delete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, deptID := middleware.GetUserID(r), ""
	if middleware.IsManager(r) {
		userID, deptID = "", middleware.ScopeDept(r)
	}
	list, err := h.svc.List(r.Context(), userID, deptID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateReviewDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if dto.UserID == middleware.GetUserID(r) {
		response.Error(w, http.StatusForbidden, "you cannot review yourself")
		return
	}
	if !middleware.CanAccessUser(r, dto.UserID) {
		response.Error(w, http.StatusForbidden, "you can only review staff in your department")
		return
	}
	rv, err := h.svc.Create(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, rv)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	rv, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !middleware.CanAccessUser(r, rv.UserID) {
		response.Error(w, http.StatusNotFound, "review not found")
		return
	}
	response.JSON(w, http.StatusOK, rv)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	existing, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !middleware.CanAccessUser(r, existing.UserID) {
		response.Error(w, http.StatusNotFound, "review not found")
		return
	}
	if existing.UserID == middleware.GetUserID(r) {
		response.Error(w, http.StatusForbidden, "you cannot edit a review of yourself")
		return
	}

	var dto UpdateReviewDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rv, err := h.svc.Update(r.Context(), existing.ID, dto)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rv)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	existing, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !middleware.CanAccessUser(r, existing.UserID) {
		response.Error(w, http.StatusNotFound, "review not found")
		return
	}
	if existing.UserID == middleware.GetUserID(r) {
		response.Error(w, http.StatusForbidden, "you cannot delete a review of yourself")
		return
	}
	if err := h.svc.Delete(r.Context(), existing.ID); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Review deleted successfully")
}
