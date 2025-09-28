package models

// NOTE: These types were migrated from pkg/models to consolidate public embedding
// under packages/engine. Original pkg/models now provides type aliases with
// deprecation comments to avoid breaking existing imports for a deprecation window.

import (
	"errors"
	"net/url"
	"time"
)

// Page represents a single scraped web page with its content and metadata.
// Stable: Field additions are allowed; existing field semantics will be preserved after baseline.
type Page struct {
	URL         *url.URL   `json:"url"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	CleanedText string     `json:"cleaned_text"`
	Markdown    string     `json:"markdown"`
	Links       []*url.URL `json:"links"`
	Images      []string   `json:"images"`
	Metadata    PageMeta   `json:"metadata"`
	CrawledAt   time.Time  `json:"crawled_at"`
	ProcessedAt time.Time  `json:"processed_at"`
}

// PageMeta contains structured metadata extracted from the page.
// Experimental: Field set may expand (front‑matter, canonical URL) prior to v1.0.
type PageMeta struct {
	Author      string            `json:"author,omitempty"`
	Description string            `json:"description,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	PublishDate time.Time         `json:"publish_date,omitempty"`
	WordCount   int               `json:"word_count"`
	Headers     map[string]string `json:"headers,omitempty"`
	OpenGraph   OpenGraphMeta     `json:"open_graph,omitempty"`
}

// OpenGraphMeta captures a subset of Open Graph tags.
// Experimental: May merge with future SocialMeta struct.
type OpenGraphMeta struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
}

// CrawlResult represents the result of processing a single URL through the pipeline.
// Experimental: Error and Stage fields may change shape (structured error) before v1.0.
type CrawlResult struct {
	URL     string `json:"url"`
	Page    *Page  `json:"page"`
	Error   error  `json:"error,omitempty"`
	Stage   string `json:"stage"`
	Success bool   `json:"success"`
	Retry   bool   `json:"retry"`
}

// CrawlStats aggregates crawl progress metrics.
// Experimental: Field set & naming may change; prefer Engine Snapshot for stable telemetry.
type CrawlStats struct {
	TotalPages     int           `json:"total_pages"`
	ProcessedPages int           `json:"processed_pages"`
	FailedPages    int           `json:"failed_pages"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time,omitempty"`
	Duration       time.Duration `json:"duration,omitempty"`
	PagesPerSec    float64       `json:"pages_per_sec,omitempty"`
}

// RateLimitConfig defines adaptive per-domain rate limiting behavior.
// Experimental: Some tuning knobs may be collapsed into policy groups.
type RateLimitConfig struct {
	Enabled             bool    `json:"enabled"`
	InitialRPS          float64 `json:"initial_rps"`
	MinRPS              float64 `json:"min_rps"`
	MaxRPS              float64 `json:"max_rps"`
	TokenBucketCapacity float64 `json:"token_bucket_capacity"`

	AIMDIncrease         float64       `json:"aimd_increase"`
	AIMDDecrease         float64       `json:"aimd_decrease"`
	LatencyTarget        time.Duration `json:"latency_target"`
	LatencyDegradeFactor float64       `json:"latency_degrade_factor"`

	ErrorRateThreshold       float64       `json:"error_rate_threshold"`
	MinSamplesToTrip         int           `json:"min_samples_to_trip"`
	ConsecutiveFailThreshold int           `json:"consecutive_fail_threshold"`
	OpenStateDuration        time.Duration `json:"open_state_duration"`
	HalfOpenProbes           int           `json:"half_open_probes"`

	RetryBaseDelay   time.Duration `json:"retry_base_delay"`
	RetryMaxDelay    time.Duration `json:"retry_max_delay"`
	RetryMaxAttempts int           `json:"retry_max_attempts"`

	StatsWindow    time.Duration `json:"stats_window"`
	StatsBucket    time.Duration `json:"stats_bucket"`
	DomainStateTTL time.Duration `json:"domain_state_ttl"`
	Shards         int           `json:"shards"`
}

// ScraperConfig holds crawler configuration formerly defined in legacy pkg/models.
// It is now the authoritative definition used by internal components. Some fields
// (like worker counts) will be reconsidered during API pruning; retained verbatim
// here to unblock root purge.
// ScraperConfig holds crawler configuration for legacy simple crawler mode.
// Experimental: This config may be reduced or replaced by Engine Config + high level policy.
type ScraperConfig struct {
	StartURL       string   `json:"start_url"`
	AllowedDomains []string `json:"allowed_domains"`
	MaxDepth       int      `json:"max_depth"`
	MaxPages       int      `json:"max_pages"`

	CrawlWorkers   int `json:"crawl_workers"`
	ExtractWorkers int `json:"extract_workers"`
	ProcessWorkers int `json:"process_workers"`

	RequestDelay time.Duration `json:"request_delay"`
	Timeout      time.Duration `json:"timeout"`

	ContentSelectors []string `json:"content_selectors"`
	RemoveSelectors  []string `json:"remove_selectors"`

	OutputDir     string   `json:"output_dir"`
	OutputFormats []string `json:"output_formats"`

	UserAgent         string `json:"user_agent"`
	IncludeImages     bool   `json:"include_images"`
	RespectRobots     bool   `json:"respect_robots"`
	EnableCheckpoints bool   `json:"enable_checkpoints"`

	RateLimit RateLimitConfig `json:"rate_limit"`
}

