package models

import (
	"time"
)

// ScraperConfig holds all configuration for the scraper
type ScraperConfig struct {
	// Target configuration
	StartURL       string   `yaml:"start_url" json:"start_url"`
	AllowedDomains []string `yaml:"allowed_domains" json:"allowed_domains"`
	MaxDepth       int      `yaml:"max_depth" json:"max_depth"`
	MaxPages       int      `yaml:"max_pages" json:"max_pages"`

	// Worker configuration
	CrawlWorkers   int `yaml:"crawl_workers" json:"crawl_workers"`
	ExtractWorkers int `yaml:"extract_workers" json:"extract_workers"`
	ProcessWorkers int `yaml:"process_workers" json:"process_workers"`

	// Rate limiting
	RequestDelay time.Duration `yaml:"request_delay" json:"request_delay"`
	Timeout      time.Duration `yaml:"timeout" json:"timeout"`

	// Content extraction
	ContentSelectors []string `yaml:"content_selectors" json:"content_selectors"`
	RemoveSelectors  []string `yaml:"remove_selectors" json:"remove_selectors"`

	// Output configuration
	OutputDir     string   `yaml:"output_dir" json:"output_dir"`
	OutputFormats []string `yaml:"output_formats" json:"output_formats"`

	// Advanced options
	UserAgent         string `yaml:"user_agent" json:"user_agent"`
	IncludeImages     bool   `yaml:"include_images" json:"include_images"`
	RespectRobots     bool   `yaml:"respect_robots" json:"respect_robots"`
	EnableCheckpoints bool   `yaml:"enable_checkpoints" json:"enable_checkpoints"`

	// Intelligent rate limiting configuration (Phase 3.2)
	RateLimit RateLimitConfig `yaml:"rate_limit" json:"rate_limit"`
}

// RateLimitConfig defines adaptive per-domain rate limiting behavior
type RateLimitConfig struct {
	Enabled             bool    `yaml:"enabled" json:"enabled"`
	InitialRPS          float64 `yaml:"initial_rps" json:"initial_rps"`
	MinRPS              float64 `yaml:"min_rps" json:"min_rps"`
	MaxRPS              float64 `yaml:"max_rps" json:"max_rps"`
	TokenBucketCapacity float64 `yaml:"token_bucket_capacity" json:"token_bucket_capacity"`

	AIMDIncrease         float64       `yaml:"aimd_increase" json:"aimd_increase"`
	AIMDDecrease         float64       `yaml:"aimd_decrease" json:"aimd_decrease"`
	LatencyTarget        time.Duration `yaml:"latency_target" json:"latency_target"`
	LatencyDegradeFactor float64       `yaml:"latency_degrade_factor" json:"latency_degrade_factor"`

	ErrorRateThreshold       float64       `yaml:"error_rate_threshold" json:"error_rate_threshold"`
	MinSamplesToTrip         int           `yaml:"min_samples_to_trip" json:"min_samples_to_trip"`
	ConsecutiveFailThreshold int           `yaml:"consecutive_fail_threshold" json:"consecutive_fail_threshold"`
	OpenStateDuration        time.Duration `yaml:"open_state_duration" json:"open_state_duration"`
	HalfOpenProbes           int           `yaml:"half_open_probes" json:"half_open_probes"`

	RetryBaseDelay   time.Duration `yaml:"retry_base_delay" json:"retry_base_delay"`
	RetryMaxDelay    time.Duration `yaml:"retry_max_delay" json:"retry_max_delay"`
	RetryMaxAttempts int           `yaml:"retry_max_attempts" json:"retry_max_attempts"`

	StatsWindow    time.Duration `yaml:"stats_window" json:"stats_window"`
	StatsBucket    time.Duration `yaml:"stats_bucket" json:"stats_bucket"`
	DomainStateTTL time.Duration `yaml:"domain_state_ttl" json:"domain_state_ttl"`
	Shards         int           `yaml:"shards" json:"shards"`
}

// DefaultConfig returns a sensible default configuration
func DefaultConfig() *ScraperConfig {
	return &ScraperConfig{
		MaxDepth:       10,
		MaxPages:       1000,
		CrawlWorkers:   1,
		ExtractWorkers: 2,
		ProcessWorkers: 4,
		RequestDelay:   1 * time.Second,
		Timeout:        30 * time.Second,
		ContentSelectors: []string{
			"article", ".content", ".main-content",
			"#content", ".post-content", "main",
		},
		RemoveSelectors: []string{
			"nav", ".nav", ".navigation", "header", "footer",
			".sidebar", ".ads", ".advertisement", "script", "style",
		},
		OutputDir:         "./output",
		OutputFormats:     []string{"markdown"},
		UserAgent:         "Site-Scraper/1.0 (Educational Purpose)",
		IncludeImages:     true,
		RespectRobots:     true,
		EnableCheckpoints: false,
		RateLimit: RateLimitConfig{
			Enabled:             true,
			InitialRPS:          2.0,
			MinRPS:              0.25,
			MaxRPS:              8.0,
			TokenBucketCapacity: 4.0,

			AIMDIncrease:         0.25,
			AIMDDecrease:         0.5,
			LatencyTarget:        1 * time.Second,
			LatencyDegradeFactor: 2.0,

			ErrorRateThreshold:       0.4,
			MinSamplesToTrip:         10,
			ConsecutiveFailThreshold: 5,
			OpenStateDuration:        15 * time.Second,
			HalfOpenProbes:           3,

			RetryBaseDelay:   200 * time.Millisecond,
			RetryMaxDelay:    5 * time.Second,
			RetryMaxAttempts: 3,

			StatsWindow:    30 * time.Second,
			StatsBucket:    2 * time.Second,
			DomainStateTTL: 2 * time.Minute,
			Shards:         16,
		},
	}
}

// Validate checks if the configuration is valid
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
