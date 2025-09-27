package assembly

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"ariadne/packages/engine/output"
	"ariadne/pkg/models"
)

// DocumentAssemblyConfig defines configuration for document assembly
type DocumentAssemblyConfig struct {
	EnableHierarchy       bool    `json:"enable_hierarchy"`
	EnableCrossReferences bool    `json:"enable_cross_references"`
	EnableDeduplication   bool    `json:"enable_deduplication"`
	IncludeMetadata       bool    `json:"include_metadata"`
	SimilarityThreshold   float64 `json:"similarity_threshold"`
}

// DocumentAssemblyStats tracks assembly statistics
type DocumentAssemblyStats struct {
	TotalPages      int           `json:"total_pages"`
	ProcessedPages  int           `json:"processed_pages"`
	CrossReferences int           `json:"cross_references"`
	DuplicateGroups int           `json:"duplicate_groups"`
	ProcessingTime  time.Duration `json:"processing_time"`
}

// HierarchyNode represents a node in the document hierarchy
type HierarchyNode struct {
	Title    string           `json:"title"`
	URL      string           `json:"url"`
	Page     *models.Page     `json:"page,omitempty"`
	Children []*HierarchyNode `json:"children"`
	Level    int              `json:"level"`
	Path     []string         `json:"path"`
}

// CrossReference represents a relationship between pages
type CrossReference struct {
	SourceURL        string  `json:"source_url"`
	TargetURL        string  `json:"target_url"`
	RelationshipType string  `json:"relationship_type"`
	Confidence       float64 `json:"confidence"`
	Context          string  `json:"context"`
}

// DuplicateGroup represents a group of pages with similar content
type DuplicateGroup struct {
	Pages           []string `json:"pages"`
	SimilarityScore float64  `json:"similarity_score"`
	CommonContent   string   `json:"common_content"`
}

// EnrichedPage represents a page with extracted metadata
type EnrichedPage struct {
	URL         string            `json:"url"`
	Title       string            `json:"title"`
	Category    string            `json:"category"`
	ContentType string            `json:"content_type"`
	Tags        []string          `json:"tags"`
	Relations   []string          `json:"relations"`
	Metadata    map[string]string `json:"metadata"`
}

// DocumentAssembler coordinates document assembly and output generation
type DocumentAssembler struct {
	config DocumentAssemblyConfig
	pages  []*models.Page
	sinks  []output.OutputSink
	stats  DocumentAssemblyStats
	mutex  sync.RWMutex
}

// DefaultDocumentAssemblyConfig returns default configuration
func DefaultDocumentAssemblyConfig() DocumentAssemblyConfig {
	return DocumentAssemblyConfig{
		EnableHierarchy:       true,
		EnableCrossReferences: true,
		EnableDeduplication:   true,
		IncludeMetadata:       true,
		SimilarityThreshold:   0.7,
	}
}

// NewDocumentAssembler creates a new assembler with default configuration
func NewDocumentAssembler() *DocumentAssembler {
	return NewDocumentAssemblerWithConfig(DefaultDocumentAssemblyConfig())
}

// NewDocumentAssemblerWithConfig creates a new assembler with custom configuration
func NewDocumentAssemblerWithConfig(config DocumentAssemblyConfig) *DocumentAssembler {
	return &DocumentAssembler{
		config: config,
		pages:  make([]*models.Page, 0),
		sinks:  make([]output.OutputSink, 0),
		stats: DocumentAssemblyStats{
			ProcessedPages: 0,
		},
	}
}

// Config returns the current configuration
func (a *DocumentAssembler) Config() DocumentAssemblyConfig {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.config
}

// Name returns the assembler identifier
func (a *DocumentAssembler) Name() string {
	return "document-assembler"
}

// RegisterSink adds an output sink to the assembler
func (a *DocumentAssembler) RegisterSink(sink output.OutputSink) {
	a.mutex.Lock()
	defer a.mutex.Unlock()
	a.sinks = append(a.sinks, sink)
}

// RegisteredSinks returns all registered output sinks
func (a *DocumentAssembler) RegisteredSinks() []output.OutputSink {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	sinks := make([]output.OutputSink, len(a.sinks))
	copy(sinks, a.sinks)
	return sinks
}