// DefaultConfig returns a baseline ScraperConfig.
// DefaultConfig returns a baseline ScraperConfig.
// Experimental: Defaults may tune between minor versions pre-v1.0.
func DefaultConfig() *ScraperConfig {
	return &ScraperConfig{
		MaxDepth:          10,
		MaxPages:          1000,
		CrawlWorkers:      1,
		ExtractWorkers:    2,
		ProcessWorkers:    4,
		RequestDelay:      1 * time.Second,
		Timeout:           30 * time.Second,
		ContentSelectors:  []string{"article", ".content", ".main-content", "#content", ".post-content", "main"},
		RemoveSelectors:   []string{"nav", ".nav", ".navigation", "header", "footer", ".sidebar", ".ads", ".advertisement", "script", "style"},
		OutputDir:         "./output",
		OutputFormats:     []string{"markdown"},
		UserAgent:         "Ariadne/1.0 (Educational Purpose)",
		IncludeImages:     true,
		RespectRobots:     true,
		EnableCheckpoints: false,
		RateLimit: RateLimitConfig{
			Enabled:                  true,
			InitialRPS:               2.0,
			MinRPS:                   0.25,
			MaxRPS:                   8.0,
			TokenBucketCapacity:      4.0,
			AIMDIncrease:             0.25,
			AIMDDecrease:             0.5,
			LatencyTarget:            1 * time.Second,
			LatencyDegradeFactor:     2.0,
			ErrorRateThreshold:       0.4,
			MinSamplesToTrip:         10,
			ConsecutiveFailThreshold: 5,
			OpenStateDuration:        15 * time.Second,
			HalfOpenProbes:           3,
			RetryBaseDelay:           200 * time.Millisecond,
			RetryMaxDelay:            5 * time.Second,
			RetryMaxAttempts:         3,
			StatsWindow:              30 * time.Second,
			StatsBucket:              2 * time.Second,
			DomainStateTTL:           2 * time.Minute,
			Shards:                   16,
		},
	}
}

// Validate performs basic sanity checks on the configuration.
// Validate performs basic sanity checks on the configuration.
// Experimental: Validation rules may evolve; callers should treat returned errors as transient guidance.
func (c *ScraperConfig) Validate() error {
	if c.StartURL == "" {
		return ErrMissingStartURL
	}
	if len(c.AllowedDomains) == 0 {
		return ErrMissingAllowedDomains
	}
	if c.MaxDepth < 1 {
		return ErrInvalidMaxDepth
	}
	if c.CrawlWorkers < 1 {
		c.CrawlWorkers = 1
	}
	return nil
}

// Domain-specific errors (copied for locality; keep values identical)
// Domain-specific errors.
// Experimental: Error list may consolidate into sentinel + typed wrappers.
var (
	ErrMissingStartURL       = errors.New("start URL is required")
	ErrMissingAllowedDomains = errors.New("at least one allowed domain is required")
	ErrInvalidMaxDepth       = errors.New("max depth must be greater than 0")
	ErrURLNotAllowed         = errors.New("URL is not in allowed domains")
	ErrMaxDepthExceeded      = errors.New("maximum crawl depth exceeded")
	ErrMaxPagesExceeded      = errors.New("maximum pages limit reached")
	ErrContentNotFound       = errors.New("main content not found on page")
	ErrHTTPError             = errors.New("HTTP request failed")
	ErrHTMLParsingFailed     = errors.New("failed to parse HTML content")
	ErrMarkdownConversion    = errors.New("failed to convert HTML to markdown")
	ErrAssetDownloadFailed   = errors.New("failed to download asset")
	ErrOutputDirCreation     = errors.New("failed to create output directory")
	ErrFileWriteFailed       = errors.New("failed to write output file")
	ErrTemplateExecution     = errors.New("failed to execute template")
)

// CrawlError wraps a stage-specific error with page context.
// Experimental: May gain more structured classification fields.
type CrawlError struct {
	URL   string
	Stage string
	Err   error
}

func (e *CrawlError) Error() string { return e.Err.Error() }
func (e *CrawlError) Unwrap() error { return e.Err }
func NewCrawlError(url, stage string, err error) *CrawlError {
	return &CrawlError{URL: url, Stage: stage, Err: err}
}
