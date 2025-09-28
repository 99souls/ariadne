package engine

import "testing"

// Deprecated test placeholder: legacy EventBus() accessor removed in C6. This test is
// intentionally skipped and will be deleted once observer coverage is sufficient.
func TestEngineEventBusInitialization(t *testing.T) {
	t.Skip("EventBus accessor removed; use RegisterEventObserver + TelemetryEvent facade; test retained temporarily during C6")
}
