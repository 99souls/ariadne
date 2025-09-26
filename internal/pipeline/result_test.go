package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"
)

// Test that focuses on receiving the expected number of results, not channel closure timing
func TestPipelineResultCounting(t *testing.T) {
	config := &PipelineConfig{
		DiscoveryWorkers:  1,
		ExtractionWorkers: 1,
		ProcessingWorkers: 1,
		OutputWorkers:     1,
		BufferSize:        2,
	}

	pipeline := NewPipeline(config)
	defer pipeline.Stop()
	
	ctx := context.Background()
	urls := []string{"https://example.com/test"}
	results := pipeline.ProcessURLs(ctx, urls)

	// Use a simpler approach - just wait for the expected number of results
	var wg sync.WaitGroup
	var resultCount int
	var resultMutex sync.Mutex
	
	wg.Add(1)
	go func() {
		defer wg.Done()
		
		for result := range results {
			resultMutex.Lock()
			resultCount++
			count := resultCount
			resultMutex.Unlock()
			
			t.Logf("Result %d: stage=%s, success=%v", count, result.Stage, result.Success)
			
			// If we got our expected result, we can return
			if count == 1 {
				return
			}
		}
	}()

	// Wait for processing with timeout
	done := make(chan bool)
	go func() {
		wg.Wait()
		done <- true
	}()

	select {
	case <-done:
		resultMutex.Lock()
		count := resultCount
		resultMutex.Unlock()
		
		if count == 1 {
			t.Logf("✅ Successfully processed %d result", count)
		} else {
			t.Errorf("Expected 1 result, got %d", count)
		}
	case <-time.After(3 * time.Second):
		resultMutex.Lock()
		count := resultCount
		resultMutex.Unlock()
		t.Errorf("Timeout: only processed %d results", count)
	}
}