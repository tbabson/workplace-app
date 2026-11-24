package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

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

const someID = "11111111-1111-1111-1111-111111111111"

// testRouter builds the real router with nil services. Every request below is
// expected to be rejected by a permission check before any service is touched.
func testRouter() http.Handler {
	allowAll := func(*http.Request) bool { return true }
	h := handlers{
		auth:       auth.NewHandler(nil, false),
		user:       user.NewHandler(nil),
		dept:       department.NewHandler(nil),
		project:    project.NewHandler(nil),
		task:       task.NewHandler(nil),
		milestone:  milestone.NewHandler(nil),
		comment:    projectcomment.NewHandler(nil),
		attendance: attendance.NewHandler(nil),
		overtime:   overtime.NewHandler(nil),
		chat:       chat.NewHandler(nil, nil, allowAll),
		memo:       memo.NewHandler(nil),
		query:      query.NewHandler(nil),
		notif:      notification.NewHandler(nil, nil, allowAll),
		leave:      leave.NewHandler(nil),
		review:     review.NewHandler(nil),
		claim:      claim.NewHandler(nil),
		asset:      asset.NewHandler(nil),
		report:     report.NewHandler(nil),
	}
	// Fake auth: identity comes from test headers.
	fakeAuth := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := middleware.WithIdentity(r.Context(), r.Header.Get("X-User"), r.Header.Get("X-Role"), "")
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
	return newRouter(h, fakeAuth, []string{"http://localhost:5173"})
}

