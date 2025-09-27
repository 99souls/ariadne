package markdown

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"ariadne/engine/output"
	"ariadne/pkg/models"
)

// MarkdownCompilerConfig defines configuration for markdown compilation
type MarkdownCompilerConfig struct {
	OutputPath   string `json:"output_path"`
	IncludeTOC   bool   `json:"include_toc"`
	TOCTitle     string `json:"toc_title"`
	TOCMaxDepth  int    `json:"toc_max_depth"`
	IncludeIndex bool   `json:"include_index"`
	IncludeStats bool   `json:"include_stats"`
	SortByURL    bool   `json:"sort_by_url"`
	SortByTitle  bool   `json:"sort_by_title"`
}

// MarkdownCompilerStats tracks compilation statistics
type MarkdownCompilerStats struct {
	TotalResults    int `json:"total_results"`
	SuccessfulPages int `json:"successful_pages"`
	FailedPages     int `json:"failed_pages"`
	TotalWordCount  int `json:"total_word_count"`
}

// TOCEntry represents a table of contents entry
type TOCEntry struct {
	Title  string
	Level  int
	Anchor string
}

// MarkdownCompiler compiles multiple crawled pages into a single markdown document
type MarkdownCompiler struct {
	config *MarkdownCompilerConfig
	pages  []*models.CrawlResult
	stats  MarkdownCompilerStats
	mu     sync.RWMutex
	closed bool
}

// DefaultMarkdownCompilerConfig returns a configuration with sensible defaults
func DefaultMarkdownCompilerConfig() *MarkdownCompilerConfig {
	return &MarkdownCompilerConfig{
		OutputPath:   "compiled-wiki.md",
		IncludeTOC:   true,
		TOCTitle:     "Table of Contents",
		TOCMaxDepth:  3,
		IncludeIndex: false,
		IncludeStats: false,
		SortByURL:    true,
		SortByTitle:  false,
	}
}

// NewMarkdownCompiler creates a new markdown compiler with default configuration
func NewMarkdownCompiler() *MarkdownCompiler {
	return NewMarkdownCompilerWithConfig(DefaultMarkdownCompilerConfig())
}

// NewMarkdownCompilerWithConfig creates a new markdown compiler with the specified configuration
func NewMarkdownCompilerWithConfig(config *MarkdownCompilerConfig) *MarkdownCompiler {
	if config == nil {
		config = DefaultMarkdownCompilerConfig()
	}

	return &MarkdownCompiler{
		config: config,
		pages:  make([]*models.CrawlResult, 0),
		stats:  MarkdownCompilerStats{},
	}
}

// Config returns the compiler's configuration
func (mc *MarkdownCompiler) Config() *MarkdownCompilerConfig {
	return mc.config
}

// Write processes a single crawl result (implements OutputSink interface)
func (mc *MarkdownCompiler) Write(result *models.CrawlResult) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.closed {
		return fmt.Errorf("compiler is closed")
	}

	mc.stats.TotalResults++

	if result == nil {
		return nil
	}

	if !result.Success || result.Page == nil {
		mc.stats.FailedPages++
		return nil
	}

	// Only include successful results with valid pages
	mc.pages = append(mc.pages, result)
	mc.stats.SuccessfulPages++

	// Update word count
	if result.Page.Markdown != "" {
		wordCount := len(strings.Fields(result.Page.Markdown))
		mc.stats.TotalWordCount += wordCount
	}

	return nil
}

// Pages returns the accumulated pages, sorted according to configuration
func (mc *MarkdownCompiler) Pages() []*models.CrawlResult {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.getSortedPagesUnlocked()
}

// getSortedPagesUnlocked returns sorted pages without acquiring locks (internal helper)
func (mc *MarkdownCompiler) getSortedPagesUnlocked() []*models.CrawlResult {
	// Create a copy to avoid mutation
	pagesCopy := make([]*models.CrawlResult, len(mc.pages))
	copy(pagesCopy, mc.pages)

	// Sort based on configuration
	if mc.config.SortByTitle && !mc.config.SortByURL {
		sort.Slice(pagesCopy, func(i, j int) bool {
			titleA := ""
			titleB := ""
			if pagesCopy[i].Page != nil {
				titleA = pagesCopy[i].Page.Title
			}
			if pagesCopy[j].Page != nil {
				titleB = pagesCopy[j].Page.Title
			}
			return titleA < titleB
		})
	} else {
		// Default: sort by URL
		sort.Slice(pagesCopy, func(i, j int) bool {
			return pagesCopy[i].URL < pagesCopy[j].URL
		})
	}

	return pagesCopy
}

// GenerateTOC generates a table of contents from all pages
func (mc *MarkdownCompiler) GenerateTOC() string {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.generateTOCUnlocked()
}

// generateTOCUnlocked generates TOC without acquiring locks (internal helper)
func (mc *MarkdownCompiler) generateTOCUnlocked() string {
	var tocEntries []TOCEntry

	// Extract headers from all pages
	for _, result := range mc.getSortedPagesUnlocked() {
		if result.Page == nil || result.Page.Markdown == "" {
			continue
		}

		entries := mc.extractTOCEntries(result.Page.Markdown)
		tocEntries = append(tocEntries, entries...)
	}

	// Build TOC markdown
	var tocBuilder strings.Builder

	if mc.config.TOCTitle != "" {
		tocBuilder.WriteString(fmt.Sprintf("# %s\n\n", mc.config.TOCTitle))
	}

	for _, entry := range tocEntries {
		if entry.Level > mc.config.TOCMaxDepth {
			continue
		}

		// Create indentation for the level
		indent := strings.Repeat("  ", entry.Level-1)
		anchor := mc.createAnchor(entry.Title)

		tocBuilder.WriteString(fmt.Sprintf("%s- [%s](#%s)\n", indent, entry.Title, anchor))
	}

	return tocBuilder.String()
}

