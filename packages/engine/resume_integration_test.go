package engine

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestEngineResumeFiltering(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "checkpoint-*.log")
	if err != nil {
		t.Fatalf("checkpoint temp: %v", err)
	}
	defer func() { _ = os.Remove(tmpFile.Name()) }()
	defer func() { _ = tmpFile.Close() }()

	seedAll := []string{"https://a.example/1", "https://a.example/2", "https://a.example/3"}
	// Pretend first two already processed
	if _, err := tmpFile.WriteString(seedAll[0] + "\n" + seedAll[1] + "\n"); err != nil {
		t.Fatalf("write checkpoint: %v", err)
	}

	cfg := Defaults()
	cfg.Resume = true
	cfg.CheckpointPath = tmpFile.Name()
	cfg.Resources.CheckpointPath = tmpFile.Name()
	cfg.Resources.CacheCapacity = 0 // keep fast
	cfg.Resources.MaxInFlight = 0

	eng, err := New(cfg)
	if err != nil {
		t.Fatalf("new engine: %v", err)
	}
	defer func() { _ = eng.Stop() }()

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	results, err := eng.Start(ctx, seedAll)
	if err != nil {
		t.Fatalf("start: %v", err)
	}

	var processed int
	for range results {
		processed++
	}
	if processed != 1 {
		t.Fatalf("expected to process only 1 remaining seed, got %d", processed)
	}

	snap := eng.Snapshot()
	if snap.Resume == nil || snap.Resume.Skipped != 2 || snap.Resume.SeedsBefore != 3 {
		t.Fatalf("unexpected resume snapshot: %+v", snap.Resume)
	}
}
