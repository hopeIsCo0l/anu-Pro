package tenant

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/auth"
	appcfg "github.com/hopeIsCo0l/anu-pro/internal/platform/config"
	"github.com/hopeIsCo0l/anu-pro/internal/platform/db"
)

type ctxKey string

const (
	userIDKey   ctxKey = "user_id"
	tenantIDKey ctxKey = "tenant_id"
	roleKey     ctxKey = "role"
)

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(userIDKey).(string)
	return v
}

func WithTenantID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, tenantIDKey, id)
}

func TenantIDFromContext(ctx context.Context) string {
	v, _ := ctx.Value(tenantIDKey).(string)
	return v
}

func WithRole(ctx context.Context, role string) context.Context {
	return context.WithValue(ctx, roleKey, role)
}

func RoleFromContext(ctx context.Context) string {
	v, _ := ctx.Value(roleKey).(string)
	return v
}

// TenantSlugFromContext returns the tenant slug set by RequireAuth.
// Delegates to db.TenantFromContext since the slug lives there.
func TenantSlugFromContext(ctx context.Context) string {
	return db.TenantFromContext(ctx)
}

// RequireAuth validates the Bearer JWT and injects tenant slug + identity into context.
func RequireAuth(cfg *appcfg.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			header := r.Header.Get("Authorization")
			if !strings.HasPrefix(header, "Bearer ") {
				writeError(w, http.StatusUnauthorized, "missing authorization header")
				return
			}
			tokenStr := strings.TrimPrefix(header, "Bearer ")

			claims, err := auth.ValidateAccessToken(cfg, tokenStr)
			if err != nil {
				if errors.Is(err, auth.ErrTokenExpired) {
					writeError(w, http.StatusUnauthorized, "token expired")
					return
				}
				writeError(w, http.StatusUnauthorized, "invalid token")
				return
			}

			ctx := db.WithTenant(r.Context(), claims.TenantSlug)
			ctx = WithUserID(ctx, claims.Subject)
			ctx = WithTenantID(ctx, claims.TenantID)
			ctx = WithRole(ctx, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func writeError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	fmt.Fprintf(w, `{"error":%q}`, msg)
}
