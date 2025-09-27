package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentValidationRules(t *testing.T) {
	rules := ContentValidationRules{
		MinWordCount:        20,
		MaxWordCount:        5000,
		MinTitleLength:      5,
		MaxTitleLength:      200,
		RequireHeadings:     false,
		AllowEmptyHeadings:  true,
		MaxHTMLTagRatio:     0.6,
		RequireMetadata:     false,
		AllowedContentTypes: []string{"article", "blog", "page"},
	}

	t.Run("valid_rules_structure", func(t *testing.T) {
		assert.Equal(t, 20, rules.MinWordCount)
		assert.Equal(t, 5000, rules.MaxWordCount)
		assert.Equal(t, 5, rules.MinTitleLength)
		assert.Equal(t, 200, rules.MaxTitleLength)
		assert.False(t, rules.RequireHeadings)
		assert.True(t, rules.AllowEmptyHeadings)
		assert.Equal(t, 0.6, rules.MaxHTMLTagRatio)
		assert.False(t, rules.RequireMetadata)
		assert.Len(t, rules.AllowedContentTypes, 3)
	})
}

func TestContentValidationResult(t *testing.T) {
	result := ContentValidationResult{
		IsValid:           true,
		QualityScore:      0.85,
		ValidationIssues:  []string{},
		WordCount:         150,
		TitleLength:       25,
		HasHeadings:       true,
		HasMetadata:       true,
		ContentDensity:    0.75,
		EstimatedReadTime: 2,
	}

	t.Run("valid_result_structure", func(t *testing.T) {
		assert.True(t, result.IsValid)
		assert.Equal(t, 0.85, result.QualityScore)
		assert.Empty(t, result.ValidationIssues)
		assert.Equal(t, 150, result.WordCount)
		assert.Equal(t, 25, result.TitleLength)
		assert.True(t, result.HasHeadings)
		assert.True(t, result.HasMetadata)
		assert.Equal(t, 0.75, result.ContentDensity)
		assert.Equal(t, 2, result.EstimatedReadTime)
	})
}

func TestContentValidationPolicyEvaluator(t *testing.T) {
	evaluator := NewContentValidationPolicyEvaluator()
	require.NotNil(t, evaluator)

	rules := ContentValidationRules{
		MinWordCount:    10,
		MaxWordCount:    1000,
		MinTitleLength:  5,
		MaxTitleLength:  100,
		RequireHeadings: false,
		MaxHTMLTagRatio: 0.5,
		RequireMetadata: false,
	}

	t.Run("validate_word_count", func(t *testing.T) {
		// Valid word count
		assert.True(t, evaluator.ValidateWordCount(50, rules))
		assert.True(t, evaluator.ValidateWordCount(10, rules))   // Minimum boundary
		assert.True(t, evaluator.ValidateWordCount(1000, rules)) // Maximum boundary

		// Invalid word count
		assert.False(t, evaluator.ValidateWordCount(5, rules))    // Below minimum
		assert.False(t, evaluator.ValidateWordCount(1500, rules)) // Above maximum
	})

	t.Run("validate_title_length", func(t *testing.T) {
		// Valid title lengths
		assert.True(t, evaluator.ValidateTitleLength("Valid Title", rules))
		assert.True(t, evaluator.ValidateTitleLength("Short", rules)) // Minimum boundary

		// Invalid title lengths
		assert.False(t, evaluator.ValidateTitleLength("Hi", rules)) // Below minimum
		longTitle := make([]byte, 150)
		for i := range longTitle {
			longTitle[i] = 'A'
		}
		assert.False(t, evaluator.ValidateTitleLength(string(longTitle), rules)) // Above maximum
	})

	t.Run("validate_content_structure", func(t *testing.T) {
		// Valid content with headings
		contentWithHeadings := "<h1>Title</h1><p>This is some content with proper structure.</p>"
		assert.True(t, evaluator.ValidateContentStructure(contentWithHeadings, rules))

		// Valid content without headings (when not required)
		contentWithoutHeadings := "<p>This is content without headings but that's okay.</p>"
		assert.True(t, evaluator.ValidateContentStructure(contentWithoutHeadings, rules))

		// Test with headings required
		rulesRequireHeadings := rules
		rulesRequireHeadings.RequireHeadings = true
		assert.True(t, evaluator.ValidateContentStructure(contentWithHeadings, rulesRequireHeadings))
		assert.False(t, evaluator.ValidateContentStructure(contentWithoutHeadings, rulesRequireHeadings))
	})

	t.Run("validate_content_density", func(t *testing.T) {
		// Good content density
		lowTagContent := "<p>This is content with a low ratio of HTML tags to actual text content.</p>"
		assert.True(t, evaluator.ValidateContentDensity(lowTagContent, rules))

		// Poor content density (too many tags)
		highTagContent := "<span><div><p><b><i><u>Text</u></i></b></p></div></span>"
		assert.False(t, evaluator.ValidateContentDensity(highTagContent, rules))

		// Edge case: no content
		assert.False(t, evaluator.ValidateContentDensity("", rules))
	})
}

