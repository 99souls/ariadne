package enhancement

import (
	"net/url"
	"strings"
	"testing"

	"github.com/99souls/ariadne/engine/internal/output/assembly"
	"github.com/99souls/ariadne/engine/models"
)

// Phase 4.3 TDD Tests: Content Enhancement System
// Testing approach: Define content enhancement behavior through tests first

func mustURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic("Invalid URL: " + rawURL)
	}
	return u
}

func TestContentEnhancerName(t *testing.T) {
	t.Run("should return correct enhancer name", func(t *testing.T) {
		enhancer := NewContentEnhancer()

		expected := "content-enhancer"
		actual := enhancer.Name()

		if actual != expected {
			t.Errorf("Expected name %q, got %q", expected, actual)
		}
	})
}

func TestContentEnhancerCreation(t *testing.T) {
	t.Run("should create enhancer with default configuration", func(t *testing.T) {
		enhancer := NewContentEnhancer()

		if enhancer == nil {
			t.Fatal("Expected enhancer to be created, got nil")
		}

		config := enhancer.Config()

		// Validate default configuration
		if !config.GenerateTOC {
			t.Error("Expected default GenerateTOC to be true")
		}

		if !config.GenerateIndex {
			t.Error("Expected default GenerateIndex to be true")
		}

		if !config.EnableSearch {
			t.Error("Expected default EnableSearch to be true")
		}

		if config.TOCMaxDepth != 6 {
			t.Errorf("Expected default TOCMaxDepth to be 6, got %d", config.TOCMaxDepth)
		}
	})

	t.Run("should create enhancer with custom configuration", func(t *testing.T) {
		config := ContentEnhancementConfig{
			GenerateTOC:     false,
			GenerateIndex:   false,
			EnableSearch:    false,
			TOCMaxDepth:     3,
			SearchMinLength: 2,
		}

		enhancer := NewContentEnhancerWithConfig(config)

		if enhancer == nil {
			t.Fatal("Expected enhancer to be created, got nil")
		}

		actualConfig := enhancer.Config()

		if actualConfig.GenerateTOC != config.GenerateTOC {
			t.Errorf("Expected GenerateTOC %v, got %v", config.GenerateTOC, actualConfig.GenerateTOC)
		}

		if actualConfig.TOCMaxDepth != config.TOCMaxDepth {
			t.Errorf("Expected TOCMaxDepth %d, got %d", config.TOCMaxDepth, actualConfig.TOCMaxDepth)
		}
	})
}

func TestEnhancedTOCGeneration(t *testing.T) {
	t.Run("should generate enhanced table of contents with cross-references", func(t *testing.T) {
		enhancer := NewContentEnhancer()

		// Add pages with headers and cross-references
		pages := []*models.Page{
			{
				URL:     mustURL("https://example.com/intro"),
				Title:   "Introduction",
				Content: "# Introduction\n\n## Overview\n\nSee the Setup Guide for details.\n\n### Key Concepts\n\nImportant concepts here.",
			},
			{
				URL:     mustURL("https://example.com/setup"),
				Title:   "Setup Guide",
				Content: "# Setup Guide\n\n## Installation\n\nInstall steps here.\n\n## Configuration\n\nRefer to API Reference.",
			},
			{
				URL:     mustURL("https://example.com/api"),
				Title:   "API Reference",
				Content: "# API Reference\n\n## Authentication\n\nAuth details.\n\n## Endpoints\n\n### GET /users\n\nUser endpoint.",
			},
		}

		// Build hierarchy for cross-reference context
		hierarchy := &assembly.HierarchyNode{
			Title: "Documentation",
			Children: []*assembly.HierarchyNode{
				{Title: "Introduction", URL: "https://example.com/intro"},
				{Title: "Setup Guide", URL: "https://example.com/setup"},
				{Title: "API Reference", URL: "https://example.com/api"},
			},
		}

		for _, page := range pages {
			enhancer.AddPage(page)
		}

		toc := enhancer.GenerateEnhancedTOC(hierarchy)

		if toc == nil {
			t.Fatal("Expected enhanced TOC to be generated, got nil")
		}

		if len(toc.Sections) == 0 {
			t.Fatal("Expected TOC to have sections")
		}

		// Verify cross-references are included
		foundCrossRef := false
		for _, section := range toc.Sections {
			if len(section.CrossReferences) > 0 {
				foundCrossRef = true
				break
			}
		}

		if !foundCrossRef {
			t.Error("Expected TOC to contain cross-references")
		}

		// Check hierarchical structure
		if len(toc.Sections) < 3 {
			t.Errorf("Expected at least 3 main sections, got %d", len(toc.Sections))
		}

		// Verify depth levels
		maxDepthFound := 0
		for _, section := range toc.Sections {
			if section.Level > maxDepthFound {
				maxDepthFound = section.Level
			}
			for _, subsection := range section.Subsections {
				if subsection.Level > maxDepthFound {
					maxDepthFound = subsection.Level
				}
			}
		}

		if maxDepthFound < 2 {
			t.Error("Expected TOC to have multiple depth levels")
		}
	})
}

