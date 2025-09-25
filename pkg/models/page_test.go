package models

import (
	"fmt"
	"net/url"
	"testing"
	"time"
)

// TestPageModel validates the Page data structure meets our requirements
func TestPageModel(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Page
		validate func(*Page) error
	}{
		{
			name: "Page should store URL correctly",
			setup: func() *Page {
				u, _ := url.Parse("https://example.com/test")
				return &Page{URL: u}
			},
			validate: func(p *Page) error {
				if p.URL.String() != "https://example.com/test" {
					return fmt.Errorf("expected URL https://example.com/test, got %s", p.URL.String())
				}
				return nil
			},
		},
		{
			name: "Page should extract title properly",
			setup: func() *Page {
				u, _ := url.Parse("https://example.com")
				return &Page{
					URL:   u,
					Title: "Test Page Title",
				}
			},
			validate: func(p *Page) error {
				if p.Title != "Test Page Title" {
					return fmt.Errorf("expected title 'Test Page Title', got '%s'", p.Title)
				}
				return nil
			},
		},
		{
			name: "Page should handle metadata correctly",
			setup: func() *Page {
				u, _ := url.Parse("https://example.com")
				return &Page{
					URL: u,
					Metadata: PageMeta{
						Description: "Test description",
						Keywords:    []string{"test", "page"},
						WordCount:   150,
					},
				}
			},
			validate: func(p *Page) error {
				if p.Metadata.Description != "Test description" {
					return fmt.Errorf("expected description 'Test description', got '%s'", p.Metadata.Description)
				}
				if len(p.Metadata.Keywords) != 2 || p.Metadata.Keywords[0] != "test" {
					return fmt.Errorf("keywords not parsed correctly: %v", p.Metadata.Keywords)
				}
				if p.Metadata.WordCount != 150 {
					return fmt.Errorf("expected word count 150, got %d", p.Metadata.WordCount)
				}
				return nil
			},
		},
		{
			name: "Page should track timestamps",
			setup: func() *Page {
				u, _ := url.Parse("https://example.com")
				now := time.Now()
				return &Page{
					URL:         u,
					CrawledAt:   now,
					ProcessedAt: now.Add(1 * time.Second),
				}
			},
			validate: func(p *Page) error {
				if p.CrawledAt.IsZero() {
					return fmt.Errorf("CrawledAt timestamp should not be zero")
				}
				if p.ProcessedAt.IsZero() {
					return fmt.Errorf("ProcessedAt timestamp should not be zero")
				}
				if !p.ProcessedAt.After(p.CrawledAt) {
					return fmt.Errorf("ProcessedAt should be after CrawledAt")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			page := tt.setup()
			if err := tt.validate(page); err != nil {
				t.Errorf("Test failed: %v", err)
			}
		})
	}
}

// TestCrawlResultModel validates CrawlResult meets pipeline requirements
func TestCrawlResultModel(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *CrawlResult
		validate func(*CrawlResult) error
	}{
		{
			name: "CrawlResult should indicate success",
			setup: func() *CrawlResult {
				u, _ := url.Parse("https://example.com")
				page := &Page{URL: u, Title: "Test"}
				return &CrawlResult{
					Page:    page,
					Success: true,
					Stage:   "crawl",
				}
			},
			validate: func(cr *CrawlResult) error {
				if !cr.Success {
					return fmt.Errorf("expected success to be true")
				}
				if cr.Stage != "crawl" {
					return fmt.Errorf("expected stage 'crawl', got '%s'", cr.Stage)
				}
				if cr.Page == nil {
					return fmt.Errorf("page should not be nil for successful result")
				}
				return nil
			},
		},
		{
			name: "CrawlResult should handle errors",
			setup: func() *CrawlResult {
				return &CrawlResult{
					Error:   NewCrawlError("https://example.com", "crawl", fmt.Errorf("connection timeout")),
					Success: false,
					Stage:   "crawl",
					Retry:   true,
				}
			},
			validate: func(cr *CrawlResult) error {
				if cr.Success {
					return fmt.Errorf("expected success to be false for error case")
				}
				if cr.Error == nil {
					return fmt.Errorf("error should not be nil for failed result")
				}
				if !cr.Retry {
					return fmt.Errorf("expected retry to be true for recoverable error")
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.setup()
			if err := tt.validate(result); err != nil {
				t.Errorf("Test failed: %v", err)
			}
		})
	}
}
