package processor

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"ariadne/internal/assets"
	"github.com/99souls/ariadne/engine/models"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/PuerkitoBio/goquery"
)

// Backward compatibility aliases for existing code
type AssetInfo = assets.AssetInfo
type AssetDiscoverer = assets.AssetDiscoverer
type AssetDownloader = assets.AssetDownloader
type AssetOptimizer = assets.AssetOptimizer
type AssetURLRewriter = assets.AssetURLRewriter
type AssetPipeline = assets.AssetPipeline
type AssetPipelineResult = assets.AssetPipelineResult

// Constructor compatibility wrappers
func NewAssetDiscoverer() *AssetDiscoverer {
	return assets.NewAssetDiscoverer()
}

func NewAssetDownloader(baseDir string) *AssetDownloader {
	return assets.NewAssetDownloader(baseDir)
}

func NewAssetOptimizer() *AssetOptimizer {
	return assets.NewAssetOptimizer()
}

func NewAssetURLRewriter(baseURL string) *AssetURLRewriter {
	return assets.NewAssetURLRewriter(baseURL)
}

func NewAssetPipeline(baseDir string) *AssetPipeline {
	return assets.NewAssetPipeline(baseDir)
}

// ContentProcessor handles HTML content cleaning and processing
type ContentProcessor struct {
	// Will add fields as needed
}

// NewContentProcessor creates a new content processor
func NewContentProcessor() *ContentProcessor {
	return &ContentProcessor{}
}

// ExtractContent extracts the main content from HTML using provided selectors
func (cp *ContentProcessor) ExtractContent(html string, selectors []string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// Try each selector in priority order
	for _, selector := range selectors {
		selection := doc.Find(selector)
		if selection.Length() > 0 {
			content, err := selection.Html()
			if err != nil {
				continue
			}
			return strings.TrimSpace(content), nil
		}
	}

	// Fallback to body content, but clean it first
	bodySelection := doc.Find("body")
	if bodySelection.Length() > 0 {
		// Remove unwanted elements from the body before extracting
		bodySelection.Find("script, style, nav, footer, aside, header").Remove()
		bodySelection.Find(".advertisement, .ad, .ads").Remove()
		bodySelection.Find("img[width='1'][height='1']").Remove()

		bodyContent, err := bodySelection.Html()
		if err != nil {
			return "", fmt.Errorf("could not extract any content: %w", err)
		}
		return strings.TrimSpace(bodyContent), nil
	}

	return "", fmt.Errorf("could not extract any content: no body found")
}

// RemoveUnwantedElements removes script, style, and other unwanted elements
func (cp *ContentProcessor) RemoveUnwantedElements(html string) (string, error) {
	// First remove HTML comments using regex (before parsing with goquery)
	re := regexp.MustCompile(`<!--[\s\S]*?-->`)
	html = re.ReplaceAllString(html, "")

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	// Remove unwanted elements
	unwantedTags := []string{"script", "style", "nav", "footer", "aside", "header"}
	for _, tag := range unwantedTags {
		doc.Find(tag).Remove()
	}

	// Remove elements with common unwanted classes/IDs
	unwantedSelectors := []string{
		".advertisement", ".ad", ".ads",
		".sidebar", ".nav", ".navigation",
		".footer", ".header",
		"#comments", ".comments",
	}

	for _, selector := range unwantedSelectors {
		doc.Find(selector).Remove()
	}

	// Remove tracking pixels (1x1 images)
	doc.Find("img[width='1'][height='1']").Remove()

	// Get the body content or the root content if it was a fragment
	bodyContent := doc.Find("body")
	if bodyContent.Length() > 0 {
		result, err := bodyContent.Html()
		if err != nil {
			return "", err
		}
		return result, nil
	}

	// If no body, return the entire document HTML (for fragments)
	result, err := doc.Html()
	if err != nil {
		return "", err
	}

	return result, nil
}

