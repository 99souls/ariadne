package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ariadne/pkg/models"
)

// Phase 4.1 TDD Tests: Markdown Compilation with Table of Contents
// Testing approach: Define markdown compilation behavior through tests first

func TestMarkdownCompilerName(t *testing.T) {
	t.Run("should return correct output sink name", func(t *testing.T) {
		compiler := NewMarkdownCompiler()

		expected := "markdown-compiler"
		actual := compiler.Name()

		if actual != expected {
			t.Errorf("Expected name %q, got %q", expected, actual)
		}
	})
}

func TestMarkdownCompilerCreation(t *testing.T) {
	t.Run("should create compiler with default configuration", func(t *testing.T) {
		compiler := NewMarkdownCompiler()
		if compiler == nil {
			t.Fatal("NewMarkdownCompiler() returned nil")
		}

		config := compiler.Config()
		if config.IncludeTOC != true {
			t.Errorf("Expected default IncludeTOC=true, got %v", config.IncludeTOC)
		}
		if config.TOCMaxDepth != 3 {
			t.Errorf("Expected default TOCMaxDepth=3, got %v", config.TOCMaxDepth)
		}
		if config.OutputPath == "" {
			t.Error("Expected non-empty default OutputPath")
		}
	})

	t.Run("should create compiler with custom configuration", func(t *testing.T) {
		config := &MarkdownCompilerConfig{
			OutputPath:   "/custom/path/output.md",
			IncludeTOC:   false,
			TOCMaxDepth:  5,
			TOCTitle:     "Custom Contents",
			IncludeIndex: true,
			SortByURL:    false,
			SortByTitle:  true,
			IncludeStats: true,
		}

		compiler := NewMarkdownCompilerWithConfig(config)
		if compiler == nil {
			t.Fatal("NewMarkdownCompilerWithConfig() returned nil")
		}

		actualConfig := compiler.Config()
		if actualConfig.OutputPath != config.OutputPath {
			t.Errorf("Expected OutputPath=%s, got %s", config.OutputPath, actualConfig.OutputPath)
		}
		if actualConfig.IncludeTOC != config.IncludeTOC {
			t.Errorf("Expected IncludeTOC=%v, got %v", config.IncludeTOC, actualConfig.IncludeTOC)
		}
		if actualConfig.TOCMaxDepth != config.TOCMaxDepth {
			t.Errorf("Expected TOCMaxDepth=%d, got %d", config.TOCMaxDepth, actualConfig.TOCMaxDepth)
		}
	})
}

func TestMarkdownCompilerWrite(t *testing.T) {
	t.Run("should write individual crawl results", func(t *testing.T) {
		compiler := NewMarkdownCompiler()
		defer compiler.Close()

		result := &models.CrawlResult{
			URL:     "https://example.com/test",
			Success: true,
			Page: &models.Page{
				Title:    "Test Page",
				Markdown: "# Test Page\n\nThis is test content.",
			},
		}

		err := compiler.Write(result)
		if err != nil {
			t.Fatalf("Write() failed: %v", err)
		}

		// Should accumulate pages for later compilation
		pages := compiler.Pages()
		if len(pages) != 1 {
			t.Fatalf("Expected 1 page, got %d", len(pages))
		}
		if pages[0].URL != result.URL {
			t.Errorf("Expected URL=%s, got %s", result.URL, pages[0].URL)
		}
	})

	t.Run("should handle failed crawl results", func(t *testing.T) {
		compiler := NewMarkdownCompiler()
		defer compiler.Close()

		result := &models.CrawlResult{
			URL:     "https://example.com/failed",
			Success: false,
			Error:   fmt.Errorf("404 not found"),
		}

		err := compiler.Write(result)
		if err != nil {
			t.Fatalf("Write() should not fail for unsuccessful results: %v", err)
		}

		// Failed results should not be included in pages
		pages := compiler.Pages()
		if len(pages) != 0 {
			t.Fatalf("Expected 0 pages for failed result, got %d", len(pages))
		}

		// But should be tracked in error stats
		stats := compiler.Stats()
		if stats.FailedPages != 1 {
			t.Errorf("Expected 1 failed page, got %d", stats.FailedPages)
		}
	})
}

func TestTableOfContentsGeneration(t *testing.T) {
	t.Run("should generate TOC from markdown headers", func(t *testing.T) {
		compiler := NewMarkdownCompiler()
		defer compiler.Close()

		// Add pages with various header levels
		pages := []*models.CrawlResult{
			{
				URL:     "https://example.com/page1",
				Success: true,
				Page: &models.Page{
					Title:    "Introduction",
					Markdown: "# Introduction\n\n## Getting Started\n\n### Installation\n\nSome content.",
				},
			},
			{
				URL:     "https://example.com/page2",
				Success: true,
				Page: &models.Page{
					Title:    "Advanced Topics",
					Markdown: "# Advanced Topics\n\n## Configuration\n\n## Best Practices\n\n### Security\n\nMore content.",
				},
			},
		}

		for _, page := range pages {
			err := compiler.Write(page)
			if err != nil {
				t.Fatalf("Write() failed: %v", err)
			}
		}

		toc := compiler.GenerateTOC()
		if toc == "" {
			t.Fatal("GenerateTOC() returned empty string")
		}

		// Check TOC contains expected elements
		expectedEntries := []string{
			"Introduction",
			"Getting Started",
			"Installation",
			"Advanced Topics",
			"Configuration",
			"Best Practices",
			"Security",
		}

		for _, entry := range expectedEntries {
			if !strings.Contains(toc, entry) {
				t.Errorf("TOC should contain '%s', but doesn't. TOC: %s", entry, toc)
			}
		}
	})
}

