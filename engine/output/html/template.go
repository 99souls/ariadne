package html

import (
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/99souls/ariadne/engine/output"
	"github.com/99souls/ariadne/engine/models"
)

// HTMLTemplateConfig defines configuration for HTML template rendering
type HTMLTemplateConfig struct {
	OutputPath        string `json:"output_path"`
	Title             string `json:"title"`
	Theme             string `json:"theme"`
	IncludeNavigation bool   `json:"include_navigation"`
	IncludeTOC        bool   `json:"include_toc"`
	CustomCSS         string `json:"custom_css"`
	CustomJS          string `json:"custom_js"`
}

// HTMLTemplateStats tracks rendering statistics
type HTMLTemplateStats struct {
	TotalPages      int           `json:"total_pages"`
	SuccessfulPages int           `json:"successful_pages"`
	FailedPages     int           `json:"failed_pages"`
	StartTime       time.Time     `json:"start_time"`
	ProcessingTime  time.Duration `json:"processing_time"`
}

// NavigationNode represents a hierarchical navigation structure
type NavigationNode struct {
	Title    string            `json:"title"`
	URL      string            `json:"url"`
	Children []*NavigationNode `json:"children"`
	Level    int               `json:"level"`
}

// HTMLTemplateRenderer implements OutputSink for HTML template rendering
type HTMLTemplateRenderer struct {
	config HTMLTemplateConfig
	pages  []*models.Page
	stats  HTMLTemplateStats
	mutex  sync.RWMutex
}

// Default configuration for HTML template rendering
func DefaultHTMLTemplateConfig() HTMLTemplateConfig {
	return HTMLTemplateConfig{
		OutputPath:        "output.html",
		Title:             "Site Documentation",
		Theme:             "default",
		IncludeNavigation: true,
		IncludeTOC:        true,
		CustomCSS:         "",
		CustomJS:          "",
	}
}

// NewHTMLTemplateRenderer creates a new renderer with default configuration
func NewHTMLTemplateRenderer() *HTMLTemplateRenderer {
	return NewHTMLTemplateRendererWithConfig(DefaultHTMLTemplateConfig())
}

// NewHTMLTemplateRendererWithConfig creates a new renderer with custom configuration
func NewHTMLTemplateRendererWithConfig(config HTMLTemplateConfig) *HTMLTemplateRenderer {
	return &HTMLTemplateRenderer{
		config: config,
		pages:  make([]*models.Page, 0),
		stats: HTMLTemplateStats{
			StartTime: time.Now(),
		},
	}
}

// Config returns the current configuration
func (r *HTMLTemplateRenderer) Config() HTMLTemplateConfig {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.config
}

// Write processes a crawl result and adds successful pages to the collection
func (r *HTMLTemplateRenderer) Write(result *models.CrawlResult) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.stats.TotalPages++

	if result.Success && result.Page != nil {
		// Create a copy of the page to avoid data races
		page := &models.Page{
			URL:         result.Page.URL,
			Title:       result.Page.Title,
			Content:     result.Page.Content,
			CleanedText: result.Page.CleanedText,
			Markdown:    result.Page.Markdown,
			Links:       result.Page.Links,
			Images:      result.Page.Images,
			Metadata:    result.Page.Metadata,
			CrawledAt:   result.Page.CrawledAt,
			ProcessedAt: result.Page.ProcessedAt,
		}

		r.pages = append(r.pages, page)
		r.stats.SuccessfulPages++
	} else {
		r.stats.FailedPages++
	}

	return nil
}

// Pages returns a copy of the processed pages
func (r *HTMLTemplateRenderer) Pages() []*models.Page {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	// Return copy to prevent external modification
	pages := make([]*models.Page, len(r.pages))
	copy(pages, r.pages)
	return pages
}

// getSortedPagesUnlocked returns pages sorted by URL (helper method without mutex)
func (r *HTMLTemplateRenderer) getSortedPagesUnlocked() []*models.Page {
	pages := make([]*models.Page, len(r.pages))
	copy(pages, r.pages)

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].URL.String() < pages[j].URL.String()
	})

	return pages
}

// GenerateNavigation creates hierarchical navigation from pages
func (r *HTMLTemplateRenderer) GenerateNavigation() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.generateNavigationUnlocked()
}

