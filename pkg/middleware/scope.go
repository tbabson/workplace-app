package middleware

import (
	"context"
	"net/http"
	"sync/atomic"
)

// noDepartment matches no department. A dept_head without a department is
// scoped to it, so they see nobody's records rather than everybody's.
const noDepartment = "00000000-0000-0000-0000-000000000000"

// ScopeDept returns the department the caller's view of staff records is
// limited to, or "" when it isn't limited by department. Department heads see
// only their own department's staff.
func ScopeDept(r *http.Request) string {
	if !HasRole(r, RoleDeptHead) {
		return ""
	}
	if d := GetDeptID(r); d != "" {
		return d
	}
	return noDepartment
}

// UserDepartmentLookup returns a user's department ID ("" if none).
type UserDepartmentLookup func(ctx context.Context, userID string) (string, error)

var userDept atomic.Pointer[UserDepartmentLookup]

// SetUserDepartmentLookup installs the lookup used by CanAccessUser. Call once at startup.
func SetUserDepartmentLookup(fn UserDepartmentLookup) { userDept.Store(&fn) }

// CanAccessUser reports whether the caller may see another user's personal
// records (attendance, leave, claims, overtime, reviews):
//   - everyone may see their own;
//   - super_admin may see everyone's;
//   - dept_head may see staff in their own department only;
//   - everyone else (staff, procurement) may see only their own.
//
// Procurement's finance access (claims, overtime) is granted separately where it applies.
func CanAccessUser(r *http.Request, targetUserID string) bool {
	if targetUserID != "" && targetUserID == GetUserID(r) {
		return true
	}
	switch GetRole(r) {
	case RoleSuperAdmin:
		return true
	case RoleDeptHead:
		myDept := GetDeptID(r)
		lookup := userDept.Load()
		if myDept == "" || lookup == nil {
			return false
		}
		dept, err := (*lookup)(r.Context(), targetUserID)
		return err == nil && dept == myDept
	default:
		return false
	}
}