// Write processes a crawl result and coordinates with registered sinks
func (a *DocumentAssembler) Write(result *models.CrawlResult) error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.stats.TotalPages++

	if result.Success && result.Page != nil {
		// Store page for assembly operations
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

		a.pages = append(a.pages, page)
		a.stats.ProcessedPages++
	}

	// Forward to all registered sinks
	for _, sink := range a.sinks {
		if err := sink.Write(result); err != nil {
			return fmt.Errorf("sink %q failed to write: %w", sink.Name(), err)
		}
	}

	return nil
}

// GenerateHierarchy creates a hierarchical document structure
func (a *DocumentAssembler) GenerateHierarchy() *HierarchyNode {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.generateHierarchyUnlocked()
}

// generateHierarchyUnlocked generates hierarchy without mutex (helper method)
func (a *DocumentAssembler) generateHierarchyUnlocked() *HierarchyNode {
	if !a.config.EnableHierarchy || len(a.pages) == 0 {
		return &HierarchyNode{
			Title:    "Root",
			Children: make([]*HierarchyNode, 0),
			Level:    0,
		}
	}

	// Create root node
	root := &HierarchyNode{
		Title:    "Root",
		Children: make([]*HierarchyNode, 0),
		Level:    0,
		Path:     []string{},
	}

	// Sort pages by URL for consistent hierarchy
	sortedPages := make([]*models.Page, len(a.pages))
	copy(sortedPages, a.pages)

	sort.Slice(sortedPages, func(i, j int) bool {
		return sortedPages[i].URL.String() < sortedPages[j].URL.String()
	})

	// Build hierarchy from URL paths
	for _, page := range sortedPages {
		a.addPageToHierarchy(root, page)
	}

	return root
}

// addPageToHierarchy adds a page to the hierarchy tree
func (a *DocumentAssembler) addPageToHierarchy(root *HierarchyNode, page *models.Page) {
	pathParts := a.extractPathParts(page.URL.Path)

	current := root
	currentPath := make([]string, 0)

	// Navigate/create path in hierarchy
	for i, part := range pathParts {
		currentPath = append(currentPath, part)

		// Find or create hierarchy node for this part
		var found *HierarchyNode
		for _, child := range current.Children {
			if strings.EqualFold(child.Title, part) {
				found = child
				break
			}
		}

		if found == nil {
			// Determine title and URL for new node
			title := part
			url := ""
			var nodePage *models.Page

			// If this is the final segment, use the page title and full URL
			if i == len(pathParts)-1 {
				if page.Title != "" {
					title = page.Title
				}
				url = page.URL.String()
				nodePage = page
			} else if part == "" || part == "home" {
				title = "Home"
			}

			found = &HierarchyNode{
				Title:    title,
				URL:      url,
				Page:     nodePage,
				Children: make([]*HierarchyNode, 0),
				Level:    current.Level + 1,
				Path:     append([]string{}, currentPath...),
			}
			current.Children = append(current.Children, found)
		}

		current = found
	}
}

// extractPathParts extracts meaningful parts from a URL path
func (a *DocumentAssembler) extractPathParts(path string) []string {
	// Clean and split path
	path = strings.Trim(path, "/")

	if path == "" {
		return []string{"home"}
	}

	parts := strings.Split(path, "/")

	// Filter out empty parts and common file extensions
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			// Remove file extensions for cleaner hierarchy
			if strings.Contains(part, ".") {
				part = strings.TrimSuffix(part, ".html")
				part = strings.TrimSuffix(part, ".php")
				part = strings.TrimSuffix(part, ".aspx")
			}
			if part != "" {
				filtered = append(filtered, part)
			}
		}
	}

	return filtered
}

// GenerateCrossReferences identifies relationships between pages
func (a *DocumentAssembler) GenerateCrossReferences() map[string][]CrossReference {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.generateCrossReferencesUnlocked()
}

