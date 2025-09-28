package crawler

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	"github.com/gocolly/colly/v2/debug"
)

// CollyFetcher implements the Fetcher interface using Colly
type CollyFetcher struct {
	collector *colly.Collector
	policy    FetchPolicy
	stats     fetcherStats
}

// fetcherStats holds atomic counters for thread-safe statistics
type fetcherStats struct {
	requestsCompleted int64
	requestsFailed    int64
	linksDiscovered   int64
	bytesDownloaded   int64
	totalLatency      int64 // in nanoseconds
}

// NewCollyFetcher creates a new Colly-based fetcher with the given policy
func NewCollyFetcher(policy FetchPolicy) (*CollyFetcher, error) {
	if err := validateFetchPolicy(policy); err != nil {
		return nil, fmt.Errorf("invalid fetch policy: %w", err)
	}

	c := colly.NewCollector(
		colly.Debugger(&debug.LogDebugger{}),
	)

	// Apply policy settings
	if policy.Timeout > 0 {
		c.SetRequestTimeout(policy.Timeout)
	}

	if policy.UserAgent != "" {
		c.UserAgent = policy.UserAgent
	}

	// Set up rate limiting
	if err := c.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       policy.RequestDelay,
	}); err != nil {
		return nil, fmt.Errorf("failed to set rate limit: %w", err)
	}

	fetcher := &CollyFetcher{
		collector: c,
		policy:    policy,
	}

	// Set up callbacks for statistics
	fetcher.setupCallbacks()

	return fetcher, nil
}

// validateFetchPolicy validates the fetch policy configuration
func validateFetchPolicy(policy FetchPolicy) error {
	if policy.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive, got %v", policy.Timeout)
	}

	if policy.MaxRetries < 0 {
		return fmt.Errorf("max retries must be non-negative, got %d", policy.MaxRetries)
	}

	if policy.RequestDelay < 0 {
		return fmt.Errorf("request delay must be non-negative, got %v", policy.RequestDelay)
	}

	return nil
}

// setupCallbacks configures Colly callbacks for statistics tracking
func (f *CollyFetcher) setupCallbacks() {
	f.collector.OnRequest(func(r *colly.Request) {
		// Track request start time for latency calculation
		r.Ctx.Put("start_time", time.Now())
	})

	f.collector.OnResponse(func(r *colly.Response) {
		atomic.AddInt64(&f.stats.requestsCompleted, 1)
		atomic.AddInt64(&f.stats.bytesDownloaded, int64(len(r.Body)))

		// Calculate and add latency
		if startTime, ok := r.Ctx.GetAny("start_time").(time.Time); ok {
			latency := time.Since(startTime)
			atomic.AddInt64(&f.stats.totalLatency, int64(latency))
		}
	})

	f.collector.OnError(func(r *colly.Response, err error) {
		atomic.AddInt64(&f.stats.requestsFailed, 1)
	})
}

// Fetch retrieves a single page from the given URL
func (f *CollyFetcher) Fetch(ctx context.Context, rawURL string) (*FetchResult, error) {
	// Parse and validate URL
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL %q: %w", rawURL, err)
	}

	// Check if URL is allowed
	if !f.isAllowedURL(u) {
		return nil, fmt.Errorf("URL not in allowed domains: %s", u.String())
	}

	result := &FetchResult{
		URL:      u,
		Headers:  make(map[string]string),
		Metadata: make(map[string]interface{}),
	}

	// Set up one-time callback for this request
	f.collector.OnHTML("html", func(e *colly.HTMLElement) {
		result.Content = e.Response.Body
		result.Status = e.Response.StatusCode

		// Copy headers
		if e.Response.Headers != nil {
			for key, values := range *e.Response.Headers {
				if len(values) > 0 {
					result.Headers[key] = values[0]
				}
			}
		}

		// Extract basic metadata
		if title := e.ChildText("title"); title != "" {
			result.Metadata["title"] = title
		}

		// Extract meta description
		e.ForEach("meta[name='description']", func(_ int, meta *colly.HTMLElement) {
			if desc := meta.Attr("content"); desc != "" {
				result.Metadata["description"] = desc
			}
		})

		// Discover links
		links, err := f.Discover(ctx, result.Content, u)
		if err == nil {
			result.Links = links
		}
	})

	// Perform the request
	err = f.collector.Visit(rawURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch %q: %w", rawURL, err)
	}

	return result, nil
}

// Discover extracts links from HTML content
func (f *CollyFetcher) Discover(ctx context.Context, content []byte, baseURL *url.URL) ([]*url.URL, error) {
	if len(content) == 0 {
		return []*url.URL{}, nil
	}

	// Parse HTML content
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(content)))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	var links []*url.URL

	// Extract all href attributes
	doc.Find("a[href]").Each(func(i int, s *goquery.Selection) {
		href, exists := s.Attr("href")
		if !exists || href == "" {
			return
		}

		// Skip non-HTTP links
		if strings.HasPrefix(href, "mailto:") ||
			strings.HasPrefix(href, "javascript:") ||
			strings.HasPrefix(href, "tel:") {
			return
		}

		// Parse and resolve relative URLs
		linkURL, err := url.Parse(href)
		if err != nil {
			return // Skip invalid URLs
		}

		// Resolve relative URLs
		if !linkURL.IsAbs() {
			linkURL = baseURL.ResolveReference(linkURL)
		}

		// Check if the resolved URL is allowed
		if f.isAllowedURL(linkURL) {
			links = append(links, linkURL)
			atomic.AddInt64(&f.stats.linksDiscovered, 1)
		}
	})

	return links, nil
}

// Configure updates the fetcher's policy
func (f *CollyFetcher) Configure(policy FetchPolicy) error {
	if err := validateFetchPolicy(policy); err != nil {
		return fmt.Errorf("invalid policy: %w", err)
	}

	f.policy = policy

	// Update collector settings
	if policy.Timeout > 0 {
		f.collector.SetRequestTimeout(policy.Timeout)
	}

	if policy.UserAgent != "" {
		f.collector.UserAgent = policy.UserAgent
	}

	// Update rate limiting
	if err := f.collector.Limit(&colly.LimitRule{
		DomainGlob:  "*",
		Parallelism: 1,
		Delay:       policy.RequestDelay,
	}); err != nil {
		return fmt.Errorf("failed to update rate limit: %w", err)
	}

	return nil
}

// Stats returns current fetch statistics
func (f *CollyFetcher) Stats() FetcherStats {
	completed := atomic.LoadInt64(&f.stats.requestsCompleted)
	failed := atomic.LoadInt64(&f.stats.requestsFailed)
	links := atomic.LoadInt64(&f.stats.linksDiscovered)
	bytes := atomic.LoadInt64(&f.stats.bytesDownloaded)
	totalLatency := atomic.LoadInt64(&f.stats.totalLatency)

	var avgLatency time.Duration
	if completed > 0 {
		avgLatency = time.Duration(totalLatency / completed)
	}

	return FetcherStats{
		RequestsCompleted: completed,
		RequestsFailed:    failed,
		LinksDiscovered:   links,
		BytesDownloaded:   bytes,
		AverageLatency:    avgLatency,
	}
}

// isAllowedURL checks if a URL is allowed based on the policy
func (f *CollyFetcher) isAllowedURL(u *url.URL) bool {
	if len(f.policy.AllowedDomains) == 0 {
		return true // No restrictions
	}

	hostname := u.Hostname()
	for _, allowed := range f.policy.AllowedDomains {
		if hostname == allowed || strings.HasSuffix(hostname, "."+allowed) {
			return true
		}
	}

	return false
}
