package models

import (
	"time"
)

// ScraperConfig holds all configuration for the scraper
type ScraperConfig struct {
	// Target configuration
	StartURL    string   `yaml:"start_url" json:"start_url"`
	AllowedDomains []string `yaml:"allowed_domains" json:"allowed_domains"`
	MaxDepth    int      `yaml:"max_depth" json:"max_depth"`
	MaxPages    int      `yaml:"max_pages" json:"max_pages"`
	
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
	UserAgent      string `yaml:"user_agent" json:"user_agent"`
	IncludeImages  bool   `yaml:"include_images" json:"include_images"`
	RespectRobots  bool   `yaml:"respect_robots" json:"respect_robots"`
	EnableCheckpoints bool `yaml:"enable_checkpoints" json:"enable_checkpoints"`
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
		OutputDir:     "./output",
		OutputFormats: []string{"markdown"},
		UserAgent:     "Site-Scraper/1.0 (Educational Purpose)",
		IncludeImages: true,
		RespectRobots: true,
		EnableCheckpoints: false,
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