package assembly

import (
	"net/url"
	"testing"
	"time"

	"github.com/99souls/ariadne/engine/output"
	"github.com/99souls/ariadne/engine/models"
)

// Phase 4.2 TDD Tests: Document Assembly System
// Testing approach: Define document assembly behavior through tests first

func mustURL(rawURL string) *url.URL {
	u, err := url.Parse(rawURL)
	if err != nil {
		panic("Invalid URL: " + rawURL)
	}
	return u
}

func TestDocumentAssemblerName(t *testing.T) {
	t.Run("should return correct assembler name", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		expected := "document-assembler"
		actual := assembler.Name()

		if actual != expected {
			t.Errorf("Expected name %q, got %q", expected, actual)
		}
	})
}

func TestDocumentAssemblerClose(t *testing.T) {
	t.Run("should close registered sinks", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		// Register mock sinks
		mockSink1 := &MockOutputSink{name: "mock1"}
		mockSink2 := &MockOutputSink{name: "mock2"}

		assembler.RegisterSink(mockSink1)
		assembler.RegisterSink(mockSink2)

		err := assembler.Close()
		if err != nil {
			t.Errorf("Expected no error on close, got %v", err)
		}

		if !mockSink1.closed {
			t.Error("Expected mock sink 1 to be closed")
		}

		if !mockSink2.closed {
			t.Error("Expected mock sink 2 to be closed")
		}
	})
}

func TestDocumentAssemblerCreation(t *testing.T) {
	t.Run("should create assembler with default configuration", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		if assembler == nil {
			t.Fatal("Expected assembler to be created, got nil")
		}

		config := assembler.Config()

		// Validate default configuration
		if !config.EnableHierarchy {
			t.Error("Expected default EnableHierarchy to be true")
		}

		if !config.EnableCrossReferences {
			t.Error("Expected default EnableCrossReferences to be true")
		}

		if !config.EnableDeduplication {
			t.Error("Expected default EnableDeduplication to be true")
		}

		if !config.IncludeMetadata {
			t.Error("Expected default IncludeMetadata to be true")
		}
	})

	t.Run("should create assembler with custom configuration", func(t *testing.T) {
		config := DocumentAssemblyConfig{
			EnableHierarchy:       false,
			EnableCrossReferences: false,
			EnableDeduplication:   false,
			IncludeMetadata:       false,
		}

		assembler := NewDocumentAssemblerWithConfig(config)

		if assembler == nil {
			t.Fatal("Expected assembler to be created, got nil")
		}

		actualConfig := assembler.Config()

		if actualConfig.EnableHierarchy != config.EnableHierarchy {
			t.Errorf("Expected EnableHierarchy %v, got %v", config.EnableHierarchy, actualConfig.EnableHierarchy)
		}

		if actualConfig.EnableCrossReferences != config.EnableCrossReferences {
			t.Errorf("Expected EnableCrossReferences %v, got %v", config.EnableCrossReferences, actualConfig.EnableCrossReferences)
		}
	})
}

func TestDocumentAssemblerSinkRegistration(t *testing.T) {
	t.Run("should register and manage output sinks", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		// Create mock sinks
		mockSink1 := &MockOutputSink{name: "mock-sink-1"}
		mockSink2 := &MockOutputSink{name: "mock-sink-2"}

		// Register sinks
		assembler.RegisterSink(mockSink1)
		assembler.RegisterSink(mockSink2)

		sinks := assembler.RegisteredSinks()
		if len(sinks) != 2 {
			t.Errorf("Expected 2 registered sinks, got %d", len(sinks))
		}

		// Verify sink names
		sinkNames := make([]string, len(sinks))
		for i, sink := range sinks {
			sinkNames[i] = sink.Name()
		}

		expectedNames := []string{"mock-sink-1", "mock-sink-2"}
		for _, expected := range expectedNames {
			found := false
			for _, actual := range sinkNames {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected to find sink %q in registered sinks", expected)
			}
		}
	})
}

