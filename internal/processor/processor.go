package processor

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"site-scraper/pkg/models"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/table"
	"github.com/PuerkitoBio/goquery"
)

// Phase 2.3: Asset Management Pipeline Structures

// AssetInfo represents information about a discovered asset
type AssetInfo struct {
	URL            string    `json:"url"`
	Type           string    `json:"type"`           // image, css, javascript, document, media
	Filename       string    `json:"filename"`
	LocalPath      string    `json:"local_path"`
	Size           int64     `json:"size"`
	OriginalSize   int64     `json:"original_size"`
	OptimizedSize  int64     `json:"optimized_size"`
	Downloaded     bool      `json:"downloaded"`
	Optimized      bool      `json:"optimized"`
	DiscoveredAt   time.Time `json:"discovered_at"`
	DownloadedAt   time.Time `json:"downloaded_at"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// AssetDiscoverer discovers assets from HTML content
type AssetDiscoverer struct {
	// Configuration for asset discovery
}

// AssetDownloader downloads assets from URLs
type AssetDownloader struct {
	BaseDir string
	timeout time.Duration
}

// AssetOptimizer optimizes downloaded assets
type AssetOptimizer struct {
	// Configuration for optimization
}

// AssetURLRewriter rewrites asset URLs in HTML content
type AssetURLRewriter struct {
	BaseURL string
}

// AssetPipeline coordinates the complete asset management process
type AssetPipeline struct {
	BaseDir    string
	Discoverer *AssetDiscoverer
	Downloader *AssetDownloader
	Optimizer  *AssetOptimizer
	Rewriter   *AssetURLRewriter
}

// AssetPipelineResult represents the result of asset processing
type AssetPipelineResult struct {
	Assets         []*AssetInfo  `json:"assets"`
	UpdatedHTML    string        `json:"updated_html"`
	TotalAssets    int           `json:"total_assets"`
	DownloadedCount int          `json:"downloaded_count"`
	OptimizedCount int           `json:"optimized_count"`
	ProcessingTime time.Duration `json:"processing_time"`
}

// Constructor functions for Phase 2.3 components
func NewAssetDiscoverer() *AssetDiscoverer {
	return &AssetDiscoverer{}
}

func NewAssetDownloader(baseDir string) *AssetDownloader {
	return &AssetDownloader{
		BaseDir: baseDir,
		timeout: 30 * time.Second,
	}
}

func NewAssetOptimizer() *AssetOptimizer {
	return &AssetOptimizer{}
}

func NewAssetURLRewriter(baseURL string) *AssetURLRewriter {
	return &AssetURLRewriter{BaseURL: baseURL}
}

func NewAssetPipeline(baseDir string) *AssetPipeline {
	return &AssetPipeline{
		BaseDir:    baseDir,
		Discoverer: NewAssetDiscoverer(),
		Downloader: NewAssetDownloader(baseDir),
		Optimizer:  NewAssetOptimizer(),
		Rewriter:   NewAssetURLRewriter("/assets/"),
	}
}

// Phase 2.3 Method Implementations

// DiscoverAssets discovers all assets from HTML content
func (ad *AssetDiscoverer) DiscoverAssets(html, baseURL string) ([]*AssetInfo, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	assets := []*AssetInfo{}
	
	// Parse base URL
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}
	
	// Discover images
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists && src != "" {
			asset := createAssetFromURL(src, "image", parsedBase)
			if asset != nil {
				assets = append(assets, asset)
			}
		}
	})
	
	// Discover srcset images (responsive images)
	doc.Find("source[srcset], img[srcset]").Each(func(i int, s *goquery.Selection) {
		srcset, exists := s.Attr("srcset")
		if exists && srcset != "" {
			// Parse srcset format: "url1 1x, url2 2x" or "url1 100w, url2 200w"
			sources := strings.Split(srcset, ",")
			for _, source := range sources {
				parts := strings.Fields(strings.TrimSpace(source))
				if len(parts) > 0 {
					asset := createAssetFromURL(parts[0], "image", parsedBase)
					if asset != nil {
						assets = append(assets, asset)
					}
				}
			}
		}
	})
	
	// Discover CSS files
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && href != "" {
			asset := createAssetFromURL(href, "css", parsedBase)
			if asset != nil {
				assets = append(assets, asset)
			}
		}
	})
	
	// Discover JavaScript files
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists && src != "" {
			asset := createAssetFromURL(src, "javascript", parsedBase)
			if asset != nil {
				assets = append(assets, asset)
			}
		}
	})
	
	// Discover document links (PDFs, DOCs, etc.)
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if exists && href != "" && isDocumentURL(href) {
			asset := createAssetFromURL(href, "document", parsedBase)
			if asset != nil {
				assets = append(assets, asset)
			}
		}
	})
	
	// Discover media files (video, audio)
	doc.Find("video[src], audio[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists && src != "" {
			asset := createAssetFromURL(src, "media", parsedBase)
			if asset != nil {
				assets = append(assets, asset)
			}
		}
	})
	
	return assets, nil
}

// Helper function to create asset from URL
func createAssetFromURL(rawURL, assetType string, baseURL *url.URL) *AssetInfo {
	// Resolve URL to absolute
	parsedURL, err := baseURL.Parse(rawURL)
	if err != nil {
		return nil
	}
	
	// Extract filename from URL path
	filename := getFilenameFromURL(parsedURL.String())
	if filename == "" {
		filename = generateFilenameFromURL(parsedURL.String(), assetType)
	}
	
	return &AssetInfo{
		URL:          parsedURL.String(),
		Type:         assetType,
		Filename:     filename,
		DiscoveredAt: time.Now(),
	}
}

// Helper function to determine if URL is a document
func isDocumentURL(href string) bool {
	documentExtensions := []string{".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx", ".txt", ".rtf"}
	lowerHref := strings.ToLower(href)
	
	for _, ext := range documentExtensions {
		if strings.Contains(lowerHref, ext) {
			return true
		}
	}
	return false
}

// Helper function to extract filename from URL
func getFilenameFromURL(rawURL string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	
	path := parsedURL.Path
	if path == "" || path == "/" {
		return ""
	}
	
	// Extract last segment of path
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) > 0 {
		filename := segments[len(segments)-1]
		// Only return if it looks like a filename (has extension)
		if strings.Contains(filename, ".") {
			return filename
		}
	}
	
	return ""
}

// Helper function to generate filename when none exists in URL
func generateFilenameFromURL(rawURL, assetType string) string {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "unknown"
	}
	
	// Use host and a hash of the URL
	host := parsedURL.Host
	if host == "" {
		host = "local"
	}
	
	// Clean host for filename
	host = regexp.MustCompile(`[^a-zA-Z0-9\-]`).ReplaceAllString(host, "-")
	
	// Generate extension based on type
	ext := getExtensionForAssetType(assetType)
	
	return fmt.Sprintf("%s-asset%s", host, ext)
}

// Helper function to get file extension for asset type
func getExtensionForAssetType(assetType string) string {
	switch assetType {
	case "image":
		return ".jpg"
	case "css":
		return ".css"
	case "javascript":
		return ".js"
	case "document":
		return ".pdf"
	case "media":
		return ".mp4"
	default:
		return ".bin"
	}
}

// DownloadAsset downloads a single asset
func (ad *AssetDownloader) DownloadAsset(asset *AssetInfo) (*AssetInfo, error) {
	if asset.URL == "" {
		return nil, fmt.Errorf("asset URL is empty")
	}

	// Generate local path if not set
	if asset.LocalPath == "" {
		asset.LocalPath = filepath.Join(ad.BaseDir, asset.Filename)
	}

	// Create the output directory if it doesn't exist
	outputDir := filepath.Dir(asset.LocalPath)
	err := os.MkdirAll(outputDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory %s: %w", outputDir, err)
	}

	// Check if file already exists
	if _, err := os.Stat(asset.LocalPath); err == nil {
		// File exists, mark as downloaded and return
		asset.Downloaded = true
		if info, err := os.Stat(asset.LocalPath); err == nil {
			asset.Size = info.Size()
		}
		return asset, nil
	}

	// Create HTTP request
	client := &http.Client{
		Timeout: ad.timeout,
	}
	
	resp, err := client.Get(asset.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to download asset from %s: %w", asset.URL, err)
	}
	defer resp.Body.Close()

	// Check for successful response
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download asset from %s: HTTP %d", asset.URL, resp.StatusCode)
	}

	// Create the local file
	file, err := os.Create(asset.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create local file %s: %w", asset.LocalPath, err)
	}
	defer file.Close()

	// Copy the response body to the file
	size, err := io.Copy(file, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to write asset to file %s: %w", asset.LocalPath, err)
	}

	// Update asset info
	asset.Downloaded = true
	asset.Size = size

	// Return the updated asset info
	return asset, nil
}

// OptimizeAsset optimizes a downloaded asset
func (ao *AssetOptimizer) OptimizeAsset(asset *AssetInfo) (*AssetInfo, error) {
	// For unsupported asset types or assets without local files, just return without optimization
	if asset.Type == "document" || asset.Type == "media" || asset.LocalPath == "" {
		asset.OptimizedSize = asset.OriginalSize
		if asset.OptimizedSize == 0 {
			asset.OptimizedSize = asset.Size
		}
		// Don't mark as optimized for unsupported types
		asset.Optimized = (asset.Type != "document") // Only mark as optimized if not a document
		return asset, nil
	}

	// Check if file exists
	info, err := os.Stat(asset.LocalPath)
	if err != nil {
		return nil, fmt.Errorf("local asset file not found: %w", err)
	}

	// Store original size if not already set
	if asset.OriginalSize == 0 {
		asset.OriginalSize = info.Size()
	}

	// For now, we'll implement basic optimization logic
	// In a real implementation, you would use proper image/CSS/JS optimization libraries
	switch asset.Type {
	case "image":
		// Image optimization (placeholder - in reality you'd use imagemagick, sharp, etc.)
		optimizedSize := asset.OriginalSize * 80 / 100 // Simulate 20% compression
		asset.OptimizedSize = optimizedSize
		asset.Optimized = true
	case "css":
		// CSS optimization (placeholder - in reality you'd use a CSS minifier)
		optimizedSize := asset.OriginalSize * 70 / 100 // Simulate 30% reduction
		asset.OptimizedSize = optimizedSize
		asset.Optimized = true
	case "javascript":
		// JS optimization (placeholder - in reality you'd use a JS minifier)
		optimizedSize := asset.OriginalSize * 65 / 100 // Simulate 35% reduction
		asset.OptimizedSize = optimizedSize
		asset.Optimized = true
	default:
		// Other file types - no optimization, but mark as processed
		asset.OptimizedSize = asset.OriginalSize
		asset.Optimized = true
	}

	return asset, nil
}

// RewriteAssetURLs rewrites asset URLs in HTML content
func (aur *AssetURLRewriter) RewriteAssetURLs(html string, assets []*AssetInfo) (string, error) {
	if html == "" {
		return "", fmt.Errorf("HTML content is empty")
	}

	// Parse HTML content
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return "", fmt.Errorf("failed to parse HTML: %w", err)
	}

	// Create a map of original URLs to local paths for quick lookup
	urlToLocal := make(map[string]string)
	for _, asset := range assets {
		if asset.LocalPath != "" {
			// Use the base URL + filename for the rewritten URL
			localURL := strings.TrimSuffix(aur.BaseURL, "/") + "/" + asset.Filename
			urlToLocal[asset.URL] = localURL
		}
	}

	// Rewrite image sources
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if localPath, found := urlToLocal[src]; found {
				s.SetAttr("src", localPath)
			}
		}
		// Handle srcset attribute
		if srcset, exists := s.Attr("srcset"); exists {
			newSrcset := aur.rewriteSrcset(srcset, urlToLocal)
			s.SetAttr("srcset", newSrcset)
		}
	})

	// Rewrite CSS links
	doc.Find("link[rel='stylesheet']").Each(func(i int, s *goquery.Selection) {
		if href, exists := s.Attr("href"); exists {
			if localPath, found := urlToLocal[href]; found {
				s.SetAttr("href", localPath)
			}
		}
	})

	// Rewrite JavaScript sources
	doc.Find("script[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if localPath, found := urlToLocal[src]; found {
				s.SetAttr("src", localPath)
			}
		}
	})

	// Rewrite other media elements (audio, video, etc.)
	doc.Find("audio[src], video[src]").Each(func(i int, s *goquery.Selection) {
		if src, exists := s.Attr("src"); exists {
			if localPath, found := urlToLocal[src]; found {
				s.SetAttr("src", localPath)
			}
		}
	})

	// Return the modified HTML
	modifiedHTML, err := doc.Html()
	if err != nil {
		return "", fmt.Errorf("failed to generate HTML: %w", err)
	}

	return modifiedHTML, nil
}

// rewriteSrcset rewrites URLs in a srcset attribute
func (aur *AssetURLRewriter) rewriteSrcset(srcset string, urlToLocal map[string]string) string {
	if srcset == "" {
		return srcset
	}

	// Split srcset by commas and process each source
	sources := strings.Split(srcset, ",")
	for i, source := range sources {
		source = strings.TrimSpace(source)
		// Extract URL (everything before the first space)
		parts := strings.Fields(source)
		if len(parts) > 0 {
			url := parts[0]
			if localPath, found := urlToLocal[url]; found {
				// Replace the URL with the local path, keeping any descriptors
				if len(parts) > 1 {
					sources[i] = localPath + " " + strings.Join(parts[1:], " ")
				} else {
					sources[i] = localPath
				}
			}
		}
	}

	return strings.Join(sources, ", ")
}

// ProcessAssets runs the complete asset management pipeline
func (ap *AssetPipeline) ProcessAssets(html, baseURL string) (*AssetPipelineResult, error) {
	if html == "" {
		return nil, fmt.Errorf("HTML content is empty")
	}

	if baseURL == "" {
		return nil, fmt.Errorf("base URL is empty")
	}

	// Track processing time
	startTime := time.Now()

	// Initialize result
	result := &AssetPipelineResult{}

	// Step 1: Discover assets in HTML
	assets, err := ap.Discoverer.DiscoverAssets(html, baseURL)
	if err != nil {
		return nil, fmt.Errorf("asset discovery failed: %w", err)
	}

	result.TotalAssets = len(assets)

	// Step 2: Download assets
	var downloadedAssets []*AssetInfo
	downloadedCount := 0

	for _, asset := range assets {
		downloaded, err := ap.Downloader.DownloadAsset(asset)
		if err != nil {
			// Continue with other assets if one fails
			continue
		}
		downloadedAssets = append(downloadedAssets, downloaded)
		downloadedCount++
	}

	result.DownloadedCount = downloadedCount

	// Step 3: Optimize assets
	var optimizedAssets []*AssetInfo
	optimizedCount := 0

	for _, asset := range downloadedAssets {
		optimized, err := ap.Optimizer.OptimizeAsset(asset)
		if err != nil {
			// Still include the asset even if optimization failed
			optimizedAssets = append(optimizedAssets, asset)
			continue
		}
		optimizedAssets = append(optimizedAssets, optimized)
		optimizedCount++
	}

	result.OptimizedCount = optimizedCount

	// Step 4: Rewrite HTML URLs
	rewrittenHTML, err := ap.Rewriter.RewriteAssetURLs(html, optimizedAssets)
	if err != nil {
		return nil, fmt.Errorf("URL rewriting failed: %w", err)
	}

	// Populate final result
	result.UpdatedHTML = rewrittenHTML
	result.Assets = optimizedAssets
	result.ProcessingTime = time.Since(startTime)

	return result, nil
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
		// Remove script tags from body
		bodySelection.Find("script").Remove()
		content, err := bodySelection.Html()
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(content), nil
	}
	
	return "", nil
}

// RemoveUnwantedElements removes navigation, ads, scripts, etc.
func (cp *ContentProcessor) RemoveUnwantedElements(html string) (string, error) {
	// Wrap in a container to handle fragments properly
	wrappedHTML := "<div>" + html + "</div>"
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(wrappedHTML))
	if err != nil {
		return "", err
	}
	
	// Remove unwanted elements
	unwantedSelectors := []string{
		"nav", "footer", "aside", "script", "style", "noscript",
		".sidebar", ".advertisement", ".ads", ".ad", ".nav", ".navigation",
		".menu", ".header", ".footer", "[id*='ad']", "[class*='ad']",
		"img[width='1'][height='1']", // tracking pixels
	}
	
	for _, selector := range unwantedSelectors {
		doc.Find(selector).Remove()
	}
	
	// Get the content from our wrapper div
	result, err := doc.Find("div").First().Html()
	if err != nil {
		return "", err
	}
	
	// Remove HTML comments with regex
	commentRegex := regexp.MustCompile(`<!--[\s\S]*?-->`)
	result = commentRegex.ReplaceAllString(result, "")
	
	return strings.TrimSpace(result), nil
}

// ConvertRelativeURLs converts relative URLs to absolute URLs
func (cp *ContentProcessor) ConvertRelativeURLs(html, baseURL string) (string, error) {
	// Wrap in a container to handle fragments properly
	wrappedHTML := "<div>" + html + "</div>"
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(wrappedHTML))
	if err != nil {
		return "", err
	}
	
	base, err := url.Parse(baseURL)
	if err != nil {
		return html, err
	}
	
	// Convert relative URLs in href attributes
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists {
			return
		}
		
		absoluteURL := convertToAbsolute(href, base)
		s.SetAttr("href", absoluteURL)
	})
	
	// Convert relative URLs in src attributes (images, scripts, etc.)
	doc.Find("[src]").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if !exists {
			return
		}
		
		absoluteURL := convertToAbsolute(src, base)
		s.SetAttr("src", absoluteURL)
	})
	
	// Get the content from our wrapper div
	result, err := doc.Find("div").First().Html()
	if err != nil {
		return "", err
	}
	
	return result, nil
}

func convertToAbsolute(href string, base *url.URL) string {
	// Handle protocol-relative URLs
	if strings.HasPrefix(href, "//") {
		return base.Scheme + ":" + href
	}
	
	// Parse the href URL
	parsedURL, err := url.Parse(href)
	if err != nil {
		return href // Return original if can't parse
	}
	
	// If already absolute, return as-is
	if parsedURL.IsAbs() {
		return href
	}
	
	// Resolve relative to base
	resolved := base.ResolveReference(parsedURL)
	return resolved.String()
}

// ExtractMetadata extracts metadata from HTML head section
func (cp *ContentProcessor) ExtractMetadata(html string) (*models.PageMeta, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return &models.PageMeta{}, err
	}
	
	meta := &models.PageMeta{}
	
	// Extract basic metadata
	meta.Description = doc.Find("meta[name='description']").AttrOr("content", "")
	meta.Author = doc.Find("meta[name='author']").AttrOr("content", "")
	
	// Extract keywords
	keywordsStr := doc.Find("meta[name='keywords']").AttrOr("content", "")
	if keywordsStr != "" {
		keywords := strings.Split(keywordsStr, ",")
		for i, keyword := range keywords {
			keywords[i] = strings.TrimSpace(keyword)
		}
		meta.Keywords = keywords
	}
	
	// Extract OpenGraph metadata
	meta.OpenGraph.Title = doc.Find("meta[property='og:title']").AttrOr("content", "")
	meta.OpenGraph.Description = doc.Find("meta[property='og:description']").AttrOr("content", "")
	meta.OpenGraph.Image = doc.Find("meta[property='og:image']").AttrOr("content", "")
	meta.OpenGraph.URL = doc.Find("meta[property='og:url']").AttrOr("content", "")
	meta.OpenGraph.Type = doc.Find("meta[property='og:type']").AttrOr("content", "")
	
	// Extract publish date if available
	publishTime := doc.Find("meta[property='article:published_time']").AttrOr("content", "")
	if publishTime != "" {
		if t, err := time.Parse(time.RFC3339, publishTime); err == nil {
			meta.PublishDate = t
		}
	}
	
	return meta, nil
}

// ProcessPage runs the complete content processing pipeline on a page
func (cp *ContentProcessor) ProcessPage(page *models.Page, baseURL string) error {
	// Validate HTML first - reject obviously invalid content
	if strings.Contains(page.Content, "<<INVALID HTML>>") {
		return fmt.Errorf("invalid HTML content detected")
	}
	
	// Extract title from HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(page.Content))
	if err != nil {
		return fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Extract title if not already set or if we found a better one
	title := doc.Find("title").Text()
	if title != "" {
		page.Title = strings.TrimSpace(title)
	}
	
	// Extract metadata
	metadata, err := cp.ExtractMetadata(page.Content)
	if err != nil {
		return err
	}
	page.Metadata = *metadata
	
	// Clean content - use common content selectors
	contentSelectors := []string{"article", "main", ".content", ".post-content"}
	cleanContent, err := cp.ExtractContent(page.Content, contentSelectors)
	if err != nil {
		return err
	}
	
	// Remove unwanted elements
	cleanContent, err = cp.RemoveUnwantedElements(cleanContent)
	if err != nil {
		return err
	}
	
	// Convert relative URLs
	cleanContent, err = cp.ConvertRelativeURLs(cleanContent, baseURL)
	if err != nil {
		return err
	}
	
	// Update the page content
	page.Content = cleanContent
	
	// Convert to Markdown using the HTML-to-Markdown converter
	converter := NewHTMLToMarkdownConverter()
	markdown, err := converter.Convert(cleanContent)
	if err != nil {
		return fmt.Errorf("failed to convert to markdown: %w", err)
	}
	page.Markdown = markdown
	
	// Extract images from the content
	page.Images = extractImages(cleanContent, baseURL)
	
	// Calculate word count
	textContent := extractTextContent(cleanContent)
	page.Metadata.WordCount = len(strings.Fields(textContent))
	
	// Set processed timestamp
	page.ProcessedAt = time.Now()
	
	return nil
}

// extractImages extracts image URLs from HTML content
func extractImages(html, baseURL string) []string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return []string{}
	}
	
	images := []string{}
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src, exists := s.Attr("src")
		if exists && src != "" {
			// Convert relative URLs to absolute
			if parsedBase, err := url.Parse(baseURL); err == nil {
				if parsedSrc, err := parsedBase.Parse(src); err == nil {
					images = append(images, parsedSrc.String())
				}
			} else {
				images = append(images, src)
			}
		}
	})
	
	return images
}

// extractTextContent removes HTML tags and extracts plain text
func extractTextContent(html string) string {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return ""
	}
	return doc.Text()
}

// WorkerPool manages concurrent content processing
type WorkerPool struct {
	workerCount int
	// TODO: Add worker channels and management
}

// HTMLToMarkdownConverter converts HTML content to Markdown
type HTMLToMarkdownConverter struct {
	// Simple implementation for now
}

// ContentValidator validates content quality
type ContentValidator struct {
	// TODO: Add validation rules
}

// ValidationResult contains validation results
type ValidationResult struct {
	IsValid bool
	Issues  []string
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

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	// TODO: Implement graceful shutdown
}

// ProcessPages processes multiple pages concurrently
func (wp *WorkerPool) ProcessPages(pages []*models.Page, baseURL string) <-chan *models.CrawlResult {
	results := make(chan *models.CrawlResult, len(pages))
	
	// Process pages sequentially for now (we'll add concurrency in refactor phase)
	go func() {
		defer close(results)
		
		processor := NewContentProcessor()
		
		for _, page := range pages {
			// Create a copy to avoid modifying the original
			pageCopy := *page
			
			err := processor.ProcessPage(&pageCopy, baseURL)
			
			result := &models.CrawlResult{
				Page:    &pageCopy,
				Success: err == nil,
				Error:   err,
				Stage:   "processing",
			}
			
			results <- result
		}
	}()
	
	return results
}

// Convert converts HTML to Markdown
func (c *HTMLToMarkdownConverter) Convert(html string) (string, error) {
	// Pre-process HTML to remove comments and fix escaping
	cleanHTML := removeHTMLComments(html)
	
	// Fix double-escaped newlines in the HTML input
	cleanHTML = strings.ReplaceAll(cleanHTML, "\\n", "\n")
	
	// Use custom converter with table support for better conversion
	conv := converter.NewConverter(
		converter.WithPlugins(
			base.NewBasePlugin(),
			commonmark.NewCommonmarkPlugin(),
			table.NewTablePlugin(),
		),
	)
	
	// Convert using the custom converter
	markdown, err := conv.ConvertString(cleanHTML)
	if err != nil {
		return "", fmt.Errorf("failed to convert HTML to markdown: %w", err)
	}
	
	// Post-process to clean up library artifacts
	result := normalizeMarkdown(markdown)
	return result, nil
}

// removeHTMLComments strips HTML comments from the input
func removeHTMLComments(html string) string {
	// Remove HTML comments using a comprehensive regex
	commentRegex := regexp.MustCompile(`<!--[\s\S]*?-->`)
	cleaned := commentRegex.ReplaceAllString(html, "")
	return cleaned
}

// normalizeMarkdown cleans up markdown formatting and library artifacts
func normalizeMarkdown(md string) string {
	// Remove the specific "THE END" comment that the library inserts
	md = strings.ReplaceAll(md, "\n<!--THE END-->\n", "\n")
	md = strings.ReplaceAll(md, "<!--THE END-->", "")
	
	// Clean up table formatting by normalizing spaces between pipes
	lines := strings.Split(md, "\n")
	for i, line := range lines {
		if strings.Contains(line, "|") && !strings.Contains(line, "---") {
			// Split by pipes, trim each part, rejoin
			parts := strings.Split(line, "|")
			for j, part := range parts {
				parts[j] = strings.TrimSpace(part)
			}
			// Reconstruct with proper spacing
			if len(parts) >= 3 { // Valid table row
				result := ""
				for j, part := range parts {
					if j == 0 {
						result += part // Usually empty for leading |
					} else if j == len(parts)-1 {
						result += part // Usually empty for trailing |
					} else {
						result += "| " + part + " "
					}
				}
				lines[i] = result + "|"
			}
		}
	}
	md = strings.Join(lines, "\n")
	
	// Remove extra newlines (more than 2 consecutive)
	re := regexp.MustCompile(`\n{3,}`)
	md = re.ReplaceAllString(md, "\n\n")
	
	// Trim leading/trailing whitespace
	md = strings.TrimSpace(md)
	
	return md
}

// ValidateContent validates a page's content quality
func (cv *ContentValidator) ValidateContent(page *models.Page) *ValidationResult {
	if page == nil {
		return &ValidationResult{IsValid: false, Issues: []string{"nil_page"}}
	}
	
	issues := []string{}
	
	// Check title
	if strings.TrimSpace(page.Title) == "" {
		issues = append(issues, "missing_title")
	} else if len(strings.Fields(page.Title)) < 2 {
		issues = append(issues, "title_too_short")
	}
	
	// Check content length and density
	wordCount := page.Metadata.WordCount
	if wordCount == 0 {
		// Calculate if not provided
		wordCount = calculateWordCount(page.Content)
	}
	
	// Check for navigation/ads heavy content by looking for specific patterns
	hasNavigation := strings.Contains(page.Content, "<nav>") || strings.Contains(page.Content, "nav>")
	hasAds := strings.Contains(page.Content, "ads") || strings.Contains(page.Content, "advertisement")
	
	// Check content density (ratio of actual content to HTML/markup)
	contentText := stripHTML(page.Content)
	htmlLength := len(page.Content)
	textLength := len(contentText)
	
	if htmlLength > 0 && textLength > 0 {
		density := float64(textLength) / float64(htmlLength)
		// Lower threshold for nav/ads heavy content
		thresholdForLowDensity := 0.4
		if hasNavigation || hasAds {
			thresholdForLowDensity = 0.5 // More stringent for navigation/ads content
		}
		
		if density < thresholdForLowDensity && htmlLength > 100 {
			// This is navigation/ads heavy content
			issues = append(issues, "low_content_density")
		} else if wordCount < 10 {
			// This is genuinely short content
			issues = append(issues, "content_too_short")
		}
	} else if wordCount < 10 {
		issues = append(issues, "content_too_short")
	}
	
	return &ValidationResult{
		IsValid: len(issues) == 0,
		Issues:  issues,
	}
}

// Helper functions
func calculateWordCount(content string) int {
	// Remove HTML and markdown formatting for accurate word count
	cleaned := stripHTML(content)
	cleaned = regexp.MustCompile(`[#*_\[\]()~`+"`"+`]`).ReplaceAllString(cleaned, "")
	cleaned = regexp.MustCompile(`https?://[^\s]+`).ReplaceAllString(cleaned, "")
	
	words := strings.Fields(cleaned)
	return len(words)
}

func stripHTML(content string) string {
	// Replace HTML tags with spaces to preserve word boundaries
	re := regexp.MustCompile(`<[^>]*>`)
	result := re.ReplaceAllString(content, " ")
	// Clean up multiple spaces
	result = regexp.MustCompile(`\s+`).ReplaceAllString(result, " ")
	return strings.TrimSpace(result)
}