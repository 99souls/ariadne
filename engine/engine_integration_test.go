package engine

import (
	"context"
	"testing"
	"time"
)

// TestEngineBasicFlow validates facade can process a small set of URLs and produce a snapshot.
func TestEngineBasicFlow(t *testing.T) {
	cfg := Defaults()
	cfg.Resources.CacheCapacity = 4
	cfg.Resources.MaxInFlight = 4

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("New engine: %v", err)
	}
	defer func() { _ = eng.Stop() }()

	urls := []string{
		"https://example.com/one",
		"https://example.com/two",
		"https://example.com/three",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	resultsCh, err := eng.Start(ctx, urls)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	var count int
	for range resultsCh {
		count++
	}
	if count != len(urls) {
		// Allow some variance while pipeline evolves, but for now expect exact
		t.Fatalf("expected %d results, got %d", len(urls), count)
	}

	snap := eng.Snapshot()
	if snap.Pipeline == nil || snap.Pipeline.TotalProcessed == 0 {
		t.Fatalf("expected pipeline metrics populated, got %#v", snap.Pipeline)
	}
	if snap.Resources == nil {
		t.Fatalf("expected resource snapshot populated")
	}
}