func TestMarkdownDocumentAssembly(t *testing.T) {
	t.Run("should compile complete markdown document with TOC", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "compiled.md")

		config := &MarkdownCompilerConfig{
			OutputPath:  outputPath,
			IncludeTOC:  true,
			TOCTitle:    "Table of Contents",
			TOCMaxDepth: 3,
		}
		compiler := NewMarkdownCompilerWithConfig(config)
		defer compiler.Close()

		// Add sample pages
		pages := []*models.CrawlResult{
			{
				URL:     "https://wiki.example.com/intro",
				Success: true,
				Page: &models.Page{
					Title:    "Introduction",
					Markdown: "# Introduction\n\nWelcome to our wiki!\n\n## Overview\n\nThis covers the basics.",
				},
			},
			{
				URL:     "https://wiki.example.com/guide",
				Success: true,
				Page: &models.Page{
					Title:    "User Guide",
					Markdown: "# User Guide\n\n## Installation\n\nSteps to install.\n\n## Configuration\n\nHow to configure.",
				},
			},
		}

		for _, page := range pages {
			err := compiler.Write(page)
			if err != nil {
				t.Fatalf("Write() failed: %v", err)
			}
		}

		// Compile the document
		err := compiler.Flush()
		if err != nil {
			t.Fatalf("Flush() failed: %v", err)
		}

		// Check that file was created
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Output file was not created: %s", outputPath)
		}

		// Read and verify content
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		contentStr := string(content)

		// Should contain TOC
		if !strings.Contains(contentStr, "Table of Contents") {
			t.Error("Compiled document should contain TOC title")
		}

		// Should contain page content
		if !strings.Contains(contentStr, "Introduction") {
			t.Error("Compiled document should contain Introduction section")
		}
		if !strings.Contains(contentStr, "User Guide") {
			t.Error("Compiled document should contain User Guide section")
		}
		if !strings.Contains(contentStr, "Welcome to our wiki!") {
			t.Error("Compiled document should contain page content")
		}
	})
}

// Phase 4.1 Integration Test
func TestMarkdownCompilerIntegration(t *testing.T) {
	t.Run("Phase 4.1: Complete markdown compilation with TOC", func(t *testing.T) {
		tempDir := t.TempDir()
		outputPath := filepath.Join(tempDir, "wiki-compiled.md")

		config := &MarkdownCompilerConfig{
			OutputPath:   outputPath,
			IncludeTOC:   true,
			TOCTitle:     "Wiki Contents",
			TOCMaxDepth:  3,
			IncludeStats: true,
			SortByURL:    true,
		}
		compiler := NewMarkdownCompilerWithConfig(config)
		defer compiler.Close()

		// Simulate wiki crawl results
		wikiPages := []*models.CrawlResult{
			{
				URL:     "https://wiki.example.com/getting-started",
				Success: true,
				Page: &models.Page{
					Title:    "Getting Started Guide",
					Markdown: "# Getting Started\n\n## Prerequisites\n\nBefore you begin...\n\n## Quick Start\n\n### Installation\n\nFollow these steps.",
				},
			},
			{
				URL:     "https://wiki.example.com/advanced",
				Success: true,
				Page: &models.Page{
					Title:    "Advanced Topics",
					Markdown: "# Advanced Topics\n\n## Configuration\n\nAdvanced settings.\n\n## Performance\n\nOptimization tips.",
				},
			},
		}

		// Write all pages
		for _, page := range wikiPages {
			err := compiler.Write(page)
			if err != nil {
				t.Fatalf("Write() failed for %s: %v", page.URL, err)
			}
		}

		// Add one failed result to test error handling
		failedResult := &models.CrawlResult{
			URL:     "https://wiki.example.com/broken-link",
			Success: false,
			Error:   fmt.Errorf("404 page not found"),
		}
		err := compiler.Write(failedResult)
		if err != nil {
			t.Fatalf("Write() failed for failed result: %v", err)
		}

		// Compile the final document
		err = compiler.Flush()
		if err != nil {
			t.Fatalf("Flush() failed: %v", err)
		}

		// Verify the output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Fatalf("Output file was not created: %s", outputPath)
		}

		// Read and analyze the compiled document
		content, err := os.ReadFile(outputPath)
		if err != nil {
			t.Fatalf("Failed to read output file: %v", err)
		}

		contentStr := string(content)
		t.Logf("Generated document length: %d characters", len(contentStr))

		// Validate TOC presence and structure
		if !strings.Contains(contentStr, "Wiki Contents") {
			t.Error("Compiled document should contain TOC title")
		}

		// Check that sections are present
		sections := []string{
			"Getting Started",
			"Prerequisites",
			"Quick Start",
			"Advanced Topics",
			"Configuration",
		}

		for _, section := range sections {
			if !strings.Contains(contentStr, section) {
				t.Errorf("Document should contain section: %s", section)
			}
		}

		// Verify statistics
		stats := compiler.Stats()
		if stats.SuccessfulPages != 2 {
			t.Errorf("Expected 2 successful pages, got %d", stats.SuccessfulPages)
		}
		if stats.FailedPages != 1 {
			t.Errorf("Expected 1 failed page, got %d", stats.FailedPages)
		}

		t.Logf("🎉 PHASE 4.1 SUCCESS: Generated markdown document with %d pages and comprehensive TOC!",
			stats.SuccessfulPages)
	})
}
