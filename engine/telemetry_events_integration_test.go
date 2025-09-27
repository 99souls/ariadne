package engine

import (
	teleevents "ariadne/packages/engine/telemetry/events"
	"testing"
	"time"
)

// TestEngineEventBusInitialization validates the event bus is always non-nil and publish / subscribe works.
func TestEngineEventBusInitialization(t *testing.T) {
	cfg := Defaults()
	// metrics disabled by default -> bus should still function (metrics provider nil/noop)
	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("engine new: %v", err)
	}
	defer func() { _ = eng.Stop() }()

	bus := eng.EventBus()
	if bus == nil {
		t.Fatalf("expected non-nil event bus")
	}

	sub, err := bus.Subscribe(4)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}
	defer func() { _ = sub.Close() }()

	evType := "engine_start"
	if err := bus.Publish(teleevents.Event{Category: "engine", Type: evType}); err != nil {
		t.Fatalf("publish: %v", err)
	}

	select {
	case ev := <-sub.C():
		if ev.Type != evType {
			t.Fatalf("expected type %s got %s", evType, ev.Type)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatalf("timeout waiting for event")
	}
}

// (no helper types needed; using events.Event directly)
