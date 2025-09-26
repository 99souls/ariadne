package processor

import (
	"strings"
	"testing"

	"ariadne/pkg/models"
)

// Phase 2.1 TDD Tests: HTML Content Cleaning

func TestContentSelector(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		selectors   []string
		expected    string
		description string
	}{
		{
			name:        "should extract main article content",
			html:        `<html><body><nav>Navigation</nav><article><h1>Title</h1><p>Content</p></article><footer>Footer</footer></body></html>`,
			selectors:   []string{"article"},
			expected:    "<h1>Title</h1><p>Content</p>",
			description: "Extract content from article tag while ignoring nav/footer",
		},
		{
			name:        "should extract content class",
			html:        `<html><body><div class="sidebar">Ads</div><div class="content"><h2>Main</h2><p>Text</p></div></body></html>`,
			selectors:   []string{".content"},
			expected:    "<h2>Main</h2><p>Text</p>",
			description: "Target specific content class selector",
		},
		{
			name:        "should try multiple selectors in priority order",
			html:        `<html><body><main><section><h1>Main Content</h1><p>Important text</p></section></main></body></html>`,
			selectors:   []string{"article", "main", "section"},
			expected:    "<section><h1>Main Content</h1><p>Important text</p></section>",
			description: "Use first matching selector from priority list",
		},
		{
			name:        "should fallback to body if no selector matches",
			html:        `<html><body><h1>Title</h1><p>Content</p><script>alert('hi')</script></body></html>`,
			selectors:   []string{"article", ".content"},
			expected:    "<h1>Title</h1><p>Content</p>", // script should be removed
			description: "Fallback to body content when no selectors match",
		},
	}

	processor := NewContentProcessor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ExtractContent(tt.html, tt.selectors)
			if err != nil {
				t.Fatalf("ExtractContent failed: %v", err)
			}

			// Normalize whitespace for comparison
			result = strings.TrimSpace(result)
			expected := strings.TrimSpace(tt.expected)

			if result != expected {
				t.Errorf("ExtractContent() = %q, expected %q", result, expected)
			}
		})
	}
}

func TestUnwantedElementRemoval(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name:     "should remove navigation elements",
			html:     `<div><nav>Menu</nav><h1>Title</h1><p>Content</p></div>`,
			expected: `<div><h1>Title</h1><p>Content</p></div>`,
		},
		{
			name:     "should remove footer elements",
			html:     `<div><h1>Title</h1><p>Content</p><footer>Copyright</footer></div>`,
			expected: `<div><h1>Title</h1><p>Content</p></div>`,
		},
		{
			name:     "should remove sidebar and ads",
			html:     `<div><aside class="sidebar">Ads</aside><h1>Title</h1><div class="advertisement">Buy now!</div><p>Content</p></div>`,
			expected: `<div><h1>Title</h1><p>Content</p></div>`,
		},
		{
			name:     "should remove script and style tags",
			html:     `<div><script>console.log('hi')</script><h1>Title</h1><style>body{}</style><p>Content</p></div>`,
			expected: `<div><h1>Title</h1><p>Content</p></div>`,
		},
		{
			name:     "should remove comments and tracking pixels",
			html:     `<div><!-- Comment --><h1>Title</h1><img src="track.gif" width="1" height="1"><p>Content</p></div>`,
			expected: `<div><h1>Title</h1><p>Content</p></div>`,
		},
	}

	processor := NewContentProcessor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.RemoveUnwantedElements(tt.html)
			if err != nil {
				t.Fatalf("RemoveUnwantedElements failed: %v", err)
			}

			// Normalize whitespace
			result = normalizeWhitespace(result)
			expected := normalizeWhitespace(tt.expected)

			if result != expected {
				t.Errorf("RemoveUnwantedElements() = %q, expected %q", result, expected)
			}
		})
	}
}

func TestRelativeURLConversion(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		baseURL  string
		expected string
	}{
		{
			name:     "should convert relative links",
			html:     `<div><a href="/page1">Link</a><a href="page2">Link2</a></div>`,
			baseURL:  "https://example.com/docs/",
			expected: `<div><a href="https://example.com/page1">Link</a><a href="https://example.com/docs/page2">Link2</a></div>`,
		},
		{
			name:     "should convert relative images",
			html:     `<div><img src="/image.jpg"><img src="thumb.png"></div>`,
			baseURL:  "https://example.com/gallery/",
			expected: ``, // We'll check content separately
		},
		{
			name:     "should preserve absolute URLs",
			html:     `<div><a href="https://other.com/page">External</a><img src="http://cdn.com/image.jpg"></div>`,
			baseURL:  "https://example.com/",
			expected: ``, // We'll check content separately
		},
		{
			name:     "should handle protocol-relative URLs",
			html:     `<div><a href="//other.com/page">Link</a></div>`,
			baseURL:  "https://example.com/",
			expected: `<div><a href="https://other.com/page">Link</a></div>`,
		},
	}

	processor := NewContentProcessor()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := processor.ConvertRelativeURLs(tt.html, tt.baseURL)
			if err != nil {
				t.Fatalf("ConvertRelativeURLs failed: %v", err)
			}

			// For specific test cases, check for expected URL conversions
			switch tt.name {
			case "should convert relative links":
				if !strings.Contains(result, "https://example.com/page1") {
					t.Error("Should convert /page1 to absolute URL")
				}
				if !strings.Contains(result, "https://example.com/docs/page2") {
					t.Error("Should convert page2 to absolute URL relative to base")
				}

			case "should convert relative images":
				if !strings.Contains(result, "https://example.com/image.jpg") {
					t.Error("Should convert /image.jpg to absolute URL")
				}
				if !strings.Contains(result, "https://example.com/gallery/thumb.png") {
					t.Error("Should convert thumb.png to absolute URL relative to base")
				}

			case "should preserve absolute URLs":
				if !strings.Contains(result, "https://other.com/page") {
					t.Error("Should preserve absolute URLs")
				}
				if !strings.Contains(result, "http://cdn.com/image.jpg") {
					t.Error("Should preserve absolute image URLs")
				}

			case "should handle protocol-relative URLs":
				if !strings.Contains(result, "https://other.com/page") {
					t.Error("Should convert protocol-relative URLs")
				}
			}
		})
	}
}

