package strategies

import (
	"context"
	"testing"
	"time"

	"ariadne/packages/engine/business/policies"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrategyBusinessPolicyIntegration(t *testing.T) {
	t.Run("end_to_end_strategy_composition", func(t *testing.T) {
		// Create comprehensive business policies
		businessPolicies := &policies.BusinessPolicies{
			CrawlingPolicy: &policies.CrawlingBusinessPolicy{
				SiteRules: map[string]*policies.SitePolicy{
					"news.com": {
						AllowedDomains: []string{"news.com", "cdn.news.com"},
						MaxDepth:       5,
						Delay:          200 * time.Millisecond,
						Selectors:      []string{".article", ".story", ".news-content"},
					},
					"blog.org": {
						AllowedDomains: []string{"blog.org"},
						MaxDepth:       3,
						Delay:          500 * time.Millisecond,
						Selectors:      []string{".post", ".content"},
					},
				},
				LinkRules: &policies.LinkFollowingPolicy{
					FollowExternalLinks: true,
					MaxDepth:            10,
				},
				ContentRules: &policies.ContentSelectionPolicy{
					DefaultSelectors: []string{".main", ".content", "article"},
					SiteSelectors: map[string][]string{
						"news.com": {".news-body", ".article-text"},
						"blog.org": {".post-content", ".entry-text"},
					},
				},
				RateRules: &policies.RateLimitingPolicy{
					DefaultDelay: 1 * time.Second,
					SiteDelays: map[string]time.Duration{
						"news.com": 200 * time.Millisecond,
						"blog.org": 500 * time.Millisecond,
					},
				},
			},
			ProcessingPolicy: &policies.ProcessingBusinessPolicy{
				ContentExtractionRules: []string{"text", "images", "links", "metadata"},
				QualityThreshold:       0.75,
				ProcessingSteps:        []string{"extract", "clean", "enhance", "validate"},
			},
			OutputPolicy: &policies.OutputBusinessPolicy{
				DefaultFormat: "json",
				Compression:   true,
				RoutingRules: map[string]string{
					"news":         "news-sink",
					"blog":         "blog-sink",
					"high-quality": "premium-sink",
					"default":      "main-sink",
				},
				QualityGates: []string{"word-count", "content-quality", "metadata-completeness"},
			},
			GlobalPolicy: &policies.GlobalBusinessPolicy{
				MaxConcurrency: 15,
				Timeout:        45 * time.Second,
				RetryPolicy: &policies.RetryPolicy{
					MaxRetries:    3,
					InitialDelay:  2 * time.Second,
					BackoffFactor: 2.5,
				},
				LoggingLevel: "info",
			},
		}

		// Create strategy composer and compose strategies
		composer := NewStrategyComposer()
		composedStrategies, err := composer.ComposeStrategies(businessPolicies)
		require.NoError(t, err)
		require.NotNil(t, composedStrategies)

		// Validate the complete composed strategy
		err = composer.ValidateComposition(composedStrategies)
		assert.NoError(t, err)

		// Verify fetching strategy composition
		fetchingStrategy := composedStrategies.FetchingStrategy
		assert.Equal(t, 15, fetchingStrategy.Concurrency)
		assert.Equal(t, 45*time.Second, fetchingStrategy.Timeout)
		assert.True(t, fetchingStrategy.RetryEnabled)
		assert.Equal(t, 3, fetchingStrategy.RetryConfig.MaxRetries)
		assert.Equal(t, 2*time.Second, fetchingStrategy.RetryConfig.InitialDelay)
		assert.Equal(t, 2.5, fetchingStrategy.RetryConfig.BackoffFactor)

		// High concurrency should trigger parallel and adaptive strategies
		assert.Contains(t, fetchingStrategy.Strategies, ParallelFetching)
		assert.Contains(t, fetchingStrategy.Strategies, FallbackFetching)

		// Verify processing strategy composition
		processingStrategy := composedStrategies.ProcessingStrategy
		assert.Equal(t, 0.75, processingStrategy.QualityThreshold)
		assert.Equal(t, []string{"extract", "clean", "enhance", "validate"}, processingStrategy.Steps)

		// Complex processing with high concurrency should use parallel processing
		assert.Contains(t, processingStrategy.Strategies, ParallelProcessing)
		assert.Contains(t, processingStrategy.Strategies, ConditionalProcessing)
		assert.True(t, processingStrategy.ParallelSteps)
		assert.Equal(t, 7, processingStrategy.Concurrency) // Half of fetching concurrency

		// Verify conditional rules were created
		assert.NotEmpty(t, processingStrategy.ConditionalRules)
		assert.Contains(t, processingStrategy.ConditionalRules, "high-quality")
		assert.Contains(t, processingStrategy.ConditionalRules, "low-quality")

		// Verify output strategy composition
		outputStrategy := composedStrategies.OutputStrategy
		assert.Equal(t, "json", outputStrategy.DefaultFormat)
		assert.True(t, outputStrategy.CompressionEnabled)
		assert.Equal(t, map[string]string{
			"news":         "news-sink",
			"blog":         "blog-sink",
			"high-quality": "premium-sink",
			"default":      "main-sink",
		}, outputStrategy.RoutingRules)

		// Multiple routing rules should use conditional routing
		assert.Contains(t, outputStrategy.Strategies, ConditionalRouting)

		// High concurrency should add multi-sink support
		assert.Contains(t, outputStrategy.Strategies, MultiSinkOutput)
		assert.NotNil(t, outputStrategy.MultiSinkConfig)
		assert.True(t, outputStrategy.MultiSinkConfig.Failover)

		// Verify metadata
		assert.NotZero(t, composedStrategies.Metadata.ComposedAt)
		assert.Equal(t, "1.0.0", composedStrategies.Metadata.Version)
		assert.Equal(t, "balanced", composedStrategies.Metadata.OptimizedFor)
	})

	t.Run("strategy_optimization_with_business_context", func(t *testing.T) {
		// Create business policies optimized for speed
		speedOptimizedPolicies := &policies.BusinessPolicies{
			CrawlingPolicy: &policies.CrawlingBusinessPolicy{
				SiteRules: map[string]*policies.SitePolicy{
					"fast-site.com": {
						AllowedDomains: []string{"fast-site.com"},
						MaxDepth:       2,                     // Shallow depth for speed
						Delay:          50 * time.Millisecond, // Low delay
					},
				},
				LinkRules: &policies.LinkFollowingPolicy{
					FollowExternalLinks: false, // Skip externals for speed
					MaxDepth:            3,     // Low depth
				},
			},
			ProcessingPolicy: &policies.ProcessingBusinessPolicy{
				QualityThreshold: 0.3,                 // Low threshold for speed
				ProcessingSteps:  []string{"extract"}, // Minimal processing
			},
			OutputPolicy: &policies.OutputBusinessPolicy{
				DefaultFormat: "text", // Simple format
				Compression:   false,  // No compression for speed
				RoutingRules:  map[string]string{"default": "fast-sink"},
			},
			GlobalPolicy: &policies.GlobalBusinessPolicy{
				MaxConcurrency: 50,               // High concurrency for speed
				Timeout:        10 * time.Second, // Short timeout
				RetryPolicy: &policies.RetryPolicy{
					MaxRetries:    1, // Minimal retries
					InitialDelay:  100 * time.Millisecond,
					BackoffFactor: 1.0, // No backoff
				},
			},
		}

		composer := NewStrategyComposer()
		composedStrategies, err := composer.ComposeStrategies(speedOptimizedPolicies)
		require.NoError(t, err)

		// Optimize for performance
		optimizedStrategies, err := composer.OptimizeComposition(composedStrategies)
		require.NoError(t, err)

		// Verify optimization applied reasonable limits
		assert.True(t, optimizedStrategies.FetchingStrategy.Concurrency <= 50)
		assert.True(t, optimizedStrategies.FetchingStrategy.Timeout >= 5*time.Second)  // Minimum safety timeout
		assert.True(t, optimizedStrategies.ProcessingStrategy.QualityThreshold >= 0.5) // Reasonable minimum

		// Verify optimization metadata
		assert.Equal(t, "performance", optimizedStrategies.Metadata.OptimizedFor)
	})

	t.Run("multi_strategy_execution_planning", func(t *testing.T) {
		// Create balanced business policies
		balancedPolicies := &policies.BusinessPolicies{
			GlobalPolicy: &policies.GlobalBusinessPolicy{
				MaxConcurrency: 8,
				Timeout:        30 * time.Second,
			},
		}

		composer := NewStrategyComposer()
		composedStrategies, err := composer.ComposeStrategies(balancedPolicies)
		require.NoError(t, err)

		// Create executor and test execution planning
		executor := NewStrategyExecutor(composedStrategies)
		urls := []string{
			"https://example.com/page1",
			"https://example.com/page2",
			"https://example.com/page3",
			"https://example.com/page4",
			"https://example.com/page5",
		}

		executionPlan, err := executor.CreateExecutionPlan(context.TODO(), urls)
		require.NoError(t, err)

		// Verify execution plan structure
		assert.Equal(t, urls, executionPlan.FetchingPlan.URLs)
		assert.Equal(t, 8, executionPlan.FetchingPlan.Concurrency)
		assert.Equal(t, 30*time.Second, executionPlan.FetchingPlan.Timeout)
		assert.Equal(t, 1, executionPlan.FetchingPlan.BatchSize) // 5 URLs / 8 concurrency = 1

		assert.NotEmpty(t, executionPlan.ProcessingPlan.Steps)
		assert.Equal(t, "json", executionPlan.OutputPlan.Format)
		assert.NotEmpty(t, executionPlan.OutputPlan.Routing)
	})

	t.Run("performance_monitoring_integration", func(t *testing.T) {
		// Create and test performance monitoring with strategy context
		monitor := NewStrategyPerformanceMonitor()

		// Record performance metrics for different strategy types
		monitor.RecordFetchingPerformance("parallel", 2*time.Second, true)
		monitor.RecordFetchingPerformance("parallel", 3*time.Second, true)
		monitor.RecordFetchingPerformance("sequential", 5*time.Second, false)

		monitor.RecordProcessingPerformance("parallel", 1*time.Second, 0.85)
		monitor.RecordProcessingPerformance("conditional", 800*time.Millisecond, 0.92)

		monitor.RecordOutputPerformance("conditional_routing", 200*time.Millisecond, true)
		monitor.RecordOutputPerformance("multi_sink", 500*time.Millisecond, true)

		// Get aggregated metrics
		metrics := monitor.GetMetrics()
		assert.NotEmpty(t, metrics.FetchingMetrics)
		assert.NotEmpty(t, metrics.ProcessingMetrics)
		assert.NotEmpty(t, metrics.OutputMetrics)

		// Verify specific metrics exist
		assert.Contains(t, metrics.FetchingMetrics, "parallel")
		assert.Contains(t, metrics.FetchingMetrics, "sequential")
		assert.Contains(t, metrics.ProcessingMetrics, "parallel")
		assert.Contains(t, metrics.ProcessingMetrics, "conditional")
		assert.Contains(t, metrics.OutputMetrics, "conditional_routing")
		assert.Contains(t, metrics.OutputMetrics, "multi_sink")

		// Test performance analysis
		recommendations := monitor.AnalyzePerformance()
		assert.NotNil(t, recommendations)

		// Should suggest retry logic for low success rate strategies
		foundRetryRecommendation := false
		for _, suggestion := range recommendations.Suggestions {
			if assert.Contains(t, suggestion, "retry") {
				foundRetryRecommendation = true
				break
			}
		}
		assert.True(t, foundRetryRecommendation, "Expected retry recommendation for low success rate strategy")
	})

	t.Run("adaptive_strategy_with_business_feedback", func(t *testing.T) {
		// Create initial strategy with adaptive configuration
		initialPolicies := &policies.BusinessPolicies{
			GlobalPolicy: &policies.GlobalBusinessPolicy{
				MaxConcurrency: 10,
				Timeout:        20 * time.Second,
			},
		}

		composer := NewStrategyComposer()
		composedStrategies, err := composer.ComposeStrategies(initialPolicies)
		require.NoError(t, err)

		// Force adaptive configuration for testing
		composedStrategies.FetchingStrategy.Strategies = append(composedStrategies.FetchingStrategy.Strategies, AdaptiveFetching)
		composedStrategies.FetchingStrategy.AdaptiveConfig = &AdaptiveConfiguration{
			InitialConcurrency: 5,
			MaxConcurrency:     20,
			AdjustmentFactor:   0.2,
		}

		// Create adaptive manager and simulate business feedback
		manager := NewAdaptiveStrategyManager()

		// Test poor performance feedback (should reduce concurrency)
		poorFeedback := &PerformanceFeedback{
			SuccessRate: 0.85, // Below 0.9 threshold
			CPUUsage:    0.95, // Very high CPU
			MemoryUsage: 0.90, // High memory
		}

		adjustedStrategy, err := manager.AdjustStrategy(composedStrategies, poorFeedback)
		require.NoError(t, err)

		// Should have reduced concurrency due to poor performance
		assert.True(t, adjustedStrategy.FetchingStrategy.Concurrency < composedStrategies.FetchingStrategy.Concurrency)

		// Test excellent performance feedback (should increase concurrency)
		excellentFeedback := &PerformanceFeedback{
			SuccessRate: 0.98, // Excellent success rate
			CPUUsage:    0.50, // Low CPU usage
			MemoryUsage: 0.40, // Low memory usage
		}

		reAdjustedStrategy, err := manager.AdjustStrategy(adjustedStrategy, excellentFeedback)
		require.NoError(t, err)

		// Should have increased concurrency due to excellent performance and available resources
		assert.True(t, reAdjustedStrategy.FetchingStrategy.Concurrency > adjustedStrategy.FetchingStrategy.Concurrency)

		// Should not exceed the maximum configured concurrency
		assert.True(t, reAdjustedStrategy.FetchingStrategy.Concurrency <= composedStrategies.FetchingStrategy.AdaptiveConfig.MaxConcurrency)
	})
}
