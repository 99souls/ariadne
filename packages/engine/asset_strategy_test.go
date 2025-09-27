package engine

import (
    "context"
    "testing"
)

// TestAssetStrategyInterfacePresence ensures Iteration 1 scaffolding exists and defaults are sane.
func TestAssetStrategyInterfacePresence(t *testing.T) {
    cfg := Defaults()
    if cfg.AssetPolicy.Enabled != false {
        t.Fatalf("expected default AssetPolicy.Enabled=false, got true")
    }

    var s AssetStrategy = &DefaultAssetStrategy{}
    if s.Name() != "noop" {
        t.Fatalf("expected noop strategy name, got %s", s.Name())
    }

    // Ensure empty returns don't panic and obey contract shapes.
    refs, err := s.Discover(context.TODO(), nil)
    if err != nil {
        t.Fatalf("unexpected error in Discover: %v", err)
    }
    if len(refs) != 0 {
        t.Fatalf("expected no refs from noop discover, got %d", len(refs))
    }

    actions, err := s.Decide(context.TODO(), refs, cfg.AssetPolicy)
    if err != nil {
        t.Fatalf("unexpected error in Decide: %v", err)
    }
    if len(actions) != 0 {
        t.Fatalf("expected no actions from noop decide, got %d", len(actions))
    }

    mats, err := s.Execute(context.TODO(), actions, cfg.AssetPolicy)
    if err != nil {
        t.Fatalf("unexpected error in Execute: %v", err)
    }
    if len(mats) != 0 {
        t.Fatalf("expected no materialized assets from noop execute, got %d", len(mats))
    }
}
