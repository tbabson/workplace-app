package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTokenQueryParamOnlyForWebSockets(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/v1/users?token=abc", nil)
	if got := TokenFromRequest(req); got != "" {
		t.Fatalf("query token accepted on a normal request: %q", got)
	}
	req.Header.Set("Upgrade", "websocket")
	if got := TokenFromRequest(req); got != "abc" {
		t.Fatalf("query token not accepted on a WebSocket handshake: %q", got)
	}
}

func TestWithRunsMiddlewareInOrder(t *testing.T) {
	var order []string
	mw := func(name string) func(http.Handler) http.Handler {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	h := With(mw("a"), mw("b"))(func(http.ResponseWriter, *http.Request) { order = append(order, "handler") })
	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/", nil))
	if got := len(order); got != 3 || order[0] != "a" || order[1] != "b" || order[2] != "handler" {
		t.Fatalf("order = %v", order)
	}
}

func TestRequireRole(t *testing.T) {
	h := RequireRole(RoleSuperAdmin)(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	for role, want := range map[string]int{RoleSuperAdmin: http.StatusTeapot, RoleStaff: http.StatusForbidden, "": http.StatusForbidden} {
		req := httptest.NewRequest("GET", "/", nil)
		req = req.WithContext(WithIdentity(req.Context(), "u", role, ""))
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != want {
			t.Errorf("role %q: got %d, want %d", role, rec.Code, want)
		}
	}
}

func TestStripSlashes(t *testing.T) {
	var got string
	h := StripSlashes(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { got = r.URL.Path }))
	for in, want := range map[string]string{"/a/b/": "/a/b", "/": "/", "/a": "/a"} {
		h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", in, nil))
		if got != want {
			t.Errorf("%q -> %q, want %q", in, got, want)
		}
	}
}
