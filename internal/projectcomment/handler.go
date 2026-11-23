package projectcomment

import (
	"encoding/json"
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

// RegisterRoutes registers comment routes. projectAccess guards routes nested
// under /projects/{projectID}.
func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string, projectAccess func(http.Handler) http.Handler) {
	inProject := middleware.With(projectAccess)

	mux.Handle("GET "+prefix+"/projects/{projectID}/comments", inProject(h.list))
	mux.Handle("POST "+prefix+"/projects/{projectID}/comments", inProject(h.create))
	mux.HandleFunc("PUT "+prefix+"/comments/{id}", h.update)
	mux.HandleFunc("DELETE "+prefix+"/comments/{id}", h.delete)
}

func (h *Handler) list(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.ListByProject(r.Context(), r.PathValue("projectID"))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, list, len(list))
}

func (h *Handler) create(w http.ResponseWriter, r *http.Request) {
	var dto CreateCommentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.svc.Create(r.Context(), r.PathValue("projectID"), dto, middleware.GetUserID(r))
	if err != nil {
		response.Error(w, http.StatusBadRequest, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, c)
}

func (h *Handler) update(w http.ResponseWriter, r *http.Request) {
	var dto UpdateCommentDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	c, err := h.svc.Update(r.Context(), r.PathValue("id"), middleware.GetUserID(r), dto)
	if err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, c)
}

func (h *Handler) delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.Delete(r.Context(), r.PathValue("id"), middleware.GetUserID(r), middleware.GetRole(r)); err != nil {
		response.Error(w, http.StatusForbidden, err.Error())
		return
	}
	response.Message(w, http.StatusOK, "Comment deleted successfully")
}
