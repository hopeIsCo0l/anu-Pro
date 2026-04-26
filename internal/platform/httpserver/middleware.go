package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/hopeIsCo0l/anu-pro/internal/platform/db"
)

// RequestLogger is a chi middleware that logs every request via slog,
// including request_id and tenant_id when present.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

		defer func() {
			attrs := []slog.Attr{
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", ww.Status()),
				slog.Int64("latency_ms", time.Since(start).Milliseconds()),
				slog.String("request_id", middleware.GetReqID(r.Context())),
			}
			if slug := db.TenantFromContext(r.Context()); slug != "" {
				attrs = append(attrs, slog.String("tenant", slug))
			}
			slog.LogAttrs(r.Context(), slog.LevelInfo, "request", attrs...)
		}()

		next.ServeHTTP(ww, r)
	})
}
