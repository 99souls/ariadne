package processor

import (
	"math"
	"strings"
	"time"
)

// ContentValidationRules defines rules for content validation
type ContentValidationRules struct {
	MinWordCount        int      `json:"min_word_count"`        // Minimum word count requirement
	MaxWordCount        int      `json:"max_word_count"`        // Maximum word count limit
	MinTitleLength      int      `json:"min_title_length"`      // Minimum title length
	MaxTitleLength      int      `json:"max_title_length"`      // Maximum title length
	RequireHeadings     bool     `json:"require_headings"`      // Whether headings are required
	AllowEmptyHeadings  bool     `json:"allow_empty_headings"`  // Whether empty headings are allowed
	MaxHTMLTagRatio     float64  `json:"max_html_tag_ratio"`    // Maximum ratio of HTML tags to content
	RequireMetadata     bool     `json:"require_metadata"`      // Whether metadata is required
	AllowedContentTypes []string `json:"allowed_content_types"` // Allowed content types
}

// ContentValidationResult represents the result of content validation
type ContentValidationResult struct {
	IsValid           bool     `json:"is_valid"`            // Whether content passes validation
	QualityScore      float64  `json:"quality_score"`       // Overall quality score (0-1)
	ValidationIssues  []string `json:"validation_issues"`   // List of validation issues found
	WordCount         int      `json:"word_count"`          // Actual word count
	TitleLength       int      `json:"title_length"`        // Actual title length
	HasHeadings       bool     `json:"has_headings"`        // Whether content has headings
	HasMetadata       bool     `json:"has_metadata"`        // Whether content has metadata
	ContentDensity    float64  `json:"content_density"`     // Ratio of content to markup
	EstimatedReadTime int      `json:"estimated_read_time"` // Estimated reading time in minutes
}

// ContentValidationBusinessPolicy combines validation rules and policy settings
type ContentValidationBusinessPolicy struct {
	ValidationRules  ContentValidationRules `json:"validation_rules"`  // Validation rules to apply
	QualityThreshold float64                `json:"quality_threshold"` // Minimum quality score required
	StrictMode       bool                   `json:"strict_mode"`       // Whether to use strict validation
}

// ValidationDecision represents a decision about content validation
type ValidationDecision struct {
	URL            string `json:"url"`
	ShouldValidate bool   `json:"should_validate"`
	ValidationType string `json:"validation_type"`
	Reason         string `json:"reason"`
}

// ValidationRulesDecision represents validation rules for a URL
type ValidationRulesDecision struct {
	URL             string                  `json:"url"`
	ValidationRules ContentValidationRules `json:"validation_rules"`
}

// ValidationContext holds context information for validation decisions
type ValidationContext struct {
	URL       string                           `json:"url"`
	Policy    ContentValidationBusinessPolicy `json:"policy"`
	CreatedAt time.Time                       `json:"created_at"`
	Status    string                          `json:"status"`
}

// ContentValidationPolicyEvaluator evaluates content validation policies
type ContentValidationPolicyEvaluator struct {
	// No fields needed for stateless evaluation
}

// ContentValidationDecisionMaker makes business decisions about content validation
type ContentValidationDecisionMaker struct {
	evaluator *ContentValidationPolicyEvaluator
	analyzer  *ContentQualityAnalyzer
}

// ContentQualityAnalyzer analyzes content quality metrics
type ContentQualityAnalyzer struct {
	// No fields needed for stateless analysis
}

// NewContentValidationPolicyEvaluator creates a new content validation policy evaluator
func NewContentValidationPolicyEvaluator() *ContentValidationPolicyEvaluator {
	return &ContentValidationPolicyEvaluator{}
}

// NewContentValidationDecisionMaker creates a new content validation decision maker
func NewContentValidationDecisionMaker() *ContentValidationDecisionMaker {
	return &ContentValidationDecisionMaker{
		evaluator: NewContentValidationPolicyEvaluator(),
		analyzer:  NewContentQualityAnalyzer(),
	}
}

// NewContentQualityAnalyzer creates a new content quality analyzer
func NewContentQualityAnalyzer() *ContentQualityAnalyzer {
	return &ContentQualityAnalyzer{}
}

// ContentValidationPolicyEvaluator methods

// ValidateWordCount checks if word count meets the requirements
func (e *ContentValidationPolicyEvaluator) ValidateWordCount(wordCount int, rules ContentValidationRules) bool {
	return wordCount >= rules.MinWordCount && wordCount <= rules.MaxWordCount
}

// ValidateTitleLength checks if title length meets the requirements
func (e *ContentValidationPolicyEvaluator) ValidateTitleLength(title string, rules ContentValidationRules) bool {
	length := len(strings.TrimSpace(title))
	return length >= rules.MinTitleLength && length <= rules.MaxTitleLength
}

