package db

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/config"
)

type ctxKey string

const tenantKey ctxKey = "tenant_slug"

// slugRe accepts lowercase alphanumeric + hyphens, 3-50 chars.
// Enforced here to guard against SQL injection in search_path.
var slugRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,48}[a-z0-9]$`)

// WithTenant returns a context carrying the tenant slug.
func WithTenant(ctx context.Context, slug string) context.Context {
	return context.WithValue(ctx, tenantKey, slug)
}

// TenantFromContext returns the tenant slug in ctx, or "" if absent.
func TenantFromContext(ctx context.Context) string {
	slug, _ := ctx.Value(tenantKey).(string)
	return slug
}

// New creates a pgxpool.Pool that switches search_path on every connection
// acquisition based on the tenant slug stored in the context, and resets it
// on release to prevent cross-tenant leaks.
func New(cfg *config.Config) (*pgxpool.Pool, error) {
	poolCfg, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("parse database url: %w", err)
	}

	poolCfg.MaxConns = 25
	poolCfg.MinConns = 2
	poolCfg.HealthCheckPeriod = 30 * time.Second
	poolCfg.MaxConnIdleTime = 5 * time.Minute

	poolCfg.BeforeAcquire = func(ctx context.Context, conn *pgx.Conn) bool {
		slug := TenantFromContext(ctx)

		var searchPath, tenantSetting string
		if slug != "" {
			if !slugRe.MatchString(slug) {
				slog.ErrorContext(ctx, "invalid tenant slug in context", "slug", slug)
				return false
			}
			// pgx.Identifier.Sanitize() double-quotes the name, handling hyphens safely.
			schema := pgx.Identifier{"tenant_" + slug}.Sanitize()
			searchPath = schema + ", public"
			tenantSetting = slug
		} else {
			searchPath = "public"
			tenantSetting = ""
		}

		_, err := conn.Exec(ctx, fmt.Sprintf(
			"SET search_path TO %s; SET app.tenant_slug = '%s'",
			searchPath, tenantSetting,
		))
		if err != nil {
			slog.ErrorContext(ctx, "BeforeAcquire: SET search_path failed", "err", err)
			return false
		}
		return true
	}

	poolCfg.AfterRelease = func(conn *pgx.Conn) bool {
		_, err := conn.Exec(context.Background(),
			"SET search_path TO public; SET app.tenant_slug = ''",
		)
		if err != nil {
			slog.Error("AfterRelease: reset search_path failed", "err", err)
			return false
		}
		return true
	}

	pool, err := pgxpool.NewWithConfig(context.Background(), poolCfg)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	return pool, nil
}
