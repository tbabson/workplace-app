package query

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
	// Anyone may submit a query to a manager; only its recipient may respond.
	mux.HandleFunc("GET "+prefix, h.list)
	mux.HandleFunc("POST "+prefix, h.create)
	mux.HandleFunc("GET "+prefix+"/{id}", h.getByID)
	mux.HandleFunc("PUT "+prefix+"/{id}/respond", h.respond)
	mux.HandleFunc("DELETE "+prefix+"/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.List(r.Context(), middleware.GetRole(r), middleware.GetUserID(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateQueryDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	q, err := h.svc.Create(r.Context(), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, q)
}

func (h *Handler) getByID(w http.ResponseWriter, r *http.Request) {
	q, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !canView(r, q) {
		response.Error(w, http.StatusNotFound, "query not found")
		return
	}
	response.JSON(w, http.StatusOK, q)
}

func (h *Handler) respond(w http.ResponseWriter, r *http.Request) {
	var dto RespondDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	q, err := h.svc.Respond(r.Context(), r.PathValue("id"), middleware.GetUserID(r), dto)
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, q)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	q, err := h.svc.GetByID(r.Context(), r.PathValue("id"))
	if err != nil || !canView(r, q) {
		response.Error(w, http.StatusNotFound, "query not found")
		return
	}
	if q.IssuerID != middleware.GetUserID(r) && !middleware.HasRole(r, middleware.RoleSuperAdmin) {
		response.Error(w, http.StatusForbidden, "only the issuer or a super admin can delete this query")
		return
	}
	if err := h.svc.Delete(r.Context(), q.ID); err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.Message(w, http.StatusOK, "Query deleted successfully")
}

// canView: a query is visible to its issuer, its recipient, and super_admins.
func canView(r *http.Request, q *Query) bool {
	userID := middleware.GetUserID(r)
	return q.IssuerID == userID || q.RecipientID == userID || middleware.HasRole(r, middleware.RoleSuperAdmin)
}
