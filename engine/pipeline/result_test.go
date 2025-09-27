package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestPipelineResultCounting_Migrated(t *testing.T){
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:2}
	pl := NewPipeline(config); defer pl.Stop()
	ctx := context.Background(); urls := []string{"https://example.com/test"}
	results := pl.ProcessURLs(ctx, urls)
	var wg sync.WaitGroup; wg.Add(1)
	var count int; var mu sync.Mutex
	go func(){ defer wg.Done(); for range results { mu.Lock(); count++; c := count; mu.Unlock(); if c == 1 { return } } }()
	done := make(chan struct{}); go func(){ wg.Wait(); close(done) }()
	select { case <-done: mu.Lock(); if count != 1 { t.Fatalf("expected 1 got %d", count) }; mu.Unlock(); case <-time.After(3*time.Second): t.Fatalf("timeout; got %d", count) }
}
