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
		if !c.allowedByRobots(r.URL) {
			log.Printf("Blocked by robots.txt: %s", r.URL.String())
			r.Abort()
			return
		}
		log.Printf("Visiting: %s", r.URL.String())
	})
	c.collector.OnHTML("html", func(e *colly.HTMLElement) {
		page := c.extractPage(e)
		e.ForEach("a[href]", func(_ int, el *colly.HTMLElement) { c.processLink(el.Attr("href"), e.Request.URL) })
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
		result := &models.CrawlResult{URL: r.Request.URL.String(), Error: models.NewCrawlError(r.Request.URL.String(), "crawl", err), Stage: "crawl", Success: false, Retry: true}
		select {
		case c.results <- result:
		default:
			log.Printf("Results channel full, dropping error result")
		}
	})
	c.collector.OnResponse(func(r *colly.Response) { c.mu.Lock(); c.stats.ProcessedPages++; c.mu.Unlock() })
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
	normalizedURL := c.normalizeURL(linkURL)
	if _, visited := c.visited.LoadOrStore(normalizedURL, true); visited {
		return
	}
	if !c.isAllowedURL(linkURL) || !c.allowedByRobots(linkURL) {
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
	normalized.RawQuery = ""
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
	close(c.queue)
	c.collector.Wait()
	close(c.results)
	c.mu.Lock()
	c.stats.EndTime = time.Now()
	c.stats.Duration = c.stats.EndTime.Sub(c.stats.StartTime)
	c.mu.Unlock()
}