func TestPageIndexGeneration(t *testing.T) {
	t.Run("should generate comprehensive page and section index", func(t *testing.T) {
		enhancer := NewContentEnhancer()

		// Add pages with various content types
		pages := []*models.Page{
			{
				URL:     mustURL("https://example.com/guide/setup"),
				Title:   "Setup Instructions",
				Content: "# Setup\n\n## Installation\nInstall the software.\n\n## Configuration\nConfigure authentication.",
			},
			{
				URL:     mustURL("https://example.com/api/auth"),
				Title:   "Authentication",
				Content: "# Authentication\n\nUse API keys for authentication.\n\n## API Keys\nGenerate keys here.",
			},
			{
				URL:     mustURL("https://example.com/tutorials/basics"),
				Title:   "Basic Tutorial",
				Content: "# Basics\n\n## Getting Started\nStart here.\n\n## First Steps\nFollow these steps.",
			},
		}

		for _, page := range pages {
			enhancer.AddPage(page)
		}

		index := enhancer.GenerateIndex()

		if index == nil {
			t.Fatal("Expected index to be generated, got nil")
		}

		// Check alphabetical sections
		if len(index.AlphabeticalSections) == 0 {
			t.Error("Expected alphabetical sections in index")
		}

		// Verify entries exist
		totalEntries := 0
		for _, section := range index.AlphabeticalSections {
			totalEntries += len(section.Entries)
		}

		if totalEntries == 0 {
			t.Error("Expected index entries to be generated")
		}

		// Check topic-based sections
		if len(index.TopicSections) == 0 {
			t.Error("Expected topic sections in index")
		}

		// Verify page-level entries
		if len(index.PageEntries) == 0 {
			t.Error("Expected page entries in index")
		}

		if len(index.PageEntries) != len(pages) {
			t.Errorf("Expected %d page entries, got %d", len(pages), len(index.PageEntries))
		}

		// Check section-level entries
		if len(index.SectionEntries) == 0 {
			t.Error("Expected section entries in index")
		}

		// Verify specific entries
		foundSetup := false
		foundAuth := false

		for _, entry := range index.PageEntries {
			if entry.Title == "Setup Instructions" {
				foundSetup = true
				if entry.URL != "https://example.com/guide/setup" {
					t.Errorf("Expected setup URL to be correct, got %q", entry.URL)
				}
			}
			if entry.Title == "Authentication" {
				foundAuth = true
			}
		}

		if !foundSetup {
			t.Error("Expected to find Setup Instructions in page index")
		}

		if !foundAuth {
			t.Error("Expected to find Authentication in page index")
		}
	})
}