// generateNavigationUnlocked generates navigation without mutex (helper method)
func (r *HTMLTemplateRenderer) generateNavigationUnlocked() string {
	if len(r.pages) == 0 {
		return ""
	}

	// Build navigation tree from URL hierarchy
	root := &NavigationNode{
		Title:    "Root",
		Children: make([]*NavigationNode, 0),
		Level:    0,
	}

	// Sort pages by URL for consistent navigation
	sortedPages := r.getSortedPagesUnlocked()

	// Build navigation tree
	for _, page := range sortedPages {
		r.addPageToNavigation(root, page)
	}

	// Generate HTML navigation
	var nav strings.Builder
	nav.WriteString("<nav class=\"navigation\">\n")
	nav.WriteString("  <h2>Navigation</h2>\n")
	nav.WriteString("  <ul class=\"nav-list\">\n")

	for _, child := range root.Children {
		r.renderNavigationNode(&nav, child, 2)
	}

	nav.WriteString("  </ul>\n")
	nav.WriteString("</nav>\n")

	return nav.String()
}

// addPageToNavigation adds a page to the navigation tree
func (r *HTMLTemplateRenderer) addPageToNavigation(root *NavigationNode, page *models.Page) {
	pathParts := strings.Split(strings.Trim(page.URL.Path, "/"), "/")
	if pathParts[0] == "" {
		pathParts = []string{"home"}
	}

	current := root
	currentPath := ""

	// Navigate/create path in navigation tree
	for i, part := range pathParts {
		if currentPath != "" {
			currentPath += "/"
		}
		currentPath += part

		// Find or create navigation node for this part
		var found *NavigationNode
		for _, child := range current.Children {
			if strings.EqualFold(child.Title, part) {
				found = child
				break
			}
		}

		if found == nil {
			// Create new navigation node
			title := part
			if i == len(pathParts)-1 && page.Title != "" {
				// Use page title for leaf nodes
				title = page.Title
			}

			found = &NavigationNode{
				Title:    title,
				URL:      page.URL.String(),
				Children: make([]*NavigationNode, 0),
				Level:    current.Level + 1,
			}
			current.Children = append(current.Children, found)
		}

		current = found
	}
}

// renderNavigationNode renders a single navigation node and its children
func (r *HTMLTemplateRenderer) renderNavigationNode(nav *strings.Builder, node *NavigationNode, indent int) {
	indentStr := strings.Repeat("  ", indent)

	_, _ = fmt.Fprintf(nav, "%s<li class=\"nav-item level-%d\">\n", indentStr, node.Level)

	if node.URL != "" && len(node.Children) == 0 {
		// Leaf node - create link
		anchor := r.createAnchor(node.Title)
		_, _ = fmt.Fprintf(nav, "%s  <a href=\"#%s\" class=\"nav-link\">%s</a>\n",
			indentStr, anchor, template.HTMLEscapeString(node.Title))
	} else {
		// Parent node - just title
		_, _ = fmt.Fprintf(nav, "%s  <span class=\"nav-title\">%s</span>\n",
			indentStr, template.HTMLEscapeString(node.Title))
	}

	if len(node.Children) > 0 {
		_, _ = fmt.Fprintf(nav, "%s  <ul class=\"nav-sublist\">\n", indentStr)
		for _, child := range node.Children {
			r.renderNavigationNode(nav, child, indent+2)
		}
		_, _ = fmt.Fprintf(nav, "%s  </ul>\n", indentStr)
	}

	_, _ = fmt.Fprintf(nav, "%s</li>\n", indentStr)
}

// createAnchor creates URL-safe anchor from text
func (r *HTMLTemplateRenderer) createAnchor(text string) string {
	// Convert to lowercase and replace spaces/special chars with hyphens
	anchor := strings.ToLower(text)
	re := regexp.MustCompile(`[^a-z0-9]+`)
	anchor = re.ReplaceAllString(anchor, "-")
	anchor = strings.Trim(anchor, "-")
	return anchor
}

