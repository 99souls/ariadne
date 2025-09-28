package crawler

import (
	"context"
	"net/url"
	"time"
)

// FetchResult represents the result of a fetch operation.
// Experimental: Field set may shrink (Metadata likely to narrow) prior to v1.0.
type FetchResult struct {
	URL      *url.URL
	Content  []byte
	Headers  map[string]string
	Status   int
	Links    []*url.URL
	Metadata map[string]interface{}
}

// (Removed Wave 3) Deprecated alias FetchedPage eliminated – use FetchResult directly.

// FetchPolicy defines configuration for fetch behavior.
// Experimental: Several fields (RespectRobots, MaxDepth) may move to higher-level policy.
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

// FetcherStats provides metrics about fetch operations.
// Experimental: Metric set may change; prefer aggregated engine snapshots when available.
type FetcherStats struct {
	RequestsCompleted int64
	RequestsFailed    int64
	LinksDiscovered   int64
	BytesDownloaded   int64
	AverageLatency    time.Duration
}

// Fetcher abstracts the act of retrieving a page + discovering outbound links.
// Stable (planned): Interface surface intended to remain small; method names may settle by v1.0.
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