func TestSearchFunctionality(t *testing.T) {
	t.Run("should generate search index and functionality", func(t *testing.T) {
		enhancer := NewContentEnhancer()

		// Add pages with searchable content
		pages := []*models.Page{
			{
				URL:     mustURL("https://example.com/auth"),
				Title:   "Authentication Guide",
				Content: "# Authentication\n\nUse JWT tokens for API authentication. Secure your applications.",
			},
			{
				URL:     mustURL("https://example.com/security"),
				Title:   "Security Best Practices",
				Content: "# Security\n\nImplement proper authentication and authorization. Use HTTPS encryption.",
			},
			{
				URL:     mustURL("https://example.com/tutorial"),
				Title:   "Getting Started Tutorial",
				Content: "# Tutorial\n\nLearn the basics of our API. Start with authentication setup.",
			},
		}

		for _, page := range pages {
			enhancer.AddPage(page)
		}

		searchIndex := enhancer.GenerateSearchIndex()

		if searchIndex == nil {
			t.Fatal("Expected search index to be generated, got nil")
		}

		// Check search entries
		if len(searchIndex.Entries) == 0 {
			t.Fatal("Expected search entries to be generated")
		}

		// Test search functionality
		results := enhancer.Search("authentication")

		if len(results) == 0 {
			t.Error("Expected search results for 'authentication'")
		}

		// Verify relevance scoring
		foundHighScore := false
		for _, result := range results {
			if result.Score > 0.5 {
				foundHighScore = true
			}

			if result.Title == "" {
				t.Error("Expected search result to have title")
			}

			if result.URL == "" {
				t.Error("Expected search result to have URL")
			}

			if result.Snippet == "" {
				t.Error("Expected search result to have snippet")
			}
		}

		if !foundHighScore {
			t.Error("Expected at least one high-relevance search result")
		}

		// Test multiple term search
		multiResults := enhancer.Search("security https")

		if len(multiResults) == 0 {
			t.Error("Expected search results for multiple terms")
		}

		// Test search with no results
		noResults := enhancer.Search("xyz123nonexistent")

		if len(noResults) != 0 {
			t.Error("Expected no results for non-existent term")
		}

		// Generate search JavaScript
		searchJS := enhancer.GenerateSearchJavaScript()

		if searchJS == "" {
			t.Error("Expected search JavaScript to be generated")
		}

		if !strings.Contains(searchJS, "function search") {
			t.Error("Expected search JavaScript to contain search function")
		}

		if !strings.Contains(searchJS, "searchIndex") {
			t.Error("Expected search JavaScript to contain search index data")
		}
	})
}

func TestCustomStylingSystem(t *testing.T) {
	t.Run("should generate custom CSS styling system", func(t *testing.T) {
		enhancer := NewContentEnhancer()

		// Configure styling options
		styleConfig := StylingConfig{
			Theme:           "professional",
			PrimaryColor:    "#2c3e50",
			SecondaryColor:  "#3498db",
			FontFamily:      "Inter, system-ui, sans-serif",
			FontSize:        "16px",
			EnableCodeTheme: true,
			EnablePrintCSS:  true,
		}

		enhancer.SetStylingConfig(styleConfig)

		css := enhancer.GenerateCustomCSS()

		if css == "" {
			t.Fatal("Expected custom CSS to be generated")
		}

		// Check theme elements
		if !strings.Contains(css, styleConfig.PrimaryColor) {
			t.Error("Expected CSS to contain primary color")
		}

		if !strings.Contains(css, styleConfig.FontFamily) {
			t.Error("Expected CSS to contain font family")
		}

		// Check responsive design
		if !strings.Contains(css, "@media") {
			t.Error("Expected CSS to contain responsive media queries")
		}

		// Check code syntax highlighting
		if !strings.Contains(css, ".code") && !strings.Contains(css, "pre") {
			t.Error("Expected CSS to contain code styling")
		}

		// Check print styles
		if !strings.Contains(css, "@media print") {
			t.Error("Expected CSS to contain print styles")
		}

		// Generate multiple themes
		themes := []string{"professional", "dark", "minimal", "academic"}

		for _, theme := range themes {
			themeConfig := styleConfig
			themeConfig.Theme = theme
			enhancer.SetStylingConfig(themeConfig)

			themeCSS := enhancer.GenerateCustomCSS()

			if themeCSS == "" {
				t.Errorf("Expected CSS to be generated for theme %q", theme)
			}

			// Verify theme-specific elements
			if theme == "dark" && !strings.Contains(themeCSS, "#333") && !strings.Contains(themeCSS, "dark") {
				t.Error("Expected dark theme to contain dark colors")
			}
		}
	})
}

