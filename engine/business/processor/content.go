package processor

import (
	"strings"
	"time"
)

// ContentProcessingPolicy defines how content should be processed
type ContentProcessingPolicy struct {
	ContentSelectors    []string `json:"content_selectors"`    // CSS selectors for main content extraction
	CleaningRules       []string `json:"cleaning_rules"`       // Elements to remove during cleaning
	URLConversionRules  []string `json:"url_conversion_rules"` // Attributes to convert from relative to absolute
	MetadataExtraction  bool     `json:"metadata_extraction"`  // Whether to extract page metadata
	ImageExtraction     bool     `json:"image_extraction"`     // Whether to extract image URLs
	MarkdownConversion  bool     `json:"markdown_conversion"`  // Whether to convert HTML to Markdown
	ContentValidation   bool     `json:"content_validation"`   // Whether to validate content quality
}

// ContentQualityPolicy defines quality requirements for processed content
type ContentQualityPolicy struct {
	MinWordCount        int     `json:"min_word_count"`        // Minimum word count for valid content
	MinTitleLength      int     `json:"min_title_length"`      // Minimum title length requirement
	RequireHeadings     bool    `json:"require_headings"`      // Whether content must have headings
	MaxHTMLTagRatio     float64 `json:"max_html_tag_ratio"`    // Maximum ratio of HTML tags to content
	ValidateOpenGraph   bool    `json:"validate_open_graph"`   // Whether to validate Open Graph metadata
	RequireDescription  bool    `json:"require_description"`   // Whether page must have description
}

// ProcessingBusinessPolicy combines content processing and quality policies
type ProcessingBusinessPolicy struct {
	ContentPolicy ContentProcessingPolicy `json:"content_policy"`
	QualityPolicy ContentQualityPolicy    `json:"quality_policy"`
}

// ProcessingDecision represents a decision about content processing
type ProcessingDecision struct {
	URL            string `json:"url"`
	ShouldProcess  bool   `json:"should_process"`
	ProcessingType string `json:"processing_type"`
	Reason         string `json:"reason"`
}

// ProcessingStepsDecision represents processing steps for a URL
type ProcessingStepsDecision struct {
	URL             string   `json:"url"`
	ProcessingSteps []string `json:"processing_steps"`
}

// ProcessingContext holds context information for processing decisions
type ProcessingContext struct {
	URL       string                    `json:"url"`
	Policy    ProcessingBusinessPolicy  `json:"policy"`
	CreatedAt time.Time                `json:"created_at"`
	Status    string                   `json:"status"`
}

// ContentProcessingPolicyEvaluator evaluates content processing policies
type ContentProcessingPolicyEvaluator struct {
	// No fields needed for stateless evaluation
}

// ContentQualityPolicyEvaluator evaluates content quality policies
type ContentQualityPolicyEvaluator struct {
	// No fields needed for stateless evaluation
}

// ProcessingDecisionMaker makes business decisions about content processing
type ProcessingDecisionMaker struct {
	contentEvaluator *ContentProcessingPolicyEvaluator
	qualityEvaluator *ContentQualityPolicyEvaluator
}

// NewContentProcessingPolicyEvaluator creates a new content processing policy evaluator
func NewContentProcessingPolicyEvaluator() *ContentProcessingPolicyEvaluator {
	return &ContentProcessingPolicyEvaluator{}
}

// NewContentQualityPolicyEvaluator creates a new content quality policy evaluator
func NewContentQualityPolicyEvaluator() *ContentQualityPolicyEvaluator {
	return &ContentQualityPolicyEvaluator{}
}

// NewProcessingDecisionMaker creates a new processing decision maker
func NewProcessingDecisionMaker() *ProcessingDecisionMaker {
	return &ProcessingDecisionMaker{
		contentEvaluator: NewContentProcessingPolicyEvaluator(),
		qualityEvaluator: NewContentQualityPolicyEvaluator(),
	}
}

// ContentProcessingPolicyEvaluator methods

// ShouldProcessContent determines if content should be processed based on URL and policy
func (e *ContentProcessingPolicyEvaluator) ShouldProcessContent(url string, policy ContentProcessingPolicy) bool {
	// Default behavior is to process all content unless specifically disabled
	return true
}

// GetContentSelectors returns the appropriate content selectors for a URL
func (e *ContentProcessingPolicyEvaluator) GetContentSelectors(url string, policy ContentProcessingPolicy) []string {
	return policy.ContentSelectors
}

// GetCleaningRules returns the cleaning rules for content processing
func (e *ContentProcessingPolicyEvaluator) GetCleaningRules(url string, policy ContentProcessingPolicy) []string {
	return policy.CleaningRules
}

// ShouldExtractMetadata determines if metadata should be extracted
func (e *ContentProcessingPolicyEvaluator) ShouldExtractMetadata(url string, policy ContentProcessingPolicy) bool {
	return policy.MetadataExtraction
}

