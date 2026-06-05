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
	CtxUserID   contextKey = "user_id"
	CtxRole     contextKey = "role"
	CtxDeptID   contextKey = "dept_id"
)

func Auth(secret string, rdb *redis.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Prefer cookie, fall back to Authorization header
			tokenStr := ""
			if cookie, err := r.Cookie("auth_token"); err == nil && cookie.Value != "" {
				tokenStr = cookie.Value
			} else {
				header := r.Header.Get("Authorization")
				if strings.HasPrefix(header, "Bearer ") {
					tokenStr = strings.TrimPrefix(header, "Bearer ")
				}
			}

			// Allow token via query param for WebSocket connections
			if tokenStr == "" {
				tokenStr = r.URL.Query().Get("token")
			}

			if tokenStr == "" {
				response.Error(w, http.StatusUnauthorized, "missing or invalid authorization header")
				return
			}

			// Check blacklist
			blacklisted, err := rdb.Get(r.Context(), "blacklist:"+tokenStr).Result()
			if err == nil && blacklisted == "1" {
				response.Error(w, http.StatusUnauthorized, "user not signed in")
				return
			}

			claims, err := utils.ParseToken(tokenStr, secret)
			if err != nil {
				response.Error(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := context.WithValue(r.Context(), CtxUserID, claims.UserID)
			ctx = context.WithValue(ctx, CtxRole, claims.Role)
			ctx = context.WithValue(ctx, CtxDeptID, claims.DepartmentID)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
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
