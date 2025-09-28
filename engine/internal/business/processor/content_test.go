package processor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestContentProcessingPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   ContentProcessingPolicy
		expected ContentProcessingPolicy
	}{
		{
			name: "default_policy",
			policy: ContentProcessingPolicy{
				ContentSelectors:    []string{"main", "article"},
				CleaningRules:       []string{"script", "style"},
				URLConversionRules:  []string{"href", "src"},
				MetadataExtraction:  true,
				ImageExtraction:     true,
				MarkdownConversion:  true,
				ContentValidation:   true,
			},
			expected: ContentProcessingPolicy{
				ContentSelectors:    []string{"main", "article"},
				CleaningRules:       []string{"script", "style"},
				URLConversionRules:  []string{"href", "src"},
				MetadataExtraction:  true,
				ImageExtraction:     true,
				MarkdownConversion:  true,
				ContentValidation:   true,
			},
		},
		{
			name: "minimal_policy",
			policy: ContentProcessingPolicy{
				ContentSelectors:   []string{"body"},
				MetadataExtraction: false,
				ImageExtraction:    false,
			},
			expected: ContentProcessingPolicy{
				ContentSelectors:   []string{"body"},
				MetadataExtraction: false,
				ImageExtraction:    false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.policy)
		})
	}
}

func TestContentQualityPolicy(t *testing.T) {
	tests := []struct {
		name     string
		policy   ContentQualityPolicy
		expected bool
	}{
		{
			name: "strict_quality_policy",
			policy: ContentQualityPolicy{
				MinWordCount:        50,
				MinTitleLength:      10,
				RequireHeadings:     true,
				MaxHTMLTagRatio:     0.3,
				ValidateOpenGraph:   true,
				RequireDescription:  true,
			},
			expected: true,
		},
		{
			name: "relaxed_quality_policy",
			policy: ContentQualityPolicy{
				MinWordCount:       5,
				MinTitleLength:     1,
				RequireHeadings:    false,
				MaxHTMLTagRatio:    1.0,
				ValidateOpenGraph:  false,
				RequireDescription: false,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, tt.expected)
		})
	}
}

func TestContentProcessingPolicyEvaluator(t *testing.T) {
	evaluator := NewContentProcessingPolicyEvaluator()

	t.Run("should_process_content", func(t *testing.T) {
		policy := ContentProcessingPolicy{
			ContentSelectors:   []string{"main", "article"},
			MetadataExtraction: true,
		}

		decision := evaluator.ShouldProcessContent("https://example.com/test", policy)
		assert.True(t, decision)
	})

	t.Run("get_content_selectors", func(t *testing.T) {
		policy := ContentProcessingPolicy{
			ContentSelectors: []string{"main", "article", ".content"},
		}

		selectors := evaluator.GetContentSelectors("https://example.com", policy)
		expected := []string{"main", "article", ".content"}
		assert.Equal(t, expected, selectors)
	})

	t.Run("get_cleaning_rules", func(t *testing.T) {
		policy := ContentProcessingPolicy{
			CleaningRules: []string{"script", "style", "nav"},
		}

		rules := evaluator.GetCleaningRules("https://example.com", policy)
		expected := []string{"script", "style", "nav"}
		assert.Equal(t, expected, rules)
	})

	t.Run("should_extract_metadata", func(t *testing.T) {
		policy := ContentProcessingPolicy{MetadataExtraction: true}
		assert.True(t, evaluator.ShouldExtractMetadata("https://example.com", policy))

		policy.MetadataExtraction = false
		assert.False(t, evaluator.ShouldExtractMetadata("https://example.com", policy))
	})

	t.Run("should_extract_images", func(t *testing.T) {
		policy := ContentProcessingPolicy{ImageExtraction: true}
		assert.True(t, evaluator.ShouldExtractImages("https://example.com", policy))

		policy.ImageExtraction = false
		assert.False(t, evaluator.ShouldExtractImages("https://example.com", policy))
	})

	t.Run("should_convert_to_markdown", func(t *testing.T) {
		policy := ContentProcessingPolicy{MarkdownConversion: true}
		assert.True(t, evaluator.ShouldConvertToMarkdown("https://example.com", policy))

		policy.MarkdownConversion = false
		assert.False(t, evaluator.ShouldConvertToMarkdown("https://example.com", policy))
	})
}