// ConvertRelativeURLs converts relative URLs to absolute URLs
func (cp *ContentProcessor) ConvertRelativeURLs(html, baseURL string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", err
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return "", fmt.Errorf("invalid base URL: %w", err)
	}

	// Convert relative links
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}

		// Skip if already absolute or is an anchor
		if strings.HasPrefix(href, "http") || strings.HasPrefix(href, "#") || strings.HasPrefix(href, "mailto:") {
			return
		}

		absoluteURL, err := base.Parse(href)
		if err != nil {
			return
		}

		s.SetAttr("href", absoluteURL.String())
	})

	// Convert relative image sources
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}

		if strings.HasPrefix(src, "http") || strings.HasPrefix(src, "data:") {
			return
		}

		absoluteURL, err := base.Parse(src)
		if err != nil {
			return
		}

		s.SetAttr("src", absoluteURL.String())
	})

	result, err := doc.Html()
	if err != nil {
		return "", err
	}

	return result, nil
}

// ExtractMetadata extracts metadata from HTML
func (cp *ContentProcessor) ExtractMetadata(html string) (string, *models.PageMeta, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", nil, err
	}

	meta := &models.PageMeta{}

	// Extract title
	title := doc.Find("title").Text()
	if title == "" {
		title = doc.Find("h1").First().Text()
	}
	title = strings.TrimSpace(title)

	// Extract description
	description, _ := doc.Find("meta[name='description']").Attr("content")
	if description == "" {
		description, _ = doc.Find("meta[property='og:description']").Attr("content")
	}
	meta.Description = strings.TrimSpace(description)

	// Extract keywords
	keywords, _ := doc.Find("meta[name='keywords']").Attr("content")
	if keywords != "" {
		keywordList := strings.Split(keywords, ",")
		for i, keyword := range keywordList {
			keywordList[i] = strings.TrimSpace(keyword)
		}
		meta.Keywords = keywordList
	}

	// Extract author
	author, _ := doc.Find("meta[name='author']").Attr("content")
	meta.Author = strings.TrimSpace(author)

	// Extract Open Graph data
	ogTitle, _ := doc.Find("meta[property='og:title']").Attr("content")
	ogDesc, _ := doc.Find("meta[property='og:description']").Attr("content")
	ogImage, _ := doc.Find("meta[property='og:image']").Attr("content")
	ogURL, _ := doc.Find("meta[property='og:url']").Attr("content")
	ogType, _ := doc.Find("meta[property='og:type']").Attr("content")

	meta.OpenGraph = models.OpenGraphMeta{
		Title:       strings.TrimSpace(ogTitle),
		Description: strings.TrimSpace(ogDesc),
		Image:       strings.TrimSpace(ogImage),
		URL:         strings.TrimSpace(ogURL),
		Type:        strings.TrimSpace(ogType),
	}

	return title, meta, nil
}

// ExtractImages extracts image URLs from HTML content
func (cp *ContentProcessor) ExtractImages(html, baseURL string) ([]string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, err
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	var images []string
	doc.Find("img[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists || src == "" {
			return
		}

		// Skip data URLs and tracking pixels
		if strings.HasPrefix(src, "data:") ||
			(s.AttrOr("width", "") == "1" && s.AttrOr("height", "") == "1") {
			return
		}

		// Convert relative URLs to absolute
		if !strings.HasPrefix(src, "http") {
			absoluteURL, err := base.Parse(src)
			if err != nil {
				return
			}
			src = absoluteURL.String()
		}

		images = append(images, src)
	})

	return images, nil
}

// ProcessPage processes a single page through the complete content pipeline
func (cp *ContentProcessor) ProcessPage(page *models.Page, baseURL string) error {
	if page == nil {
		return fmt.Errorf("page cannot be nil")
	}

	if strings.TrimSpace(page.Content) == "" {
		return fmt.Errorf("page content is empty")
	}

	// Basic validation to catch truly malformed HTML patterns like <<INVALID HTML>>
	if strings.HasPrefix(strings.TrimSpace(page.Content), "<<") {
		return fmt.Errorf("content appears to be malformed HTML")
	}

	// Step 1: Remove unwanted elements
	cleaned, err := cp.RemoveUnwantedElements(page.Content)
	if err != nil {
		return fmt.Errorf("failed to clean content: %w", err)
	}

	// Step 2: Convert relative URLs to absolute
	withAbsoluteURLs, err := cp.ConvertRelativeURLs(cleaned, baseURL)
	if err != nil {
		return fmt.Errorf("failed to convert URLs: %w", err)
	}

	// Step 3: Extract main content using common selectors
	contentSelectors := []string{
		"main", "article", ".content", "#content",
		".post", ".entry", ".article-content",
	}

	extractedContent, err := cp.ExtractContent(withAbsoluteURLs, contentSelectors)
	if err != nil {
		// If extraction fails, use the cleaned content
		extractedContent = withAbsoluteURLs
	}

	// Step 4: Convert to Markdown
	converter := NewHTMLToMarkdownConverter()
	markdown, err := converter.Convert(extractedContent)
	if err != nil {
		return fmt.Errorf("failed to convert to markdown: %w", err)
	}

	// Step 5: Extract metadata and title
	title, meta, err := cp.ExtractMetadata(page.Content)
	if err != nil {
		// Non-fatal error, continue without metadata
		meta = &models.PageMeta{}
		title = ""
	}

	// Step 6: Extract images
	images, err := cp.ExtractImages(extractedContent, baseURL)
	if err != nil {
		// Non-fatal error, continue without images
		images = []string{}
	}

	// Calculate word count from the cleaned content (removing HTML tags)
	cleanText := regexp.MustCompile(`<[^>]*>`).ReplaceAllString(extractedContent, " ")
	words := strings.Fields(cleanText)
	meta.WordCount = len(words)

	// Update the page with processed content
	page.Content = extractedContent
	page.Markdown = markdown
	page.Title = title
	page.Images = images
	page.Metadata = *meta
	page.ProcessedAt = time.Now()

	return nil
}