// ShouldExtractImages determines if images should be extracted
func (e *ContentProcessingPolicyEvaluator) ShouldExtractImages(url string, policy ContentProcessingPolicy) bool {
	return policy.ImageExtraction
}

// ShouldConvertToMarkdown determines if content should be converted to Markdown
func (e *ContentProcessingPolicyEvaluator) ShouldConvertToMarkdown(url string, policy ContentProcessingPolicy) bool {
	return policy.MarkdownConversion
}

// ContentQualityPolicyEvaluator methods

// MeetsWordCountRequirement checks if word count meets the minimum requirement
func (e *ContentQualityPolicyEvaluator) MeetsWordCountRequirement(wordCount int, policy ContentQualityPolicy) bool {
	return wordCount >= policy.MinWordCount
}

// MeetsTitleLengthRequirement checks if title length meets the minimum requirement
func (e *ContentQualityPolicyEvaluator) MeetsTitleLengthRequirement(title string, policy ContentQualityPolicy) bool {
	return len(strings.TrimSpace(title)) >= policy.MinTitleLength
}

// MeetsHeadingsRequirement checks if content has required headings
func (e *ContentQualityPolicyEvaluator) MeetsHeadingsRequirement(content string, policy ContentQualityPolicy) bool {
	if !policy.RequireHeadings {
		return true
	}
	
	return strings.Contains(content, "<h1") || strings.Contains(content, "<h2")
}

// MeetsHTMLTagRatioRequirement checks if HTML tag ratio is within acceptable limits
func (e *ContentQualityPolicyEvaluator) MeetsHTMLTagRatioRequirement(content string, policy ContentQualityPolicy) bool {
	if policy.MaxHTMLTagRatio >= 1.0 {
		return true // No restriction
	}
	
	tagCount := strings.Count(content, "<")
	if tagCount == 0 {
		return true // No tags, ratio is 0
	}
	
	// Count words (approximate by splitting on whitespace and filtering out HTML tags)
	words := strings.Fields(content)
	wordCount := 0
	for _, word := range words {
		if !strings.HasPrefix(word, "<") {
			wordCount++
		}
	}
	
	if wordCount == 0 {
		return false // All tags, no content
	}
	
	ratio := float64(tagCount) / float64(wordCount)
	return ratio <= policy.MaxHTMLTagRatio
}

// MeetsDescriptionRequirement checks if description meets requirements
func (e *ContentQualityPolicyEvaluator) MeetsDescriptionRequirement(description string, policy ContentQualityPolicy) bool {
	if !policy.RequireDescription {
		return true
	}
	
	return strings.TrimSpace(description) != ""
}

// ProcessingDecisionMaker methods

// ShouldProcessContent makes a decision about whether to process specific content
func (d *ProcessingDecisionMaker) ShouldProcessContent(url string, policy ProcessingBusinessPolicy) ProcessingDecision {
	shouldProcess := d.contentEvaluator.ShouldProcessContent(url, policy.ContentPolicy)
	
	return ProcessingDecision{
		URL:            url,
		ShouldProcess:  shouldProcess,
		ProcessingType: "content_processing",
		Reason:         "policy_evaluation",
	}
}

// GetProcessingSteps returns the processing steps for a given URL and policy
func (d *ProcessingDecisionMaker) GetProcessingSteps(url string, policy ProcessingBusinessPolicy) ProcessingStepsDecision {
	steps := []string{}
	
	// Always include basic content extraction and cleaning
	steps = append(steps, "content_extraction", "content_cleaning")
	
	if policy.ContentPolicy.MetadataExtraction {
		steps = append(steps, "metadata_extraction")
	}
	
	if policy.ContentPolicy.ImageExtraction {
		steps = append(steps, "image_extraction")
	}
	
	if policy.ContentPolicy.MarkdownConversion {
		steps = append(steps, "markdown_conversion")
	}
	
	if policy.ContentPolicy.ContentValidation {
		steps = append(steps, "content_validation")
	}
	
	return ProcessingStepsDecision{
		URL:             url,
		ProcessingSteps: steps,
	}
}

// CreateProcessingContext creates a processing context for a URL
func (d *ProcessingDecisionMaker) CreateProcessingContext(url string, policy ProcessingBusinessPolicy) ProcessingContext {
	return ProcessingContext{
		URL:       url,
		Policy:    policy,
		CreatedAt: time.Now(),
		Status:    "pending",
	}
}

// BatchShouldProcess makes processing decisions for multiple URLs
func (d *ProcessingDecisionMaker) BatchShouldProcess(urls []string, policy ProcessingBusinessPolicy) []ProcessingDecision {
	decisions := make([]ProcessingDecision, len(urls))
	
	for i, url := range urls {
		decisions[i] = d.ShouldProcessContent(url, policy)
	}
	
	return decisions
}