package tracing

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"sync"
	"time"
)

// Span represents an active unit of work.
// Minimal interface for Iteration 3: attributes + end time tracking.
type Span interface {
	End()
	SetAttribute(key string, value any)
	Context() SpanContext
	IsEnded() bool
}

// SpanContext carries identifiers for correlation.
type SpanContext struct {
	TraceID      string
	SpanID       string
	ParentSpanID string
	Start        time.Time
	End          time.Time
}

// Tracer creates spans.
type Tracer interface {
	StartSpan(ctx context.Context, name string) (context.Context, Span)
	Noop() bool
}

// noop implementations --------------------------------------------------------------------------------

type noopTracer struct{}

type noopSpan struct{}

func (n noopTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	return ctx, noopSpan{}
}
func (n noopTracer) Noop() bool                       { return true }
func (n noopSpan) End()                               {}
func (n noopSpan) SetAttribute(key string, value any) {}
func (n noopSpan) Context() SpanContext               { return SpanContext{} }
func (n noopSpan) IsEnded() bool                      { return true }

// simple in-process tracer -----------------------------------------------------------------------------

type simpleTracer struct{ enabled bool }

type simpleSpan struct {
	ctx   SpanContext
	mu    sync.Mutex
	ended bool
	attrs map[string]any
}

// NewTracer returns a simple in-process tracer (always enabled for now).
func NewTracer(enabled bool) Tracer {
	if !enabled {
		return noopTracer{}
	}
	return simpleTracer{enabled: true}
}

// StartSpan creates a span and stores it in the context.
func (t simpleTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	parent := SpanFromContext(ctx)
	traceID := parent.ctx.TraceID
	if traceID == "" {
		traceID = newID(16)
	}
	sp := &simpleSpan{ctx: SpanContext{TraceID: traceID, SpanID: newID(8), ParentSpanID: parent.ctx.SpanID, Start: time.Now()}, attrs: make(map[string]any)}
	ctx = context.WithValue(ctx, spanKey{}, sp)
	return ctx, sp
}
func (t simpleTracer) Noop() bool { return !t.enabled }

func (s *simpleSpan) End() {
	s.mu.Lock()
	if !s.ended {
		s.ctx.End = time.Now()
		s.ended = true
	}
	s.mu.Unlock()
}
func (s *simpleSpan) SetAttribute(key string, value any) {
	s.mu.Lock()
	if s.attrs != nil {
		s.attrs[key] = value
	}
	s.mu.Unlock()
}
func (s *simpleSpan) Context() SpanContext { return s.ctx }
func (s *simpleSpan) IsEnded() bool        { s.mu.Lock(); ended := s.ended; s.mu.Unlock(); return ended }

// context helpers -------------------------------------------------------------------------------------

type spanKey struct{}

// SpanFromContext returns the active span or a noop span if absent.
func SpanFromContext(ctx context.Context) *simpleSpan {
	if ctx == nil {
		return &simpleSpan{}
	}
	if sp, ok := ctx.Value(spanKey{}).(*simpleSpan); ok {
		return sp
	}
	return &simpleSpan{}
}

// ExtractIDs returns active trace/span ids from context (empty if none).
func ExtractIDs(ctx context.Context) (traceID, spanID string) {
	sp := SpanFromContext(ctx)
	return sp.ctx.TraceID, sp.ctx.SpanID
}

// newID generates n random bytes hex encoded (2n length string).
func newID(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