// generateCrossReferencesUnlocked generates cross-references without mutex
func (a *DocumentAssembler) generateCrossReferencesUnlocked() map[string][]CrossReference {
	crossRefs := make(map[string][]CrossReference)

	if !a.config.EnableCrossReferences || len(a.pages) < 2 {
		return crossRefs
	}

	// Build URL to page mapping
	urlToPage := make(map[string]*models.Page)
	for _, page := range a.pages {
		urlToPage[page.URL.String()] = page
	}

	// Analyze each page for references to other pages
	for _, sourcePage := range a.pages {
		sourceURL := sourcePage.URL.String()
		refs := make([]CrossReference, 0)

		// Look for references in content
		content := strings.ToLower(sourcePage.Content + " " + sourcePage.CleanedText)

		for _, targetPage := range a.pages {
			if sourcePage.URL.String() == targetPage.URL.String() {
				continue // Skip self-references
			}

			targetURL := targetPage.URL.String()

			// Check for explicit URL references
			if strings.Contains(content, strings.ToLower(targetURL)) {
				refs = append(refs, CrossReference{
					SourceURL:        sourceURL,
					TargetURL:        targetURL,
					RelationshipType: "explicit_link",
					Confidence:       1.0,
					Context:          "Direct URL reference",
				})
				continue
			}

			// Check for title references
			if targetPage.Title != "" && strings.Contains(content, strings.ToLower(targetPage.Title)) {
				refs = append(refs, CrossReference{
					SourceURL:        sourceURL,
					TargetURL:        targetURL,
					RelationshipType: "mentions",
					Confidence:       0.8,
					Context:          fmt.Sprintf("Mentions title: %q", targetPage.Title),
				})
				continue
			}

			// Check for keyword/topic relationships
			if a.hasTopicOverlap(sourcePage, targetPage) {
				refs = append(refs, CrossReference{
					SourceURL:        sourceURL,
					TargetURL:        targetURL,
					RelationshipType: "related_topic",
					Confidence:       0.6,
					Context:          "Shared keywords or topics",
				})
			}
		}

		if len(refs) > 0 {
			crossRefs[sourceURL] = refs
			a.stats.CrossReferences += len(refs)
		}
	}

	return crossRefs
}

// hasTopicOverlap checks if two pages share common topics or keywords
func (a *DocumentAssembler) hasTopicOverlap(page1, page2 *models.Page) bool {
	// Extract keywords from URLs
	keywords1 := a.extractKeywords(page1)
	keywords2 := a.extractKeywords(page2)

	// Check for overlap
	commonCount := 0
	for _, kw1 := range keywords1 {
		for _, kw2 := range keywords2 {
			if kw1 == kw2 {
				commonCount++
				break
			}
		}
	}

	// Require at least 2 common keywords for relationship
	return commonCount >= 2
}

// extractKeywords extracts meaningful keywords from a page
func (a *DocumentAssembler) extractKeywords(page *models.Page) []string {
	keywords := make([]string, 0)

	// Extract from URL path
	pathParts := a.extractPathParts(page.URL.Path)
	keywords = append(keywords, pathParts...)

	// Extract from metadata keywords
	keywords = append(keywords, page.Metadata.Keywords...)

	// Extract common technical terms from content
	content := strings.ToLower(page.Content)
	commonTerms := []string{
		"api", "guide", "setup", "install", "config", "auth", "security",
		"docs", "documentation", "reference", "tutorial", "example",
	}

	for _, term := range commonTerms {
		if strings.Contains(content, term) {
			keywords = append(keywords, term)
		}
	}

	return a.deduplicateStrings(keywords)
}

// DetectDuplicateContent identifies pages with similar content
func (a *DocumentAssembler) DetectDuplicateContent() []DuplicateGroup {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.detectDuplicateContentUnlocked()
}