// GenerateHTML creates the complete HTML document
func (r *HTMLTemplateRenderer) GenerateHTML() string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var html strings.Builder

	// HTML document structure
	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html lang=\"en\">\n")
	html.WriteString("<head>\n")
	html.WriteString("  <meta charset=\"UTF-8\">\n")
	html.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	html.WriteString(fmt.Sprintf("  <title>%s</title>\n", template.HTMLEscapeString(r.config.Title)))

	// Default CSS
	html.WriteString("  <style>\n")
	html.WriteString(r.generateDefaultCSS())

	// Custom CSS
	if r.config.CustomCSS != "" {
		html.WriteString("    /* Custom CSS */\n")
		html.WriteString("    " + r.config.CustomCSS + "\n")
	}

	html.WriteString("  </style>\n")
	html.WriteString("</head>\n")
	html.WriteString("<body>\n")

	// Header
	html.WriteString("  <header class=\"site-header\">\n")
	html.WriteString(fmt.Sprintf("    <h1>%s</h1>\n", template.HTMLEscapeString(r.config.Title)))
	html.WriteString("  </header>\n")

	// Main content container
	html.WriteString("  <div class=\"container\">\n")

	// Navigation
	if r.config.IncludeNavigation {
		html.WriteString("    <aside class=\"sidebar\">\n")
		html.WriteString("      " + r.generateNavigationUnlocked() + "\n")
		html.WriteString("    </aside>\n")
	}

	// Main content
	html.WriteString("    <main class=\"content\">\n")

	// Generate content for each page
	sortedPages := r.getSortedPagesUnlocked()
	for _, page := range sortedPages {
		html.WriteString(r.renderPageHTML(page))
	}

	html.WriteString("    </main>\n")
	html.WriteString("  </div>\n")

	// Footer
	html.WriteString("  <footer class=\"site-footer\">\n")
	html.WriteString(fmt.Sprintf("    <p>Generated on %s | %d pages processed</p>\n",
		time.Now().Format("2006-01-02 15:04:05"), r.stats.TotalPages))
	html.WriteString("  </footer>\n")

	// Custom JavaScript
	if r.config.CustomJS != "" {
		html.WriteString("  <script>\n")
		html.WriteString("    " + r.config.CustomJS + "\n")
		html.WriteString("  </script>\n")
	}

	html.WriteString("</body>\n")
	html.WriteString("</html>\n")

	return html.String()
}

// renderPageHTML renders a single page as HTML
func (r *HTMLTemplateRenderer) renderPageHTML(page *models.Page) string {
	var pageHTML strings.Builder

	anchor := r.createAnchor(page.Title)

	pageHTML.WriteString(fmt.Sprintf("      <section id=\"%s\" class=\"page-section\">\n", anchor))
	pageHTML.WriteString(fmt.Sprintf("        <h2>%s</h2>\n", template.HTMLEscapeString(page.Title)))
	pageHTML.WriteString(fmt.Sprintf("        <div class=\"page-url\"><small>URL: <a href=\"%s\">%s</a></small></div>\n",
		page.URL.String(), template.HTMLEscapeString(page.URL.String())))

	// Render content (assuming markdown or HTML)
	content := page.Content
	if page.Markdown != "" {
		content = page.Markdown
	}

	pageHTML.WriteString("        <div class=\"page-content\">\n")
	pageHTML.WriteString("          <pre>" + template.HTMLEscapeString(content) + "</pre>\n")
	pageHTML.WriteString("        </div>\n")
	pageHTML.WriteString("      </section>\n")

	return pageHTML.String()
}

// generateDefaultCSS creates default styling
func (r *HTMLTemplateRenderer) generateDefaultCSS() string {
	return `
    body {
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
      line-height: 1.6;
      color: #333;
      margin: 0;
      padding: 0;
      background-color: #f5f5f5;
    }
    .site-header {
      background: #2c3e50;
      color: white;
      padding: 1rem 2rem;
      text-align: center;
    }
    .container {
      display: flex;
      max-width: 1200px;
      margin: 0 auto;
      min-height: calc(100vh - 140px);
    }
    .sidebar {
      width: 250px;
      background: white;
      padding: 1rem;
      box-shadow: 2px 0 5px rgba(0,0,0,0.1);
    }
    .navigation h2 {
      margin-top: 0;
      color: #2c3e50;
      border-bottom: 2px solid #3498db;
      padding-bottom: 0.5rem;
    }
    .nav-list, .nav-sublist {
      list-style: none;
      padding-left: 0;
    }
    .nav-sublist {
      padding-left: 1rem;
    }
    .nav-item {
      margin: 0.25rem 0;
    }
    .nav-link {
      color: #3498db;
      text-decoration: none;
      padding: 0.25rem 0;
      display: block;
      border-radius: 3px;
    }
    .nav-link:hover {
      background-color: #ecf0f1;
      padding-left: 0.5rem;
    }
    .content {
      flex: 1;
      padding: 1rem;
      background: white;
      margin-left: 1rem;
    }
    .page-section {
      margin-bottom: 2rem;
      border-bottom: 1px solid #ecf0f1;
      padding-bottom: 1rem;
    }
    .page-url {
      margin-bottom: 1rem;
      color: #666;
    }
    .page-content {
      background: #f8f9fa;
      border: 1px solid #e9ecef;
      border-radius: 4px;
      padding: 1rem;
    }
    .page-content pre {
      margin: 0;
      white-space: pre-wrap;
      word-wrap: break-word;
    }
    .site-footer {
      background: #2c3e50;
      color: white;
      text-align: center;
      padding: 1rem;
      margin-top: auto;
    }
`
}

