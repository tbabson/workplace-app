package middleware

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"
)

func TestScopeDept(t *testing.T) {
	for _, c := range []struct{ role, dept, want string }{
		{RoleSuperAdmin, "d1", ""},
		{RoleProcurement, "d1", ""},
		{RoleDeptHead, "d1", "d1"},
		{RoleDeptHead, "", noDepartment}, // no department → sees nobody
	} {
		req := httptest.NewRequest("GET", "/", nil)
		req = req.WithContext(WithIdentity(req.Context(), "me", c.role, c.dept))
		if got := ScopeDept(req); got != c.want {
			t.Errorf("%s/%q: got %q, want %q", c.role, c.dept, got, c.want)
		}
	}
}

func TestCanAccessUser(t *testing.T) {
	// carol is in "hr", a sub-department of "admin".
	depts := map[string][2]string{"alice": {"sales", ""}, "bob": {"press", ""}, "carol": {"hr", "admin"}, "dave": {"admin", ""}}
	SetUserDepartmentLookup(func(_ context.Context, id string) (string, string, error) {
		if d, ok := depts[id]; ok {
			return d[0], d[1], nil
		}
		return "", "", errors.New("no such user")
	})

	cases := []struct {
		name, role, dept, target string
		want                     bool
	}{
		{"self", RoleStaff, "sales", "me", true},
		{"staff → colleague", RoleStaff, "sales", "alice", false},
		{"dept head → own dept", RoleDeptHead, "sales", "alice", true},
		{"dept head → other dept", RoleDeptHead, "sales", "bob", false},
		{"parent dept head → sub-dept staff", RoleDeptHead, "admin", "carol", true},
		{"sub-dept head → own staff", RoleDeptHead, "hr", "carol", true},
		{"sub-dept head → parent dept staff", RoleDeptHead, "hr", "dave", false},
		{"dept head without dept", RoleDeptHead, "", "alice", false},
		{"dept head → unknown user", RoleDeptHead, "sales", "ghost", false},
		{"super admin → anyone", RoleSuperAdmin, "", "bob", true},
		{"procurement → others", RoleProcurement, "", "alice", false},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/", nil)
		req = req.WithContext(WithIdentity(req.Context(), "me", c.role, c.dept))
		if got := CanAccessUser(req, c.target); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
