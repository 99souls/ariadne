package output

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOutputProcessingPolicy(t *testing.T) {
	policy := OutputProcessingPolicy{
		OutputFormat:         "json",
		CompressionEnabled:   true,
		CompressionLevel:     6,
		BufferingEnabled:     true,
		BufferSize:          1000,
		FlushInterval:       5 * time.Second,
		RetryEnabled:        true,
		MaxRetries:          3,
		RetryDelay:          100 * time.Millisecond,
		TransformationRules: []string{"remove_empty", "normalize_urls"},
		ValidationEnabled:   true,
	}

	t.Run("valid_policy_structure", func(t *testing.T) {
		assert.Equal(t, "json", policy.OutputFormat)
		assert.True(t, policy.CompressionEnabled)
		assert.Equal(t, 6, policy.CompressionLevel)
		assert.True(t, policy.BufferingEnabled)
		assert.Equal(t, 1000, policy.BufferSize)
		assert.Equal(t, 5*time.Second, policy.FlushInterval)
		assert.True(t, policy.RetryEnabled)
		assert.Equal(t, 3, policy.MaxRetries)
		assert.Len(t, policy.TransformationRules, 2)
	})
}

func TestOutputQualityPolicy(t *testing.T) {
	policy := OutputQualityPolicy{
		RequireValidation:     true,
		MaxOutputSize:         10485760, // 10MB
		MinContentLength:      100,
		AllowedFormats:        []string{"json", "markdown", "html"},
		RequireMetadata:       true,
		ValidateSchema:        true,
		ContentIntegrityCheck: true,
	}

	t.Run("quality_policy_configuration", func(t *testing.T) {
		assert.True(t, policy.RequireValidation)
		assert.Equal(t, 10485760, policy.MaxOutputSize)
		assert.Equal(t, 100, policy.MinContentLength)
		assert.Len(t, policy.AllowedFormats, 3)
		assert.True(t, policy.RequireMetadata)
		assert.True(t, policy.ValidateSchema)
		assert.True(t, policy.ContentIntegrityCheck)
	})
}

func TestOutputProcessingPolicyEvaluator(t *testing.T) {
	evaluator := NewOutputProcessingPolicyEvaluator()
	require.NotNil(t, evaluator)

	policy := OutputProcessingPolicy{
		OutputFormat:       "json",
		CompressionEnabled: true,
		BufferingEnabled:   true,
		RetryEnabled:       true,
		ValidationEnabled:  true,
	}

	t.Run("should_process_output", func(t *testing.T) {
		decision := evaluator.ShouldProcessOutput("https://example.com/test", policy)
		assert.True(t, decision)
	})

	t.Run("get_output_format", func(t *testing.T) {
		format := evaluator.GetOutputFormat("https://example.com", policy)
		assert.Equal(t, "json", format)
	})

	t.Run("should_enable_compression", func(t *testing.T) {
		assert.True(t, evaluator.ShouldEnableCompression("https://example.com", policy))

		policy.CompressionEnabled = false
		assert.False(t, evaluator.ShouldEnableCompression("https://example.com", policy))
	})

	t.Run("should_enable_buffering", func(t *testing.T) {
		assert.True(t, evaluator.ShouldEnableBuffering("https://example.com", policy))

		policy.BufferingEnabled = false
		assert.False(t, evaluator.ShouldEnableBuffering("https://example.com", policy))
	})

	t.Run("should_enable_retry", func(t *testing.T) {
		assert.True(t, evaluator.ShouldEnableRetry("https://example.com", policy))

		policy.RetryEnabled = false
		assert.False(t, evaluator.ShouldEnableRetry("https://example.com", policy))
	})

	t.Run("get_transformation_rules", func(t *testing.T) {
		policy.TransformationRules = []string{"rule1", "rule2", "rule3"}
		rules := evaluator.GetTransformationRules("https://example.com", policy)
		expected := []string{"rule1", "rule2", "rule3"}
		assert.Equal(t, expected, rules)
	})
}

func TestOutputQualityPolicyEvaluator(t *testing.T) {
	evaluator := NewOutputQualityPolicyEvaluator()
	require.NotNil(t, evaluator)

	policy := OutputQualityPolicy{
		RequireValidation:     true,
		MaxOutputSize:         1000,
		MinContentLength:      10,
		AllowedFormats:        []string{"json", "markdown"},
		RequireMetadata:       false,
		ValidateSchema:        true,
		ContentIntegrityCheck: true,
	}

	t.Run("meets_size_requirements", func(t *testing.T) {
		assert.True(t, evaluator.MeetsSizeRequirements(500, policy))  // Within limits
		assert.True(t, evaluator.MeetsSizeRequirements(1000, policy)) // At max limit
		assert.False(t, evaluator.MeetsSizeRequirements(1500, policy)) // Above limit
	})

	t.Run("meets_content_length_requirements", func(t *testing.T) {
		assert.True(t, evaluator.MeetsContentLengthRequirements(50, policy))  // Above minimum
		assert.True(t, evaluator.MeetsContentLengthRequirements(10, policy))  // At minimum
		assert.False(t, evaluator.MeetsContentLengthRequirements(5, policy))  // Below minimum
	})

	t.Run("is_format_allowed", func(t *testing.T) {
		assert.True(t, evaluator.IsFormatAllowed("json", policy))
		assert.True(t, evaluator.IsFormatAllowed("markdown", policy))
		assert.False(t, evaluator.IsFormatAllowed("xml", policy))
		assert.False(t, evaluator.IsFormatAllowed("csv", policy))
	})

	t.Run("meets_metadata_requirements", func(t *testing.T) {
		// When metadata not required, should always pass
		assert.True(t, evaluator.MeetsMetadataRequirements(true, policy))
		assert.True(t, evaluator.MeetsMetadataRequirements(false, policy))

		// When metadata is required
		policy.RequireMetadata = true
		assert.True(t, evaluator.MeetsMetadataRequirements(true, policy))
		assert.False(t, evaluator.MeetsMetadataRequirements(false, policy))
	})
}

