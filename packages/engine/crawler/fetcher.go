package crawler

import (
	"context"
	"net/url"
	"time"
)

// FetchResult represents the result of a fetch operation
// Renamed from FetchedPage for consistency with test naming
type FetchResult struct {
	URL      *url.URL
	Content  []byte
	Headers  map[string]string
	Status   int
	Links    []*url.URL
	Metadata map[string]interface{}
}

// FetchedPage is deprecated, use FetchResult instead
// Keeping for backward compatibility
type FetchedPage = FetchResult

// FetchPolicy defines configuration for fetch behavior
type FetchPolicy struct {
	UserAgent       string
	RequestDelay    time.Duration
	Timeout         time.Duration
	MaxRetries      int
	RespectRobots   bool
	FollowRedirects bool
	AllowedDomains  []string
	MaxDepth        int
}

// FetcherStats provides metrics about fetch operations
type FetcherStats struct {
	RequestsCompleted int64
	RequestsFailed    int64
	LinksDiscovered   int64
	BytesDownloaded   int64
	AverageLatency    time.Duration
}

// Fetcher abstracts the act of retrieving a page + discovering outbound links.
type Fetcher interface {
	// Fetch retrieves a single page from the given URL
	Fetch(ctx context.Context, rawURL string) (*FetchResult, error)

	// Discover extracts links from HTML content
	Discover(ctx context.Context, content []byte, baseURL *url.URL) ([]*url.URL, error)

	// Configure updates the fetcher's policy
	Configure(policy FetchPolicy) error

	// Stats returns current fetch statistics
	Stats() FetcherStats
}
