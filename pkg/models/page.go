package models

import (
	"net/url"
	"time"
)

// Page represents a single scraped web page with its content and metadata
type Page struct {
	URL         *url.URL   `json:"url"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`      // Raw HTML content
	CleanedText string     `json:"cleaned_text"` // Processed text content
	Markdown    string     `json:"markdown"`     // Markdown conversion
	Links       []*url.URL `json:"links"`        // Internal links found on this page
	Images      []string   `json:"images"`       // Image URLs found on this page
	Metadata    PageMeta   `json:"metadata"`
	CrawledAt   time.Time  `json:"crawled_at"`
	ProcessedAt time.Time  `json:"processed_at"`
}

// PageMeta contains additional metadata extracted from the page
type PageMeta struct {
	Author      string            `json:"author,omitempty"`
	Description string            `json:"description,omitempty"`
	Keywords    []string          `json:"keywords,omitempty"`
	PublishDate time.Time         `json:"publish_date,omitempty"`
	WordCount   int               `json:"word_count"`
	Headers     map[string]string `json:"headers,omitempty"` // HTTP headers
	OpenGraph   OpenGraphMeta     `json:"open_graph,omitempty"`
}

// OpenGraphMeta contains Open Graph metadata
type OpenGraphMeta struct {
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	Image       string `json:"image,omitempty"`
	URL         string `json:"url,omitempty"`
	Type        string `json:"type,omitempty"`
}

// CrawlResult represents the result of processing a single URL through the pipeline
type CrawlResult struct {
	URL     string `json:"url"`
	Page    *Page  `json:"page"`
	Error   error  `json:"error,omitempty"`
	Stage   string `json:"stage"` // Which stage produced this result
	Success bool   `json:"success"`
	Retry   bool   `json:"retry"` // Whether this should be retried
}

// CrawlStats tracks statistics about the crawling process
type CrawlStats struct {
	TotalPages     int           `json:"total_pages"`
	ProcessedPages int           `json:"processed_pages"`
	FailedPages    int           `json:"failed_pages"`
	StartTime      time.Time     `json:"start_time"`
	EndTime        time.Time     `json:"end_time,omitempty"`
	Duration       time.Duration `json:"duration,omitempty"`
	PagesPerSec    float64       `json:"pages_per_sec,omitempty"`
}
