package pipeline

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"site-scraper/internal/ratelimit"
	"site-scraper/internal/resources"
)

// Phase 3.1 TDD Tests: Multi-Stage Pipeline Architecture
// Testing approach: Define pipeline behavior through tests first

func TestPipelineStages(t *testing.T) {
	t.Run("should create pipeline with configurable stage workers", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 2,
			ProcessingWorkers: 4,
			OutputWorkers:     2,
			BufferSize:        100,
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		if pipeline.Config().DiscoveryWorkers != 1 {
			t.Errorf("Expected 1 discovery worker, got %d", pipeline.Config().DiscoveryWorkers)
		}
		if pipeline.Config().ProcessingWorkers != 4 {
			t.Errorf("Expected 4 processing workers, got %d", pipeline.Config().ProcessingWorkers)
		}
	})

	t.Run("should handle pipeline stages independently", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 1,
			ProcessingWorkers: 1,
			OutputWorkers:     1,
			BufferSize:        10,
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		// Test each stage can be controlled independently
		stages := []string{"discovery", "extraction", "processing", "output"}
		for _, stage := range stages {
			status := pipeline.StageStatus(stage)
			if status.Workers == 0 {
				t.Errorf("Stage %s should have workers", stage)
			}
			if !status.Active {
				t.Errorf("Stage %s should be active", stage)
			}
		}
	})
}

func TestPipelineDataFlow(t *testing.T) {
	t.Run("should process URLs through complete pipeline", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 1,
			ProcessingWorkers: 1,
			OutputWorkers:     1,
			BufferSize:        10,
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		// Input: URLs to discover
		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2",
		}

		// Process through pipeline
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		results := pipeline.ProcessURLs(ctx, urls)

		processedCount := 0
		for result := range results {
			processedCount++

			// Should have gone through all stages
			if result.Stage != "output" {
				t.Errorf("Expected final stage 'output', got %s", result.Stage)
			}

			// Should have content from extraction stage
			if result.Page == nil {
				t.Error("Result should have page data")
			}

			// Should be marked as success if no errors
			if result.Error == nil && !result.Success {
				t.Error("Successful processing should be marked as success")
			}
		}

		if processedCount != len(urls) {
			t.Errorf("Expected %d processed results, got %d", len(urls), processedCount)
		}
	})

	t.Run("should handle errors in specific pipeline stages", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 1,
			ProcessingWorkers: 1,
			OutputWorkers:     1,
			BufferSize:        10,
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		// Invalid URL should fail at discovery stage
		urls := []string{"invalid-url"}

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		results := pipeline.ProcessURLs(ctx, urls)

		errorFound := false
		for result := range results {
			if result.Error != nil {
				errorFound = true
				// Error should indicate which stage failed
				if result.Stage == "" {
					t.Error("Error result should indicate which stage failed")
				}
			}
		}

		if !errorFound {
			t.Error("Invalid URL should produce error")
		}
	})
}

func TestPipelineBackpressure(t *testing.T) {
	t.Run("should handle backpressure between stages", func(t *testing.T) {
		// Slow processing stage with fast input
		config := &PipelineConfig{
			DiscoveryWorkers:  2, // Fast discovery
			ExtractionWorkers: 1, // Slow extraction bottleneck
			ProcessingWorkers: 2, // Fast processing waiting
			OutputWorkers:     2, // Fast output waiting
			BufferSize:        5, // Small buffer to force backpressure
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		// Large number of URLs to test backpressure
		urls := make([]string, 20)
		for i := range urls {
			urls[i] = "https://example.com/page" + string(rune(i+'0'))
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		start := time.Now()
		results := pipeline.ProcessURLs(ctx, urls)

		processedCount := 0
		for range results {
			processedCount++
		}
		duration := time.Since(start)

		// Should process all URLs despite backpressure
		if processedCount != len(urls) {
			t.Errorf("Expected %d processed results, got %d", len(urls), processedCount)
		}

		// Should take reasonable time (not instant, showing backpressure working)
		if duration < 100*time.Millisecond {
			t.Error("Processing too fast - backpressure not working")
		}
	})
}

func TestPipelineGracefulShutdown(t *testing.T) {
	t.Run("should stop gracefully with context cancellation", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 1,
			ProcessingWorkers: 1,
			OutputWorkers:     1,
			BufferSize:        10,
		}

		pipeline := NewPipeline(config)

		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2",
			"https://example.com/page3",
		}

		ctx, cancel := context.WithCancel(context.Background())
		results := pipeline.ProcessURLs(ctx, urls)

		// Cancel after short time
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		processedCount := 0
		for range results {
			processedCount++
		}

		// Should stop processing when context cancelled
		// May process some results before cancellation
		if processedCount > len(urls) {
			t.Error("Should not process more URLs than input")
		}

		// Should stop cleanly
		pipeline.Stop()
	})

	t.Run("should complete in-flight work during shutdown", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 1,
			ProcessingWorkers: 1,
			OutputWorkers:     1,
			BufferSize:        10,
		}

		pipeline := NewPipeline(config)

		urls := []string{"https://example.com/page1"}

		ctx := context.Background()
		results := pipeline.ProcessURLs(ctx, urls)

		// Start shutdown while processing
		go func() {
			time.Sleep(50 * time.Millisecond)
			pipeline.Stop()
		}()

		processedCount := 0
		for range results {
			processedCount++
		}

		// Should complete the work that was in flight
		if processedCount == 0 {
			t.Error("Should complete in-flight work before stopping")
		}
	})
}