func TestDocumentHierarchyGeneration(t *testing.T) {
	t.Run("should organize pages by URL hierarchy", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		// Add pages with hierarchical URLs
		pages := []*models.Page{
			{
				URL:     mustURL("https://example.com/"),
				Title:   "Home",
				Content: "# Welcome\n\nThis is the home page.",
			},
			{
				URL:     mustURL("https://example.com/docs/"),
				Title:   "Documentation",
				Content: "# Documentation\n\nMain documentation section.",
			},
			{
				URL:     mustURL("https://example.com/docs/guide/"),
				Title:   "User Guide",
				Content: "# User Guide\n\nDetailed user instructions.",
			},
			{
				URL:     mustURL("https://example.com/docs/guide/setup"),
				Title:   "Setup Instructions",
				Content: "# Setup\n\nHow to get started.",
			},
			{
				URL:     mustURL("https://example.com/api/"),
				Title:   "API Reference",
				Content: "# API\n\nAPI documentation.",
			},
		}

		for _, page := range pages {
			_ = assembler.Write(&models.CrawlResult{Page: page, Success: true})
		}

		hierarchy := assembler.GenerateHierarchy()

		if hierarchy == nil {
			t.Fatal("Expected hierarchy to be generated, got nil")
		}

		// Check root level
		if hierarchy.Title != "Root" {
			t.Errorf("Expected root title 'Root', got %q", hierarchy.Title)
		}

		if len(hierarchy.Children) == 0 {
			t.Fatal("Expected root to have children")
		}

		// Verify hierarchical structure
		homeFound := false
		docsFound := false
		apiFound := false

		for _, child := range hierarchy.Children {
			switch child.Title {
			case "Home":
				homeFound = true
			case "docs":
				docsFound = true
				// Check docs has guide child
				guideFound := false
				for _, grandchild := range child.Children {
					if grandchild.Title == "guide" {
						guideFound = true
						// Check guide has setup child
						setupFound := false
						for _, greatgrandchild := range grandchild.Children {
							if greatgrandchild.Title == "Setup Instructions" {
								setupFound = true
							}
						}
						if !setupFound {
							t.Error("Expected guide to contain setup page")
						}
					}
				}
				if !guideFound {
					t.Error("Expected docs to contain guide section")
				}
			case "API Reference", "api":
				apiFound = true
			}
		}

		if !homeFound {
			t.Error("Expected to find Home in hierarchy")
		}
		if !docsFound {
			t.Error("Expected to find docs in hierarchy")
		}
		if !apiFound {
			t.Error("Expected to find API Reference in hierarchy")
		}
	})
}

func TestCrossReferenceGeneration(t *testing.T) {
	t.Run("should generate cross-references between related pages", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		// Add pages with potential cross-references
		pages := []*models.Page{
			{
				URL:     mustURL("https://example.com/docs/setup"),
				Title:   "Setup Guide",
				Content: "# Setup\n\nSee the API documentation for more details.",
			},
			{
				URL:     mustURL("https://example.com/docs/api"),
				Title:   "API Reference",
				Content: "# API\n\nRefer to the setup guide first.",
			},
			{
				URL:     mustURL("https://example.com/docs/examples"),
				Title:   "Examples",
				Content: "# Examples\n\nCheck setup and API docs for context.",
			},
		}

		for _, page := range pages {
			_ = assembler.Write(&models.CrawlResult{Page: page, Success: true})
		}

		crossRefs := assembler.GenerateCrossReferences()

		if len(crossRefs) == 0 {
			t.Error("Expected cross-references to be generated")
		}

		// Check that cross-references contain expected relationships
		setupURL := "https://example.com/docs/setup"
		apiURL := "https://example.com/docs/api"

		setupRefs, exists := crossRefs[setupURL]
		if !exists {
			t.Error("Expected cross-references for setup page")
		} else {
			apiRefFound := false
			for _, ref := range setupRefs {
				if ref.TargetURL == apiURL {
					apiRefFound = true
					if ref.RelationshipType != "mentions" && ref.RelationshipType != "references" && ref.RelationshipType != "related_topic" {
						t.Errorf("Expected relationship type 'mentions', 'references', or 'related_topic', got %q", ref.RelationshipType)
					}
				}
			}
			if !apiRefFound {
				t.Error("Expected setup page to reference API page")
			}
		}
	})
}