// ValidateContentStructure checks if content structure meets the requirements
func (e *ContentValidationPolicyEvaluator) ValidateContentStructure(content string, rules ContentValidationRules) bool {
	if !rules.RequireHeadings {
		return true
	}
	
	hasHeadings := strings.Contains(content, "<h1") || strings.Contains(content, "<h2") || 
		strings.Contains(content, "<h3") || strings.Contains(content, "<h4") || 
		strings.Contains(content, "<h5") || strings.Contains(content, "<h6")
	
	return hasHeadings
}

// ValidateContentDensity checks if content density meets the requirements
func (e *ContentValidationPolicyEvaluator) ValidateContentDensity(content string, rules ContentValidationRules) bool {
	if strings.TrimSpace(content) == "" {
		return false
	}
	
	tagCount := strings.Count(content, "<")
	if tagCount == 0 {
		return true // No tags, density is perfect
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
	return ratio <= rules.MaxHTMLTagRatio
}

// ContentValidationDecisionMaker methods

// ShouldValidateContent makes a decision about whether to validate specific content
func (d *ContentValidationDecisionMaker) ShouldValidateContent(url string, policy ContentValidationBusinessPolicy) ValidationDecision {
	// Default behavior is to validate all content
	return ValidationDecision{
		URL:            url,
		ShouldValidate: true,
		ValidationType: "content_validation",
		Reason:         "policy_evaluation",
	}
}

// GetValidationRules returns the validation rules for a given URL and policy
func (d *ContentValidationDecisionMaker) GetValidationRules(url string, policy ContentValidationBusinessPolicy) ValidationRulesDecision {
	return ValidationRulesDecision{
		URL:             url,
		ValidationRules: policy.ValidationRules,
	}
}

// CreateValidationContext creates a validation context for a URL
func (d *ContentValidationDecisionMaker) CreateValidationContext(url string, policy ContentValidationBusinessPolicy) ValidationContext {
	return ValidationContext{
		URL:       url,
		Policy:    policy,
		CreatedAt: time.Now(),
		Status:    "pending",
	}
}

// BatchShouldValidate makes validation decisions for multiple URLs
func (d *ContentValidationDecisionMaker) BatchShouldValidate(urls []string, policy ContentValidationBusinessPolicy) []ValidationDecision {
	decisions := make([]ValidationDecision, len(urls))
	
	for i, url := range urls {
		decisions[i] = d.ShouldValidateContent(url, policy)
	}
	
	return decisions
}

// ContentQualityAnalyzer methods

// AnalyzeQualityScore analyzes content and returns a quality score (0-1)
func (a *ContentQualityAnalyzer) AnalyzeQualityScore(content string, title string, wordCount int) float64 {
	if strings.TrimSpace(content) == "" {
		return 0.0
	}
	
	score := 1.0
	
	// Title quality (20% of score)
	titleLength := len(strings.TrimSpace(title))
	if titleLength == 0 {
		score -= 0.2
	} else if titleLength < 10 {
		score -= 0.1
	}
	
	// Word count quality (30% of score)
	if wordCount < 10 {
		score -= 0.3
	} else if wordCount < 50 {
		score -= 0.15
	}
	
	// Content structure quality (25% of score)
	hasHeadings := strings.Contains(content, "<h1") || strings.Contains(content, "<h2")
	if !hasHeadings && wordCount > 100 {
		score -= 0.15
	}
	
	// Content density quality (25% of score)
	density := a.CalculateContentDensity(content)
	if density < 0.3 {
		score -= 0.25
	} else if density < 0.5 {
		score -= 0.1
	}
	
	// Ensure score stays within bounds
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

// CalculateContentDensity calculates the ratio of content to markup
func (a *ContentQualityAnalyzer) CalculateContentDensity(content string) float64 {
	if strings.TrimSpace(content) == "" {
		return 0.0
	}
	
	tagCount := strings.Count(content, "<")
	if tagCount == 0 {
		return 1.0 // Pure content, no markup
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
		return 0.0 // All markup, no content
	}
	
	// Return the ratio of words to total elements (words + tags)
	return float64(wordCount) / float64(wordCount + tagCount)
}

// EstimateReadingTime estimates reading time in minutes (assumes 200 words per minute)
func (a *ContentQualityAnalyzer) EstimateReadingTime(wordCount int) int {
	if wordCount <= 0 {
		return 0
	}
	
	// Assume average reading speed of 200 words per minute
	minutes := math.Ceil(float64(wordCount) / 200.0)
	
	// Minimum 1 minute for any content
	if minutes < 1 {
		return 1
	}
	
	return int(minutes)
}