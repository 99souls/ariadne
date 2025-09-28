package html

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/99souls/ariadne/engine/models"
)

// Phase 4.1 TDD Tests: HTML Template System for Web Output
// Testing approach: Define HTML template behavior through tests first

func mustURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic(fmt.Sprintf("Invalid URL %q: %v", rawURL, err))
	}
	return u
}

func TestHTMLTemplateRendererName(t *testing.T) {
	t.Run("should return correct output sink name", func(t *testing.T) {
		renderer := NewHTMLTemplateRenderer()

		expected := "html-template-renderer"
		actual := renderer.Name()

		if actual != expected {
			t.Errorf("Expected name %q, got %q", expected, actual)
		}
	})
}

func TestHTMLTemplateRendererUtilityMethods(t *testing.T) {
	t.Run("should set output path", func(t *testing.T) {
		renderer := NewHTMLTemplateRenderer()

		newPath := "/tmp/custom_output.html"
		renderer.SetOutputPath(newPath)

		config := renderer.Config()
		if config.OutputPath != newPath {
			t.Errorf("Expected output path %q, got %q", newPath, config.OutputPath)
		}
	})

	t.Run("should close without error", func(t *testing.T) {
		renderer := NewHTMLTemplateRenderer()

		err := renderer.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got %v", err)
		}
	})
}

func TestHTMLTemplateRendererCreation(t *testing.T) {
	t.Run("should create renderer with default configuration", func(t *testing.T) {
		renderer := NewHTMLTemplateRenderer()

		if renderer == nil {
			t.Fatal("Expected renderer to be created, got nil")
		}

		config := renderer.Config()

		// Validate default configuration
		if config.OutputPath != "output.html" {
			t.Errorf("Expected default output path 'output.html', got %q", config.OutputPath)
		}

		if !config.IncludeNavigation {
			t.Error("Expected default IncludeNavigation to be true")
		}

		if !config.IncludeTOC {
			t.Error("Expected default IncludeTOC to be true")
		}

		if config.Theme != "default" {
			t.Errorf("Expected default theme 'default', got %q", config.Theme)
		}
	})

	t.Run("should create renderer with custom configuration", func(t *testing.T) {
		config := HTMLTemplateConfig{
			OutputPath:        "custom.html",
			Title:             "Custom Site Documentation",
			Theme:             "dark",
			IncludeNavigation: false,
			IncludeTOC:        false,
			CustomCSS:         "body { background: #333; }",
		}

		renderer := NewHTMLTemplateRendererWithConfig(config)

		if renderer == nil {
			t.Fatal("Expected renderer to be created, got nil")
		}

		actualConfig := renderer.Config()

		if actualConfig.OutputPath != config.OutputPath {
			t.Errorf("Expected output path %q, got %q", config.OutputPath, actualConfig.OutputPath)
		}

		if actualConfig.Theme != config.Theme {
			t.Errorf("Expected theme %q, got %q", config.Theme, actualConfig.Theme)
		}

		if actualConfig.IncludeNavigation != config.IncludeNavigation {
			t.Errorf("Expected IncludeNavigation %v, got %v", config.IncludeNavigation, actualConfig.IncludeNavigation)
		}
	})
}

func TestHTMLTemplateRendererWrite(t *testing.T) {
	t.Run("should write individual crawl results", func(t *testing.T) {
		renderer := NewHTMLTemplateRenderer()

		page := &models.Page{
			URL:     mustURL("https://example.com/page1"),
			Title:   "Test Page 1",
			Content: "# Test Content\n\nThis is a test page with some content.",
		}

		result := &models.CrawlResult{
			Page:    page,
			Success: true,
		}

		err := renderer.Write(result)
		if err != nil {
			t.Errorf("Expected no error writing crawl result, got %v", err)
		}

		pages := renderer.Pages()
		if len(pages) != 1 {
			t.Errorf("Expected 1 page, got %d", len(pages))
		}

		if pages[0].URL != page.URL {
			t.Errorf("Expected page URL %q, got %q", page.URL, pages[0].URL)
		}
	})

	t.Run("should handle failed crawl results", func(t *testing.T) {
		renderer := NewHTMLTemplateRenderer()

		result := &models.CrawlResult{
			Page: &models.Page{
				URL: mustURL("https://example.com/failed"),
			},
			Success: false,
			Error:   errors.New("Connection timeout"),
		}

		err := renderer.Write(result)
		if err != nil {
			t.Errorf("Expected no error writing failed result, got %v", err)
		}

		// Failed results should not be included in pages
		pages := renderer.Pages()
		if len(pages) != 0 {
			t.Errorf("Expected 0 pages for failed result, got %d", len(pages))
		}

		stats := renderer.Stats()
		if stats.FailedPages != 1 {
			t.Errorf("Expected 1 failed page, got %d", stats.FailedPages)
		}
	})
}

func TestHTMLNavigationGeneration(t *testing.T) {
	t.Run("should generate navigation from page hierarchy", func(t *testing.T) {
		renderer := NewHTMLTemplateRenderer()

		// Add pages with hierarchical URLs
		pages := []*models.Page{
			{URL: mustURL("https://example.com/docs/intro"), Title: "Introduction"},
			{URL: mustURL("https://example.com/docs/guide/setup"), Title: "Setup Guide"},
			{URL: mustURL("https://example.com/docs/guide/advanced"), Title: "Advanced Guide"},
			{URL: mustURL("https://example.com/docs/api"), Title: "API Reference"},
		}

		for _, page := range pages {
			_ = renderer.Write(&models.CrawlResult{Page: page, Success: true})
		}

		navigation := renderer.GenerateNavigation()

		if navigation == "" {
			t.Error("Expected navigation to be generated, got empty string")
		}

		// Check that navigation contains hierarchical structure
		if !strings.Contains(navigation, "docs") {
			t.Error("Expected navigation to contain 'docs' section")
		}

		if !strings.Contains(navigation, "guide") {
			t.Error("Expected navigation to contain 'guide' subsection")
		}

		// Check that all page titles are included
		for _, page := range pages {
			if !strings.Contains(navigation, page.Title) {
				t.Errorf("Expected navigation to contain page title %q", page.Title)
			}
		}
	})
}

