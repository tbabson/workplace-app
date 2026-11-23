package main

import (
	"net/http"

	"workplace/internal/asset"
	"workplace/internal/attendance"
	"workplace/internal/auth"
	"workplace/internal/chat"
	"workplace/internal/claim"
	"workplace/internal/department"
	"workplace/internal/leave"
	"workplace/internal/memo"
	"workplace/internal/milestone"
	"workplace/internal/notification"
	"workplace/internal/overtime"
	"workplace/internal/project"
	"workplace/internal/projectcomment"
	"workplace/internal/query"
	"workplace/internal/report"
	"workplace/internal/review"
	"workplace/internal/task"
	"workplace/internal/user"
	"workplace/pkg/middleware"
)

const (
	apiPrefix    = "/api/v1"
	maxBodyBytes = 1 << 20 // 1 MiB
)

type handlers struct {
	auth       *auth.Handler
	user       *user.Handler
	dept       *department.Handler
	project    *project.Handler
	task       *task.Handler
	milestone  *milestone.Handler
	comment    *projectcomment.Handler
	attendance *attendance.Handler
	overtime   *overtime.Handler
	chat       *chat.Handler
	memo       *memo.Handler
	query      *query.Handler
	notif      *notification.Handler
	leave      *leave.Handler
	review     *review.Handler
	claim      *claim.Handler
	asset      *asset.Handler
	report     *report.Handler
}

// newRouter wires every route. authMw authenticates the caller and stores their
// identity in the request context for everything under /api/v1 except /auth.
func newRouter(h handlers, authMw func(http.Handler) http.Handler, allowedOrigins []string) http.Handler {
	const api = apiPrefix

	// Public routes.
	mux := http.NewServeMux()

	mux.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	h.auth.RegisterRoutes(mux, api+"/auth")

	// Authenticated routes: everything else under /api/v1 goes through authMw.
	// The public mux prefers its more specific patterns (e.g. /api/v1/auth/login),
	// so those are never shadowed by this catch-all.
	protected := http.NewServeMux()
	projectAccess := h.project.RequireAccess("projectID")

	h.user.RegisterRoutes(protected, api+"/users")
	h.dept.RegisterRoutes(protected, api+"/departments")
	h.project.RegisterRoutes(protected, api+"/projects")
	h.task.RegisterRoutes(protected, api, projectAccess)
	h.milestone.RegisterRoutes(protected, api, projectAccess)
	h.comment.RegisterRoutes(protected, api, projectAccess)
	h.attendance.RegisterRoutes(protected, api+"/attendance")
	h.overtime.RegisterRoutes(protected, api+"/overtime")
	h.chat.RegisterRoutes(protected, api+"/chat")
	h.memo.RegisterRoutes(protected, api+"/memos")
	h.query.RegisterRoutes(protected, api+"/queries")
	h.notif.RegisterRoutes(protected, api+"/notifications")
	h.leave.RegisterRoutes(protected, api+"/leaves")
	h.review.RegisterRoutes(protected, api+"/reviews")
	h.claim.RegisterRoutes(protected, api+"/claims")
	h.asset.RegisterRoutes(protected, api+"/assets")
	h.report.RegisterRoutes(protected, api+"/reports")

	mux.Handle(api+"/", authMw(protected))

	// Global middleware, outermost first.
	return middleware.Recoverer(
		middleware.Logger(
			middleware.CORS(allowedOrigins)(
				middleware.MaxBodyBytes(maxBodyBytes)(
					middleware.StripSlashes(mux),
				),
			),
		),
	)
}