// detectDuplicateContentUnlocked detects duplicates without mutex
func (a *DocumentAssembler) detectDuplicateContentUnlocked() []DuplicateGroup {
	if !a.config.EnableDeduplication || len(a.pages) < 2 {
		return []DuplicateGroup{}
	}

	duplicateGroups := make([]DuplicateGroup, 0)
	processed := make(map[string]bool)

	// Compare each page with every other page
	for i, page1 := range a.pages {
		url1 := page1.URL.String()
		if processed[url1] {
			continue
		}

		group := DuplicateGroup{
			Pages:           []string{url1},
			SimilarityScore: 0.0,
			CommonContent:   "",
		}

		// Find similar pages
		for j := i + 1; j < len(a.pages); j++ {
			page2 := a.pages[j]
			url2 := page2.URL.String()

			if processed[url2] {
				continue
			}

			similarity := a.calculateContentSimilarity(page1, page2)
			if similarity >= a.config.SimilarityThreshold {
				group.Pages = append(group.Pages, url2)
				if similarity > group.SimilarityScore {
					group.SimilarityScore = similarity
					group.CommonContent = a.extractCommonContent(page1, page2)
				}
				processed[url2] = true
			}
		}

		// Add group if it contains duplicates
		if len(group.Pages) > 1 {
			duplicateGroups = append(duplicateGroups, group)
			a.stats.DuplicateGroups++
		}

		processed[url1] = true
	}

	return duplicateGroups
}

// calculateContentSimilarity calculates similarity between two pages
func (a *DocumentAssembler) calculateContentSimilarity(page1, page2 *models.Page) float64 {
	// Simple similarity based on common words
	words1 := a.extractWords(page1.Content)
	words2 := a.extractWords(page2.Content)

	if len(words1) == 0 || len(words2) == 0 {
		return 0.0
	}

	// Count common words
	common := 0
	wordSet1 := make(map[string]bool)
	for _, word := range words1 {
		wordSet1[word] = true
	}

	for _, word := range words2 {
		if wordSet1[word] {
			common++
		}
	}

	// Calculate Jaccard similarity
	union := len(words1) + len(words2) - common
	if union == 0 {
		return 0.0
	}

	return float64(common) / float64(union)
}

// extractWords extracts meaningful words from content
func (a *DocumentAssembler) extractWords(content string) []string {
	// Remove markdown/HTML markup and normalize
	re := regexp.MustCompile(`[<>#*\-_\[\](){}]`)
	clean := re.ReplaceAllString(content, " ")

	// Split into words and filter
	words := strings.Fields(strings.ToLower(clean))
	filtered := make([]string, 0, len(words))

	for _, word := range words {
		// Skip short words and common words
		if len(word) > 3 && !a.isStopWord(word) {
			filtered = append(filtered, word)
		}
	}

	return filtered
}

// extractCommonContent finds common content between two pages
func (a *DocumentAssembler) extractCommonContent(page1, page2 *models.Page) string {
	// Simple approach: find longest common substring
	content1 := strings.TrimSpace(page1.Content)
	content2 := strings.TrimSpace(page2.Content)

	// Find common lines
	lines1 := strings.Split(content1, "\n")
	lines2 := strings.Split(content2, "\n")

	common := make([]string, 0)
	for _, line1 := range lines1 {
		line1 = strings.TrimSpace(line1)
		if len(line1) > 10 { // Skip very short lines
			for _, line2 := range lines2 {
				if strings.TrimSpace(line2) == line1 {
					common = append(common, line1)
					break
				}
			}
		}
	}

	return strings.Join(common, "\n")
}

// isStopWord checks if a word is a common stop word
func (a *DocumentAssembler) isStopWord(word string) bool {
	stopWords := map[string]bool{
		"the": true, "and": true, "for": true, "are": true, "but": true,
		"not": true, "you": true, "all": true, "can": true, "had": true,
		"her": true, "was": true, "one": true, "our": true, "out": true,
		"day": true, "get": true, "has": true, "him": true, "his": true,
		"how": true, "its": true, "new": true, "now": true, "old": true,
		"see": true, "two": true, "way": true, "who": true, "boy": true,
		"did": true, "use": true, "may": true, "say": true, "she": true,
		"oil": true, "sit": true, "set": true, "run": true, "eat": true,
	}
	return stopWords[word]
}

// ExtractMetadata enriches pages with extracted metadata
func (a *DocumentAssembler) ExtractMetadata() []EnrichedPage {
	a.mutex.RLock()
	defer a.mutex.RUnlock()

	return a.extractMetadataUnlocked()
}

