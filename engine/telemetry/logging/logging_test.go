package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	internaltracing "github.com/99souls/ariadne/engine/internal/telemetry/tracing"
)

func TestCorrelatedLoggerAddsTraceSpan(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{AddSource: false})
	base := slog.New(handler)
	log := New(base)

	tr := internaltracing.NewTracer(true)
	ctx, span := tr.StartSpan(context.Background(), "op")
	defer span.End()
	log.InfoCtx(ctx, "hello", "k", "v")
	out := buf.String()
	if !strings.Contains(out, "trace_id=") || !strings.Contains(out, "span_id=") {
		t.Fatalf("expected trace/span in log: %s", out)
	}
}

func TestCorrelatedLoggerNoSpan(t *testing.T) {
	var buf bytes.Buffer
	handler := slog.NewTextHandler(&buf, nil)
	log := New(slog.New(handler))
	log.InfoCtx(context.Background(), "plain")
	if strings.Contains(buf.String(), "trace_id=") {
		t.Fatalf("unexpected trace id present")
	}
}
