package report

import (
	"net/http"

	"workplace/pkg/middleware"
	"workplace/pkg/response"
)

type Handler struct{ svc *Service }

func NewHandler(svc *Service) *Handler { return &Handler{svc: svc} }

func (h *Handler) RegisterRoutes(mux *http.ServeMux, prefix string) {
	// Reports: managers, plus procurement (which handles overtime and payments).
	manager := middleware.With(middleware.RequireRole(middleware.RoleSuperAdmin, middleware.RoleDeptHead, middleware.RoleProcurement))

	mux.HandleFunc("GET "+prefix+"/dashboard", h.dashboard)
	mux.Handle("GET "+prefix+"/attendance", manager(h.attendance))
	mux.Handle("GET "+prefix+"/attendance/export", manager(h.attendanceCSV))
	mux.Handle("GET "+prefix+"/overtime", manager(h.overtime))
	mux.Handle("GET "+prefix+"/overtime/export", manager(h.overtimeCSV))
}

func (h *Handler) dashboard(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r)
	role := middleware.GetRole(r)
	deptID := middleware.GetDeptID(r)
	data, err := h.svc.Dashboard(r.Context(), userID, role, deptID)
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, data)
}

func (h *Handler) attendance(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	rows, err := h.svc.AttendanceReport(r.Context(), from, to, middleware.ScopeDept(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, rows, len(rows))
}

func (h *Handler) attendanceCSV(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	data, err := h.svc.AttendanceCSV(r.Context(), from, to, middleware.ScopeDept(r))
	if err != nil {
		response.ServerError(w, r, err)
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
	rows, err := h.svc.OvertimeReport(r.Context(), from, to, middleware.ScopeDept(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	response.JSON(w, http.StatusOK, rows, len(rows))
}

func (h *Handler) overtimeCSV(w http.ResponseWriter, r *http.Request) {
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	data, err := h.svc.OvertimeCSV(r.Context(), from, to, middleware.ScopeDept(r))
	if err != nil {
		response.ServerError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=overtime.csv")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}