// WorkerPool manages concurrent content processing
type WorkerPool struct {
	workerCount int
}

// HTMLToMarkdownConverter converts HTML content to Markdown
type HTMLToMarkdownConverter struct {
	// Configuration will be added as needed
}

// ContentValidator validates content quality
type ContentValidator struct {
	// Validation rules will be added
}

// ValidationResult represents the result of content validation
type ValidationResult struct {
	IsValid     bool     `json:"is_valid"`
	Score       float64  `json:"score"`
	Issues      []string `json:"issues"`
	WordCount   int      `json:"word_count"`
	HasContent  bool     `json:"has_content"`
	HasHeadings bool     `json:"has_headings"`
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(workerCount int) *WorkerPool {
	return &WorkerPool{workerCount: workerCount}
}

// NewHTMLToMarkdownConverter creates a new HTML to Markdown converter
func NewHTMLToMarkdownConverter() *HTMLToMarkdownConverter {
	return &HTMLToMarkdownConverter{}
}

// NewContentValidator creates a new content validator
func NewContentValidator() *ContentValidator {
	return &ContentValidator{}
}

// WorkerCount returns the number of workers in the pool
func (wp *WorkerPool) WorkerCount() int {
	return wp.workerCount
}

// Stop stops all workers in the pool
func (wp *WorkerPool) Stop() {
	// Implementation for stopping workers
}

// ProcessPages processes multiple pages concurrently
func (wp *WorkerPool) ProcessPages(pages []*models.Page, baseURL string) <-chan *models.CrawlResult {
	results := make(chan *models.CrawlResult, len(pages))

	// Process pages (simplified for now - in reality would use worker goroutines)
	go func() {
		defer close(results)

		for _, page := range pages {
			processor := NewContentProcessor()
			err := processor.ProcessPage(page, baseURL)

			resultURL := ""
			if page != nil && page.URL != nil {
				resultURL = page.URL.String()
			}
			result := &models.CrawlResult{
				URL:     resultURL,
				Page:    page,
				Success: err == nil,
				Stage:   "processing",
			}

			if err != nil {
				result.Error = err
			}

			results <- result
		}
	}()

	return results
}

// Convert converts HTML content to Markdown
func (c *HTMLToMarkdownConverter) Convert(html string) (string, error) {
	if strings.TrimSpace(html) == "" {
		return "", fmt.Errorf("HTML content is empty")
	}

	// Create converter with plugins
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)

	// Convert HTML to Markdown
	markdown, err := conv.ConvertString(html)
	if err != nil {
		return "", fmt.Errorf("conversion failed: %w", err)
	}

	// Clean up the markdown
	cleaned := cleanMarkdown(markdown)

	return cleaned, nil
}