func TestContentEnhancerIntegration(t *testing.T) {
	t.Run("Phase 4.3: Complete content enhancement with TOC, index, search, and styling", func(t *testing.T) {
		enhancer := NewContentEnhancer()

		// Configure comprehensive enhancement
		config := ContentEnhancementConfig{
			GenerateTOC:     true,
			GenerateIndex:   true,
			EnableSearch:    true,
			TOCMaxDepth:     4,
			SearchMinLength: 2,
		}

		enhancer.SetConfig(config)

		// Add comprehensive test data
		testPages := []*models.Page{
			{
				URL:     mustURL("https://docs.example.com/"),
				Title:   "Documentation Home",
				Content: "# Welcome to Our Documentation\n\n## Overview\nComprehensive guide.\n\n### Getting Started\nStart here.",
			},
			{
				URL:     mustURL("https://docs.example.com/guide/setup"),
				Title:   "Setup Guide",
				Content: "# Setup Instructions\n\n## Prerequisites\nWhat you need.\n\n## Installation\nHow to install.",
			},
			{
				URL:     mustURL("https://docs.example.com/guide/config"),
				Title:   "Configuration",
				Content: "# Configuration Guide\n\n## Basic Config\nBasic setup.\n\n## Advanced Options\nAdvanced configuration.",
			},
			{
				URL:     mustURL("https://docs.example.com/api/"),
				Title:   "API Reference",
				Content: "# API Documentation\n\n## Authentication\nHow to authenticate.\n\n## Endpoints\n\n### Users\nUser management.",
			},
			{
				URL:     mustURL("https://docs.example.com/api/auth"),
				Title:   "Authentication Details",
				Content: "# Authentication\n\n## JWT Tokens\nUse JWT for auth.\n\n## API Keys\nAlternative method.",
			},
		}

		// Add pages to enhancer
		for _, page := range testPages {
			enhancer.AddPage(page)
		}

		// Create mock hierarchy
		hierarchy := &assembly.HierarchyNode{
			Title: "Documentation",
			Children: []*assembly.HierarchyNode{
				{
					Title: "Guide",
					Children: []*assembly.HierarchyNode{
						{Title: "Setup Guide", URL: "https://docs.example.com/guide/setup"},
						{Title: "Configuration", URL: "https://docs.example.com/guide/config"},
					},
				},
				{
					Title: "API",
					Children: []*assembly.HierarchyNode{
						{Title: "API Reference", URL: "https://docs.example.com/api/"},
						{Title: "Authentication Details", URL: "https://docs.example.com/api/auth"},
					},
				},
			},
		}

		// Test Enhanced TOC Generation
		toc := enhancer.GenerateEnhancedTOC(hierarchy)
		if toc == nil {
			t.Fatal("Expected enhanced TOC to be generated")
		}

		if len(toc.Sections) == 0 {
			t.Fatal("Expected TOC to have sections")
		}

		t.Logf("Generated enhanced TOC with %d main sections", len(toc.Sections))

		// Test Index Generation
		index := enhancer.GenerateIndex()
		if index == nil {
			t.Fatal("Expected index to be generated")
		}

		if len(index.PageEntries) != len(testPages) {
			t.Errorf("Expected %d page entries, got %d", len(testPages), len(index.PageEntries))
		}

		t.Logf("Generated index with %d page entries and %d section entries",
			len(index.PageEntries), len(index.SectionEntries))

		// Test Search Functionality
		searchIndex := enhancer.GenerateSearchIndex()
		if searchIndex == nil {
			t.Fatal("Expected search index to be generated")
		}

		searchResults := enhancer.Search("authentication")
		if len(searchResults) == 0 {
			t.Error("Expected search results for 'authentication'")
		}

		t.Logf("Search for 'authentication' returned %d results", len(searchResults))

		// Test Custom Styling
		styleConfig := StylingConfig{
			Theme:        "professional",
			PrimaryColor: "#2c3e50",
			FontFamily:   "Inter, sans-serif",
		}

		enhancer.SetStylingConfig(styleConfig)
		customCSS := enhancer.GenerateCustomCSS()

		if customCSS == "" {
			t.Fatal("Expected custom CSS to be generated")
		}

		t.Logf("Generated custom CSS with %d characters", len(customCSS))

		// Test Enhancement Statistics
		stats := enhancer.Stats()

		if stats.TOCSections == 0 {
			t.Error("Expected TOC sections in statistics")
		}

		if stats.IndexEntries == 0 {
			t.Error("Expected index entries in statistics")
		}

		if stats.SearchablePages == 0 {
			t.Error("Expected searchable pages in statistics")
		}

		// Generate complete enhanced output
		enhancedOutput := enhancer.GenerateEnhancedOutput(hierarchy)

		if enhancedOutput == nil {
			t.Fatal("Expected enhanced output to be generated")
		}

		if enhancedOutput.TOC == nil {
			t.Error("Expected enhanced output to contain TOC")
		}

		if enhancedOutput.Index == nil {
			t.Error("Expected enhanced output to contain index")
		}

		if enhancedOutput.SearchData == nil {
			t.Error("Expected enhanced output to contain search data")
		}

		if enhancedOutput.CustomCSS == "" {
			t.Error("Expected enhanced output to contain custom CSS")
		}

		t.Logf("🎉 PHASE 4.3 SUCCESS: Content enhancement completed with enhanced TOC (%d sections), comprehensive index (%d entries), search functionality (%d results), and custom styling!",
			stats.TOCSections, stats.IndexEntries, len(searchResults))
	})
}
