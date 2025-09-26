package pipeline

import (
	"testing"
)

// Test individual pipeline components to isolate the issue
func TestPipelineComponents(t *testing.T) {
	t.Run("URL validation should work", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 1,
			ProcessingWorkers: 1,
			OutputWorkers:     1,
			BufferSize:        2,
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		// Test URL validation logic directly
		if !pipeline.isValidURL("https://example.com/test") {
			t.Error("Valid URL should pass validation")
		}

		if pipeline.isValidURL("invalid-url") {
			t.Error("Invalid URL should fail validation")
		}

		if pipeline.isValidURL("") {
			t.Error("Empty URL should fail validation")
		}
	})

	t.Run("content extraction should work", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 1,
			ProcessingWorkers: 1,
			OutputWorkers:     1,
			BufferSize:        2,
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		page := pipeline.extractContent("https://example.com/test")
		if page == nil {
			t.Error("Content extraction should return a page")
		}

		if page.Title == "" {
			t.Error("Extracted page should have a title")
		}
	})

	t.Run("content processing should work", func(t *testing.T) {
		config := &PipelineConfig{
			DiscoveryWorkers:  1,
			ExtractionWorkers: 1,
			ProcessingWorkers: 1,
			OutputWorkers:     1,
			BufferSize:        2,
		}

		pipeline := NewPipeline(config)
		defer pipeline.Stop()

		page := pipeline.extractContent("https://example.com/test")
		result := pipeline.processContent(page)

		if result == nil {
			t.Error("Content processing should return a result")
		}

		if !result.Success {
			t.Error("Content processing should succeed for valid page")
		}

		if result.Page != page {
			t.Error("Result should contain the original page")
		}
	})
}