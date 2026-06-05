package report

import (
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"

	"github.com/go-chi/chi/v5"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Get("/dashboard", h.dashboard)
	r.Get("/attendance", h.attendance)
	r.Get("/attendance/export", h.attendanceCSV)
	r.Get("/overtime", h.overtime)
	r.Get("/overtime/export", h.overtimeCSV)
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	role := middleware.GetRole(r)
	deptID := middleware.GetDeptID(r)
	data, err := h.svc.Dashboard(r.Context(), userID, role, deptID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, data)
}

func (h *Handler) attendance(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	rows, err := h.svc.AttendanceReport(r.Context(), from, to)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rows, len(rows))
}

func (h *Handler) attendanceCSV(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	data, err := h.svc.AttendanceCSV(r.Context(), from, to)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=attendance.csv")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (h *Handler) overtime(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	rows, err := h.svc.OvertimeReport(r.Context(), from, to)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, rows, len(rows))
}

func (h *Handler) overtimeCSV(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	data, err := h.svc.OvertimeCSV(r.Context(), from, to)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=overtime.csv")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
