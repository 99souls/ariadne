package output

import (
	"testing"
	"time"

	"github.com/99souls/ariadne/engine/models"
)

// TestEnhancedOutputSink validates the enhanced OutputSink interface
func TestEnhancedOutputSink(t *testing.T) {
	t.Run("should maintain backward compatibility with existing interface", func(t *testing.T) {
		// Ensure existing OutputSink interface is still supported
		var _ OutputSink = (*EnhancedSink)(nil)

		sink := NewEnhancedSink()
		if sink == nil {
			t.Fatal("NewEnhancedSink should return non-nil instance")
		}
	})

	t.Run("should support enhanced interface methods", func(t *testing.T) {
		// Verify enhanced interface methods exist
		var _ EnhancedOutputSink = (*EnhancedSink)(nil)

		sink := NewEnhancedSink()

		// Test policy configuration
		err := sink.Configure(DefaultSinkPolicy())
		if err != nil {
			t.Errorf("Configure should accept default policy: %v", err)
		}

		// Test statistics
		stats := sink.Stats()
		if stats.WriteCount < 0 {
			t.Error("Stats should return valid statistics")
		}

		// Test health check
		healthy := sink.IsHealthy()
		if !healthy {
			t.Error("New sink should be healthy")
		}
	})
}

// TestSinkPolicy validates sink policy configuration
func TestSinkPolicy(t *testing.T) {
	t.Run("should define comprehensive sink policies", func(t *testing.T) {
		policy := SinkPolicy{
			MaxRetries:        3,
			RetryDelay:        100 * time.Millisecond,
			BufferSize:        1000,
			FlushInterval:     5 * time.Second,
			EnableCompression: true,
			FilterPattern:     "success",
			TransformRules:    []string{"clean-urls", "normalize-metadata"},
		}

		if policy.MaxRetries != 3 {
			t.Error("SinkPolicy should support MaxRetries")
		}
		if policy.BufferSize != 1000 {
			t.Error("SinkPolicy should support BufferSize")
		}
		if len(policy.TransformRules) == 0 {
			t.Error("SinkPolicy should support TransformRules")
		}
	})
}

// TestSinkStats validates statistics tracking
func TestSinkStats(t *testing.T) {
	t.Run("should track comprehensive sink statistics", func(t *testing.T) {
		stats := SinkStats{
			WriteCount:        100,
			WriteErrors:       2,
			FlushCount:        10,
			BytesProcessed:    50000,
			AverageLatency:    50 * time.Millisecond,
			LastWrite:         time.Now(),
			BufferUtilization: 0.75,
		}

		if stats.WriteCount != 100 {
			t.Error("SinkStats should track WriteCount")
		}
		if stats.WriteErrors != 2 {
			t.Error("SinkStats should track WriteErrors")
		}
		if stats.BufferUtilization != 0.75 {
			t.Error("SinkStats should track BufferUtilization")
		}

		// Calculate success rate
		successRate := float64(stats.WriteCount-stats.WriteErrors) / float64(stats.WriteCount)
		if successRate != 0.98 {
			t.Errorf("Success rate should be calculable: expected 0.98, got %f", successRate)
		}
	})
}

// TestCompositeSink validates multi-sink composition
func TestCompositeSink(t *testing.T) {
	t.Run("should compose multiple sinks", func(t *testing.T) {
		// Create individual sinks
		sink1 := NewEnhancedSink()
		sink2 := NewEnhancedSink()

		// Create composite sink
		composite := NewCompositeSink(sink1, sink2)

		if composite == nil {
			t.Fatal("NewCompositeSink should return non-nil instance")
		}

		// Verify it implements both interfaces
		var _ OutputSink = composite
		var _ EnhancedOutputSink = composite

		// Test write to multiple sinks
		result := &models.CrawlResult{URL: "https://example.com", Success: true}
		err := composite.Write(result)
		if err != nil {
			t.Errorf("Composite write should succeed: %v", err)
		}

		// Verify both sinks received the write
		stats1 := sink1.Stats()
		stats2 := sink2.Stats()

		if stats1.WriteCount == 0 {
			t.Error("First sink should have received write")
		}
		if stats2.WriteCount == 0 {
			t.Error("Second sink should have received write")
		}
	})
}