func TestRolePermissions(t *testing.T) {
	router := testRouter()

	cases := []struct {
		method, path string
		roles        []string // roles that must be refused
		body         string
	}{
		// Users & departments: super_admin only.
		{"POST", "/api/v1/users", []string{"staff", "dept_head", "procurement"}, `{"role":"super_admin"}`},
		{"DELETE", "/api/v1/users/" + someID, []string{"staff", "dept_head", "procurement"}, ""},
		{"PUT", "/api/v1/users/" + someID, []string{"staff", "dept_head", "procurement"}, `{"is_active":false}`},
		{"POST", "/api/v1/departments", []string{"staff", "dept_head", "procurement"}, `{}`},
		{"PUT", "/api/v1/departments/" + someID, []string{"staff", "dept_head", "procurement"}, `{}`},
		{"DELETE", "/api/v1/departments/" + someID, []string{"staff", "dept_head", "procurement"}, ""},

		// Claims: managers review, super_admin/procurement pay.
		{"POST", "/api/v1/claims/" + someID + "/approve", []string{"staff", "procurement"}, ""},
		{"POST", "/api/v1/claims/" + someID + "/reject", []string{"staff", "procurement"}, ""},
		{"POST", "/api/v1/claims/" + someID + "/pay", []string{"staff", "dept_head"}, ""},

		// Leave.
		{"PUT", "/api/v1/leaves/" + someID + "/approve", []string{"staff", "procurement"}, ""},
		{"PUT", "/api/v1/leaves/" + someID + "/reject", []string{"staff", "procurement"}, ""},
		{"POST", "/api/v1/leaves/balance/" + someID, []string{"staff", "dept_head", "procurement"}, `{}`},
		{"GET", "/api/v1/leaves/balance/" + someID, []string{"staff", "procurement"}, ""},

		// Overtime.
		{"POST", "/api/v1/overtime", []string{"staff", "procurement"}, `{}`},
		{"PUT", "/api/v1/overtime/" + someID, []string{"staff", "procurement"}, `{}`},
		{"PUT", "/api/v1/overtime/" + someID + "/mark-paid", []string{"staff", "procurement"}, ""},
		{"DELETE", "/api/v1/overtime/" + someID, []string{"staff", "procurement"}, ""},

		// Reviews.
		{"POST", "/api/v1/reviews", []string{"staff", "procurement"}, `{}`},
		{"PUT", "/api/v1/reviews/" + someID, []string{"staff", "procurement"}, `{}`},
		{"DELETE", "/api/v1/reviews/" + someID, []string{"staff", "procurement"}, ""},

		// Assets: super_admin/procurement manage.
		{"POST", "/api/v1/assets", []string{"staff", "dept_head"}, `{}`},
		{"PUT", "/api/v1/assets/" + someID, []string{"staff", "dept_head"}, `{}`},
		{"POST", "/api/v1/assets/" + someID + "/assign", []string{"staff", "dept_head"}, `{}`},
		{"DELETE", "/api/v1/assets/" + someID, []string{"staff", "dept_head"}, ""},

		// Memos: managers send.
		{"POST", "/api/v1/memos", []string{"staff", "procurement"}, `{}`},

		// Reports & attendance history: managers.
		{"GET", "/api/v1/reports/attendance", []string{"staff"}, ""},
		{"GET", "/api/v1/reports/attendance/export", []string{"staff"}, ""},
		{"GET", "/api/v1/reports/overtime", []string{"staff"}, ""},
		{"GET", "/api/v1/reports/overtime/export", []string{"staff"}, ""},
		{"GET", "/api/v1/attendance", []string{"staff", "procurement"}, ""},
		{"GET", "/api/v1/attendance/user/" + someID, []string{"staff", "procurement"}, ""},
		{"PUT", "/api/v1/attendance/" + someID + "/sign-out", []string{"staff", "procurement"}, ""},

		// Projects, tasks, milestones.
		{"POST", "/api/v1/projects", []string{"staff", "dept_head", "procurement"}, `{}`},
		{"PUT", "/api/v1/projects/" + someID + "/budget", []string{"staff", "dept_head"}, `{}`},
		{"POST", "/api/v1/projects/" + someID + "/tasks", []string{"staff", "dept_head", "procurement"}, `{}`},
		{"DELETE", "/api/v1/tasks/" + someID, []string{"staff", "dept_head", "procurement"}, ""},
		{"POST", "/api/v1/projects/" + someID + "/milestones", []string{"staff", "dept_head", "procurement"}, `{}`},
	}

	for _, c := range cases {
		for _, role := range c.roles {
			req := httptest.NewRequest(c.method, c.path, strings.NewReader(c.body))
			req.Header.Set("X-Role", role)
			req.Header.Set("X-User", "22222222-2222-2222-2222-222222222222")
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)
			if rec.Code != http.StatusForbidden {
				t.Errorf("%s %s as %s: got %d, want 403", c.method, c.path, role, rec.Code)
			}
		}
	}
}

func TestSelfServiceUserUpdateRejectsOthers(t *testing.T) {
	router := testRouter()
	req := httptest.NewRequest("PUT", "/api/v1/users/"+someID, strings.NewReader(`{"name":"x"}`))
	req.Header.Set("X-Role", "dept_head")
	req.Header.Set("X-User", "22222222-2222-2222-2222-222222222222")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("dept_head editing another user: got %d, want 403", rec.Code)
	}
}

func TestCORSAllowlist(t *testing.T) {
	router := testRouter()

	for _, tc := range []struct {
		origin  string
		allowed bool
	}{
		{"http://localhost:5173", true},
		{"https://evil.example", false},
	} {
		req := httptest.NewRequest("GET", "/health", nil)
		req.Header.Set("Origin", tc.origin)
		rec := httptest.NewRecorder()
		router.ServeHTTP(rec, req)
		got := rec.Header().Get("Access-Control-Allow-Origin")
		if tc.allowed && got != tc.origin {
			t.Errorf("origin %s: Allow-Origin = %q, want it echoed", tc.origin, got)
		}
		if !tc.allowed && got != "" {
			t.Errorf("origin %s: Allow-Origin = %q, want empty", tc.origin, got)
		}
	}
}

func TestBodyLimit(t *testing.T) {
	router := testRouter()
	big := `{"email":"` + strings.Repeat("a", maxBodyBytes+1) + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(big))
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("oversized body: got %d, want 400", rec.Code)
	}
}
