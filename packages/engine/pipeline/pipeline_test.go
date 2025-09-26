package pipeline

import (
	"context"
	"path/filepath"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	engresources "ariadne/packages/engine/resources"
)

func TestPipelineBasicFlow(t *testing.T){
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:10}
	pl := NewPipeline(config); defer pl.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second); defer cancel()
	urls := []string{"https://example.com/one","https://example.com/two"}
	results := pl.ProcessURLs(ctx, urls)
	count := 0
	for r := range results { if r.Stage != "output" { t.Fatalf("expected final stage output got %s", r.Stage) }; count++ }
	if count != len(urls) { t.Fatalf("expected %d results got %d", len(urls), count) }
}

func TestPipelineRetriesFailure(t *testing.T){
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:4,RetryBaseDelay:1*time.Millisecond,RetryMaxDelay:2*time.Millisecond,RetryMaxAttempts:2}
	pl := NewPipeline(config); defer pl.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel()
	urls := []string{"https://example.com/fail-extraction"}
	results := pl.ProcessURLs(ctx, urls)
	fails := 0
	for r := range results { if r.Error == nil { t.Fatalf("expected failure") }; fails++ }
	if fails != 1 { t.Fatalf("expected 1 failure got %d", fails) }
}

func TestPipelineResourceCacheHit(t *testing.T){
	temp := t.TempDir()
	cfg := engresources.Config{CacheCapacity:2,MaxInFlight:4,SpillDirectory: filepath.Join(temp,"spill"),CheckpointPath: filepath.Join(temp,"checkpoint.log"),CheckpointInterval:5*time.Millisecond}
	mgr, err := engresources.NewManager(cfg); if err != nil { t.Fatalf("rm: %v", err) }
	defer mgr.Close()
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:4,ResourceManager:mgr}
	pl := NewPipeline(config); defer pl.Stop()
	urls := []string{"https://example.com/cache","https://example.com/cache"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel()
	results := pl.ProcessURLs(ctx, urls)
	processed := 0
	for range results { processed++ }
	if processed != len(urls) { t.Fatalf("expected %d got %d", len(urls), processed) }
	m := pl.Metrics(); if m.StageMetrics["extraction"].Processed != 1 { t.Fatalf("expected 1 extraction") }; if m.StageMetrics["cache"].Processed != 1 { t.Fatalf("expected 1 cache hit") }
}

func TestPipelineResourceSpill(t *testing.T){
	temp := t.TempDir()
	cfg := engresources.Config{CacheCapacity:1,MaxInFlight:2,SpillDirectory: filepath.Join(temp,"spill"),CheckpointPath: filepath.Join(temp,"checkpoint.log"),CheckpointInterval:5*time.Millisecond}
	mgr, err := engresources.NewManager(cfg); if err != nil { t.Fatalf("rm: %v", err) }
	defer mgr.Close()
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:4,ResourceManager:mgr}
	pl := NewPipeline(config); defer pl.Stop()
	urls := []string{"https://example.com/a","https://example.com/b","https://example.com/c"}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second); defer cancel()
	results := pl.ProcessURLs(ctx, urls); for range results {}
	entries, err := os.ReadDir(filepath.Join(temp,"spill")); if err != nil { t.Fatalf("spill dir error: %v", err) }
	found := false; for _, e := range entries { if strings.HasSuffix(e.Name(), ".spill.json") { found = true; break } }
	if !found { t.Fatalf("expected at least one spill file") }
}

func TestPipelineCheckpointing(t *testing.T){
	temp := t.TempDir()
	cfg := engresources.Config{CacheCapacity:4,MaxInFlight:4,SpillDirectory: filepath.Join(temp,"spill"),CheckpointPath: filepath.Join(temp,"cp.log"),CheckpointInterval:1*time.Millisecond}
	mgr, err := engresources.NewManager(cfg); if err != nil { t.Fatalf("rm: %v", err) }
	defer mgr.Close()
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:4,ResourceManager:mgr}
	pl := NewPipeline(config); defer pl.Stop()
	urls := []string{"https://example.com/cp1","https://example.com/cp2"}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel(); results := pl.ProcessURLs(ctx, urls); for range results {}
	time.Sleep(10*time.Millisecond)
	data, err := os.ReadFile(filepath.Join(temp,"cp.log")); if err != nil { t.Fatalf("cp read: %v", err) }
	lines := strings.Split(strings.TrimSpace(string(data)), "\n"); if len(lines) != len(urls) { t.Fatalf("expected %d cp entries got %d", len(urls), len(lines)) }
}

func TestPipelineResultCounting(t *testing.T){
	config := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:2}
	pl := NewPipeline(config); defer pl.Stop()
	ctx := context.Background(); urls := []string{"https://example.com/test"}; results := pl.ProcessURLs(ctx, urls)
	var wg sync.WaitGroup; var count int; var mu sync.Mutex
	wg.Add(1)
	go func(){ defer wg.Done(); for range results { mu.Lock(); count++; mu.Unlock(); if count == 1 { return } } }()
	ch := make(chan struct{}); go func(){ wg.Wait(); close(ch) }()
	select { case <-ch: mu.Lock(); c := count; mu.Unlock(); if c != 1 { t.Fatalf("expected 1 got %d", c) }; case <-time.After(3*time.Second): t.Fatalf("timeout waiting result") }
}