func TestPipelineMetrics(t *testing.T) {
	t.Run("should collect pipeline stage metrics", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 2,
			ProcessingWorkers: 2,
			OutputWorkers:     1,
			BufferSize:        10,
			RetryBaseDelay:    1 * time.Millisecond,
			RetryMaxDelay:     2 * time.Millisecond,
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2",
		}

		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		results := pipeline.ProcessURLs(ctx, urls)

		// Consume results
		for range results {
		}

		// Check stage metrics
		metrics := pipeline.Metrics()

		if metrics.TotalProcessed < len(urls) {
			t.Errorf("Expected at least %d processed, got %d", len(urls), metrics.TotalProcessed)
		}

		stages := []string{"discovery", "extraction", "processing", "output"}
		for _, stage := range stages {
			stageMetrics := metrics.StageMetrics[stage]
			if stageMetrics.Processed == 0 {
				t.Errorf("Stage %s should have processed some items", stage)
			}
		}
	})
}

func TestPipelineRateLimiterIntegration(t *testing.T) {
	limiter := newStubLimiter()
	config := &PipelineConfig{
		DiscoveryWorkers:  1,
		ExtractionWorkers: 1,
		ProcessingWorkers: 1,
		OutputWorkers:     1,
		BufferSize:        4,
		RateLimiter:       limiter,
		RetryBaseDelay:    1 * time.Millisecond,
		RetryMaxDelay:     2 * time.Millisecond,
		RetryMaxAttempts:  3,
	}

	pipeline := NewPipeline(config)
	defer pipeline.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	urls := []string{"https://example.com/limited"}
	results := pipeline.ProcessURLs(ctx, urls)

	processed := 0
	for result := range results {
		processed++
		if result.Error != nil {
			t.Fatalf("expected success, got error: %v", result.Error)
		}
	}

	if processed != len(urls) {
		t.Fatalf("expected %d results, got %d", len(urls), processed)
	}

	if attempts := limiter.Attempts("example.com"); attempts < 2 {
		t.Fatalf("expected limiter to be retried due to circuit open, attempts=%d", attempts)
	}
}

func TestPipelineExtractionRetriesAndFailure(t *testing.T) {
	config := &PipelineConfig{
		DiscoveryWorkers:  1,
		ExtractionWorkers: 1,
		ProcessingWorkers: 1,
		OutputWorkers:     1,
		BufferSize:        4,
		RetryBaseDelay:    1 * time.Millisecond,
		RetryMaxDelay:     2 * time.Millisecond,
		RetryMaxAttempts:  2,
	}

	pipeline := NewPipeline(config)
	defer pipeline.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	urls := []string{"https://example.com/fail-extraction"}
	results := pipeline.ProcessURLs(ctx, urls)

	failures := 0
	for result := range results {
		failures++
		if result.Error == nil {
			t.Fatalf("expected failure result")
		}
		if result.Retry {
			t.Fatalf("final failure should not be marked retryable")
		}
	}

	if failures != 1 {
		t.Fatalf("expected 1 failure result, got %d", failures)
	}
}