// TestRoutingSink validates conditional routing
func TestRoutingSink(t *testing.T) {
	t.Run("should route based on conditions", func(t *testing.T) {
		successSink := NewEnhancedSink()
		errorSink := NewEnhancedSink()

		// Create routing sink
		router := NewRoutingSink()
		router.AddRoute(func(result *models.CrawlResult) bool {
			return result.Success
		}, successSink)
		router.AddRoute(func(result *models.CrawlResult) bool {
			return !result.Success
		}, errorSink)

		// Test successful result routing
		successResult := &models.CrawlResult{URL: "https://example.com", Success: true}
		err := router.Write(successResult)
		if err != nil {
			t.Errorf("Router write should succeed: %v", err)
		}

		// Test error result routing
		errorResult := &models.CrawlResult{URL: "https://error.com", Success: false}
		err = router.Write(errorResult)
		if err != nil {
			t.Errorf("Router write should succeed: %v", err)
		}

		// Verify routing worked
		successStats := successSink.Stats()
		errorStats := errorSink.Stats()

		if successStats.WriteCount != 1 {
			t.Errorf("Success sink should have 1 write, got %d", successStats.WriteCount)
		}
		if errorStats.WriteCount != 1 {
			t.Errorf("Error sink should have 1 write, got %d", errorStats.WriteCount)
		}
	})
}

// TestSinkPipeline validates processing pipeline integration
func TestSinkPipeline(t *testing.T) {
	t.Run("should process results through pipeline", func(t *testing.T) {
		sink := NewEnhancedSink()

		// Configure pipeline with transformations
		policy := SinkPolicy{
			TransformRules: []string{"clean-urls", "normalize-metadata"},
			BufferSize:     10,
			FlushInterval:  100 * time.Millisecond,
		}

		err := sink.Configure(policy)
		if err != nil {
			t.Errorf("Configure should accept pipeline policy: %v", err)
		}

		// Process result through pipeline
		result := &models.CrawlResult{
			URL:     "https://example.com/path/../clean",
			Success: true,
		}

		err = sink.Write(result)
		if err != nil {
			t.Errorf("Pipeline write should succeed: %v", err)
		}

		// Verify statistics updated
		stats := sink.Stats()
		if stats.WriteCount != 1 {
			t.Error("Pipeline should track writes")
		}
	})
}

// TestBackwardCompatibility ensures existing sinks work with enhanced interface
func TestBackwardCompatibility(t *testing.T) {
	t.Run("should work with existing stdout sink", func(t *testing.T) {
		// Import existing stdout sink and test it works with enhancements
		// This test validates that existing OutputSink implementations
		// can be composed with enhanced sinks

		enhanced := NewEnhancedSink()

		// Test that we can create compositions with mixed sink types
		composite := NewCompositeSink(enhanced)

		if composite == nil {
			t.Fatal("Should be able to compose enhanced and basic sinks")
		}

		// Test write operation
		result := &models.CrawlResult{URL: "https://example.com", Success: true}
		err := composite.Write(result)
		if err != nil {
			t.Errorf("Composite with mixed sink types should work: %v", err)
		}
	})
}

// TestSinkTransformations validates data transformation capabilities
func TestSinkTransformations(t *testing.T) {
	t.Run("should apply preprocessing transformations", func(t *testing.T) {
		sink := NewEnhancedSink()

		// Configure a preprocessor that modifies URLs
		sink.SetPreprocessor(func(result *models.CrawlResult) (*models.CrawlResult, error) {
			modified := *result
			modified.URL = "transformed-" + result.URL
			return &modified, nil
		})

		result := &models.CrawlResult{URL: "https://example.com", Success: true}
		err := sink.Write(result)
		if err != nil {
			t.Errorf("Preprocessing should succeed: %v", err)
		}

		// Verify statistics were updated
		stats := sink.Stats()
		if stats.WriteCount != 1 {
			t.Error("Write count should be updated after preprocessing")
		}
	})

	t.Run("should apply postprocessing operations", func(t *testing.T) {
		sink := NewEnhancedSink()

		postprocessCalled := false
		sink.SetPostprocessor(func(result *models.CrawlResult) error {
			postprocessCalled = true
			return nil
		})

		result := &models.CrawlResult{URL: "https://example.com", Success: true}
		err := sink.Write(result)
		if err != nil {
			t.Errorf("Postprocessing should succeed: %v", err)
		}

		if !postprocessCalled {
			t.Error("Postprocessor should have been called")
		}
	})
}
