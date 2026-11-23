package claim

import (
	"encoding/json"
	"errors"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	reviewer := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleDeptHead))
	payer := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleProcurement))

	mux.HandleFunc("GET "+prefix, h.list)
	mux.HandleFunc("POST "+prefix, h.create)
	mux.HandleFunc("GET "+prefix+"/{id}", h.getByID)
	mux.Handle("POST "+prefix+"/{id}/approve", reviewer(h.approve))
	mux.Handle("POST "+prefix+"/{id}/reject", reviewer(h.reject))
	mux.Handle("POST "+prefix+"/{id}/pay", payer(h.markPaid))
	mux.HandleFunc("DELETE "+prefix+"/{id}", h.delete)
}

// seeAll: managers review claims and procurement pays them, so they see every claim.
func seeAll(r *http.Request) bool {
	return middleware.HasRole(r, middleware.RoleSuperAdmin, middleware.RoleDeptHead, middleware.RoleProcurement)
}

func writeErr(w http.ResponseWriter, r *http.Request, err error) {
	switch {
	case errors.Is(err, ErrNotFound):
		response.Error(w, http.StatusNotFound, err.Error())
	case errors.Is(err, ErrForbidden), errors.Is(err, ErrOwnClaim):
		response.Error(w, http.StatusForbidden, err.Error())
	case errors.Is(err, ErrWrongState):
		response.Error(w, http.StatusConflict, err.Error())
	default:
		response.ServerError(w, r, err)
	}
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), seeAll(r), middleware.GetUserID(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateClaimDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.svc.Create(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Get(r.Context(), r.PathValue("id"), middleware.GetUserID(r), seeAll(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) approve(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Approve(r.Context(), r.PathValue("id"), middleware.GetUserID(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) reject(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.Reject(r.Context(), r.PathValue("id"), middleware.GetUserID(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) markPaid(w http.ResponseWriter, r *http.Request) {
	c, err := h.svc.MarkPaid(r.Context(), r.PathValue("id"), middleware.GetUserID(r))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	err := h.svc.Delete(r.Context(), r.PathValue("id"), middleware.GetUserID(r),
		middleware.HasRole(r, middleware.RoleSuperAdmin))
	if err != nil {
		writeErr(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Claim deleted successfully")
}
