package events

import (
	"context"
	"testing"
	"time"

	internaltracing "github.com/99souls/ariadne/engine/internal/telemetry/tracing"
	metrics "github.com/99souls/ariadne/engine/telemetry/metrics"
)

func TestBusBasicPublishSubscribe(t *testing.T) {
	bus := NewBus(metrics.NewNoopProvider())
	sub, err := bus.Subscribe(10)
	if err != nil {
		t.Fatalf("subscribe err: %v", err)
	}
	defer func() { _ = sub.Close() }()

	ev := Event{Category: CategoryAssets, Type: "asset_discovered"}
	if err := bus.Publish(ev); err != nil {
		t.Fatalf("publish err: %v", err)
	}

	select {
	case got := <-sub.C():
		if got.Type != ev.Type || got.Category != ev.Category {
			t.Fatalf("unexpected event %+v", got)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("timeout waiting for event")
	}
}

func TestBusDropBehavior(t *testing.T) {
	bus := NewBus(metrics.NewNoopProvider())
	sub, err := bus.Subscribe(1)
	if err != nil {
		t.Fatalf("subscribe err: %v", err)
	}
	// Don't consume from sub to force drops
	defer func() { _ = sub.Close() }()

	for i := 0; i < 5; i++ {
		_ = bus.Publish(Event{Category: CategoryPipeline, Type: "tick"})
	}
	stats := bus.Stats()
	if stats.Published == 0 {
		t.Fatalf("expected published >0")
	}
	if stats.Dropped == 0 {
		t.Fatalf("expected drops >0, got %#v", stats)
	}
}

func TestMultipleSubscribers(t *testing.T) {
	bus := NewBus(metrics.NewNoopProvider())
	sub1, _ := bus.Subscribe(2)
	sub2, _ := bus.Subscribe(2)
	defer func() { _ = sub1.Close() }()
	defer func() { _ = sub2.Close() }()

	_ = bus.Publish(Event{Category: CategoryRateLimit, Type: "decision"})

	recv := func(ch <-chan Event) bool {
		select {
		case <-ch:
			return true
		case <-time.After(200 * time.Millisecond):
			return false
		}
	}
	if !recv(sub1.C()) || !recv(sub2.C()) {
		t.Fatalf("both subscribers should receive event")
	}
}

func TestPublishCtxTracingCorrelation(t *testing.T) {
	bus := NewBus(metrics.NewNoopProvider())
	tr := internaltracing.NewTracer(true)
	ctx, span := tr.StartSpan(context.Background(), "root")
	defer span.End()
	sub, err := bus.Subscribe(2)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer func() { _ = sub.Close() }()
	if err := bus.PublishCtx(ctx, Event{Category: CategoryPipeline, Type: "stage_start"}); err != nil {
		t.Fatalf("publishctx: %v", err)
	}
	select {
	case ev := <-sub.C():
		if ev.TraceID == "" || ev.SpanID == "" {
			t.Fatalf("expected trace/span ids on event: %+v", ev)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("timeout")
	}
}

// BenchmarkPublishContextVsSimple provides a rough comparison of overhead.
func BenchmarkPublishContextVsSimple(b *testing.B) {
	bus := NewBus(metrics.NewNoopProvider())
	sub, _ := bus.Subscribe(128)
	defer func() { _ = sub.Close() }()
	tr := internaltracing.NewTracer(true)
	ctx, span := tr.StartSpan(context.Background(), "bench")
	defer span.End()
	ev := Event{Category: CategoryPipeline, Type: "tick"}
	b.Run("plain", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bus.Publish(ev)
		}
	})
	b.Run("ctx", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = bus.PublishCtx(ctx, ev)
		}
	})
}
