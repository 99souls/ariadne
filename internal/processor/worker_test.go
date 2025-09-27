package processor

import (
	"strings"
	"testing"

	"github.com/99souls/ariadne/engine/models"
)

// Phase 2.2 TDD Tests: Content Processing Workers & HTML-to-Markdown Pipeline

func TestWorkerPool(t *testing.T) {
	t.Run("should create worker pool with configurable size", func(t *testing.T) {
		pool := NewWorkerPool(3)
		defer pool.Stop()

		if pool.WorkerCount() != 3 {
			t.Errorf("Expected 3 workers, got %d", pool.WorkerCount())
		}
	})

	t.Run("should process pages concurrently", func(t *testing.T) {
		pool := NewWorkerPool(2)
		defer pool.Stop()

		pages := []*models.Page{
			{Content: `<html><head><title>Page 1</title></head><body><h1>Title 1</h1><p>Content 1</p></body></html>`},
			{Content: `<html><head><title>Page 2</title></head><body><h1>Title 2</h1><p>Content 2</p></body></html>`},
			{Content: `<html><head><title>Page 3</title></head><body><h1>Title 3</h1><p>Content 3</p></body></html>`},
		}

		results := pool.ProcessPages(pages, "https://example.com/")

		processedCount := 0
		for result := range results {
			if result.Success {
				processedCount++
				if result.Page.Title == "" {
					t.Error("Page title should be extracted")
				}
			} else {
				t.Errorf("Processing should succeed: %v", result.Error)
			}
		}

		if processedCount != 3 {
			t.Errorf("Expected 3 processed pages, got %d", processedCount)
		}
	})

	t.Run("should handle worker errors gracefully", func(t *testing.T) {
		pool := NewWorkerPool(1)
		defer pool.Stop()

		// Invalid HTML should be handled gracefully
		pages := []*models.Page{
			{Content: `<html><head><title>Valid</title></head><body><p>Good content</p></body></html>`},
			{Content: `<<INVALID HTML>>`},
			{Content: `<html><head><title>Also Valid</title></head><body><p>More good content</p></body></html>`},
		}

		results := pool.ProcessPages(pages, "https://example.com/")

		successCount := 0
		errorCount := 0

		for result := range results {
			if result.Success {
				successCount++
			} else {
				errorCount++
			}
		}

		// Should handle valid pages successfully and invalid ones with errors
		if successCount != 2 {
			t.Errorf("Expected 2 successful pages, got %d", successCount)
		}
		if errorCount != 1 {
			t.Errorf("Expected 1 error, got %d", errorCount)
		}
	})
}

func TestHTMLToMarkdownConverter(t *testing.T) {
	tests := []struct {
		name     string
		html     string
		expected string
	}{
		{
			name: "should convert headings",
			html: `<h1>Main Title</h1><h2>Subtitle</h2><h3>Section</h3>`,
			expected: `# Main Title

## Subtitle

### Section`,
		},
		{
			name:     "should convert paragraphs and emphasis",
			html:     `<p>This is <strong>bold</strong> and <em>italic</em> text.</p>`,
			expected: `This is **bold** and *italic* text.`,
		},
		{
			name: "should convert lists",
			html: `<ul><li>Item 1</li><li>Item 2</li></ul><ol><li>First</li><li>Second</li></ol>`,
			expected: `- Item 1
- Item 2

1. First
2. Second`,
		},
		{
			name:     "should convert links and preserve URLs",
			html:     `<p>Visit <a href="https://example.com">our site</a> and <a href="/local">local page</a>.</p>`,
			expected: `Visit [our site](https://example.com) and [local page](/local).`,
		},
		{
			name:     "should convert code blocks and inline code",
			html:     `<p>Use <code>console.log()</code> for debugging.</p><pre><code>function hello() {\n  console.log("Hello!");\n}</code></pre>`,
			expected: "Use `console.log()` for debugging.\n\n```\nfunction hello() {\n  console.log(\"Hello!\");\n}\n```",
		},
		{
			name: "should convert tables",
			html: `<table><thead><tr><th>Name</th><th>Age</th></tr></thead><tbody><tr><td>John</td><td>25</td></tr><tr><td>Jane</td><td>30</td></tr></tbody></table>`,
			expected: `| Name | Age |
|------|-----|
| John | 25 |
| Jane | 30 |`,
		},
		{
			name:     "should handle images with alt text",
			html:     `<p>Here's an image: <img src="/image.jpg" alt="Description"> and more text.</p>`,
			expected: `Here's an image: ![Description](/image.jpg) and more text.`,
		},
		{
			name:     "should convert blockquotes",
			html:     `<blockquote><p>This is a quoted text with multiple sentences. It should be properly formatted.</p></blockquote>`,
			expected: `> This is a quoted text with multiple sentences. It should be properly formatted.`,
		},
	}

	converter := NewHTMLToMarkdownConverter()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := converter.Convert(tt.html)
			if err != nil {
				t.Fatalf("Convert failed: %v", err)
			}

			// Normalize whitespace for comparison
			result = normalizeMarkdownWhitespace(result)
			expected := normalizeMarkdownWhitespace(tt.expected)

			if result != expected {
				t.Errorf("Convert() = %q, expected %q", result, expected)
			}
		})
	}
}