func TestContentDeduplication(t *testing.T) {
	t.Run("should identify and handle duplicate content", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		// Add pages with duplicate content
		duplicateContent := "# Common Section\n\nThis content appears on multiple pages."

		pages := []*models.Page{
			{
				URL:     mustURL("https://example.com/page1"),
				Title:   "Page One",
				Content: duplicateContent + "\n\n## Page 1 Specific\n\nUnique content for page 1.",
			},
			{
				URL:     mustURL("https://example.com/page2"),
				Title:   "Page Two",
				Content: duplicateContent + "\n\n## Page 2 Specific\n\nUnique content for page 2.",
			},
			{
				URL:     mustURL("https://example.com/page3"),
				Title:   "Page Three",
				Content: "# Different Content\n\nThis page has completely different content.",
			},
		}

		for _, page := range pages {
			_ = assembler.Write(&models.CrawlResult{Page: page, Success: true})
		}

		duplicates := assembler.DetectDuplicateContent()

		if len(duplicates) == 0 {
			t.Error("Expected duplicate content to be detected")
		}

		// Check that duplicate group contains expected pages
		foundDuplicateGroup := false
		for _, group := range duplicates {
			if len(group.Pages) >= 2 {
				page1Found := false
				page2Found := false

				for _, pageURL := range group.Pages {
					if pageURL == "https://example.com/page1" {
						page1Found = true
					}
					if pageURL == "https://example.com/page2" {
						page2Found = true
					}
				}

				if page1Found && page2Found {
					foundDuplicateGroup = true

					if group.SimilarityScore < 0.5 {
						t.Errorf("Expected high similarity score for duplicates, got %f", group.SimilarityScore)
					}
				}
			}
		}

		if !foundDuplicateGroup {
			t.Error("Expected to find duplicate group containing page1 and page2")
		}
	})
}

func TestMetadataExtraction(t *testing.T) {
	t.Run("should extract and enrich page metadata", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		// Add pages with various metadata
		pages := []*models.Page{
			{
				URL:     mustURL("https://example.com/blog/post1"),
				Title:   "First Blog Post",
				Content: "# First Post\n\nPublished on January 1, 2024\n\nBy: John Doe",
				Metadata: models.PageMeta{
					Author:      "John Doe",
					Description: "First blog post",
					WordCount:   150,
					PublishDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				},
			},
			{
				URL:     mustURL("https://example.com/docs/api"),
				Title:   "API Documentation",
				Content: "# API Reference\n\nVersion 2.0\n\nLast updated: March 2024",
				Metadata: models.PageMeta{
					Description: "API reference documentation",
					Keywords:    []string{"API", "reference", "documentation"},
					WordCount:   500,
				},
			},
		}

		for _, page := range pages {
			_ = assembler.Write(&models.CrawlResult{Page: page, Success: true})
		}

		enrichedPages := assembler.ExtractMetadata()

		if len(enrichedPages) != 2 {
			t.Errorf("Expected 2 enriched pages, got %d", len(enrichedPages))
		}

		// Check metadata enrichment
		for _, enriched := range enrichedPages {
			if enriched.URL == "https://example.com/blog/post1" {
				if enriched.Category != "blog" {
					t.Errorf("Expected blog category for blog post, got %q", enriched.Category)
				}

				if enriched.ContentType != "article" {
					t.Errorf("Expected article content type for blog post, got %q", enriched.ContentType)
				}
			}

			if enriched.URL == "https://example.com/docs/api" {
				if enriched.Category != "documentation" {
					t.Errorf("Expected documentation category for API docs, got %q", enriched.Category)
				}

				if enriched.ContentType != "reference" {
					t.Errorf("Expected reference content type for API docs, got %q", enriched.ContentType)
				}
			}
		}
	})
}

