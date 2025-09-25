package models

import "errors"

// Domain-specific errors for the scraper
var (
	// Configuration errors
	ErrMissingStartURL       = errors.New("start URL is required")
	ErrMissingAllowedDomains = errors.New("at least one allowed domain is required")
	ErrInvalidMaxDepth       = errors.New("max depth must be greater than 0")
	
	// Crawling errors
	ErrURLNotAllowed      = errors.New("URL is not in allowed domains")
	ErrMaxDepthExceeded   = errors.New("maximum crawl depth exceeded")
	ErrMaxPagesExceeded   = errors.New("maximum pages limit reached")
	ErrContentNotFound    = errors.New("main content not found on page")
	ErrHTTPError          = errors.New("HTTP request failed")
	
	// Processing errors
	ErrHTMLParsingFailed  = errors.New("failed to parse HTML content")
	ErrMarkdownConversion = errors.New("failed to convert HTML to markdown")
	ErrAssetDownloadFailed = errors.New("failed to download asset")
	
	// Output errors
	ErrOutputDirCreation = errors.New("failed to create output directory")
	ErrFileWriteFailed   = errors.New("failed to write output file")
	ErrTemplateExecution = errors.New("failed to execute template")
)

// CrawlError wraps errors with additional context
type CrawlError struct {
	URL   string
	Stage string
	Err   error
}

func (e *CrawlError) Error() string {
	return e.Err.Error()
}

func (e *CrawlError) Unwrap() error {
	return e.Err
}

// NewCrawlError creates a new CrawlError with context
func NewCrawlError(url, stage string, err error) *CrawlError {
	return &CrawlError{
		URL:   url,
		Stage: stage,
		Err:   err,
	}
}