package models

import (
	"testing"
	"time"
)

// TestScraperConfig validates configuration functionality
func TestScraperConfig(t *testing.T) {
	t.Run("DefaultConfig should provide sensible defaults", func(t *testing.T) {
		config := DefaultConfig()
		
		// Validate core defaults
		if config.MaxDepth != 10 {
			t.Errorf("expected MaxDepth 10, got %d", config.MaxDepth)
		}
		if config.MaxPages != 1000 {
			t.Errorf("expected MaxPages 1000, got %d", config.MaxPages)
		}
		if config.RequestDelay != 1*time.Second {
			t.Errorf("expected RequestDelay 1s, got %v", config.RequestDelay)
		}
		if config.Timeout != 30*time.Second {
			t.Errorf("expected Timeout 30s, got %v", config.Timeout)
		}
		
		// Validate worker defaults
		if config.CrawlWorkers != 1 {
			t.Errorf("expected CrawlWorkers 1, got %d", config.CrawlWorkers)
		}
		if config.ExtractWorkers != 2 {
			t.Errorf("expected ExtractWorkers 2, got %d", config.ExtractWorkers)
		}
		if config.ProcessWorkers != 4 {
			t.Errorf("expected ProcessWorkers 4, got %d", config.ProcessWorkers)
		}
		
		// Validate selectors
		if len(config.ContentSelectors) == 0 {
			t.Error("ContentSelectors should not be empty")
		}
		if len(config.RemoveSelectors) == 0 {
			t.Error("RemoveSelectors should not be empty")
		}
		
		// Validate output defaults
		if config.OutputDir != "./output" {
			t.Errorf("expected OutputDir './output', got '%s'", config.OutputDir)
		}
		if len(config.OutputFormats) != 1 || config.OutputFormats[0] != "markdown" {
			t.Errorf("expected OutputFormats ['markdown'], got %v", config.OutputFormats)
		}
		
		// Validate behavior flags
		if !config.IncludeImages {
			t.Error("expected IncludeImages to be true by default")
		}
		if !config.RespectRobots {
			t.Error("expected RespectRobots to be true by default")
		}
	})
	
	t.Run("Validate should enforce required fields", func(t *testing.T) {
		config := &ScraperConfig{}
		
		err := config.Validate()
		if err == nil {
			t.Error("expected validation error for empty config")
		}
		
		// Should fail on missing StartURL
		config.AllowedDomains = []string{"example.com"}
		config.MaxDepth = 5
		err = config.Validate()
		if err != ErrMissingStartURL {
			t.Errorf("expected ErrMissingStartURL, got %v", err)
		}
		
		// Should fail on missing AllowedDomains
		config.StartURL = "https://example.com"
		config.AllowedDomains = []string{}
		err = config.Validate()
		if err != ErrMissingAllowedDomains {
			t.Errorf("expected ErrMissingAllowedDomains, got %v", err)
		}
		
		// Should fail on invalid MaxDepth
		config.AllowedDomains = []string{"example.com"}
		config.MaxDepth = 0
		err = config.Validate()
		if err != ErrInvalidMaxDepth {
			t.Errorf("expected ErrInvalidMaxDepth, got %v", err)
		}
	})
	
	t.Run("Validate should fix invalid worker counts", func(t *testing.T) {
		config := &ScraperConfig{
			StartURL:       "https://example.com",
			AllowedDomains: []string{"example.com"},
			MaxDepth:       5,
			CrawlWorkers:   0, // Invalid
		}
		
		err := config.Validate()
		if err != nil {
			t.Errorf("validation should succeed and fix worker count, got error: %v", err)
		}
		
		if config.CrawlWorkers != 1 {
			t.Errorf("expected CrawlWorkers to be corrected to 1, got %d", config.CrawlWorkers)
		}
	})
	
	t.Run("Config should support wiki-specific settings", func(t *testing.T) {
		config := DefaultConfig()
		
		// Verify wiki-friendly content selectors
		expectedSelectors := []string{"article", ".content", ".main-content", "#content", ".post-content", "main"}
		for i, expected := range expectedSelectors {
			if i >= len(config.ContentSelectors) || config.ContentSelectors[i] != expected {
				t.Errorf("expected ContentSelector[%d] '%s', got '%s'", i, expected, config.ContentSelectors[i])
			}
		}
		
		// Verify removal of navigation elements
		removeSelectors := config.RemoveSelectors
		expectedRemove := []string{"nav", ".nav", ".navigation", "header", "footer", ".sidebar"}
		for _, expected := range expectedRemove {
			found := false
			for _, selector := range removeSelectors {
				if selector == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("expected RemoveSelector '%s' not found in %v", expected, removeSelectors)
			}
		}
	})
}