func TestOutputBusinessPolicy(t *testing.T) {
	policy := OutputBusinessPolicy{
		ProcessingPolicy: OutputProcessingPolicy{
			OutputFormat:         "markdown",
			CompressionEnabled:   false,
			BufferingEnabled:     true,
			BufferSize:          500,
			RetryEnabled:        true,
			MaxRetries:          5,
			ValidationEnabled:   true,
		},
		QualityPolicy: OutputQualityPolicy{
			RequireValidation:     true,
			MaxOutputSize:         5242880, // 5MB
			MinContentLength:      50,
			AllowedFormats:        []string{"markdown", "html"},
			RequireMetadata:       false,
			ValidateSchema:        false,
			ContentIntegrityCheck: true,
		},
	}

	t.Run("combined_policy_structure", func(t *testing.T) {
		assert.Equal(t, "markdown", policy.ProcessingPolicy.OutputFormat)
		assert.False(t, policy.ProcessingPolicy.CompressionEnabled)
		assert.Equal(t, 500, policy.ProcessingPolicy.BufferSize)
		assert.Equal(t, 5, policy.ProcessingPolicy.MaxRetries)
		
		assert.True(t, policy.QualityPolicy.RequireValidation)
		assert.Equal(t, 5242880, policy.QualityPolicy.MaxOutputSize)
		assert.Equal(t, 50, policy.QualityPolicy.MinContentLength)
		assert.Len(t, policy.QualityPolicy.AllowedFormats, 2)
	})
}

func TestOutputDecisionMaker(t *testing.T) {
	decisionMaker := NewOutputDecisionMaker()
	require.NotNil(t, decisionMaker)

	policy := OutputBusinessPolicy{
		ProcessingPolicy: OutputProcessingPolicy{
			OutputFormat:         "json",
			CompressionEnabled:   true,
			BufferingEnabled:     true,
			RetryEnabled:        true,
			ValidationEnabled:   true,
			TransformationRules: []string{"normalize", "validate"},
		},
		QualityPolicy: OutputQualityPolicy{
			RequireValidation:     true,
			MaxOutputSize:         1048576, // 1MB
			MinContentLength:      20,
			AllowedFormats:        []string{"json", "markdown"},
			RequireMetadata:       false,
			ValidateSchema:        true,
			ContentIntegrityCheck: true,
		},
	}

	t.Run("should_process_output_decision", func(t *testing.T) {
		decision := decisionMaker.ShouldProcessOutput("https://example.com/article", policy)
		assert.True(t, decision.ShouldProcess)
		assert.Equal(t, "output_processing", decision.ProcessingType)
		assert.Equal(t, "https://example.com/article", decision.URL)
	})

	t.Run("get_output_configuration_decision", func(t *testing.T) {
		config := decisionMaker.GetOutputConfiguration("https://example.com/article", policy)
		
		assert.Equal(t, "https://example.com/article", config.URL)
		assert.Equal(t, "json", config.OutputFormat)
		assert.True(t, config.CompressionEnabled)
		assert.True(t, config.BufferingEnabled)
		assert.True(t, config.RetryEnabled)
		assert.Len(t, config.TransformationRules, 2)
	})

	t.Run("create_output_context", func(t *testing.T) {
		context := decisionMaker.CreateOutputContext("https://example.com/test", policy)
		
		assert.Equal(t, "https://example.com/test", context.URL)
		assert.Equal(t, policy, context.Policy)
		assert.NotZero(t, context.CreatedAt)
		assert.Equal(t, "pending", context.Status)
	})

	t.Run("batch_output_decisions", func(t *testing.T) {
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
			assert.Equal(t, "output_processing", decision.ProcessingType)
		}
	})
}

func TestOutputRoutingDecisionMaker(t *testing.T) {
	routingMaker := NewOutputRoutingDecisionMaker()
	require.NotNil(t, routingMaker)

	routingRules := OutputRoutingRules{
		DefaultSink: "stdout",
		Rules: []RoutingRule{
			{Pattern: "*.json", SinkName: "json-sink"},
			{Pattern: "*.md", SinkName: "markdown-sink"},
		},
		FallbackSink: "error-sink",
	}

	t.Run("get_sink_destination", func(t *testing.T) {
		// Test pattern matching
		destination := routingMaker.GetSinkDestination("file.json", routingRules)
		assert.Equal(t, "json-sink", destination.SinkName)
		assert.Equal(t, "pattern_match", destination.Reason)

		destination = routingMaker.GetSinkDestination("document.md", routingRules)
		assert.Equal(t, "markdown-sink", destination.SinkName)

		// Test default fallback
		destination = routingMaker.GetSinkDestination("unknown.txt", routingRules)
		assert.Equal(t, "stdout", destination.SinkName)
		assert.Equal(t, "default_route", destination.Reason)
	})

	t.Run("should_route_to_multiple_sinks", func(t *testing.T) {
		decision := routingMaker.ShouldRouteToMultipleSinks("https://example.com/test", routingRules)
		assert.Equal(t, "https://example.com/test", decision.URL)
		// Default behavior is single sink routing
		assert.False(t, decision.ShouldUseMultipleSinks)
	})
}