// extractMetadataUnlocked extracts metadata without mutex
func (a *DocumentAssembler) extractMetadataUnlocked() []EnrichedPage {
	if !a.config.IncludeMetadata {
		return []EnrichedPage{}
	}

	enriched := make([]EnrichedPage, 0, len(a.pages))

	for _, page := range a.pages {
		enrichedPage := EnrichedPage{
			URL:       page.URL.String(),
			Title:     page.Title,
			Tags:      make([]string, 0),
			Relations: make([]string, 0),
			Metadata:  make(map[string]string),
		}

		// Classify by URL pattern
		enrichedPage.Category = a.classifyPageCategory(page)
		enrichedPage.ContentType = a.classifyContentType(page)

		// Extract tags from keywords and content
		enrichedPage.Tags = append(enrichedPage.Tags, page.Metadata.Keywords...)
		enrichedPage.Tags = append(enrichedPage.Tags, a.extractKeywords(page)...)
		enrichedPage.Tags = a.deduplicateStrings(enrichedPage.Tags)

		// Add metadata from page
		if page.Metadata.Author != "" {
			enrichedPage.Metadata["author"] = page.Metadata.Author
		}
		if page.Metadata.Description != "" {
			enrichedPage.Metadata["description"] = page.Metadata.Description
		}
		if page.Metadata.WordCount > 0 {
			enrichedPage.Metadata["word_count"] = fmt.Sprintf("%d", page.Metadata.WordCount)
		}

		enriched = append(enriched, enrichedPage)
	}

	return enriched
}

// classifyPageCategory determines the category of a page
func (a *DocumentAssembler) classifyPageCategory(page *models.Page) string {
	path := strings.ToLower(page.URL.Path)

	if strings.Contains(path, "blog") || strings.Contains(path, "post") {
		return "blog"
	}
	if strings.Contains(path, "doc") || strings.Contains(path, "guide") {
		return "documentation"
	}
	if strings.Contains(path, "api") {
		return "api"
	}
	if strings.Contains(path, "tutorial") || strings.Contains(path, "example") {
		return "tutorial"
	}
	if path == "/" || path == "" {
		return "home"
	}

	return "general"
}

// classifyContentType determines the content type of a page
func (a *DocumentAssembler) classifyContentType(page *models.Page) string {
	content := strings.ToLower(page.Content)
	title := strings.ToLower(page.Title)

	if strings.Contains(title, "reference") || strings.Contains(content, "reference") {
		return "reference"
	}
	if strings.Contains(title, "guide") || strings.Contains(content, "guide") {
		return "guide"
	}
	if strings.Contains(title, "tutorial") || strings.Contains(content, "tutorial") {
		return "tutorial"
	}
	if strings.Contains(title, "example") || strings.Contains(content, "example") {
		return "example"
	}
	if strings.Contains(content, "api") && strings.Contains(content, "method") {
		return "api"
	}

	return "article"
}

// deduplicateStrings removes duplicates from a string slice
func (a *DocumentAssembler) deduplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0, len(slice))

	for _, item := range slice {
		if !seen[item] && item != "" {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

// Stats returns current assembly statistics
func (a *DocumentAssembler) Stats() DocumentAssemblyStats {
	a.mutex.RLock()
	defer a.mutex.RUnlock()
	return a.stats
}

// Flush coordinates flushing of all registered sinks
func (a *DocumentAssembler) Flush() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	startTime := time.Now()

	// Flush all registered sinks
	for _, sink := range a.sinks {
		if err := sink.Flush(); err != nil {
			return fmt.Errorf("sink %q failed to flush: %w", sink.Name(), err)
		}
	}

	a.stats.ProcessingTime = time.Since(startTime)
	return nil
}

// Close cleans up all registered sinks
func (a *DocumentAssembler) Close() error {
	a.mutex.Lock()
	defer a.mutex.Unlock()

	// Close all registered sinks
	for _, sink := range a.sinks {
		if err := sink.Close(); err != nil {
			return fmt.Errorf("sink %q failed to close: %w", sink.Name(), err)
		}
	}

	return nil
}

// Implement OutputSink interface for coordination purposes
var _ output.OutputSink = (*DocumentAssembler)(nil)
