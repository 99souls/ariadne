package processor

import (
	"context"
	"fmt"
	"net/url"
	"testing"
	"time"

	"ariadne/packages/engine/models"
)

// TestProcessorInterface validates the core Processor interface contract
func TestProcessorInterface(t *testing.T) {
	t.Run("should define comprehensive processor contract", func(t *testing.T) {
		// Interface validation through type assertion
		var _ Processor = (*ContentProcessor)(nil)

		// Verify interface methods exist
		processor := NewContentProcessor()

		// Basic type check
		if processor == nil {
			t.Fatal("NewContentProcessor should return non-nil instance")
		}

		// Verify the interface can be used polymorphically
		var p Processor = processor
		_ = p // Verify interface assignment works
	})
}

// TestProcessRequest validates the ProcessRequest type structure
func TestProcessRequest(t *testing.T) {
	t.Run("should contain required fields for processing configuration", func(t *testing.T) {
		baseURL, _ := url.Parse("https://example.com")

		request := ProcessRequest{
			Page:    &models.Page{Content: "<html><body>Test</body></html>"},
			BaseURL: baseURL,
			Policy:  ProcessPolicy{},
			Context: context.Background(),
		}

		if request.Page == nil {
			t.Error("ProcessRequest should support Page field")
		}
		if request.BaseURL == nil {
			t.Error("ProcessRequest should support BaseURL field")
		}
		if request.Context == nil {
			t.Error("ProcessRequest should support Context field")
		}
	})
}

// TestProcessPolicy validates processing policy configuration
func TestProcessPolicy(t *testing.T) {
	t.Run("should define comprehensive processing policies", func(t *testing.T) {
		policy := ProcessPolicy{
			ExtractContent:     true,
			ConvertToMarkdown:  true,
			ExtractMetadata:    true,
			ExtractImages:      true,
			ValidateContent:    true,
			ContentSelectors:   []string{"article", "main", ".content"},
			RemoveSelectors:    []string{"script", "style", "nav"},
			PreserveFormatting: true,
			MaxWordCount:       10000,
			MinWordCount:       50,
			AllowedDomains:     []string{"example.com"},
		}

		if !policy.ExtractContent {
			t.Error("ProcessPolicy should support ExtractContent flag")
		}
		if !policy.ConvertToMarkdown {
			t.Error("ProcessPolicy should support ConvertToMarkdown flag")
		}
		if len(policy.ContentSelectors) == 0 {
			t.Error("ProcessPolicy should support ContentSelectors")
		}
		if policy.MaxWordCount == 0 {
			t.Error("ProcessPolicy should support MaxWordCount limit")
		}
	})
}

// TestContentProcessor validates the concrete implementation
func TestContentProcessor(t *testing.T) {
	processor := NewContentProcessor()

	t.Run("should implement Processor interface", func(t *testing.T) {
		var _ Processor = processor
	})

	t.Run("should validate policy configuration", func(t *testing.T) {
		// Test invalid policy
		invalidPolicy := ProcessPolicy{
			MaxWordCount: -1, // Invalid
			MinWordCount: 1000,
		}
		err := processor.Configure(invalidPolicy)
		if err == nil {
			t.Error("Configure should reject invalid policy with MaxWordCount < MinWordCount")
		}

		// Test valid policy
		validPolicy := ProcessPolicy{
			ExtractContent:    true,
			ConvertToMarkdown: true,
			MaxWordCount:      10000,
			MinWordCount:      50,
		}
		err = processor.Configure(validPolicy)
		if err != nil {
			t.Errorf("Configure should accept valid policy: %v", err)
		}
	})

	t.Run("should process basic content successfully", func(t *testing.T) {
		baseURL, _ := url.Parse("https://example.com")

		page := &models.Page{
			Content: `<html>
				<head><title>Test Page</title></head>
				<body>
					<h1>Main Title</h1>
					<p>This is test content.</p>
				</body>
			</html>`,
		}

		request := ProcessRequest{
			Page:    page,
			BaseURL: baseURL,
			Policy: ProcessPolicy{
				ExtractContent:    true,
				ConvertToMarkdown: true,
				ExtractMetadata:   true,
			},
			Context: context.Background(),
		}

		result, err := processor.Process(request)
		if err != nil {
			t.Fatalf("Process should succeed: %v", err)
		}

		if result == nil || result.Page == nil {
			t.Fatal("Process should return valid result with page")
		}

		if !result.Success {
			t.Error("Process should report success for valid input")
		}
	})

	t.Run("should provide processing statistics", func(t *testing.T) {
		stats := processor.Stats()

		if stats.PagesProcessed < 0 {
			t.Error("Stats should track pages processed")
		}
		if stats.TotalWordCount < 0 {
			t.Error("Stats should track total word count")
		}
	})
}