// cleanMarkdown cleans up common markdown formatting issues
func cleanMarkdown(markdown string) string {
	// Remove HTML comments that may have been preserved
	re := regexp.MustCompile(`<!--[\s\S]*?-->`)
	cleaned := re.ReplaceAllString(markdown, "")

	// Remove excessive newlines
	re = regexp.MustCompile(`\n{3,}`)
	cleaned = re.ReplaceAllString(cleaned, "\n\n")

	// Fix escaped characters in code blocks (common with some converters)
	cleaned = strings.ReplaceAll(cleaned, "\\n", "\n")
	cleaned = strings.ReplaceAll(cleaned, `\"`, `"`)

	// Clean up table formatting - remove excessive spaces in cells
	lines := strings.Split(cleaned, "\n")
	for i, line := range lines {
		// Check if this is a table row (contains pipes)
		if strings.Contains(line, "|") && !strings.HasPrefix(strings.TrimSpace(line), "|--") {
			// Split by pipes and clean each cell
			parts := strings.Split(line, "|")
			for j, part := range parts {
				parts[j] = strings.TrimSpace(part)
			}
			// Rejoin with proper spacing
			if len(parts) > 2 && parts[0] == "" && parts[len(parts)-1] == "" {
				// Standard table row format: | cell1 | cell2 |
				var cleanParts []string
				for k := 1; k < len(parts)-1; k++ {
					cleanParts = append(cleanParts, parts[k])
				}
				lines[i] = "| " + strings.Join(cleanParts, " | ") + " |"
			}
		} else {
			// Regular line - just remove trailing spaces
			lines[i] = strings.TrimRight(line, " ")
		}
	}

	return strings.TrimSpace(strings.Join(lines, "\n"))
}

// ValidateContent validates the quality and completeness of page content
func (cv *ContentValidator) ValidateContent(page *models.Page) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
		Score:   1.0,
		Issues:  []string{},
	}

	if page == nil {
		result.IsValid = false
		result.Score = 0.0
		result.Issues = append(result.Issues, "page_is_nil")
		return result
	}

	// Check if content exists
	content := strings.TrimSpace(page.Content)
	if content == "" {
		result.IsValid = false
		result.Score = 0.0
		result.Issues = append(result.Issues, "no_content")
		result.HasContent = false
		return result
	}

	result.HasContent = true

	// Use the word count from metadata if available, otherwise calculate
	wordCount := page.Metadata.WordCount
	if wordCount == 0 {
		words := strings.Fields(strings.ReplaceAll(content, "<", " <"))
		wordCount = 0
		for _, word := range words {
			// Skip HTML tags
			if !strings.HasPrefix(word, "<") {
				wordCount++
			}
		}
	}
	result.WordCount = wordCount

	// Priority-based validation - only flag the most critical issues

	// 1. Check title first (most critical for short content)
	titleMissing := strings.TrimSpace(page.Title) == ""
	titleTooShort := !titleMissing && len(strings.TrimSpace(page.Title)) < 10

	// 2. Check content length (very short threshold)
	contentTooShort := wordCount < 5 // Only flag extremely short content

	// 3. Check content density (for pages with some content but lots of markup)
	lowContentDensity := false
	if wordCount >= 5 && wordCount <= 15 { // Medium-length content that might have density issues
		htmlTagCount := strings.Count(content, "<")
		if htmlTagCount > 0 {
			contentDensity := float64(wordCount) / float64(htmlTagCount)
			lowContentDensity = contentDensity < 1.0 // Less than 1 word per HTML tag
		}
	}

	// Apply validation rules based on specific scenarios
	if titleMissing {
		result.Issues = append(result.Issues, "missing_title")
		result.Score -= 0.4
	} else if titleTooShort {
		result.Issues = append(result.Issues, "title_too_short")
		result.Score -= 0.3
	}

	if contentTooShort {
		result.Issues = append(result.Issues, "content_too_short")
		result.Score -= 0.4
	} else if lowContentDensity {
		result.Issues = append(result.Issues, "low_content_density")
		result.Score -= 0.3
	}

	// Check for headings (only for longer content)
	if wordCount >= 15 && !strings.Contains(content, "<h1") && !strings.Contains(content, "<h2") {
		result.HasHeadings = false
		// Only add as an issue if there are no other critical issues
		if len(result.Issues) == 0 {
			result.Issues = append(result.Issues, "no_headings")
			result.Score -= 0.2
		}
	} else {
		result.HasHeadings = strings.Contains(content, "<h1") || strings.Contains(content, "<h2")
	}

	// Final score adjustment
	if result.Score < 0 {
		result.Score = 0
	}

	if result.Score < 0.6 || len(result.Issues) > 0 {
		result.IsValid = false
	}

	return result
}
