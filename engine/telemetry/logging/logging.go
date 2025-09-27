package logging

import (
	tracing "ariadne/packages/engine/telemetry/tracing"
	"context"
	"log/slog"
)

// Logger is a minimal interface wrapper allowing correlation injection.
type Logger interface {
	InfoCtx(ctx context.Context, msg string, attrs ...any)
	ErrorCtx(ctx context.Context, msg string, attrs ...any)
}

type correlatedLogger struct{ base *slog.Logger }

// New returns a correlated Logger wrapper.
func New(base *slog.Logger) Logger {
	if base == nil {
		base = slog.Default()
	}
	return &correlatedLogger{base: base}
}

func (l *correlatedLogger) InfoCtx(ctx context.Context, msg string, attrs ...any) {
	traceID, spanID := tracing.ExtractIDs(ctx)
	if traceID != "" || spanID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID), slog.String("span_id", spanID))
	}
	l.base.InfoContext(ctx, msg, attrs...)
}

func (l *correlatedLogger) ErrorCtx(ctx context.Context, msg string, attrs ...any) {
	traceID, spanID := tracing.ExtractIDs(ctx)
	if traceID != "" || spanID != "" {
		attrs = append(attrs, slog.String("trace_id", traceID), slog.String("span_id", spanID))
	}
	l.base.ErrorContext(ctx, msg, attrs...)
}
