package overtime

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
	validate := middleware.ValidateUUIDParams("id")
	validID := middleware.With(validate)
	manager := middleware.With(validate, middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleDeptHead))

	mux.HandleFunc("GET "+prefix, h.list)
	mux.Handle("POST "+prefix, manager(h.create))
	mux.Handle("GET "+prefix+"/{id}", validID(h.getByID))
	mux.Handle("PUT "+prefix+"/{id}", manager(h.update))
	mux.Handle("PUT "+prefix+"/{id}/mark-paid", manager(h.markPaid))
	mux.Handle("DELETE "+prefix+"/{id}", manager(h.delete))
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID := ""
	if middleware.GetRole(r) == middleware.RoleStaff {
		userID = middleware.GetUserID(r)
	}

	list, err := h.svc.List(r.Context(), userID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateOvertimeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	o, err := h.svc.Create(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, o)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || (middleware.GetRole(r) == middleware.RoleStaff && o.UserID != middleware.GetUserID(r)) {
		response.Error(w, http.StatusNotFound, "overtime record not found")
		return
	}
	response.JSON(w, http.StatusOK, o)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var dto UpdateOvertimeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	o, err := h.svc.Update(r.Context(), r.PathValue("id"), dto)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, o)
}

func (h *Handler) markPaid(w http.ResponseWriter, r *http.Request) {
	o, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil {
		response.Error(w, http.StatusNotFound, "overtime record not found")
		return
	}
	if o.UserID == middleware.GetUserID(r) {
		response.Error(w, http.StatusForbidden, "you cannot mark your own overtime as paid")
		return
	}
	if err := h.svc.MarkPaid(r.Context(), r.PathValue("id")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Overtime marked as paid successfully")
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id")); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Overtime record deleted successfully")
}