func TestContentValidator(t *testing.T) {
	validator := NewContentValidator()

	t.Run("should validate content quality metrics", func(t *testing.T) {
		tests := []struct {
			name    string
			page    *models.Page
			isValid bool
			issues  []string
		}{
			{
				name: "high quality content should pass",
				page: &models.Page{
					Title:    "Comprehensive Guide to Web Development",
					Content:  `<h1>Web Development</h1><p>This is a detailed article about web development with substantial content. It covers multiple topics and provides valuable information to readers. The content is well-structured and informative.</p><h2>Frontend</h2><p>Frontend development involves creating user interfaces.</p>`,
					Metadata: models.PageMeta{WordCount: 45},
				},
				isValid: true,
				issues:  []string{},
			},
			{
				name: "short content should be flagged",
				page: &models.Page{
					Title:    "Short",
					Content:  `<p>Too short.</p>`,
					Metadata: models.PageMeta{WordCount: 2},
				},
				isValid: false,
				issues:  []string{"content_too_short", "title_too_short"},
			},
			{
				name: "missing title should be flagged",
				page: &models.Page{
					Title:    "",
					Content:  `<p>This content has no title but is otherwise substantial enough to meet content length requirements for processing.</p>`,
					Metadata: models.PageMeta{WordCount: 18},
				},
				isValid: false,
				issues:  []string{"missing_title"},
			},
			{
				name: "low content density should be flagged",
				page: &models.Page{
					Title:    "Navigation Heavy Page",
					Content:  `<nav><ul><li><a href="/">Home</a></li><li><a href="/about">About</a></li></ul></nav><div class="ads">Advertisement content here</div><p>Small content.</p>`,
					Metadata: models.PageMeta{WordCount: 8},
				},
				isValid: false,
				issues:  []string{"low_content_density"},
			},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				result := validator.ValidateContent(tt.page)

				if result.IsValid != tt.isValid {
					t.Errorf("Expected IsValid=%t, got %t", tt.isValid, result.IsValid)
				}

				if len(result.Issues) != len(tt.issues) {
					t.Errorf("Expected %d issues, got %d: %v", len(tt.issues), len(result.Issues), result.Issues)
				}

				for _, expectedIssue := range tt.issues {
					found := false
					for _, actualIssue := range result.Issues {
						if actualIssue == expectedIssue {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected issue '%s' not found in %v", expectedIssue, result.Issues)
					}
				}
			})
		}
	})
}

func TestSpecialContentHandling(t *testing.T) {
	processor := NewContentProcessor()

	t.Run("should preserve code blocks with syntax highlighting info", func(t *testing.T) {
		html := `<div><pre><code class="language-go">func main() {
	fmt.Println("Hello World")
}</code></pre></div>`

		page := &models.Page{Content: html}
		err := processor.ProcessPage(page, "https://example.com/")
		if err != nil {
			t.Fatalf("ProcessPage failed: %v", err)
		}

		// Should preserve the code and language info
		if !containsCodeBlock(page.Markdown) {
			t.Error("Code block should be preserved in markdown")
		}
		if !containsLanguageInfo(page.Markdown, "go") {
			t.Error("Language information should be preserved")
		}
	})

	t.Run("should handle complex tables with formatting", func(t *testing.T) {
		html := `<table>
			<thead>
				<tr><th>Feature</th><th>Status</th><th>Notes</th></tr>
			</thead>
			<tbody>
				<tr><td><strong>Crawling</strong></td><td>✅ Complete</td><td>Basic implementation</td></tr>
				<tr><td><em>Processing</em></td><td>🔄 In Progress</td><td>Advanced features</td></tr>
			</tbody>
		</table>`

		page := &models.Page{Content: html}
		err := processor.ProcessPage(page, "https://example.com/")
		if err != nil {
			t.Fatalf("ProcessPage failed: %v", err)
		}

		// Should convert to markdown table while preserving formatting
		if !containsMarkdownTable(page.Markdown) {
			t.Error("HTML table should be converted to markdown table")
		}
		if !containsTableFormatting(page.Markdown) {
			t.Error("Table formatting (bold, italic) should be preserved")
		}
	})

	t.Run("should extract and catalog images", func(t *testing.T) {
		html := `<div>
			<p>Article with images:</p>
			<img src="/diagram.png" alt="Architecture Diagram" width="500">
			<p>Some text</p>
			<figure>
				<img src="/photo.jpg" alt="Team Photo" class="responsive">
				<figcaption>Our development team</figcaption>
			</figure>
		</div>`

		page := &models.Page{Content: html}
		err := processor.ProcessPage(page, "https://example.com/")
		if err != nil {
			t.Fatalf("ProcessPage failed: %v", err)
		}

		// Should catalog images
		if len(page.Images) != 2 {
			t.Errorf("Expected 2 images, found %d", len(page.Images))
		}

		// Should preserve images in markdown
		if !containsMarkdownImage(page.Markdown, "Architecture Diagram") {
			t.Error("Image should be converted to markdown format")
		}
	})
}

