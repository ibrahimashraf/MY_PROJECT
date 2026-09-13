package httputil

import (
	"context"
	"log/slog"
)

type loggerContextKey struct{}

// WithLogger injects a contextual *slog.Logger into the context.
func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey{}, logger)
}

// LoggerFromContext retrieves the scoped logger from context, or returns slog.Default().
func LoggerFromContext(ctx context.Context) *slog.Logger {
	if ctx == nil {
		return slog.Default()
	}
	if l, ok := ctx.Value(loggerContextKey{}).(*slog.Logger); ok && l != nil {
		return l
	}
	return slog.Default()
}

// ScopedLogger returns a logger pre-enriched with common tenant and operational metadata using logger.With().
func ScopedLogger(ctx context.Context, tenantID, orgID, subsystem string, attrs ...any) *slog.Logger {
	base := LoggerFromContext(ctx)
	fields := []any{
		"subsystem", subsystem,
	}
	if tenantID != "" {
		fields = append(fields, "tenant_id", tenantID)
	}
	if orgID != "" {
		fields = append(fields, "org_id", orgID)
	}
	fields = append(fields, attrs...)
	return base.With(fields...)
}