func TestPipelineResourceCacheHit(t *testing.T) {
	tempDir := t.TempDir()

	resourceCfg := resources.Config{
		CacheCapacity:      2,
		MaxInFlight:        4,
		SpillDirectory:     filepath.Join(tempDir, "spill"),
		CheckpointPath:     filepath.Join(tempDir, "checkpoint.log"),
		CheckpointInterval: 5 * time.Millisecond,
	}

	manager, err := resources.NewManager(resourceCfg)
	if err != nil {
		t.Fatalf("failed to create resource manager: %v", err)
	}
	defer manager.Close()

	config := &PipelineConfig{
		DiscoveryWorkers:  1,
		ExtractionWorkers: 1,
		ProcessingWorkers: 1,
		OutputWorkers:     1,
		BufferSize:        4,
		ResourceManager:   manager,
	}

	pipeline := NewPipeline(config)
	defer pipeline.Stop()

	urls := []string{
		"https://example.com/cache", // first pass populates cache
		"https://example.com/cache", // second pass should hit cache
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	results := pipeline.ProcessURLs(ctx, urls)

	processed := 0
	for range results {
		processed++
	}

	if processed != len(urls) {
		t.Fatalf("expected %d results, got %d", len(urls), processed)
	}

	metrics := pipeline.Metrics()
	extraction := metrics.StageMetrics["extraction"].Processed
	cacheHits := metrics.StageMetrics["cache"].Processed

	if extraction != 1 {
		t.Fatalf("expected 1 extraction, got %d", extraction)
	}
	if cacheHits != 1 {
		t.Fatalf("expected 1 cache hit, got %d", cacheHits)
	}
}

func TestPipelineResourceSpillover(t *testing.T) {
	tempDir := t.TempDir()

	resourceCfg := resources.Config{
		CacheCapacity:      1,
		MaxInFlight:        2,
		SpillDirectory:     filepath.Join(tempDir, "spill"),
		CheckpointPath:     filepath.Join(tempDir, "checkpoint.log"),
		CheckpointInterval: 5 * time.Millisecond,
	}

	manager, err := resources.NewManager(resourceCfg)
	if err != nil {
		t.Fatalf("failed to create resource manager: %v", err)
	}
	defer manager.Close()

	config := &PipelineConfig{
		DiscoveryWorkers:  1,
		ExtractionWorkers: 1,
		ProcessingWorkers: 1,
		OutputWorkers:     1,
		BufferSize:        4,
		ResourceManager:   manager,
	}

	pipeline := NewPipeline(config)
	defer pipeline.Stop()

	urls := []string{
		"https://example.com/resource/1",
		"https://example.com/resource/2",
		"https://example.com/resource/3",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	results := pipeline.ProcessURLs(ctx, urls)
	for range results {
	}

	spillDir := filepath.Join(tempDir, "spill")
	entries, err := os.ReadDir(spillDir)
	if err != nil {
		t.Fatalf("expected spill directory, got error: %v", err)
	}

	found := false
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".spill.json") {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected at least one spill file")
	}
}

func TestPipelineResourceCheckpointing(t *testing.T) {
	tempDir := t.TempDir()
	checkpointPath := filepath.Join(tempDir, "checkpoint.log")

	resourceCfg := resources.Config{
		CacheCapacity:      4,
		MaxInFlight:        4,
		SpillDirectory:     filepath.Join(tempDir, "spill"),
		CheckpointPath:     checkpointPath,
		CheckpointInterval: 1 * time.Millisecond,
	}

	manager, err := resources.NewManager(resourceCfg)
	if err != nil {
		t.Fatalf("failed to create resource manager: %v", err)
	}
	defer manager.Close()

	config := &PipelineConfig{
		DiscoveryWorkers:  1,
		ExtractionWorkers: 1,
		ProcessingWorkers: 1,
		OutputWorkers:     1,
		BufferSize:        4,
		ResourceManager:   manager,
	}

	pipeline := NewPipeline(config)
	defer pipeline.Stop()

	urls := []string{
		"https://example.com/checkpoint/1",
		"https://example.com/checkpoint/2",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	results := pipeline.ProcessURLs(ctx, urls)
	for range results {
	}

	time.Sleep(10 * time.Millisecond)

	data, err := os.ReadFile(checkpointPath)
	if err != nil {
		t.Fatalf("expected checkpoint file, got error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != len(urls) {
		t.Fatalf("expected %d checkpoint entries, got %d", len(urls), len(lines))
	}
}

type stubLimiter struct {
	mu       sync.Mutex
	attempts map[string]int
}

func newStubLimiter() *stubLimiter {
	return &stubLimiter{attempts: make(map[string]int)}
}

func (s *stubLimiter) Acquire(ctx context.Context, domain string) (ratelimit.Permit, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	count := s.attempts[domain]
	count++
	s.attempts[domain] = count

	if count == 1 {
		return nil, ratelimit.ErrCircuitOpen
	}

	return stubPermit{}, nil
}

func (s *stubLimiter) Feedback(domain string, fb ratelimit.Feedback) {}

func (s *stubLimiter) Snapshot() ratelimit.LimiterSnapshot { return ratelimit.LimiterSnapshot{} }

func (s *stubLimiter) Attempts(domain string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.attempts[domain]
}

type stubPermit struct{}

func (stubPermit) Release() {}
