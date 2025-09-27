package processor

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"

	// TODO(phase5f-wave3): Reintroduce legacy processor bridging within engine/internal without cross-module import.
	// Previously used "ariadne/internal/processor"; now temporarily replaced by minimal parser here.
	"github.com/99souls/ariadne/engine/models"
)

// CompatibilityAdapter bridges the new Processor interface with existing internal processor
type CompatibilityAdapter struct {
	// contentProcessor *processor.ContentProcessor // removed pending internal relocation
	policy ProcessPolicy
	stats  ProcessorStats
}

// extractTitleAndMarkdown performs a very lightweight HTML walk to get the first <title>
// and produce naive markdown by mapping <h1>/<h2>/<p> elements.
func extractTitleAndMarkdown(htmlStr string) (string, string) {
	if htmlStr == "" {
		return "", ""
	}
	z := html.NewTokenizer(strings.NewReader(htmlStr))
	var (
		titleFound bool
		title      string
		bldr       strings.Builder
		inP        bool
	)
	for {
		tt := z.Next()
		if tt == html.ErrorToken {
			break
		}
		tok := z.Token()
		switch tt {
		case html.StartTagToken:
			switch tok.Data {
			case "title":
				if !titleFound {
					// Next token should be text
					if z.Next() == html.TextToken {
						title = strings.TrimSpace(z.Token().Data)
						titleFound = true
					}
				}
			case "h1":
				if z.Next() == html.TextToken {
					h := strings.TrimSpace(z.Token().Data)
					if h != "" {
						bldr.WriteString("# " + h + "\n\n")
					}
				}
			case "h2":
				if z.Next() == html.TextToken {
					h := strings.TrimSpace(z.Token().Data)
					if h != "" {
						bldr.WriteString("## " + h + "\n\n")
					}
				}
			case "p":
				inP = true
			}
		case html.EndTagToken:
			if tok.Data == "p" && inP {
				bldr.WriteString("\n\n")
				inP = false
			}
		case html.TextToken:
			if inP {
				txt := strings.TrimSpace(tok.Data)
				if txt != "" {
					bldr.WriteString(txt + " ")
				}
			}
		}
	}
	md := strings.TrimSpace(bldr.String())
	return title, md
}

// NewCompatibilityAdapter creates a new adapter for existing processor logic
func NewCompatibilityAdapter() *CompatibilityAdapter {
	return &CompatibilityAdapter{policy: DefaultProcessPolicy(), stats: ProcessorStats{}}
}

// Process transforms content using the existing internal processor
func (ca *CompatibilityAdapter) Process(request ProcessRequest) (*ProcessResult, error) {
	startTime := time.Now()

	if request.Page == nil {
		return nil, fmt.Errorf("page cannot be nil")
	}

	if request.Context == nil {
		request.Context = context.Background()
	}

	// Use effective policy (request policy overrides adapter defaults)
	_ = ca.mergePolicy(request.Policy)

	// Create a copy of the page to avoid modifying the original
	page := &models.Page{
		URL:         request.Page.URL,
		Title:       request.Page.Title,
		Content:     request.Page.Content,
		CleanedText: request.Page.CleanedText,
		Markdown:    request.Page.Markdown,
		Links:       request.Page.Links,
		Images:      request.Page.Images,
		Metadata:    request.Page.Metadata,
		CrawledAt:   request.Page.CrawledAt,
		ProcessedAt: request.Page.ProcessedAt,
	}

	var warnings []string

	// Use the existing processor logic
	// BaseURL ignored in transitional no-op path (will be reintegrated when legacy content processor ported).

	// Minimal inline processing (temporary) replacing legacy pipeline:
	// - Extract <title>
	// - Convert headings & paragraphs to naive markdown
	if request.Policy.ExtractContent || request.Policy.ConvertToMarkdown {
		title, markdown := extractTitleAndMarkdown(page.Content)
		if title != "" {
			page.Title = title
		}
		if request.Policy.ConvertToMarkdown {
			page.Markdown = markdown
		}
	}

	processingTime := time.Since(startTime)
	ca.stats.ProcessingTime += processingTime
	ca.stats.PagesProcessed++

	// Always succeed in minimal adapter path
	ca.stats.PagesSucceeded++

	// Calculate metrics
	wordCount := ca.calculateWordCount(page.Content)
	linksFound := len(page.Links)
	imagesFound := len(page.Images)

	ca.stats.TotalWordCount += int64(wordCount)
	if ca.stats.PagesProcessed > 0 {
		ca.stats.AverageWordCount = ca.stats.TotalWordCount / ca.stats.PagesProcessed
	}

	return &ProcessResult{
		Page:           page,
		Success:        true,
		ProcessingTime: processingTime,
		WordCount:      wordCount,
		LinksFound:     linksFound,
		ImagesFound:    imagesFound,
		Warnings:       warnings,
	}, nil
}

// Configure updates the adapter's policy settings
func (ca *CompatibilityAdapter) Configure(policy ProcessPolicy) error {
	// Validate policy
	if policy.MaxWordCount < 0 || policy.MinWordCount < 0 {
		return fmt.Errorf("word count limits cannot be negative: MaxWordCount=%d, MinWordCount=%d",
			policy.MaxWordCount, policy.MinWordCount)
	}

	if policy.MaxWordCount > 0 && policy.MinWordCount > 0 && policy.MaxWordCount < policy.MinWordCount {
		return fmt.Errorf("MaxWordCount (%d) cannot be less than MinWordCount (%d)",
			policy.MaxWordCount, policy.MinWordCount)
	}

	ca.policy = policy
	return nil
}

// Stats returns current processing statistics
func (ca *CompatibilityAdapter) Stats() ProcessorStats {
	return ca.stats
}

// mergePolicy combines adapter policy with request-specific policy
func (ca *CompatibilityAdapter) mergePolicy(requestPolicy ProcessPolicy) ProcessPolicy {
	// Start with adapter's policy as base
	merged := ca.policy

	// Override with non-zero request values
	if requestPolicy.ExtractContent {
		merged.ExtractContent = requestPolicy.ExtractContent
	}
	if requestPolicy.ConvertToMarkdown {
		merged.ConvertToMarkdown = requestPolicy.ConvertToMarkdown
	}
	if requestPolicy.ExtractMetadata {
		merged.ExtractMetadata = requestPolicy.ExtractMetadata
	}
	if requestPolicy.ExtractImages {
		merged.ExtractImages = requestPolicy.ExtractImages
	}
	if len(requestPolicy.ContentSelectors) > 0 {
		merged.ContentSelectors = requestPolicy.ContentSelectors
	}
	if len(requestPolicy.RemoveSelectors) > 0 {
		merged.RemoveSelectors = requestPolicy.RemoveSelectors
	}

	return merged
}

// calculateWordCount counts words in HTML content
func (ca *CompatibilityAdapter) calculateWordCount(content string) int {
	// Remove HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(content, " ")

	// Split by whitespace and count non-empty words
	words := strings.Fields(text)
	count := 0
	for _, word := range words {
		if strings.TrimSpace(word) != "" {
			count++
		}
	}

	return count
}
