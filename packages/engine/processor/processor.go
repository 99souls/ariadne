package processor

import (
	"context"
	"net/url"
	"time"

	"ariadne/packages/engine/models"
)

// ProcessRequest encapsulates all parameters needed for content processing
type ProcessRequest struct {
	Page    *models.Page
	BaseURL *url.URL
	Policy  ProcessPolicy
	Context context.Context
}

// ProcessPolicy defines configuration for content processing behavior
type ProcessPolicy struct {
	// Content extraction settings
	ExtractContent     bool
	ContentSelectors   []string
	RemoveSelectors    []string
	PreserveFormatting bool

	// Conversion settings
	ConvertToMarkdown bool
	ExtractMetadata   bool
	ExtractImages     bool
	ExtractLinks      bool

	// Validation settings
	ValidateContent bool
	MinWordCount    int
	MaxWordCount    int
	AllowedDomains  []string

	// Processing limits
	TimeoutDuration time.Duration
	MaxRetries      int
}

// ProcessResult contains the outcome of content processing
type ProcessResult struct {
	Page           *models.Page
	Success        bool
	Error          error
	ProcessingTime time.Duration
	WordCount      int
	LinksFound     int
	ImagesFound    int
	Warnings       []string
}

// ProcessorStats provides metrics about processing operations
type ProcessorStats struct {
	PagesProcessed   int64
	PagesSucceeded   int64
	PagesFailed      int64
	TotalWordCount   int64
	AverageWordCount int64
	ProcessingTime   time.Duration
}

// Processor abstracts content processing operations with configurable policies
type Processor interface {
	// Process transforms raw page content according to the specified policy
	Process(request ProcessRequest) (*ProcessResult, error)

	// Configure updates the processor's default policy settings
	Configure(policy ProcessPolicy) error

	// Stats returns current processing statistics
	Stats() ProcessorStats
}

// ContentProcessor implements the Processor interface by delegating to CompatibilityAdapter
type ContentProcessor struct {
	adapter *CompatibilityAdapter
}

// NewContentProcessor creates a new content processor with default settings
func NewContentProcessor() *ContentProcessor {
	return &ContentProcessor{
		adapter: NewCompatibilityAdapter(),
	}
}

// Process transforms raw page content according to the specified policy
func (cp *ContentProcessor) Process(request ProcessRequest) (*ProcessResult, error) {
	return cp.adapter.Process(request)
}

// Configure updates the processor's default policy settings
func (cp *ContentProcessor) Configure(policy ProcessPolicy) error {
	return cp.adapter.Configure(policy)
}

// Stats returns current processing statistics
func (cp *ContentProcessor) Stats() ProcessorStats {
	return cp.adapter.Stats()
}

// DefaultProcessPolicy returns a sensible default processing policy
func DefaultProcessPolicy() ProcessPolicy {
	return ProcessPolicy{
		ExtractContent:     true,
		ConvertToMarkdown:  true,
		ExtractMetadata:    true,
		ExtractImages:      true,
		ExtractLinks:       true,
		ValidateContent:    true,
		ContentSelectors:   []string{"article", "main", ".content", "#content"},
		RemoveSelectors:    []string{"script", "style", "nav", "footer", "aside", "header"},
		PreserveFormatting: true,
		MinWordCount:       10,
		MaxWordCount:       100000,
		TimeoutDuration:    30 * time.Second,
		MaxRetries:         3,
	}
}