func TestDocumentAssemblerIntegration(t *testing.T) {
	t.Run("Phase 4.2: Complete document assembly with hierarchy and cross-references", func(t *testing.T) {
		assembler := NewDocumentAssembler()

		// Register mock sinks to test output coordination
		mockMarkdown := &MockOutputSink{name: "markdown"}
		mockHTML := &MockOutputSink{name: "html"}

		assembler.RegisterSink(mockMarkdown)
		assembler.RegisterSink(mockHTML)

		// Add comprehensive test data with hierarchical structure
		testPages := []*models.Page{
			{
				URL:     mustURL("https://example.com/"),
				Title:   "Documentation Home",
				Content: "# Welcome\n\nMain documentation portal. See user guide and API reference.",
			},
			{
				URL:     mustURL("https://example.com/guide/"),
				Title:   "User Guide",
				Content: "# User Guide\n\nComprehensive user documentation. Check setup first.",
			},
			{
				URL:     mustURL("https://example.com/guide/setup"),
				Title:   "Setup Instructions",
				Content: "# Setup\n\nFollow these steps. Then proceed to advanced configuration.",
			},
			{
				URL:     mustURL("https://example.com/guide/advanced"),
				Title:   "Advanced Configuration",
				Content: "# Advanced Setup\n\nAdvanced options. Requires basic setup completion.",
			},
			{
				URL:     mustURL("https://example.com/api/"),
				Title:   "API Reference",
				Content: "# API Documentation\n\nComplete API reference. See authentication guide.",
			},
			{
				URL:     mustURL("https://example.com/api/auth"),
				Title:   "Authentication",
				Content: "# API Authentication\n\nHow to authenticate API requests.",
			},
		}

		// Write all pages
		for _, page := range testPages {
			err := assembler.Write(&models.CrawlResult{Page: page, Success: true})
			if err != nil {
				t.Fatalf("Failed to write page %q: %v", page.URL, err)
			}
		}

		// Test hierarchy generation
		hierarchy := assembler.GenerateHierarchy()
		if hierarchy == nil {
			t.Fatal("Expected hierarchy to be generated")
		}

		if len(hierarchy.Children) == 0 {
			t.Fatal("Expected hierarchy to have children")
		}

		t.Logf("Generated hierarchy with %d top-level sections", len(hierarchy.Children))

		// Test cross-reference generation
		crossRefs := assembler.GenerateCrossReferences()
		if len(crossRefs) == 0 {
			t.Error("Expected cross-references to be generated")
		}

		t.Logf("Generated %d cross-reference relationships", len(crossRefs))

		// Test deduplication
		duplicates := assembler.DetectDuplicateContent()
		t.Logf("Detected %d duplicate content groups", len(duplicates))

		// Test metadata extraction
		enrichedPages := assembler.ExtractMetadata()
		if len(enrichedPages) != len(testPages) {
			t.Errorf("Expected %d enriched pages, got %d", len(testPages), len(enrichedPages))
		}

		// Test output coordination - flush to all sinks
		err := assembler.Flush()
		if err != nil {
			t.Fatalf("Failed to flush assembler: %v", err)
		}

		// Verify sinks received data
		if mockMarkdown.writeCount == 0 {
			t.Error("Expected markdown sink to receive data")
		}

		if mockHTML.writeCount == 0 {
			t.Error("Expected HTML sink to receive data")
		}

		// Verify statistics
		stats := assembler.Stats()
		if stats.TotalPages != len(testPages) {
			t.Errorf("Expected %d total pages, got %d", len(testPages), stats.TotalPages)
		}

		if stats.CrossReferences == 0 {
			t.Error("Expected cross-references to be generated")
		}

		t.Logf("🎉 PHASE 4.2 SUCCESS: Document assembly completed with %d pages, %d cross-references, and hierarchical organization!",
			stats.TotalPages, stats.CrossReferences)
	})
}

// MockOutputSink for testing
type MockOutputSink struct {
	name       string
	writeCount int
	closed     bool
}

func (m *MockOutputSink) Write(result *models.CrawlResult) error {
	m.writeCount++
	return nil
}

func (m *MockOutputSink) Flush() error {
	return nil
}

func (m *MockOutputSink) Close() error {
	m.closed = true
	return nil
}

func (m *MockOutputSink) Name() string {
	return m.name
}

var _ output.OutputSink = (*MockOutputSink)(nil) // Compile-time interface check
