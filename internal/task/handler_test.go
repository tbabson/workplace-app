package task

import (
	"context"
	"errors"
	"net/http/httptest"
	"testing"

	"workplace/pkg/middleware"
)

func TestVisible(t *testing.T) {
	// Departments: "press"; "hr" sits under "admin".
	depts := map[string][2]string{"pressStaff": {"press", ""}, "hrStaff": {"hr", "admin"}, "adminStaff": {"admin", ""}}
	middleware.SetUserDepartmentLookup(func(_ context.Context, id string) (string, string, error) {
		if d, ok := depts[id]; ok {
			return d[0], d[1], nil
		}
		return "", "", errors.New("unknown")
	})
	str := func(s string) *string { return &s }

	cases := []struct {
		name, role, dept string
		task             Task
		want             bool
	}{
		{"head: own task", "dept_head", "admin", Task{AssignedTo: str("me"), CreatedBy: "boss"}, true},
		{"head: own dept staff", "dept_head", "admin", Task{AssignedTo: str("adminStaff"), CreatedBy: "boss"}, true},
		{"head: sub-dept staff", "dept_head", "admin", Task{AssignedTo: str("hrStaff"), CreatedBy: "boss"}, true},
		{"head: other dept staff", "dept_head", "admin", Task{AssignedTo: str("pressStaff"), CreatedBy: "boss"}, false},
		{"head: unassigned, not mine", "dept_head", "admin", Task{CreatedBy: "boss"}, false},
		{"head: unassigned, created by me", "dept_head", "admin", Task{CreatedBy: "me"}, true},
		{"admin sees everything", "super_admin", "", Task{AssignedTo: str("pressStaff")}, true},
		{"staff: own task", "staff", "press", Task{AssignedTo: str("me"), CreatedBy: "boss"}, true},
		{"staff: colleague's task", "staff", "press", Task{AssignedTo: str("pressStaff"), CreatedBy: "boss"}, false},
		{"staff: unassigned task", "staff", "press", Task{CreatedBy: "boss"}, false},
		{"procurement: someone else's task", "procurement", "", Task{AssignedTo: str("hrStaff"), CreatedBy: "boss"}, false},
		{"procurement: own task", "procurement", "", Task{AssignedTo: str("me"), CreatedBy: "boss"}, true},
	}
	for _, c := range cases {
		req := httptest.NewRequest("GET", "/", nil)
		req = req.WithContext(middleware.WithIdentity(req.Context(), "me", c.role, c.dept))
		task := c.task
		if got := visible(req, &task); got != c.want {
			t.Errorf("%s: got %v, want %v", c.name, got, c.want)
		}
	}
}
