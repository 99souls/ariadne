package models

// NOTE: These types were migrated from pkg/models to consolidate public embedding
// under packages/engine. Original pkg/models now provides type aliases with
// deprecation comments to avoid breaking existing imports for a deprecation window.

import (
	"errors"
	"net/url"
	"time"
)

// Page represents a single scraped web page with its content and metadata
// (lifted verbatim from legacy pkg/models). Keep structure identical for stability.
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

type PageMeta struct {
	Author      string            `json:"author,omitempty"`
	Description string            `json:"description,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	PublishDate time.Time         `json:"publish_date,omitempty"`
	WordCount   int               `json:"word_count"`
	Headers     map[string]string `json:"headers,omitempty"`
	OpenGraph   OpenGraphMeta     `json:"open_graph,omitempty"`
}

type OpenGraphMeta struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
}

// CrawlResult represents the result of processing a single URL through the pipeline
// (error remains an interface field; JSON marshalling will omit detailed error stack).
type CrawlResult struct {
	URL     string `json:"url"`
	Page    *Page  `json:"page"`
	Error   error  `json:"error,omitempty"`
	Stage   string `json:"stage"`
	Success bool   `json:"success"`
	Retry   bool   `json:"retry"`
}

type CrawlStats struct {
	TotalPages     int           `json:"total_pages"`
	ProcessedPages int           `json:"processed_pages"`
	FailedPages    int           `json:"failed_pages"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time,omitempty"`
	Duration       time.Duration `json:"duration,omitempty"`
	PagesPerSec    float64       `json:"pages_per_sec,omitempty"`
}

// RateLimitConfig defines adaptive per-domain rate limiting behavior
// (kept identical to legacy version for compatibility).
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

// Domain-specific errors (copied for locality; keep values identical)
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

type CrawlError struct {
	URL   string
	Stage string
	Err   error
}

func (e *CrawlError) Error() string { return e.Err.Error() }
func (e *CrawlError) Unwrap() error { return e.Err }
func NewCrawlError(url, stage string, err error) *CrawlError { return &CrawlError{URL: url, Stage: stage, Err: err} }
