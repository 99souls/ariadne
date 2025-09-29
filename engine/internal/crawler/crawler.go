package crawler

import (
	"fmt"
	"log"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/99souls/ariadne/engine/models"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

type Crawler struct {
	config    *models.ScraperConfig
	collector *colly.Collector
	visited   sync.Map
	queue     chan string
	results   chan *models.CrawlResult
	stats     *models.CrawlStats
	mu        sync.RWMutex
	robots    *robotsCache
	stopping  bool
}

func New(config *models.ScraperConfig) *Crawler {
	if err := config.Validate(); err != nil {
		panic(fmt.Sprintf("invalid config: %v", err))
	}
	c := colly.NewCollector(colly.Debugger(&debug.LogDebugger{}))
	_ = c.Limit(&colly.LimitRule{DomainGlob: "*", Parallelism: 1, Delay: config.RequestDelay})
	c.SetRequestTimeout(config.Timeout)
	c.UserAgent = config.UserAgent
	crawler := &Crawler{config: config, collector: c, queue: make(chan string, 1000), results: make(chan *models.CrawlResult, 100), stats: &models.CrawlStats{StartTime: time.Now()}, robots: newRobotsCache()}
	crawler.setupCallbacks()
	return crawler
}

func (c *Crawler) setupCallbacks() {
	c.collector.OnRequest(func(r *colly.Request) {
		if !c.isAllowedURL(r.URL) {
			log.Printf("Blocked URL not in allowed domains: %s", r.URL.String())
			r.Abort()
			return
		}
		// Depth check (treat depth as number of non-empty path segments). Allow root depth=0.
		if c.config.MaxDepth > 0 && pathDepth(r.URL) > c.config.MaxDepth {
			log.Printf("Blocked by max depth (%d): %s", c.config.MaxDepth, r.URL.String())
			r.Abort()
			return
		}
		if !c.allowedByRobots(r.URL) {
			log.Printf("Blocked by robots.txt: %s", r.URL.String())
			r.Abort()
			return
		}
		log.Printf("Visiting: %s", r.URL.String())
	})
	c.collector.OnHTML("html", func(e *colly.HTMLElement) {
		page := c.extractPage(e)
		// Normalize the page URL before emitting so cosmetic query params (e.g. theme, utm_*)
		// do not cause separate logical pages in results. We intentionally only update the
		// emitted Page + CrawlResult URL; the underlying Colly request URL (with original
		// query) remains for fetch/accounting purposes.
		if page.URL != nil {
			if norm := c.normalizeURL(page.URL); norm != page.URL.String() {
				if u2, err := url.Parse(norm); err == nil {
					page.URL = u2
				}
			}
		}
		e.ForEach("a[href]", func(_ int, el *colly.HTMLElement) { c.processLink(el.Attr("href"), e.Request.URL) })
		// Also enqueue common asset references (img[src]) so tests can observe 404s.
		e.ForEach("img[src]", func(_ int, el *colly.HTMLElement) { c.processLink(el.Attr("src"), e.Request.URL) })
		// Stylesheets and scripts
		e.ForEach("link[href]", func(_ int, el *colly.HTMLElement) { c.processLink(el.Attr("href"), e.Request.URL) })
		e.ForEach("script[src]", func(_ int, el *colly.HTMLElement) { c.processLink(el.Attr("src"), e.Request.URL) })
		resultURL := ""
		if page.URL != nil {
			resultURL = page.URL.String()
		}
		result := &models.CrawlResult{URL: resultURL, Page: page, Stage: "crawl", Success: true}
		select {
		case c.results <- result:
		default:
			log.Printf("Results channel full, dropping result for %s", page.URL.String())
		}
	})
	c.collector.OnError(func(r *colly.Response, err error) {
		log.Printf("Error crawling %s: %v", r.Request.URL, err)
		stage := "crawl"
		ct := strings.ToLower(r.Headers.Get("Content-Type"))
		if strings.Contains(r.Request.URL.Path, "/static/") || (ct != "" && !strings.Contains(ct, "text/html")) {
			stage = "asset"
		}
		normURL := c.normalizeURL(r.Request.URL)
		result := &models.CrawlResult{URL: normURL, Error: models.NewCrawlError(normURL, stage, err), Stage: stage, Success: false, Retry: false, StatusCode: r.StatusCode}
		select {
		case c.results <- result:
		default:
			log.Printf("Results channel full, dropping error result")
		}
	})
	c.collector.OnResponse(func(r *colly.Response) {
		c.mu.Lock()
		c.stats.ProcessedPages++
		c.mu.Unlock()
		// For non-HTML (e.g., images) we emit a CrawlResult to allow tests to observe asset status codes (404, etc.).
		ct := strings.ToLower(r.Headers.Get("Content-Type"))
		if !strings.Contains(ct, "text/html") {
			normURL := c.normalizeURL(r.Request.URL)
			result := &models.CrawlResult{URL: normURL, Stage: "asset", Success: r.StatusCode < 400, StatusCode: r.StatusCode}
			if r.StatusCode >= 400 {
				result.Error = fmt.Errorf("asset status %d", r.StatusCode)
			}
			select {
			case c.results <- result:
			default:
			}
		}
	})
}

func (c *Crawler) extractPage(e *colly.HTMLElement) *models.Page {
	pageHTML, _ := e.DOM.Html()
	page := &models.Page{URL: e.Request.URL, Title: c.extractTitle(e), Content: pageHTML, CrawledAt: time.Now(), Links: make([]*url.URL, 0), Images: make([]string, 0)}
	page.Metadata = models.PageMeta{Description: e.ChildAttr("meta[name='description']", "content"), WordCount: len(strings.Fields(e.Text))}
	keywords := e.ChildAttr("meta[name='keywords']", "content")
	if keywords != "" {
		page.Metadata.Keywords = strings.Split(keywords, ",")
		for i, k := range page.Metadata.Keywords {
			page.Metadata.Keywords[i] = strings.TrimSpace(k)
		}
	}
	return page
}

func (c *Crawler) extractTitle(e *colly.HTMLElement) string {
	if title := e.ChildText("title"); title != "" {
		return strings.TrimSpace(title)
	}
	if h1 := e.ChildText("h1"); h1 != "" {
		return strings.TrimSpace(h1)
	}
	if ogTitle := e.ChildAttr("meta[property='og:title']", "content"); ogTitle != "" {
		return strings.TrimSpace(ogTitle)
	}
	return "Untitled"
}

func (c *Crawler) processLink(link string, base *url.URL) {
	linkURL, err := base.Parse(link)
	if err != nil {
		return
	}
	// If stopping, avoid enqueueing new work to prevent race with collector.Wait.
	c.mu.RLock()
	if c.stopping {
		c.mu.RUnlock()
		return
	}
	c.mu.RUnlock()
	// Early check: if path includes "/static/img/missing.png" ensure we always attempt fetch
	// (Even if previously visited we don't requeue; this just documents intent.)
	normalizedURL := c.normalizeURL(linkURL)
	if _, visited := c.visited.LoadOrStore(normalizedURL, true); visited {
		return
	}
	// Enforce domain, robots, and depth limits prior to queueing.
	if !c.isAllowedURL(linkURL) || !c.allowedByRobots(linkURL) {
		return
	}
	if c.config.MaxDepth > 0 && pathDepth(linkURL) > c.config.MaxDepth {
		return
	}
	select {
	case c.queue <- linkURL.String():
	default:
		log.Printf("Queue full, dropping URL: %s", linkURL.String())
	}
}

func (c *Crawler) normalizeURL(u *url.URL) string {
	normalized := *u
	normalized.Fragment = ""
	if normalized.RawQuery != "" {
		q := normalized.Query()
		// Drop cosmetic / non-content-affecting parameters.
		q.Del("theme")
		// If other cosmetic keys are added in future (e.g., utm_* tracking), strip them here.
		for key := range q {
			if strings.HasPrefix(key, "utm_") {
				q.Del(key)
			}
		}
		if len(q) == 0 {
			normalized.RawQuery = ""
		} else {
			normalized.RawQuery = q.Encode()
		}
	}
	return normalized.String()
}
func (c *Crawler) isAllowedURL(u *url.URL) bool {
	for _, domain := range c.config.AllowedDomains {
		if u.Host == domain || strings.HasSuffix(u.Host, "."+domain) {
			return true
		}
	}
	return false
}
func (c *Crawler) Start(startURL string) error {
	log.Printf("Starting crawl from: %s", startURL)
	c.queue <- startURL
	go c.processQueue()
	return nil
}
func (c *Crawler) processQueue() {
	for url := range c.queue {
		if c.shouldStop() {
			break
		}
		if err := c.collector.Visit(url); err != nil {
			log.Printf("Failed to visit %s: %v", url, err)
		}
	}
}
func (c *Crawler) shouldStop() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config.MaxPages > 0 && c.stats.ProcessedPages >= c.config.MaxPages
}
func (c *Crawler) Results() <-chan *models.CrawlResult { return c.results }
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
func (c *Crawler) Stop() {
	log.Println("Stopping crawler...")
	c.mu.Lock()
	c.stopping = true
	c.mu.Unlock()
	close(c.queue)
	c.collector.Wait()
	close(c.results)
	c.mu.Lock()
	c.stats.EndTime = time.Now()
	c.stats.Duration = c.stats.EndTime.Sub(c.stats.StartTime)
	c.mu.Unlock()
}

// pathDepth returns the number of non-empty path segments in the URL path.
// Example: "/labs/depth/depth2/depth3/leaf" => 5, "/" => 0
func pathDepth(u *url.URL) int {
	if u == nil {
		return 0
	}
	p := strings.Trim(u.Path, "/")
	if p == "" {
		return 0
	}
	return len(strings.Split(p, "/"))
}
