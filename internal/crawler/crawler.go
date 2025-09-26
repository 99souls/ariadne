package crawler

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"ariadne/pkg/models"

	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

// Crawler manages the web crawling process
type Crawler struct {
	config    *models.ScraperConfig
	collector *colly.Collector
	visited   sync.Map
	queue     chan string
	results   chan *models.CrawlResult
	stats     *models.CrawlStats
	mu        sync.RWMutex
}

// New creates a new crawler instance
func New(config *models.ScraperConfig) *Crawler {
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid config: %v", err))
	}

	c := colly.NewCollector(
		colly.Debugger(&debug.LogDebugger{}),
		// Don't set AllowedDomains here - we'll handle it in our custom logic
	)

	// Set up collector options
	_ = c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1, // Start with single worker for Phase 1
		Delay:       config.RequestDelay,
	})

	c.SetRequestTimeout(config.Timeout)
	c.UserAgent = config.UserAgent

	crawler := &Crawler{
		config:    config,
		collector: c,
		queue:     make(chan string, 1000),
		results:   make(chan *models.CrawlResult, 100),
		stats: &models.CrawlStats{
			StartTime: time.Now(),
		},
	}

	// Set up collector callbacks
	crawler.setupCallbacks()

	return crawler
}

// setupCallbacks configures the Colly collector callbacks
func (c *Crawler) setupCallbacks() {
	// Request callback - log every request and check domain
	c.collector.OnRequest(func(r *colly.Request) {
		if !c.isAllowedURL(r.URL) {
			log.Printf("Blocked URL not in allowed domains: %s", r.URL.String())
			r.Abort()
			return
		}
		log.Printf("Visiting: %s", r.URL.String())
	})

	// HTML callback - extract content and links
	c.collector.OnHTML("html", func(e *colly.HTMLElement) {
		page := c.extractPage(e)

		// Extract internal links
		e.ForEach("a[href]", func(_ int, el *colly.HTMLElement) {
			link := el.Attr("href")
			c.processLink(link, e.Request.URL)
		})

		resultURL := ""
		if page.URL != nil {
			resultURL = page.URL.String()
		}
		result := &models.CrawlResult{
			URL:     resultURL,
			Page:    page,
			Stage:   "crawl",
			Success: true,
		}

		select {
		case c.results <- result:
		default:
			log.Printf("Results channel full, dropping result for %s", page.URL.String())
		}
	})

	// Error callback
	c.collector.OnError(func(r *colly.Response, err error) {
		log.Printf("Error crawling %s: %v", r.Request.URL, err)

		result := &models.CrawlResult{
			URL:     r.Request.URL.String(),
			Error:   models.NewCrawlError(r.Request.URL.String(), "crawl", err),
			Stage:   "crawl",
			Success: false,
			Retry:   true,
		}

		select {
		case c.results <- result:
		default:
			log.Printf("Results channel full, dropping error result")
		}
	})

	// Response callback - update stats
	c.collector.OnResponse(func(r *colly.Response) {
		c.mu.Lock()
		c.stats.ProcessedPages++
		c.mu.Unlock()
	})
}

// extractPage creates a Page model from the HTML element
func (c *Crawler) extractPage(e *colly.HTMLElement) *models.Page {
	pageHTML, _ := e.DOM.Html()

	page := &models.Page{
		URL:       e.Request.URL,
		Title:     c.extractTitle(e),
		Content:   pageHTML,
		CrawledAt: time.Now(),
		Links:     make([]*url.URL, 0),
		Images:    make([]string, 0),
	}

	// Extract basic metadata
	page.Metadata = models.PageMeta{
		Description: e.ChildAttr("meta[name='description']", "content"),
		WordCount:   len(strings.Fields(e.Text)),
	}

	// Extract keywords
	keywords := e.ChildAttr("meta[name='keywords']", "content")
	if keywords != "" {
		page.Metadata.Keywords = strings.Split(keywords, ",")
		for i, k := range page.Metadata.Keywords {
			page.Metadata.Keywords[i] = strings.TrimSpace(k)
		}
	}

	return page
}

// extractTitle gets the page title from various sources
func (c *Crawler) extractTitle(e *colly.HTMLElement) string {
	// Try title tag first
	if title := e.ChildText("title"); title != "" {
		return strings.TrimSpace(title)
	}

	// Try h1 tag
	if h1 := e.ChildText("h1"); h1 != "" {
		return strings.TrimSpace(h1)
	}

	// Try og:title meta tag
	if ogTitle := e.ChildAttr("meta[property='og:title']", "content"); ogTitle != "" {
		return strings.TrimSpace(ogTitle)
	}

	return "Untitled"
}

// processLink processes a discovered link and adds it to the queue if valid
func (c *Crawler) processLink(link string, base *url.URL) {
	// Resolve relative URLs
	linkURL, err := base.Parse(link)
	if err != nil {
		return
	}

	// Normalize URL (remove fragment, query params for deduplication)
	normalizedURL := c.normalizeURL(linkURL)

	// Check if already visited
	if _, visited := c.visited.LoadOrStore(normalizedURL, true); visited {
		return
	}

	// Check if URL is allowed
	if !c.isAllowedURL(linkURL) {
		return
	}

	// Add to queue for processing
	select {
	case c.queue <- linkURL.String():
	default:
		log.Printf("Queue full, dropping URL: %s", linkURL.String())
	}
}

// normalizeURL normalizes a URL for deduplication
func (c *Crawler) normalizeURL(u *url.URL) string {
	// Create a copy
	normalized := *u
	// Remove fragment and query for deduplication
	normalized.Fragment = ""
	normalized.RawQuery = ""
	return normalized.String()
}

// isAllowedURL checks if a URL is within allowed domains
func (c *Crawler) isAllowedURL(u *url.URL) bool {
	for _, domain := range c.config.AllowedDomains {
		// Exact match or subdomain match
		if u.Host == domain || strings.HasSuffix(u.Host, "."+domain) {
			return true
		}
	}
	return false
}

// Start begins the crawling process
func (c *Crawler) Start(startURL string) error {
	log.Printf("Starting crawl from: %s", startURL)

	// Add start URL to queue
	c.queue <- startURL

	// Start processing queue
	go c.processQueue()

	return nil
}

// processQueue processes URLs from the queue
func (c *Crawler) processQueue() {
	for url := range c.queue {
		if c.shouldStop() {
			break
		}

		err := c.collector.Visit(url)
		if err != nil {
			log.Printf("Failed to visit %s: %v", url, err)
		}
	}
}

// shouldStop checks if crawling should stop based on limits
func (c *Crawler) shouldStop() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.config.MaxPages > 0 && c.stats.ProcessedPages >= c.config.MaxPages {
		return true
	}

	return false
}

// Results returns the results channel
func (c *Crawler) Results() <-chan *models.CrawlResult {
	return c.results
}

// Stats returns current crawling statistics
func (c *Crawler) Stats() *models.CrawlStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := *c.stats
	stats.Duration = time.Since(c.stats.StartTime)
	if stats.Duration > 0 {
		stats.PagesPerSec = float64(stats.ProcessedPages) / stats.Duration.Seconds()
	}

	return &stats
}

// Stop gracefully stops the crawler
func (c *Crawler) Stop() {
	log.Println("Stopping crawler...")
	close(c.queue)
	c.collector.Wait()
	close(c.results)

	c.mu.Lock()
	c.stats.EndTime = time.Now()
	c.stats.Duration = c.stats.EndTime.Sub(c.stats.StartTime)
	c.mu.Unlock()
}
