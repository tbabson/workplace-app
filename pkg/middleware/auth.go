package middleware

import (
	"context"
	"net/http"
	"strings"

	"workplace/pkg/response"
	"workplace/pkg/utils"

	"github.com/redis/go-redis/v9"
)

type contextKey string

const (
	CtxUserID contextKey = "user_id"
	CtxRole   contextKey = "role"
	CtxDeptID contextKey = "dept_id"
)

// Roles.
const (
	RoleSuperAdmin  = "super_admin"
	RoleDeptHead    = "dept_head"
	RoleStaff       = "staff"
	RoleProcurement = "procurement"
)

// Identity is the current state of an authenticated user, loaded on every request
// so that deactivation and role changes take effect immediately rather than
// when the JWT expires.
type Identity struct {
	Role         string
	DepartmentID string
	Active       bool
}

// IdentityLookup loads the current identity for a user ID.
type IdentityLookup func(ctx context.Context, userID string) (*Identity, error)

// TokenFromRequest extracts the JWT from the auth cookie, the Authorization
// header, or (for WebSocket handshakes only) the "token" query parameter.
func TokenFromRequest(r *http.Request) string {
	if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
		return cookie.Value
	}
	if header := r.Header.Get("Authorization"); strings.HasPrefix(header, "Bearer ") {
		return strings.TrimPrefix(header, "Bearer ")
	}
	// Browsers can't set headers on WebSocket handshakes, so allow a query
	// param there only — never on normal requests, where it would leak into logs.
	if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		return r.URL.Query().Get("token")
	}
	return ""
}

func Auth(secret string, rdb *redis.Client, lookup IdentityLookup) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := TokenFromRequest(r)
			if tokenStr == "" {
				response.Error(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			claims, err := utils.ParseToken(tokenStr, secret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}

			// Check blacklist. Fail closed: if Redis is unreachable we can't tell
			// whether the token was revoked.
			blacklisted, err := rdb.Exists(r.Context(), "blacklist:"+tokenStr).Result()
			if err != nil {
				response.Error(w, http.StatusServiceUnavailable, "authentication temporarily unavailable")
				return
			}
			if blacklisted > 0 {
				response.Error(w, http.StatusUnauthorized, "user not signed in")
				return
			}

			id, err := lookup(r.Context(), claims.UserID)
			if err != nil || !id.Active {
				response.Error(w, http.StatusUnauthorized, "account is not active")
				return
			}

			next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), claims.UserID, id.Role, id.DepartmentID)))
		})
	}
}

// WithIdentity stores the caller's identity in the context. Used by Auth and by tests.
func WithIdentity(ctx context.Context, userID, role, deptID string) context.Context {
	ctx = context.WithValue(ctx, CtxUserID, userID)
	ctx = context.WithValue(ctx, CtxRole, role)
	return context.WithValue(ctx, CtxDeptID, deptID)
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	allowed := make(map[string]bool, len(roles))
	for _, r := range roles {
		allowed[r] = true
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, _ := r.Context().Value(CtxRole).(string)
			if !allowed[role] {
				response.Error(w, http.StatusForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// HasRole reports whether the caller has one of the given roles.
func HasRole(r *http.Request, roles ...string) bool {
	role := GetRole(r)
	for _, want := range roles {
		if role == want {
			return true
		}
	}
	return false
}

// IsManager reports whether the caller is a super_admin or dept_head.
func IsManager(r *http.Request) bool {
	return HasRole(r, RoleSuperAdmin, RoleDeptHead)
}

func GetUserID(r *http.Request) string {
	id, _ := r.Context().Value(CtxUserID).(string)
	return id
}

func GetRole(r *http.Request) string {
	role, _ := r.Context().Value(CtxRole).(string)
	return role
}

func GetDeptID(r *http.Request) string {
	id, _ := r.Context().Value(CtxDeptID).(string)
	return id
}
