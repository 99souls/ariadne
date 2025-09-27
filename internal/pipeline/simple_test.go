//go:build legacyremoved
// +build legacyremoved

// Deprecated legacy test file retained temporarily until filesystem deletion recognized.
package pipeline

import (
	"context"
	"testing"
	"time"
)

// Simple pipeline test to debug the deadlock
func TestSimplePipeline(t *testing.T) {
	config := &PipelineConfig{
		DiscoveryWorkers:  1,
		ExtractionWorkers: 1,
		ProcessingWorkers: 1,
		OutputWorkers:     1,
		BufferSize:        2,
	}

	pipeline := NewPipeline(config)
	defer pipeline.Stop() // Clean up at the end

	// Test with one URL - use longer timeout for processing
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	urls := []string{"https://example.com/test"}
	results := pipeline.ProcessURLs(ctx, urls)

	// Count results with proper synchronization
	var resultCount int
	completed := make(chan bool, 1) // Buffered to prevent blocking

	go func() {
		defer func() {
			completed <- true
		}()

		for result := range results {
			t.Logf("Received result: stage=%s, success=%v", result.Stage, result.Success)
			resultCount++
		}
		t.Logf("Results channel closed, total count: %d", resultCount)
	}()

	// Wait for either completion or timeout - give more time for processing
	select {
	case <-completed:
		t.Logf("✅ Pipeline completed successfully! Processed %d results", resultCount)
		if resultCount != 1 {
			t.Errorf("Expected 1 result, got %d", resultCount)
		}
	case <-time.After(5 * time.Second):
		t.Errorf("Test timed out - pipeline processed %d results so far", resultCount)
	}
}