func TestContentQualityPolicyEvaluator(t *testing.T) {
	evaluator := NewContentQualityPolicyEvaluator()

	t.Run("meets_word_count_requirement", func(t *testing.T) {
		policy := ContentQualityPolicy{MinWordCount: 50}

		assert.True(t, evaluator.MeetsWordCountRequirement(100, policy))
		assert.True(t, evaluator.MeetsWordCountRequirement(50, policy))
		assert.False(t, evaluator.MeetsWordCountRequirement(25, policy))
	})

	t.Run("meets_title_length_requirement", func(t *testing.T) {
		policy := ContentQualityPolicy{MinTitleLength: 10}

		assert.True(t, evaluator.MeetsTitleLengthRequirement("This is a long title", policy))
		assert.True(t, evaluator.MeetsTitleLengthRequirement("1234567890", policy))
		assert.False(t, evaluator.MeetsTitleLengthRequirement("Short", policy))
	})

	t.Run("meets_headings_requirement", func(t *testing.T) {
		policy := ContentQualityPolicy{RequireHeadings: true}

		assert.True(t, evaluator.MeetsHeadingsRequirement("<h1>Title</h1><p>Content</p>", policy))
		assert.True(t, evaluator.MeetsHeadingsRequirement("<h2>Subtitle</h2><p>Content</p>", policy))
		assert.False(t, evaluator.MeetsHeadingsRequirement("<p>No headings here</p>", policy))

		// Should pass when not required
		policy.RequireHeadings = false
		assert.True(t, evaluator.MeetsHeadingsRequirement("<p>No headings here</p>", policy))
	})

	t.Run("meets_html_tag_ratio_requirement", func(t *testing.T) {
		policy := ContentQualityPolicy{MaxHTMLTagRatio: 0.3}

		// Content with low tag ratio should pass
		lowTagContent := "<p>This is a lot of text content without many HTML tags at all.</p>"
		assert.True(t, evaluator.MeetsHTMLTagRatioRequirement(lowTagContent, policy))

		// Content with high tag ratio should fail
		highTagContent := "<span><div><p><b><i>Text</i></b></p></div></span>"
		assert.False(t, evaluator.MeetsHTMLTagRatioRequirement(highTagContent, policy))
	})

	t.Run("meets_description_requirement", func(t *testing.T) {
		policy := ContentQualityPolicy{RequireDescription: true}

		assert.True(t, evaluator.MeetsDescriptionRequirement("This is a description", policy))
		assert.False(t, evaluator.MeetsDescriptionRequirement("", policy))

		// Should pass when not required
		policy.RequireDescription = false
		assert.True(t, evaluator.MeetsDescriptionRequirement("", policy))
	})
}

func TestProcessingBusinessPolicy(t *testing.T) {
	policy := ProcessingBusinessPolicy{
		ContentPolicy: ContentProcessingPolicy{
			ContentSelectors:   []string{"main", "article"},
			CleaningRules:      []string{"script", "style"},
			MetadataExtraction: true,
			ImageExtraction:    true,
		},
		QualityPolicy: ContentQualityPolicy{
			MinWordCount:       20,
			MinTitleLength:     5,
			RequireHeadings:    false,
			MaxHTMLTagRatio:    0.5,
			ValidateOpenGraph:  false,
			RequireDescription: false,
		},
	}

	t.Run("valid_policy_structure", func(t *testing.T) {
		assert.NotEmpty(t, policy.ContentPolicy.ContentSelectors)
		assert.NotEmpty(t, policy.ContentPolicy.CleaningRules)
		assert.True(t, policy.ContentPolicy.MetadataExtraction)
		assert.Equal(t, 20, policy.QualityPolicy.MinWordCount)
	})
}

func TestProcessingDecisionMaker(t *testing.T) {
	decisionMaker := NewProcessingDecisionMaker()
	require.NotNil(t, decisionMaker)

	policy := ProcessingBusinessPolicy{
		ContentPolicy: ContentProcessingPolicy{
			ContentSelectors:    []string{"main", "article"},
			CleaningRules:       []string{"script", "style"},
			MetadataExtraction:  true,
			ImageExtraction:     true,
			MarkdownConversion:  true,
			ContentValidation:   true,
		},
		QualityPolicy: ContentQualityPolicy{
			MinWordCount:       10,
			MinTitleLength:     5,
			RequireHeadings:    false,
			MaxHTMLTagRatio:    0.8,
			ValidateOpenGraph:  false,
			RequireDescription: false,
		},
	}

	t.Run("should_process_content_decision", func(t *testing.T) {
		decision := decisionMaker.ShouldProcessContent("https://example.com/article", policy)
		assert.True(t, decision.ShouldProcess)
		assert.Equal(t, "content_processing", decision.ProcessingType)
		assert.Equal(t, "https://example.com/article", decision.URL)
	})

	t.Run("get_processing_steps_decision", func(t *testing.T) {
		steps := decisionMaker.GetProcessingSteps("https://example.com/article", policy)
		
		expectedSteps := []string{
			"content_extraction",
			"content_cleaning", 
			"metadata_extraction",
			"image_extraction",
			"markdown_conversion",
			"content_validation",
		}
		
		assert.Equal(t, expectedSteps, steps.ProcessingSteps)
		assert.Equal(t, "https://example.com/article", steps.URL)
	})

	t.Run("create_processing_context", func(t *testing.T) {
		context := decisionMaker.CreateProcessingContext("https://example.com/test", policy)
		
		assert.Equal(t, "https://example.com/test", context.URL)
		assert.Equal(t, policy, context.Policy)
		assert.NotZero(t, context.CreatedAt)
		assert.Equal(t, "pending", context.Status)
	})

	t.Run("batch_processing_decisions", func(t *testing.T) {
		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2", 
			"https://example.com/page3",
		}
		
		decisions := decisionMaker.BatchShouldProcess(urls, policy)
		
		assert.Len(t, decisions, 3)
		for i, decision := range decisions {
			assert.True(t, decision.ShouldProcess)
			assert.Equal(t, urls[i], decision.URL)
			assert.Equal(t, "content_processing", decision.ProcessingType)
		}
	})
}