func TestContentValidationBusinessPolicy(t *testing.T) {
	policy := ContentValidationBusinessPolicy{
		ValidationRules: ContentValidationRules{
			MinWordCount:        15,
			MaxWordCount:        2000,
			MinTitleLength:      8,
			MaxTitleLength:      120,
			RequireHeadings:     true,
			AllowEmptyHeadings:  false,
			MaxHTMLTagRatio:     0.4,
			RequireMetadata:     true,
			AllowedContentTypes: []string{"article", "blog"},
		},
		QualityThreshold: 0.7,
		StrictMode:       false,
	}

	t.Run("policy_configuration", func(t *testing.T) {
		assert.Equal(t, 15, policy.ValidationRules.MinWordCount)
		assert.Equal(t, 0.7, policy.QualityThreshold)
		assert.False(t, policy.StrictMode)
		assert.True(t, policy.ValidationRules.RequireHeadings)
		assert.True(t, policy.ValidationRules.RequireMetadata)
	})
}

func TestContentValidationDecisionMaker(t *testing.T) {
	decisionMaker := NewContentValidationDecisionMaker()
	require.NotNil(t, decisionMaker)

	policy := ContentValidationBusinessPolicy{
		ValidationRules: ContentValidationRules{
			MinWordCount:    10,
			MaxWordCount:    500,
			MinTitleLength:  5,
			MaxTitleLength:  80,
			RequireHeadings: false,
			MaxHTMLTagRatio: 0.5,
			RequireMetadata: false,
		},
		QualityThreshold: 0.6,
		StrictMode:       false,
	}

	t.Run("should_validate_content", func(t *testing.T) {
		decision := decisionMaker.ShouldValidateContent("https://example.com/article", policy)
		assert.True(t, decision.ShouldValidate)
		assert.Equal(t, "content_validation", decision.ValidationType)
		assert.Equal(t, "https://example.com/article", decision.URL)
	})

	t.Run("get_validation_rules", func(t *testing.T) {
		rules := decisionMaker.GetValidationRules("https://example.com/article", policy)
		assert.Equal(t, policy.ValidationRules, rules.ValidationRules)
		assert.Equal(t, "https://example.com/article", rules.URL)
	})

	t.Run("create_validation_context", func(t *testing.T) {
		context := decisionMaker.CreateValidationContext("https://example.com/test", policy)

		assert.Equal(t, "https://example.com/test", context.URL)
		assert.Equal(t, policy, context.Policy)
		assert.NotZero(t, context.CreatedAt)
		assert.Equal(t, "pending", context.Status)
	})

	t.Run("batch_validation_decisions", func(t *testing.T) {
		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2",
			"https://example.com/page3",
		}

		decisions := decisionMaker.BatchShouldValidate(urls, policy)

		assert.Len(t, decisions, 3)
		for i, decision := range decisions {
			assert.True(t, decision.ShouldValidate)
			assert.Equal(t, urls[i], decision.URL)
			assert.Equal(t, "content_validation", decision.ValidationType)
		}
	})
}

func TestContentQualityAnalyzer(t *testing.T) {
	analyzer := NewContentQualityAnalyzer()
	require.NotNil(t, analyzer)

	t.Run("analyze_quality_score", func(t *testing.T) {
		// High quality content
		goodContent := "<h1>Great Article</h1><p>This is a well-structured article with good content density and proper formatting.</p><h2>Section</h2><p>More quality content here.</p>"
		score := analyzer.AnalyzeQualityScore(goodContent, "Great Article", 25)
		assert.True(t, score > 0.7, "Expected high quality score, got %f", score)

		// Low quality content
		poorContent := "<div><span><p>Poor</p></span></div>"
		score = analyzer.AnalyzeQualityScore(poorContent, "X", 1)
		assert.True(t, score < 0.4, "Expected low quality score, got %f", score)
	})

	t.Run("calculate_content_density", func(t *testing.T) {
		// Good density content
		goodDensityContent := "<p>This paragraph has a good ratio of actual text content to HTML markup tags.</p>"
		density := analyzer.CalculateContentDensity(goodDensityContent)
		assert.True(t, density > 0.5, "Expected high content density, got %f", density)

		// Poor density content
		poorDensityContent := "<div><span><b><i><u><em>Text</em></u></i></b></span></div>"
		density = analyzer.CalculateContentDensity(poorDensityContent)
		assert.True(t, density < 0.3, "Expected low content density, got %f", density)
	})

	t.Run("estimate_reading_time", func(t *testing.T) {
		// Test reading time calculation (assuming 200 words per minute)
		readingTime := analyzer.EstimateReadingTime(400) // 400 words
		assert.Equal(t, 2, readingTime)                  // Should be 2 minutes

		readingTime = analyzer.EstimateReadingTime(100) // 100 words
		assert.Equal(t, 1, readingTime)                 // Should be 1 minute (minimum)

		readingTime = analyzer.EstimateReadingTime(50) // 50 words
		assert.Equal(t, 1, readingTime)                // Should be 1 minute (minimum)
	})
}
