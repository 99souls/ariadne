package tracing

import (
    "context"
    "testing"
    "time"
)

func TestNoopTracer(t *testing.T) {
    tr := NewTracer(false)
    if !tr.Noop() { t.Fatalf("expected noop") }
    ctx, sp := tr.StartSpan(context.Background(), "noop")
    if ctx == nil || sp == nil { t.Fatalf("expected span and ctx") }
    sp.End()
}

func TestSimpleTracerHierarchy(t *testing.T) {
    tr := NewTracer(true)
    if tr.Noop() { t.Fatalf("should be enabled") }
    ctx, root := tr.StartSpan(context.Background(), "root")
    if root.Context().TraceID == "" || root.Context().SpanID == "" { t.Fatalf("missing ids") }
    ctx2, child := tr.StartSpan(ctx, "child")
    _ = ctx2
    if child.Context().TraceID != root.Context().TraceID { t.Fatalf("trace mismatch") }
    if child.Context().ParentSpanID != root.Context().SpanID { t.Fatalf("parent mismatch") }
    child.End(); root.End()
    if !root.IsEnded() || !child.IsEnded() { t.Fatalf("expected spans ended") }
    if root.Context().End.IsZero() || child.Context().End.IsZero() { t.Fatalf("end timestamps not set") }
}

func TestSpanAttributes(t *testing.T) {
    tr := NewTracer(true)
    _, sp := tr.StartSpan(context.Background(), "work")
    sp.SetAttribute("stage", "pipeline")
    sp.SetAttribute("ok", true)
    sp.End()
    if !sp.IsEnded() { t.Fatalf("span should be ended") }
    // internal attrs not exposed yet; future iteration could expose a snapshot for tests
}

func TestSpanTimingOrder(t *testing.T) {
    tr := NewTracer(true)
    _, sp := tr.StartSpan(context.Background(), "timing")
    time.Sleep(5 * time.Millisecond)
    sp.End()
    if sp.Context().End.Before(sp.Context().Start) { t.Fatalf("end before start") }
}
