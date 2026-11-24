package user

import (
	"context"
	"testing"
)

func TestDepartmentChangedHook(t *testing.T) {
	str := func(s string) *string { return &s }
	user := func(dept *string, active bool) *User {
		return &User{ID: "u1", DepartmentID: dept, IsActive: active}
	}

	cases := []struct {
		name           string
		before, after  *User
		wantCall       bool
		wantFrom, want string
	}{
		{"created in a department", nil, user(str("a"), true), true, "", "a"},
		{"created without a department", nil, user(nil, true), false, "", ""},
		{"moved between departments", user(str("a"), true), user(str("b"), true), true, "a", "b"},
		{"removed from department", user(str("a"), true), user(nil, true), true, "a", ""},
		{"unchanged", user(str("a"), true), user(str("a"), true), false, "", ""},
		{"deactivated", user(str("a"), true), user(str("a"), false), true, "a", ""},
		{"reactivated", user(str("a"), false), user(str("a"), true), true, "", "a"},
		{"moved while inactive", user(str("a"), false), user(str("b"), false), false, "", ""},
	}
	for _, c := range cases {
		var called bool
		var from, to string
		s := &Service{}
		s.OnDepartmentChanged(func(_ context.Context, _ string, f, t string) { called, from, to = true, f, t })
		s.departmentChanged(context.Background(), c.before, c.after)
		if called != c.wantCall || from != c.wantFrom || to != c.want {
			t.Errorf("%s: called=%v from=%q to=%q; want called=%v from=%q to=%q",
				c.name, called, from, to, c.wantCall, c.wantFrom, c.want)
		}
	}
}