// Stats returns current rendering statistics
func (r *HTMLTemplateRenderer) Stats() HTMLTemplateStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	return r.stats
}

// SetOutputPath updates the output file path
func (r *HTMLTemplateRenderer) SetOutputPath(path string) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	r.config.OutputPath = path
}

// Flush writes the HTML content to the configured output file
func (r *HTMLTemplateRenderer) Flush() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	startTime := time.Now()

	// Generate complete HTML
	html := r.generateHTMLUnlocked()

	// Create output directory if needed
	outputDir := filepath.Dir(r.config.OutputPath)
	if outputDir != "." {
		if err := os.MkdirAll(outputDir, 0755); err != nil {
			return fmt.Errorf("failed to create output directory %q: %w", outputDir, err)
		}
	}

	// Write HTML to file
	if err := os.WriteFile(r.config.OutputPath, []byte(html), 0644); err != nil {
		return fmt.Errorf("failed to write HTML file %q: %w", r.config.OutputPath, err)
	}

	r.stats.ProcessingTime = time.Since(startTime)
	return nil
}

// generateHTMLUnlocked generates HTML without acquiring mutex (helper method)
func (r *HTMLTemplateRenderer) generateHTMLUnlocked() string {
	var html strings.Builder

	// HTML document structure
	html.WriteString("<!DOCTYPE html>\n")
	html.WriteString("<html lang=\"en\">\n")
	html.WriteString("<head>\n")
	html.WriteString("  <meta charset=\"UTF-8\">\n")
	html.WriteString("  <meta name=\"viewport\" content=\"width=device-width, initial-scale=1.0\">\n")
	html.WriteString(fmt.Sprintf("  <title>%s</title>\n", template.HTMLEscapeString(r.config.Title)))

	// Default CSS
	html.WriteString("  <style>\n")
	html.WriteString(r.generateDefaultCSS())

	// Custom CSS
	if r.config.CustomCSS != "" {
		html.WriteString("    /* Custom CSS */\n")
		html.WriteString("    " + r.config.CustomCSS + "\n")
	}

	html.WriteString("  </style>\n")
	html.WriteString("</head>\n")
	html.WriteString("<body>\n")

	// Header
	html.WriteString("  <header class=\"site-header\">\n")
	html.WriteString(fmt.Sprintf("    <h1>%s</h1>\n", template.HTMLEscapeString(r.config.Title)))
	html.WriteString("  </header>\n")

	// Main content container
	html.WriteString("  <div class=\"container\">\n")

	// Navigation
	if r.config.IncludeNavigation {
		html.WriteString("    <aside class=\"sidebar\">\n")
		html.WriteString("      " + r.generateNavigationUnlocked() + "\n")
		html.WriteString("    </aside>\n")
	}

	// Main content
	html.WriteString("    <main class=\"content\">\n")

	// Generate content for each page
	sortedPages := r.getSortedPagesUnlocked()
	for _, page := range sortedPages {
		html.WriteString(r.renderPageHTML(page))
	}

	html.WriteString("    </main>\n")
	html.WriteString("  </div>\n")

	// Footer
	html.WriteString("  <footer class=\"site-footer\">\n")
	html.WriteString(fmt.Sprintf("    <p>Generated on %s | %d pages processed</p>\n",
		time.Now().Format("2006-01-02 15:04:05"), r.stats.TotalPages))
	html.WriteString("  </footer>\n")

	// Custom JavaScript
	if r.config.CustomJS != "" {
		html.WriteString("  <script>\n")
		html.WriteString("    " + r.config.CustomJS + "\n")
		html.WriteString("  </script>\n")
	}

	html.WriteString("</body>\n")
	html.WriteString("</html>\n")

	return html.String()
}

// Close cleans up resources (implements OutputSink interface)
func (r *HTMLTemplateRenderer) Close() error {
	// No resources to clean up for HTML renderer
	return nil
}

// Name returns the sink identifier (implements OutputSink interface)
func (r *HTMLTemplateRenderer) Name() string {
	return "html-template-renderer"
}

var _ output.OutputSink = (*HTMLTemplateRenderer)(nil) // Compile-time interface check