// extractTOCEntries extracts table of contents entries from markdown content
func (mc *MarkdownCompiler) extractTOCEntries(markdown string) []TOCEntry {
	var entries []TOCEntry

	// Regular expression to match markdown headers
	headerRegex := regexp.MustCompile(`^(#{1,6})\s+(.+)$`)

	lines := strings.Split(markdown, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if matches := headerRegex.FindStringSubmatch(line); matches != nil {
			level := len(matches[1])
			title := strings.TrimSpace(matches[2])

			entries = append(entries, TOCEntry{
				Title:  title,
				Level:  level,
				Anchor: mc.createAnchor(title),
			})
		}
	}

	return entries
}

// createAnchor creates a GitHub-style anchor from a title
func (mc *MarkdownCompiler) createAnchor(title string) string {
	// Convert to lowercase and replace spaces with hyphens
	anchor := strings.ToLower(title)
	anchor = regexp.MustCompile(`[^\w\s-]`).ReplaceAllString(anchor, "")
	anchor = regexp.MustCompile(`\s+`).ReplaceAllString(anchor, "-")
	anchor = strings.Trim(anchor, "-")
	return anchor
}

// Stats returns current compilation statistics
func (mc *MarkdownCompiler) Stats() MarkdownCompilerStats {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return mc.stats
}

// SetOutputPath updates the output path
func (mc *MarkdownCompiler) SetOutputPath(path string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.config.OutputPath = path
}

// Flush compiles and writes the final document (implements OutputSink interface)
func (mc *MarkdownCompiler) Flush() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if mc.closed {
		return fmt.Errorf("compiler is closed")
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(mc.config.OutputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	file, err := os.Create(mc.config.OutputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	// Generate document header
	if _, err := file.WriteString(mc.generateDocumentHeader()); err != nil {
		return fmt.Errorf("failed to write document header: %w", err)
	}

	// Generate table of contents
	if mc.config.IncludeTOC {
		toc := mc.generateTOCUnlocked()
		if toc != "" {
			if _, err := file.WriteString(toc + "\n---\n\n"); err != nil {
				return fmt.Errorf("failed to write TOC: %w", err)
			}
		}
	}

	// Write page content
	sortedPages := mc.getSortedPagesUnlocked()
	for i, result := range sortedPages {
		if result.Page == nil {
			continue
		}

		// Add page separator
		if i > 0 {
			if _, err := file.WriteString("\n\n---\n\n"); err != nil {
				return fmt.Errorf("failed to write page separator: %w", err)
			}
		}

		// Write page content
		if _, err := file.WriteString(result.Page.Markdown); err != nil {
			return fmt.Errorf("failed to write page content: %w", err)
		}
	}

	// Generate document footer
	if mc.config.IncludeStats {
		footer := mc.generateDocumentFooter()
		if footer != "" {
			if _, err := file.WriteString("\n\n---\n\n" + footer); err != nil {
				return fmt.Errorf("failed to write document footer: %w", err)
			}
		}
	}

	return nil
}

// generateDocumentHeader creates a header for the compiled document
func (mc *MarkdownCompiler) generateDocumentHeader() string {
	var header strings.Builder

	header.WriteString("# Compiled Wiki Documentation\n\n")
	header.WriteString(fmt.Sprintf("Generated on: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if mc.config.IncludeStats {
		stats := mc.stats
		header.WriteString("**Statistics:**\n")
		header.WriteString(fmt.Sprintf("- Total pages: %d\n", stats.SuccessfulPages))
		header.WriteString(fmt.Sprintf("- Failed pages: %d\n", stats.FailedPages))
		header.WriteString(fmt.Sprintf("- Total words: %d\n\n", stats.TotalWordCount))
	}

	return header.String()
}

// generateDocumentFooter creates a footer for the compiled document
func (mc *MarkdownCompiler) generateDocumentFooter() string {
	if !mc.config.IncludeStats {
		return ""
	}

	var footer strings.Builder
	footer.WriteString("## Document Statistics\n\n")
	footer.WriteString(fmt.Sprintf("- **Total Results**: %d\n", mc.stats.TotalResults))
	footer.WriteString(fmt.Sprintf("- **Successful Pages**: %d\n", mc.stats.SuccessfulPages))
	footer.WriteString(fmt.Sprintf("- **Failed Pages**: %d\n", mc.stats.FailedPages))
	footer.WriteString(fmt.Sprintf("- **Total Word Count**: %d\n", mc.stats.TotalWordCount))
	footer.WriteString(fmt.Sprintf("- **Compilation Time**: %s\n", time.Now().Format("2006-01-02 15:04:05")))

	return footer.String()
}

// Close closes the compiler (implements OutputSink interface)
func (mc *MarkdownCompiler) Close() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.closed = true
	return nil
}

// Name returns the name of this output sink (implements OutputSink interface)
func (mc *MarkdownCompiler) Name() string {
	return "markdown-compiler"
}

// Ensure MarkdownCompiler implements OutputSink interface
var _ output.OutputSink = (*MarkdownCompiler)(nil)