func TestPipelineIntegration(t *testing.T) {
	t.Run("Phase 2.2: Complete processing pipeline with workers and markdown conversion", func(t *testing.T) {
		// Create a realistic test scenario
		pages := []*models.Page{
			{
				Content: `<!DOCTYPE html>
				<html>
				<head>
					<title>Advanced Web Scraping Guide</title>
					<meta name="description" content="Comprehensive guide to web scraping">
					<meta name="author" content="Tech Team">
				</head>
				<body>
					<nav><a href="/">Home</a></nav>
					<article>
						<h1>Web Scraping with Go</h1>
						<p>Web scraping is the process of <strong>extracting data</strong> from websites. 
						See our <a href="/tutorial">complete tutorial</a> for more details.</p>
						<h2>Key Concepts</h2>
						<ul>
							<li>HTML parsing with <code>goquery</code></li>
							<li>Concurrent processing</li>
							<li>Rate limiting</li>
						</ul>
						<blockquote>
							<p>Always respect robots.txt and rate limits!</p>
						</blockquote>
						<pre><code class="language-go">
func main() {
	fmt.Println("Hello Scraper!")
}
						</code></pre>
					</article>
					<footer>Copyright 2024</footer>
				</body>
				</html>`,
			},
			{
				Content: `<html>
				<head><title>Data Processing Patterns</title></head>
				<body>
					<h1>Worker Pool Pattern</h1>
					<p>The worker pool pattern is essential for concurrent processing.
					Check out <a href="/examples">our examples</a> to learn more.</p>
					<table>
						<thead><tr><th>Pattern</th><th>Use Case</th></tr></thead>
						<tbody>
							<tr><td>Worker Pool</td><td>CPU-intensive tasks</td></tr>
							<tr><td>Pipeline</td><td>Stream processing</td></tr>
						</tbody>
					</table>
				</body>
				</html>`,
			},
		}

		// Process through complete pipeline
		pool := NewWorkerPool(2)
		defer pool.Stop()

		results := pool.ProcessPages(pages, "https://example.com/docs/")

		successCount := 0
		for result := range results {
			if !result.Success {
				t.Errorf("Processing should succeed: %v", result.Error)
				continue
			}

			page := result.Page
			successCount++

			// Validate content cleaning
			if containsUnwantedElements(page.Content) {
				t.Error("Unwanted elements should be removed")
			}

			// Validate markdown conversion
			if page.Markdown == "" {
				t.Error("Markdown should be generated")
			}
			if !containsMarkdownHeadings(page.Markdown) {
				t.Error("Headings should be converted to markdown")
			}

			// Validate metadata extraction
			if page.Metadata.WordCount == 0 {
				t.Error("Word count should be calculated")
			}
			if page.Metadata.Description == "" && successCount == 1 {
				t.Error("Metadata should be extracted from first page")
			}

			// Validate URL conversion
			if !containsAbsoluteURLs(page.Content) {
				t.Error("Relative URLs should be converted to absolute")
			}
		}

		if successCount != 2 {
			t.Errorf("Expected 2 successful pages, got %d", successCount)
		}

		t.Logf("🔥 PHASE 2.2 SUCCESS: Complete pipeline processing %d pages!", successCount)
	})
}

// Helper functions for test validation
func normalizeMarkdownWhitespace(s string) string {
	// Normalize whitespace and newlines for markdown comparison
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")

	// Trim whitespace from each line and remove empty lines at start/end
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, " \t")
	}

	// Remove leading/trailing empty lines
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	return strings.Join(lines, "\n")
}

func containsCodeBlock(markdown string) bool {
	return strings.Contains(markdown, "```")
}

func containsLanguageInfo(markdown, language string) bool {
	return strings.Contains(markdown, "```"+language)
}

func containsMarkdownTable(markdown string) bool {
	return strings.Contains(markdown, "|") && strings.Contains(markdown, "---")
}

func containsTableFormatting(markdown string) bool {
	return strings.Contains(markdown, "**") || strings.Contains(markdown, "*")
}

func containsMarkdownImage(markdown, altText string) bool {
	return strings.Contains(markdown, "!["+altText+"]")
}

func containsUnwantedElements(html string) bool {
	unwanted := []string{"<nav>", "<footer>", "<script>", "<style>"}
	for _, element := range unwanted {
		if strings.Contains(html, element) {
			return true
		}
	}
	return false
}

func containsMarkdownHeadings(markdown string) bool {
	return strings.Contains(markdown, "# ") || strings.Contains(markdown, "## ")
}

func containsAbsoluteURLs(html string) bool {
	return strings.Contains(html, "https://example.com/")
}
