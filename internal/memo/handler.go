package memo

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
	manager := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleDeptHead))

	mux.HandleFunc("GET "+prefix, h.list)
	mux.Handle("POST "+prefix, manager(h.create))
	mux.HandleFunc("GET "+prefix+"/{id}", h.getByID)
	mux.HandleFunc("PUT "+prefix+"/{id}/read", h.markRead)
	mux.HandleFunc("POST "+prefix+"/{id}/acknowledge", h.acknowledge)
	mux.HandleFunc("GET "+prefix+"/{id}/acknowledgements", h.listAcknowledgements)
	mux.HandleFunc("DELETE "+prefix+"/{id}", h.delete)
}

// canView mirrors ListForUser: a memo is visible to its sender, its individual
// recipient, members of its target department, and everyone when it is a broadcast.
func canView(r *http.Request, m *Memo) bool {
	userID := middleware.GetUserID(r)
	switch {
	case middleware.HasRole(r, middleware.RoleSuperAdmin), m.SenderID == userID:
		return true
	case m.RecipientID != nil:
		return *m.RecipientID == userID
	case m.DepartmentID != nil:
		return *m.DepartmentID == middleware.GetDeptID(r)
	default:
		return true
	}
}

func canManage(r *http.Request, m *Memo) bool {
	return middleware.HasRole(r, middleware.RoleSuperAdmin) || m.SenderID == middleware.GetUserID(r)
}

// load fetches the memo and writes a 404 if it doesn't exist or isn't visible.
func (h *Handler) load(w http.ResponseWriter, r *http.Request) (*Memo, bool) {
	m, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !canView(r, m) {
		response.Error(w, http.StatusNotFound, "memo not found")
		return nil, false
	}
	return m, true
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(),
		middleware.GetRole(r),
		middleware.GetUserID(r),
		middleware.GetDeptID(r),
	)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateMemoDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// Department heads may message individuals or their own department, but
	// company-wide broadcasts and other departments are for super_admin only.
	if middleware.HasRole(r, middleware.RoleDeptHead) && dto.RecipientID == nil {
		own := middleware.GetDeptID(r)
		if dto.DepartmentID == nil || *dto.DepartmentID != own || own == "" {
			response.Error(w, http.StatusForbidden, "department heads can only send memos to individuals or their own department")
			return
		}
	}

	m, err := h.svc.Create(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, m)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	m, ok := h.load(w, r)
	if !ok {
		return
	}
	response.JSON(w, http.StatusOK, m)
}

func (h *Handler) markRead(w http.ResponseWriter, r *http.Request) {
	m, ok := h.load(w, r)
	if !ok {
		return
	}
	if err := h.svc.MarkRead(r.Context(), m.ID); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Memo marked as read successfully")
}

func (h *Handler) acknowledge(w http.ResponseWriter, r *http.Request) {
	m, ok := h.load(w, r)
	if !ok {
		return
	}
	if err := h.svc.Acknowledge(r.Context(), m.ID, middleware.GetUserID(r)); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Memo acknowledged")
}

func (h *Handler) listAcknowledgements(w http.ResponseWriter, r *http.Request) {
	m, ok := h.load(w, r)
	if !ok {
		return
	}
	if !canManage(r, m) {
		response.Error(w, http.StatusForbidden, "only the sender can view acknowledgements")
		return
	}
	list, err := h.svc.GetAcknowledgements(r.Context(), m.ID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	m, ok := h.load(w, r)
	if !ok {
		return
	}
	if !canManage(r, m) {
		response.Error(w, http.StatusForbidden, "only the sender or a super admin can delete this memo")
		return
	}
	if err := h.svc.Delete(r.Context(), m.ID); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Memo deleted successfully")
}