func TestHTMLDocumentGeneration(t *testing.T) {
	t.Run("should generate complete HTML document", func(t *testing.T) {
		renderer := NewHTMLTemplateRenderer()

		// Add test pages
		pages := []*models.Page{
			{
				URL:     mustURL("https://example.com/page1"),
				Title:   "Page 1",
				Content: "# Page 1\n\nContent for page 1",
			},
			{
				URL:     mustURL("https://example.com/page2"),
				Title:   "Page 2",
				Content: "# Page 2\n\nContent for page 2",
			},
		}

		for _, page := range pages {
			_ = renderer.Write(&models.CrawlResult{Page: page, Success: true})
		}

		html := renderer.GenerateHTML()

		if html == "" {
			t.Error("Expected HTML to be generated, got empty string")
		}

		// Check HTML document structure
		if !strings.Contains(html, "<!DOCTYPE html>") {
			t.Error("Expected HTML to contain DOCTYPE declaration")
		}

		if !strings.Contains(html, "<html") {
			t.Error("Expected HTML to contain html tag")
		}

		if !strings.Contains(html, "<head>") {
			t.Error("Expected HTML to contain head section")
		}

		if !strings.Contains(html, "<body>") {
			t.Error("Expected HTML to contain body section")
		}

		// Check that page content is included
		for _, page := range pages {
			if !strings.Contains(html, page.Title) {
				t.Errorf("Expected HTML to contain page title %q", page.Title)
			}
		}

		// Check for navigation
		if !strings.Contains(html, "navigation") || !strings.Contains(html, "nav") {
			t.Error("Expected HTML to contain navigation elements")
		}
	})
}

func TestHTMLTemplateRendererIntegration(t *testing.T) {
	t.Run("Phase 4.1: Complete HTML template rendering with navigation", func(t *testing.T) {
		// Create temporary directory for test output
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "test_output.html")

		config := HTMLTemplateConfig{
			OutputPath:        outputPath,
			Title:             "Test Site Documentation",
			Theme:             "default",
			IncludeNavigation: true,
			IncludeTOC:        true,
		}

		renderer := NewHTMLTemplateRendererWithConfig(config)

		// Add comprehensive test data
		testPages := []*models.Page{
			{
				URL:     mustURL("https://example.com/"),
				Title:   "Home Page",
				Content: "# Welcome\n\nThis is the home page of our documentation.",
			},
			{
				URL:     mustURL("https://example.com/docs/intro"),
				Title:   "Introduction",
				Content: "# Introduction\n\n## Overview\n\nThis section introduces the concepts.",
			},
			{
				URL:     mustURL("https://example.com/docs/guide/setup"),
				Title:   "Setup Guide",
				Content: "# Setup Guide\n\n## Installation\n\nFollow these steps to install.",
			},
			{
				URL:     mustURL("https://example.com/api/reference"),
				Title:   "API Reference",
				Content: "# API Reference\n\n## Methods\n\nAvailable API methods.",
			},
		}

		// Write all pages
		for _, page := range testPages {
			err := renderer.Write(&models.CrawlResult{Page: page, Success: true})
			if err != nil {
				t.Fatalf("Failed to write page %q: %v", page.URL, err)
			}
		}

		// Add a failed page for stats
		err := renderer.Write(&models.CrawlResult{
			Page:    &models.Page{URL: mustURL("https://example.com/broken")},
			Success: false,
			Error:   errors.New("404 Not Found"),
		})
		if err != nil {
			t.Fatalf("Failed to write failed page: %v", err)
		}

		// Flush to generate the HTML file
		err = renderer.Flush()
		if err != nil {
			t.Fatalf("Failed to flush renderer: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Expected HTML file to be created at %q", outputPath)
		}

		// Read and verify file content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read generated HTML file: %v", err)
		}

		html := string(content)

		if len(html) == 0 {
			t.Fatal("Generated HTML file is empty")
		}

		t.Logf("Generated HTML length: %d characters", len(html))

		// Comprehensive validation
		requiredElements := []string{
			"<!DOCTYPE html>",
			"<html",
			"<head>",
			"<title>Test Site Documentation</title>",
			"<body>",
			"navigation",
			"Home Page",
			"Introduction",
			"Setup Guide",
			"API Reference",
		}

		for _, element := range requiredElements {
			if !strings.Contains(html, element) {
				t.Errorf("Expected HTML to contain %q", element)
			}
		}

		// Verify statistics
		stats := renderer.Stats()
		if stats.TotalPages != 5 {
			t.Errorf("Expected 5 total pages (4 successful + 1 failed), got %d", stats.TotalPages)
		}

		if stats.SuccessfulPages != 4 {
			t.Errorf("Expected 4 successful pages, got %d", stats.SuccessfulPages)
		}

		if stats.FailedPages != 1 {
			t.Errorf("Expected 1 failed page, got %d", stats.FailedPages)
		}

		t.Logf("🎉 PHASE 4.1 SUCCESS: Generated HTML document with %d successful pages and comprehensive navigation!", stats.SuccessfulPages)
	})
}
