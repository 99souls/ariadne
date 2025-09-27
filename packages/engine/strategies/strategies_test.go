package strategies

import (
	"context"
	"testing"
	"time"

	"ariadne/packages/engine/business/policies"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStrategyComposer(t *testing.T) {
	composer := NewStrategyComposer()
	require.NotNil(t, composer)

	t.Run("compose_strategies", func(t *testing.T) {
		businessPolicies := &policies.BusinessPolicies{
			CrawlingPolicy: &policies.CrawlingBusinessPolicy{
				SiteRules: map[string]*policies.SitePolicy{
					"news.com": {
						AllowedDomains: []string{"news.com"},
						MaxDepth:       3,
						Delay:          500 * time.Millisecond,
					},
				},
				LinkRules: &policies.LinkFollowingPolicy{
					FollowExternalLinks: true,
					MaxDepth:            5,
				},
				ContentRules: &policies.ContentSelectionPolicy{
					DefaultSelectors: []string{".content", ".article"},
					SiteSelectors: map[string][]string{
						"news.com": {".news-content", ".story"},
					},
				},
				RateRules: &policies.RateLimitingPolicy{
					DefaultDelay: 1 * time.Second,
					SiteDelays: map[string]time.Duration{
						"news.com": 500 * time.Millisecond,
					},
				},
			},
			ProcessingPolicy: &policies.ProcessingBusinessPolicy{
				ContentExtractionRules: []string{"text", "images", "links"},
				QualityThreshold:       0.7,
				ProcessingSteps:        []string{"extract", "clean", "validate"},
			},
			OutputPolicy: &policies.OutputBusinessPolicy{
				DefaultFormat: "json",
				Compression:   true,
				RoutingRules: map[string]string{
					"news":    "news-sink",
					"default": "main-sink",
				},
				QualityGates: []string{"content-length", "metadata"},
			},
			GlobalPolicy: &policies.GlobalBusinessPolicy{
				MaxConcurrency: 10,
				Timeout:        30 * time.Second,
				RetryPolicy: &policies.RetryPolicy{
					MaxRetries:    3,
					InitialDelay:  1 * time.Second,
					BackoffFactor: 2.0,
				},
				LoggingLevel: "info",
			},
		}

		composedStrategies, err := composer.ComposeStrategies(businessPolicies)
		assert.NoError(t, err)
		assert.NotNil(t, composedStrategies)

		// Verify strategy composition structure
		assert.NotNil(t, composedStrategies.FetchingStrategy)
		assert.NotNil(t, composedStrategies.ProcessingStrategy)
		assert.NotNil(t, composedStrategies.OutputStrategy)

		// Verify fetching strategy configuration
		assert.Equal(t, 10, composedStrategies.FetchingStrategy.Concurrency)
		assert.Equal(t, 30*time.Second, composedStrategies.FetchingStrategy.Timeout)
		assert.True(t, composedStrategies.FetchingStrategy.RetryEnabled)

		// Verify processing strategy configuration
		assert.Equal(t, 0.7, composedStrategies.ProcessingStrategy.QualityThreshold)
		assert.Contains(t, composedStrategies.ProcessingStrategy.Steps, "extract")
		assert.Contains(t, composedStrategies.ProcessingStrategy.Steps, "clean")
		assert.Contains(t, composedStrategies.ProcessingStrategy.Steps, "validate")

		// Verify output strategy configuration
		assert.Equal(t, "json", composedStrategies.OutputStrategy.DefaultFormat)
		assert.True(t, composedStrategies.OutputStrategy.CompressionEnabled)
		assert.NotEmpty(t, composedStrategies.OutputStrategy.RoutingRules)
	})

	t.Run("validate_composition", func(t *testing.T) {
		validComposition := &ComposedStrategies{
			FetchingStrategy: ComposedFetchingStrategy{
				Strategies:   []FetchingStrategyType{ParallelFetching, FallbackFetching},
				Concurrency:  5,
				Timeout:      30 * time.Second,
				RetryEnabled: true,
				RetryConfig: RetryConfiguration{
					MaxRetries:    3,
					InitialDelay:  1 * time.Second,
					BackoffFactor: 2.0,
				},
			},
			ProcessingStrategy: ComposedProcessingStrategy{
				Strategies:       []ProcessingStrategyType{SequentialProcessing, ConditionalProcessing},
				QualityThreshold: 0.8,
				Steps:            []string{"extract", "validate"},
				ParallelSteps:    false,
			},
			OutputStrategy: ComposedOutputStrategy{
				Strategies:         []OutputStrategyType{ConditionalRouting, MultiSinkOutput},
				DefaultFormat:      "json",
				CompressionEnabled: true,
				RoutingRules:       map[string]string{"news": "news-sink"},
			},
		}

		err := composer.ValidateComposition(validComposition)
		assert.NoError(t, err)

		// Test invalid composition (negative concurrency)
		invalidComposition := &ComposedStrategies{
			FetchingStrategy: ComposedFetchingStrategy{
				Concurrency: -1,
			},
		}

		err = composer.ValidateComposition(invalidComposition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "concurrency must be positive")
	})

	t.Run("optimize_composition", func(t *testing.T) {
		originalComposition := &ComposedStrategies{
			FetchingStrategy: ComposedFetchingStrategy{
				Strategies:  []FetchingStrategyType{ParallelFetching},
				Concurrency: 100,             // Very high concurrency
				Timeout:     1 * time.Second, // Very short timeout
			},
			ProcessingStrategy: ComposedProcessingStrategy{
				Strategies:       []ProcessingStrategyType{SequentialProcessing},
				QualityThreshold: 0.1, // Very low threshold
				ParallelSteps:    false,
			},
			OutputStrategy: ComposedOutputStrategy{
				Strategies:         []OutputStrategyType{SimpleOutput},
				CompressionEnabled: false,
			},
		}

		optimizedComposition, err := composer.OptimizeComposition(originalComposition)
		assert.NoError(t, err)
		assert.NotNil(t, optimizedComposition)

		// Verify optimization occurred
		assert.True(t, optimizedComposition.FetchingStrategy.Concurrency <= 50)         // Reasonable concurrency
		assert.True(t, optimizedComposition.FetchingStrategy.Timeout >= 5*time.Second)  // Reasonable timeout
		assert.True(t, optimizedComposition.ProcessingStrategy.QualityThreshold >= 0.5) // Reasonable threshold
	})
}

func TestComposedFetchingStrategy(t *testing.T) {
	t.Run("parallel_fetching_strategy", func(t *testing.T) {
		strategy := ComposedFetchingStrategy{
			Strategies:  []FetchingStrategyType{ParallelFetching},
			Concurrency: 5,
			Timeout:     30 * time.Second,
		}

		assert.Equal(t, 5, strategy.Concurrency)
		assert.Equal(t, 30*time.Second, strategy.Timeout)
		assert.Contains(t, strategy.Strategies, ParallelFetching)
	})

	t.Run("fallback_fetching_strategy", func(t *testing.T) {
		strategy := ComposedFetchingStrategy{
			Strategies:   []FetchingStrategyType{ParallelFetching, FallbackFetching},
			Concurrency:  3,
			RetryEnabled: true,
			RetryConfig: RetryConfiguration{
				MaxRetries:    2,
				InitialDelay:  500 * time.Millisecond,
				BackoffFactor: 1.5,
			},
		}

		assert.Contains(t, strategy.Strategies, ParallelFetching)
		assert.Contains(t, strategy.Strategies, FallbackFetching)
		assert.True(t, strategy.RetryEnabled)
		assert.Equal(t, 2, strategy.RetryConfig.MaxRetries)
	})

	t.Run("adaptive_fetching_strategy", func(t *testing.T) {
		strategy := ComposedFetchingStrategy{
			Strategies:  []FetchingStrategyType{AdaptiveFetching},
			Concurrency: 10,
			AdaptiveConfig: &AdaptiveConfiguration{
				InitialConcurrency: 5,
				MaxConcurrency:     15,
				AdjustmentFactor:   0.1,
				PerformanceWindow:  10,
			},
		}

		assert.Contains(t, strategy.Strategies, AdaptiveFetching)
		assert.NotNil(t, strategy.AdaptiveConfig)
		assert.Equal(t, 5, strategy.AdaptiveConfig.InitialConcurrency)
		assert.Equal(t, 15, strategy.AdaptiveConfig.MaxConcurrency)
	})
}

func TestComposedProcessingStrategy(t *testing.T) {
	t.Run("sequential_processing_strategy", func(t *testing.T) {
		strategy := ComposedProcessingStrategy{
			Strategies:       []ProcessingStrategyType{SequentialProcessing},
			Steps:            []string{"extract", "clean", "transform"},
			QualityThreshold: 0.8,
			ParallelSteps:    false,
		}

		assert.Contains(t, strategy.Strategies, SequentialProcessing)
		assert.Equal(t, 0.8, strategy.QualityThreshold)
		assert.False(t, strategy.ParallelSteps)
		assert.Len(t, strategy.Steps, 3)
	})

	t.Run("parallel_processing_strategy", func(t *testing.T) {
		strategy := ComposedProcessingStrategy{
			Strategies:    []ProcessingStrategyType{ParallelProcessing},
			Steps:         []string{"extract", "validate"},
			ParallelSteps: true,
			Concurrency:   4,
		}

		assert.Contains(t, strategy.Strategies, ParallelProcessing)
		assert.True(t, strategy.ParallelSteps)
		assert.Equal(t, 4, strategy.Concurrency)
	})

	t.Run("conditional_processing_strategy", func(t *testing.T) {
		strategy := ComposedProcessingStrategy{
			Strategies: []ProcessingStrategyType{ConditionalProcessing},
			ConditionalRules: map[string]ProcessingCondition{
				"high-quality": {
					Condition: "quality > 0.8",
					Actions:   []string{"detailed-extract", "metadata-enhance"},
				},
				"low-quality": {
					Condition: "quality < 0.3",
					Actions:   []string{"skip", "log"},
				},
			},
		}

		assert.Contains(t, strategy.Strategies, ConditionalProcessing)
		assert.NotEmpty(t, strategy.ConditionalRules)
		assert.Contains(t, strategy.ConditionalRules, "high-quality")
		assert.Contains(t, strategy.ConditionalRules, "low-quality")
	})
}

func TestComposedOutputStrategy(t *testing.T) {
	t.Run("simple_output_strategy", func(t *testing.T) {
		strategy := ComposedOutputStrategy{
			Strategies:         []OutputStrategyType{SimpleOutput},
			DefaultFormat:      "json",
			CompressionEnabled: false,
		}

		assert.Contains(t, strategy.Strategies, SimpleOutput)
		assert.Equal(t, "json", strategy.DefaultFormat)
		assert.False(t, strategy.CompressionEnabled)
	})

	t.Run("conditional_routing_strategy", func(t *testing.T) {
		strategy := ComposedOutputStrategy{
			Strategies: []OutputStrategyType{ConditionalRouting},
			RoutingRules: map[string]string{
				"news":    "news-sink",
				"blog":    "blog-sink",
				"default": "main-sink",
			},
		}

		assert.Contains(t, strategy.Strategies, ConditionalRouting)
		assert.NotEmpty(t, strategy.RoutingRules)
		assert.Equal(t, "news-sink", strategy.RoutingRules["news"])
		assert.Equal(t, "main-sink", strategy.RoutingRules["default"])
	})

	t.Run("multi_sink_output_strategy", func(t *testing.T) {
		strategy := ComposedOutputStrategy{
			Strategies: []OutputStrategyType{MultiSinkOutput},
			MultiSinkConfig: &MultiSinkConfiguration{
				SinkTypes: []string{"file", "database", "api"},
				FanOut:    true,
				Failover:  true,
			},
		}

		assert.Contains(t, strategy.Strategies, MultiSinkOutput)
		assert.NotNil(t, strategy.MultiSinkConfig)
		assert.True(t, strategy.MultiSinkConfig.FanOut)
		assert.True(t, strategy.MultiSinkConfig.Failover)
		assert.Contains(t, strategy.MultiSinkConfig.SinkTypes, "file")
	})
}

func TestStrategyExecution(t *testing.T) {
	t.Run("execute_composed_strategy", func(t *testing.T) {
		ctx := context.Background()

		composedStrategies := &ComposedStrategies{
			FetchingStrategy: ComposedFetchingStrategy{
				Strategies:  []FetchingStrategyType{ParallelFetching},
				Concurrency: 3,
				Timeout:     10 * time.Second,
			},
			ProcessingStrategy: ComposedProcessingStrategy{
				Strategies:       []ProcessingStrategyType{SequentialProcessing},
				Steps:            []string{"extract", "clean"},
				QualityThreshold: 0.6,
			},
			OutputStrategy: ComposedOutputStrategy{
				Strategies:    []OutputStrategyType{SimpleOutput},
				DefaultFormat: "json",
			},
		}

		executor := NewStrategyExecutor(composedStrategies)
		require.NotNil(t, executor)

		// Test strategy execution planning
		executionPlan, err := executor.CreateExecutionPlan(ctx, []string{"https://example.com"})
		assert.NoError(t, err)
		assert.NotNil(t, executionPlan)
		assert.NotEmpty(t, executionPlan.FetchingPlan.URLs)
		assert.Equal(t, 3, executionPlan.FetchingPlan.Concurrency)
		assert.Len(t, executionPlan.ProcessingPlan.Steps, 2)
	})

	t.Run("strategy_performance_monitoring", func(t *testing.T) {
		monitor := NewStrategyPerformanceMonitor()
		require.NotNil(t, monitor)

		// Record some performance metrics
		monitor.RecordFetchingPerformance("parallel", 5*time.Second, true)
		monitor.RecordProcessingPerformance("sequential", 2*time.Second, 0.8)
		monitor.RecordOutputPerformance("simple", 1*time.Second, true)

		metrics := monitor.GetMetrics()
		assert.NotNil(t, metrics)
		assert.NotEmpty(t, metrics.FetchingMetrics)
		assert.NotEmpty(t, metrics.ProcessingMetrics)
		assert.NotEmpty(t, metrics.OutputMetrics)

		// Test performance analysis
		recommendations := monitor.AnalyzePerformance()
		assert.NotNil(t, recommendations)
		assert.NotEmpty(t, recommendations.Suggestions)
	})
}

func TestStrategyOptimization(t *testing.T) {
	t.Run("performance_based_optimization", func(t *testing.T) {
		optimizer := NewStrategyOptimizer()
		require.NotNil(t, optimizer)

		performanceMetrics := &PerformanceMetrics{
			FetchingMetrics: map[string]*StrategyMetrics{
				"parallel": {
					AverageLatency: 3 * time.Second,
					SuccessRate:    0.95,
					ThroughputRPM:  100,
				},
			},
			ProcessingMetrics: map[string]*StrategyMetrics{
				"sequential": {
					AverageLatency: 1 * time.Second,
					SuccessRate:    0.98,
					ThroughputRPM:  200,
				},
			},
		}

		currentStrategy := &ComposedStrategies{
			FetchingStrategy: ComposedFetchingStrategy{
				Strategies:  []FetchingStrategyType{ParallelFetching},
				Concurrency: 10,
				Timeout:     5 * time.Second,
			},
		}

		optimizedStrategy, err := optimizer.OptimizeBasedOnMetrics(currentStrategy, performanceMetrics)
		assert.NoError(t, err)
		assert.NotNil(t, optimizedStrategy)

		// Verify optimization suggestions were applied
		assert.True(t, optimizedStrategy.FetchingStrategy.Concurrency <= 10)
		assert.True(t, optimizedStrategy.FetchingStrategy.Timeout >= 5*time.Second)
	})

	t.Run("adaptive_strategy_adjustment", func(t *testing.T) {
		adapter := NewAdaptiveStrategyManager()
		require.NotNil(t, adapter)

		initialStrategy := &ComposedStrategies{
			FetchingStrategy: ComposedFetchingStrategy{
				Strategies:  []FetchingStrategyType{AdaptiveFetching},
				Concurrency: 5,
				AdaptiveConfig: &AdaptiveConfiguration{
					InitialConcurrency: 5,
					MaxConcurrency:     20,
					AdjustmentFactor:   0.2,
				},
			},
		}

		// Simulate poor performance feedback that should trigger adjustment
		feedback := &PerformanceFeedback{
			Latency:     2 * time.Second,
			SuccessRate: 0.85, // Below 0.9 threshold
			ErrorRate:   0.15, // High error rate
			CPUUsage:    0.9,  // High CPU usage
			MemoryUsage: 0.8,  // High memory usage
		}

		adjustedStrategy, err := adapter.AdjustStrategy(initialStrategy, feedback)
		assert.NoError(t, err)
		assert.NotNil(t, adjustedStrategy)

		// Verify adaptive adjustments were made
		assert.NotEqual(t, initialStrategy.FetchingStrategy.Concurrency, adjustedStrategy.FetchingStrategy.Concurrency)
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("invalid_policy_handling", func(t *testing.T) {
		composer := NewStrategyComposer()

		invalidPolicies := &policies.BusinessPolicies{
			GlobalPolicy: &policies.GlobalBusinessPolicy{
				MaxConcurrency: -1, // Invalid
				Timeout:        0,  // Invalid
			},
		}

		_, err := composer.ComposeStrategies(invalidPolicies)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid business policies")
	})

	t.Run("composition_conflict_detection", func(t *testing.T) {
		composer := NewStrategyComposer()

		conflictingComposition := &ComposedStrategies{
			FetchingStrategy: ComposedFetchingStrategy{
				Strategies:  []FetchingStrategyType{ParallelFetching, SequentialFetching}, // Conflicting
				Concurrency: 10,
				Timeout:     10 * time.Second, // Valid timeout
			},
			ProcessingStrategy: ComposedProcessingStrategy{
				QualityThreshold: 0.8, // Valid threshold
			},
			OutputStrategy: ComposedOutputStrategy{
				DefaultFormat: "json", // Valid format
			},
		}

		err := composer.ValidateComposition(conflictingComposition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conflicting strategies")
	})

	t.Run("resource_constraint_validation", func(t *testing.T) {
		composer := NewStrategyComposer()

		resourceExceedingComposition := &ComposedStrategies{
			FetchingStrategy: ComposedFetchingStrategy{
				Concurrency: 1000,             // Exceeds reasonable limits
				Timeout:     10 * time.Second, // Valid timeout
			},
			ProcessingStrategy: ComposedProcessingStrategy{
				Concurrency:      500, // Exceeds reasonable limits
				QualityThreshold: 0.8, // Valid threshold
			},
			OutputStrategy: ComposedOutputStrategy{
				DefaultFormat: "json", // Valid format
			},
		}

		err := composer.ValidateComposition(resourceExceedingComposition)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "resource constraints")
	})
}
