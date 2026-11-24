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

// canSee: super_admin and procurement (overtime is a payroll matter) see all
// records; dept heads see their own department's; staff see their own.
func canSee(r *http.Request, o *Overtime) bool {
	return middleware.HasRole(r, middleware.RoleProcurement) || middleware.CanAccessUser(r, o.UserID)
}

// load returns the record in {id} if the caller may see it, else writes 404.
func (h *Handler) load(w http.ResponseWriter, r *http.Request) (*Overtime, bool) {
	o, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !canSee(r, o) {
		response.Error(w, http.StatusNotFound, "overtime record not found")
		return nil, false
	}
	return o, true
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	userID, deptID := middleware.GetUserID(r), ""
	switch {
	case middleware.HasRole(r, middleware.RoleSuperAdmin, middleware.RoleProcurement):
		userID = ""
	case middleware.HasRole(r, middleware.RoleDeptHead):
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
	var dto CreateOvertimeDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	// Dept heads may only log overtime for staff in their own department.
	if dto.UserID != "" && !middleware.CanAccessUser(r, dto.UserID) {
		response.Error(w, http.StatusForbidden, "you can only log overtime for staff in your department")
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
	o, ok := h.load(w, r)
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, o)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	if _, ok := h.load(w, r); !ok {
		return
	}
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
	o, ok := h.load(w, r)
	if !ok {
		return
	}
	if o.UserID == middleware.GetUserID(r) {
		response.Error(w, http.StatusForbidden, "you cannot mark your own overtime as paid")
		return
	}
	if err := h.svc.MarkPaid(r.Context(), o.ID); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Overtime marked as paid successfully")
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	o, ok := h.load(w, r)
	if !ok {
		return
	}
	if err := h.svc.Delete(r.Context(), o.ID); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Overtime record deleted successfully")
}