// TestProcessorStats validates statistics tracking
func TestProcessorStats(t *testing.T) {
	t.Run("should track comprehensive processing statistics", func(t *testing.T) {
		stats := ProcessorStats{
			PagesProcessed:   100,
			PagesSucceeded:   95,
			PagesFailed:      5,
			TotalWordCount:   15000,
			AverageWordCount: 150,
			ProcessingTime:   5 * time.Second,
		}

		if stats.PagesProcessed != 100 {
			t.Error("ProcessorStats should track PagesProcessed")
		}
		if stats.PagesSucceeded != 95 {
			t.Error("ProcessorStats should track PagesSucceeded")
		}
		if stats.TotalWordCount != 15000 {
			t.Error("ProcessorStats should track TotalWordCount")
		}
		if stats.ProcessingTime != 5*time.Second {
			t.Error("ProcessorStats should track ProcessingTime")
		}

		// Calculate success rate
		successRate := float64(stats.PagesSucceeded) / float64(stats.PagesProcessed)
		if successRate != 0.95 {
			t.Errorf("Success rate should be calculable: expected 0.95, got %f", successRate)
		}
	})
}

// TestProcessorIntegration tests real content processing with the existing processor
func TestProcessorIntegration(t *testing.T) {
	t.Run("should process realistic HTML content", func(t *testing.T) {
		processor := NewContentProcessor()
		baseURL, _ := url.Parse("https://example.com")

		page := &models.Page{
			Content: `<!DOCTYPE html>
				<html>
				<head>
					<title>Sample Article</title>
					<meta name="description" content="A test article">
				</head>
				<body>
					<nav><ul><li><a href="/">Home</a></li></ul></nav>
					<aside class="sidebar">Advertisement</aside>
					<article>
						<h1>Main Article Title</h1>
						<p>This is the main content with <a href="/related">relative link</a>.</p>
						<img src="image.jpg" alt="Test image">
						<h2>Subsection</h2>
						<p>More content here with <strong>bold text</strong>.</p>
					</article>
					<footer>Copyright 2024</footer>
					<script>analytics.track();</script>
				</body>
				</html>`,
		}

		request := ProcessRequest{
			Page:    page,
			BaseURL: baseURL,
			Policy: ProcessPolicy{
				ExtractContent:    true,
				ConvertToMarkdown: true,
				ExtractMetadata:   true,
				ExtractImages:     true,
			},
			Context: context.Background(),
		}

		result, err := processor.Process(request)
		if err != nil {
			t.Fatalf("Process should succeed: %v", err)
		}

		if result == nil || result.Page == nil {
			t.Fatal("Process should return valid result with page")
		}

		if !result.Success {
			t.Error("Process should report success")
		}

		// Verify processing actually occurred
		if result.ProcessingTime <= 0 {
			t.Error("ProcessingTime should be recorded")
		}

		// Check that content was processed (title extracted)
		if result.Page.Title == "" {
			t.Error("Title should be extracted from HTML")
		}

		// Check that markdown was generated if policy requested it
		if result.Page.Markdown == "" {
			t.Error("Markdown should be generated when ConvertToMarkdown is true")
		}
	})

	t.Run("should handle processing errors gracefully", func(t *testing.T) {
		processor := NewContentProcessor()

		// Test with nil page
		request := ProcessRequest{
			Page:    nil,
			Context: context.Background(),
		}

		result, err := processor.Process(request)
		if err == nil {
			t.Error("Process should return error for nil page")
		}
		if result != nil {
			t.Error("Result should be nil when error occurs")
		}
	})

	t.Run("should accumulate processing statistics", func(t *testing.T) {
		processor := NewContentProcessor()
		baseURL, _ := url.Parse("https://example.com")

		// Process multiple pages
		for i := 0; i < 3; i++ {
			page := &models.Page{
				Content: fmt.Sprintf("<html><body><h1>Page %d</h1><p>Content for page %d</p></body></html>", i, i),
			}

			request := ProcessRequest{
				Page:    page,
				BaseURL: baseURL,
				Policy:  DefaultProcessPolicy(),
				Context: context.Background(),
			}

			_, err := processor.Process(request)
			if err != nil {
				t.Fatalf("Process should succeed for page %d: %v", i, err)
			}
		}

		stats := processor.Stats()
		if stats.PagesProcessed < 3 {
			t.Errorf("Should have processed at least 3 pages, got %d", stats.PagesProcessed)
		}
		if stats.PagesSucceeded < 3 {
			t.Errorf("Should have succeeded on at least 3 pages, got %d", stats.PagesSucceeded)
		}
		if stats.TotalWordCount <= 0 {
			t.Error("Should have accumulated word count")
		}
	})
}