func TestMetadataExtraction(t *testing.T) {
	html := `<html>
		<head>
			<title>Article Title</title>
			<meta name="description" content="Article description">
			<meta name="author" content="John Doe">
			<meta name="keywords" content="test, article">
			<meta property="og:title" content="Social Title">
			<meta property="article:published_time" content="2024-01-01T12:00:00Z">
		</head>
		<body><h1>Content</h1></body>
	</html>`

	processor := NewContentProcessor()

	// Test that we can extract a complete page with title and metadata
	page := &models.Page{Content: html}
	err := processor.ProcessPage(page, "https://example.com/")
	if err != nil {
		t.Fatalf("ProcessPage failed: %v", err)
	}

	// Test title extraction (separate from metadata)
	if page.Title != "Article Title" {
		t.Errorf("Expected title 'Article Title', got '%s'", page.Title)
	}

	// Test basic metadata
	if page.Metadata.Description != "Article description" {
		t.Errorf("Expected description 'Article description', got '%s'", page.Metadata.Description)
	}

	if page.Metadata.Author != "John Doe" {
		t.Errorf("Expected author 'John Doe', got '%s'", page.Metadata.Author)
	}

	// Test keywords
	expectedKeywords := []string{"test", "article"}
	if len(page.Metadata.Keywords) != len(expectedKeywords) {
		t.Errorf("Expected %d keywords, got %d", len(expectedKeywords), len(page.Metadata.Keywords))
	}

	// Test OpenGraph data
	if page.Metadata.OpenGraph.Title != "Social Title" {
		t.Errorf("Expected OG title 'Social Title', got '%s'", page.Metadata.OpenGraph.Title)
	}
}

func TestContentProcessingIntegration(t *testing.T) {
	t.Run("Phase 2.1: Complete content cleaning pipeline", func(t *testing.T) {
		// Simulate a real webpage with various elements
		html := `<!DOCTYPE html>
		<html>
		<head>
			<title>Wiki Article</title>
			<meta name="description" content="A test wiki article">
		</head>
		<body>
			<nav><ul><li><a href="/">Home</a></li></ul></nav>
			<aside class="sidebar">Advertisement</aside>
			<article>
				<h1>Main Article Title</h1>
				<p>This is the main content with <a href="/related">relative link</a>.</p>
				<img src="image.jpg" alt="Test image">
				<h2>Subsection</h2>
				<p>More content here.</p>
			</article>
			<footer>Copyright 2024</footer>
			<script>analytics.track();</script>
		</body>
		</html>`

		processor := NewContentProcessor()

		// Process the content through the complete pipeline
		page := &models.Page{
			Title:   "Original Title",
			Content: html,
		}

		err := processor.ProcessPage(page, "https://example.com/wiki/")
		if err != nil {
			t.Fatalf("ProcessPage failed: %v", err)
		}

		// Validate results
		if page.Title != "Wiki Article" {
			t.Errorf("Title should be extracted: expected 'Wiki Article', got '%s'", page.Title)
		}

		// Content should be cleaned and have absolute URLs
		if !strings.Contains(page.Content, "https://example.com/related") {
			t.Error("Relative URLs should be converted to absolute")
		}

		if strings.Contains(page.Content, "<nav>") {
			t.Error("Navigation elements should be removed")
		}

		if strings.Contains(page.Content, "<script>") {
			t.Error("Script elements should be removed")
		}

		// Metadata should be populated
		if page.Metadata.Description != "A test wiki article" {
			t.Errorf("Description should be extracted: got '%s'", page.Metadata.Description)
		}

		if page.Metadata.WordCount == 0 {
			t.Error("Word count should be calculated")
		}
	})
}

// Helper function to normalize whitespace for comparisons
func normalizeWhitespace(s string) string {
	// Replace multiple whitespaces with single space
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	for strings.Contains(s, "  ") {
		s = strings.ReplaceAll(s, "  ", " ")
	}
	return strings.TrimSpace(s)
}
