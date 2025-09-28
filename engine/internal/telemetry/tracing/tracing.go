package tracing

// INTERNAL: moved from engine/telemetry/tracing; simple adaptive tracer retained for internal spans.

import (
	"context"
	randcrypto "crypto/rand"
	"encoding/hex"
	"math/rand"
	"sync"
	"time"
)

type Span interface { End(); SetAttribute(key string, value any); Context() SpanContext; IsEnded() bool }

type SpanContext struct { TraceID, SpanID, ParentSpanID string; Start, End time.Time }

type Tracer interface { StartSpan(ctx context.Context, name string) (context.Context, Span); Noop() bool }

type noopTracer struct{}

type noopSpan struct{}

func (n noopTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) { return ctx, noopSpan{} }
func (n noopTracer) Noop() bool { return true }
func (n noopSpan) End() {} ; func (n noopSpan) SetAttribute(key string, value any) {}; func (n noopSpan) Context() SpanContext { return SpanContext{} }; func (n noopSpan) IsEnded() bool { return true }

type simpleTracer struct{ enabled bool }

type adaptiveTracer struct { policyFn func() float64 }

type simpleSpan struct { ctx SpanContext; mu sync.Mutex; ended bool; attrs map[string]any }

func NewTracer(enabled bool) Tracer { if !enabled { return noopTracer{} }; return simpleTracer{enabled:true} }

func NewAdaptiveTracer(percentFn func() float64) Tracer { if percentFn == nil { return noopTracer{} }; return &adaptiveTracer{policyFn: percentFn} }

func (t simpleTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	parent := SpanFromContext(ctx); traceID := parent.ctx.TraceID; if traceID == "" { traceID = newID(16) }
	sp := &simpleSpan{ctx: SpanContext{TraceID: traceID, SpanID: newID(8), ParentSpanID: parent.ctx.SpanID, Start: time.Now()}, attrs: make(map[string]any)}
	ctx = context.WithValue(ctx, spanKey{}, sp); return ctx, sp }
func (t simpleTracer) Noop() bool { return !t.enabled }

func (a *adaptiveTracer) StartSpan(ctx context.Context, name string) (context.Context, Span) {
	parent := SpanFromContext(ctx); traceID := parent.ctx.TraceID
	if traceID == "" { pct := a.policyFn(); if pct <= 0 || rand.Float64()*100 > pct { return ctx, noopSpan{} }; traceID = newID(16) }
	sp := &simpleSpan{ctx: SpanContext{TraceID: traceID, SpanID: newID(8), ParentSpanID: parent.ctx.SpanID, Start: time.Now()}, attrs: make(map[string]any)}
	ctx = context.WithValue(ctx, spanKey{}, sp); return ctx, sp }
func (a *adaptiveTracer) Noop() bool { return false }

func (s *simpleSpan) End() { s.mu.Lock(); if !s.ended { s.ctx.End = time.Now(); s.ended = true }; s.mu.Unlock() }
func (s *simpleSpan) SetAttribute(key string, value any) { s.mu.Lock(); if s.attrs != nil { s.attrs[key] = value }; s.mu.Unlock() }
func (s *simpleSpan) Context() SpanContext { return s.ctx }
func (s *simpleSpan) IsEnded() bool { s.mu.Lock(); ended := s.ended; s.mu.Unlock(); return ended }

type spanKey struct{}

func SpanFromContext(ctx context.Context) *simpleSpan { if ctx == nil { return &simpleSpan{} }; if sp, ok := ctx.Value(spanKey{}).(*simpleSpan); ok { return sp }; return &simpleSpan{} }

func ExtractIDs(ctx context.Context) (traceID, spanID string) { sp := SpanFromContext(ctx); return sp.ctx.TraceID, sp.ctx.SpanID }

func newID(n int) string { b := make([]byte, n); _, _ = randcrypto.Read(b); return hex.EncodeToString(b) }
