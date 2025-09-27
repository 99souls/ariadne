package engine

import (
	"testing"
)

// TestEngineModuleBootstrap validates that the engine module can be imported
// and basic functionality works without depending on the root module.
// This ensures the module bootstrap is correct for Wave1.
func TestEngineModuleBootstrap(t *testing.T) {
	// Test that we can create an engine config
	cfg := Defaults()
	if cfg.DiscoveryWorkers <= 0 {
		t.Error("Expected positive DiscoveryWorkers")
	}
	if cfg.ExtractionWorkers <= 0 {
		t.Error("Expected positive ExtractionWorkers")
	}

	// Test that we can create an engine instance (basic constructor validation)
	engine, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create engine: %v", err)
	}
	if engine == nil {
		t.Fatal("Engine should not be nil")
	}

	// Clean shutdown
	err = engine.Stop()
	if err != nil {
		t.Errorf("Failed to stop engine: %v", err)
	}
}