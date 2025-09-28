// Package strategies provides experimental strategy composition, execution, and
// adaptive optimization structures. Experimental: All exported types and
// functions in this package may change or be removed before v1.0; the public
// facade does not yet depend on this surface. Treat as a preview extension API.
package strategies

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/99souls/ariadne/engine/business/policies"
)

// Strategy types and enums

// Experimental: FetchingStrategyType enumerates fetching strategy styles.
// May be reduced or renamed.
type FetchingStrategyType string

const (
	ParallelFetching   FetchingStrategyType = "parallel"
	SequentialFetching FetchingStrategyType = "sequential"
	FallbackFetching   FetchingStrategyType = "fallback"
	AdaptiveFetching   FetchingStrategyType = "adaptive"
)

// Experimental: ProcessingStrategyType enumerates processing orchestration modes.
type ProcessingStrategyType string

const (
	SequentialProcessing  ProcessingStrategyType = "sequential"
	ParallelProcessing    ProcessingStrategyType = "parallel"
	ConditionalProcessing ProcessingStrategyType = "conditional"
	PipelineProcessing    ProcessingStrategyType = "pipeline"
)

// Experimental: OutputStrategyType enumerates output routing modes.
type OutputStrategyType string

const (
	SimpleOutput       OutputStrategyType = "simple"
	ConditionalRouting OutputStrategyType = "conditional_routing"
	MultiSinkOutput    OutputStrategyType = "multi_sink"
	BufferedOutput     OutputStrategyType = "buffered"
)

// Core strategy composition structures

// Experimental: StrategyComposer composes policies into executable strategies.
// Interface shape may change; optimization & validation may relocate.
type StrategyComposer interface {
	ComposeStrategies(policies *policies.BusinessPolicies) (*ComposedStrategies, error)
	ValidateComposition(*ComposedStrategies) error
	OptimizeComposition(*ComposedStrategies) (*ComposedStrategies, error)
}

// Experimental: ComposedStrategies aggregates composed strategy configurations.
type ComposedStrategies struct {
	FetchingStrategy   ComposedFetchingStrategy   `json:"fetching_strategy"`
	ProcessingStrategy ComposedProcessingStrategy `json:"processing_strategy"`
	OutputStrategy     ComposedOutputStrategy     `json:"output_strategy"`
	Metadata           StrategyMetadata           `json:"metadata"`
}

// Experimental: ComposedFetchingStrategy configuration; field set may shrink.
type ComposedFetchingStrategy struct {
	Strategies     []FetchingStrategyType `json:"strategies"`
	Concurrency    int                    `json:"concurrency"`
	Timeout        time.Duration          `json:"timeout"`
	RetryEnabled   bool                   `json:"retry_enabled"`
	RetryConfig    RetryConfiguration     `json:"retry_config"`
	AdaptiveConfig *AdaptiveConfiguration `json:"adaptive_config,omitempty"`
}

// Experimental: ComposedProcessingStrategy configuration; subject to rename.
type ComposedProcessingStrategy struct {
	Strategies       []ProcessingStrategyType       `json:"strategies"`
	Steps            []string                       `json:"steps"`
	QualityThreshold float64                        `json:"quality_threshold"`
	ParallelSteps    bool                           `json:"parallel_steps"`
	Concurrency      int                            `json:"concurrency"`
	ConditionalRules map[string]ProcessingCondition `json:"conditional_rules,omitempty"`
}

// Experimental: ComposedOutputStrategy configuration; subject to simplification.
type ComposedOutputStrategy struct {
	Strategies         []OutputStrategyType    `json:"strategies"`
	DefaultFormat      string                  `json:"default_format"`
	CompressionEnabled bool                    `json:"compression_enabled"`
	RoutingRules       map[string]string       `json:"routing_rules"`
	MultiSinkConfig    *MultiSinkConfiguration `json:"multi_sink_config,omitempty"`
}

// Supporting configuration structures

// Experimental: RetryConfiguration for fetching; may move under facade policy.
type RetryConfiguration struct {
	MaxRetries    int           `json:"max_retries"`
	InitialDelay  time.Duration `json:"initial_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// Experimental: AdaptiveConfiguration for concurrency adaptation.
type AdaptiveConfiguration struct {
	InitialConcurrency int     `json:"initial_concurrency"`
	MaxConcurrency     int     `json:"max_concurrency"`
	AdjustmentFactor   float64 `json:"adjustment_factor"`
	PerformanceWindow  int     `json:"performance_window"`
}

// Experimental: ProcessingCondition expresses conditional actions.
type ProcessingCondition struct {
	Condition string   `json:"condition"`
	Actions   []string `json:"actions"`
}

// Experimental: MultiSinkConfiguration for multi-sink output.
type MultiSinkConfiguration struct {
	SinkTypes []string `json:"sink_types"`
	FanOut    bool     `json:"fan_out"`
	Failover  bool     `json:"failover"`
}

// Experimental: StrategyMetadata metadata; diagnostic only pre-v1.0.
type StrategyMetadata struct {
	ComposedAt    time.Time `json:"composed_at"`
	Version       string    `json:"version"`
	OptimizedFor  string    `json:"optimized_for"`
	EstimatedLoad string    `json:"estimated_load"`
}

// Execution and monitoring structures

// Experimental: StrategyExecutor executes composed strategies; API not stable.
type StrategyExecutor struct {
	strategies *ComposedStrategies
	mutex      sync.RWMutex
}

// Experimental: ExecutionPlan derived execution details.
type ExecutionPlan struct {
	FetchingPlan   FetchingExecutionPlan   `json:"fetching_plan"`
	ProcessingPlan ProcessingExecutionPlan `json:"processing_plan"`
	OutputPlan     OutputExecutionPlan     `json:"output_plan"`
}

// Experimental: FetchingExecutionPlan execution details.
type FetchingExecutionPlan struct {
	URLs        []string      `json:"urls"`
	Concurrency int           `json:"concurrency"`
	Timeout     time.Duration `json:"timeout"`
	BatchSize   int           `json:"batch_size"`
}

// Experimental: ProcessingExecutionPlan execution details.
type ProcessingExecutionPlan struct {
	Steps       []string `json:"steps"`
	Concurrency int      `json:"concurrency"`
	BatchSize   int      `json:"batch_size"`
}

// Experimental: OutputExecutionPlan execution details.
type OutputExecutionPlan struct {
	Format      string            `json:"format"`
	Compression bool              `json:"compression"`
	Routing     map[string]string `json:"routing"`
}

// Performance monitoring structures

// Experimental: StrategyPerformanceMonitor collects runtime metrics.
type StrategyPerformanceMonitor struct {
	metrics map[string]*StrategyMetrics
	mutex   sync.RWMutex
}

// Experimental: StrategyMetrics metrics snapshot.
type StrategyMetrics struct {
	AverageLatency time.Duration `json:"average_latency"`
	SuccessRate    float64       `json:"success_rate"`
	ThroughputRPM  int           `json:"throughput_rpm"`
	ErrorRate      float64       `json:"error_rate"`
	LastUpdated    time.Time     `json:"last_updated"`
}

// Experimental: PerformanceMetrics aggregated metrics.
type PerformanceMetrics struct {
	FetchingMetrics   map[string]*StrategyMetrics `json:"fetching_metrics"`
	ProcessingMetrics map[string]*StrategyMetrics `json:"processing_metrics"`
	OutputMetrics     map[string]*StrategyMetrics `json:"output_metrics"`
}

// Experimental: PerformanceRecommendations suggestions; heuristic.
type PerformanceRecommendations struct {
	Suggestions    []string               `json:"suggestions"`
	OptimalConfigs map[string]interface{} `json:"optimal_configs"`
	Warnings       []string               `json:"warnings"`
}

// Optimization structures

// Experimental: StrategyOptimizer heuristic optimizer.
type StrategyOptimizer struct {
	optimizationRules map[string]OptimizationRule
}

// Experimental: OptimizationRule rule definition.
type OptimizationRule struct {
	Condition string      `json:"condition"`
	Action    string      `json:"action"`
	Priority  int         `json:"priority"`
	Value     interface{} `json:"value"`
}

// Experimental: AdaptiveStrategyManager dynamic adjustment manager.
type AdaptiveStrategyManager struct {
	adjustmentHistory []StrategyAdjustment
	mutex             sync.RWMutex
}

// Experimental: StrategyAdjustment record.
type StrategyAdjustment struct {
	Timestamp      time.Time   `json:"timestamp"`
	PreviousConfig interface{} `json:"previous_config"`
	NewConfig      interface{} `json:"new_config"`
	Reason         string      `json:"reason"`
	Impact         string      `json:"impact"`
}

// Experimental: PerformanceFeedback input to adaptive adjustments.
type PerformanceFeedback struct {
	Latency     time.Duration `json:"latency"`
	SuccessRate float64       `json:"success_rate"`
	ErrorRate   float64       `json:"error_rate"`
	CPUUsage    float64       `json:"cpu_usage"`
	MemoryUsage float64       `json:"memory_usage"`
}

// Implementation

// NewStrategyComposer creates a new strategy composer.
// Experimental.
func NewStrategyComposer() StrategyComposer {
	return &strategyComposerImpl{}
}

type strategyComposerImpl struct{}

// ComposeStrategies creates composed strategies from business policies.
// Experimental.
func (sc *strategyComposerImpl) ComposeStrategies(businessPolicies *policies.BusinessPolicies) (*ComposedStrategies, error) {
	if businessPolicies == nil {
		return nil, errors.New("business policies cannot be nil")
	}

	// Validate business policies first
	if businessPolicies.GlobalPolicy == nil {
		return nil, errors.New("invalid business policies: global policy is required")
	}

	if businessPolicies.GlobalPolicy.MaxConcurrency <= 0 {
		return nil, errors.New("invalid business policies: maxConcurrency must be positive")
	}

	if businessPolicies.GlobalPolicy.Timeout <= 0 {
		return nil, errors.New("invalid business policies: timeout must be positive")
	}

	composed := &ComposedStrategies{
		Metadata: StrategyMetadata{
			ComposedAt:    time.Now(),
			Version:       "1.0.0",
			OptimizedFor:  "balanced",
			EstimatedLoad: "medium",
		},
	}

	// Compose fetching strategy
	if err := sc.composeFetchingStrategy(businessPolicies, composed); err != nil {
		return nil, fmt.Errorf("failed to compose fetching strategy: %w", err)
	}

	// Compose processing strategy
	if err := sc.composeProcessingStrategy(businessPolicies, composed); err != nil {
		return nil, fmt.Errorf("failed to compose processing strategy: %w", err)
	}

	// Compose output strategy
	if err := sc.composeOutputStrategy(businessPolicies, composed); err != nil {
		return nil, fmt.Errorf("failed to compose output strategy: %w", err)
	}

	return composed, nil
}

func (sc *strategyComposerImpl) composeFetchingStrategy(policies *policies.BusinessPolicies, composed *ComposedStrategies) error {
	fetchingStrategy := ComposedFetchingStrategy{
		Concurrency:  policies.GlobalPolicy.MaxConcurrency,
		Timeout:      policies.GlobalPolicy.Timeout,
		RetryEnabled: policies.GlobalPolicy.RetryPolicy != nil,
	}

	// Configure retry if enabled
	if fetchingStrategy.RetryEnabled {
		fetchingStrategy.RetryConfig = RetryConfiguration{
			MaxRetries:    policies.GlobalPolicy.RetryPolicy.MaxRetries,
			InitialDelay:  policies.GlobalPolicy.RetryPolicy.InitialDelay,
			BackoffFactor: policies.GlobalPolicy.RetryPolicy.BackoffFactor,
		}
	}

	// Determine appropriate fetching strategies
	if policies.CrawlingPolicy != nil && len(policies.CrawlingPolicy.SiteRules) > 1 {
		// Multiple sites suggest parallel fetching
		fetchingStrategy.Strategies = []FetchingStrategyType{ParallelFetching}

		// Add fallback for reliability
		if fetchingStrategy.RetryEnabled {
			fetchingStrategy.Strategies = append(fetchingStrategy.Strategies, FallbackFetching)
		}
	} else {
		// Single site or simple crawling
		fetchingStrategy.Strategies = []FetchingStrategyType{SequentialFetching}
	}

	// Add adaptive strategy for high concurrency scenarios
	if fetchingStrategy.Concurrency > 20 {
		fetchingStrategy.Strategies = append(fetchingStrategy.Strategies, AdaptiveFetching)
		fetchingStrategy.AdaptiveConfig = &AdaptiveConfiguration{
			InitialConcurrency: fetchingStrategy.Concurrency / 2,
			MaxConcurrency:     fetchingStrategy.Concurrency * 2,
			AdjustmentFactor:   0.1,
			PerformanceWindow:  10,
		}
	}

	composed.FetchingStrategy = fetchingStrategy
	return nil
}

func (sc *strategyComposerImpl) composeProcessingStrategy(policies *policies.BusinessPolicies, composed *ComposedStrategies) error {
	processingStrategy := ComposedProcessingStrategy{
		QualityThreshold: 0.5, // Default threshold
		ParallelSteps:    false,
		Concurrency:      1,
	}

	// Configure based on processing policies
	if policies.ProcessingPolicy != nil {
		processingStrategy.QualityThreshold = policies.ProcessingPolicy.QualityThreshold
		processingStrategy.Steps = policies.ProcessingPolicy.ProcessingSteps

		// Determine processing strategy type
		if len(processingStrategy.Steps) > 3 && composed.FetchingStrategy.Concurrency > 5 {
			// Complex processing with high concurrency suggests parallel processing
			processingStrategy.Strategies = []ProcessingStrategyType{ParallelProcessing}
			processingStrategy.ParallelSteps = true
			processingStrategy.Concurrency = composed.FetchingStrategy.Concurrency / 2
		} else {
			// Simple processing suggests sequential
			processingStrategy.Strategies = []ProcessingStrategyType{SequentialProcessing}
		}

		// Add conditional processing for quality-based decisions
		if processingStrategy.QualityThreshold > 0.3 {
			processingStrategy.Strategies = append(processingStrategy.Strategies, ConditionalProcessing)
			processingStrategy.ConditionalRules = map[string]ProcessingCondition{
				"high-quality": {
					Condition: fmt.Sprintf("quality > %.2f", processingStrategy.QualityThreshold),
					Actions:   []string{"detailed-extract", "metadata-enhance"},
				},
				"low-quality": {
					Condition: fmt.Sprintf("quality < %.2f", processingStrategy.QualityThreshold*0.5),
					Actions:   []string{"basic-extract", "log-quality"},
				},
			}
		}
	} else {
		// Default processing strategy
		processingStrategy.Strategies = []ProcessingStrategyType{SequentialProcessing}
		processingStrategy.Steps = []string{"extract", "clean", "validate"}
	}

	composed.ProcessingStrategy = processingStrategy
	return nil
}

func (sc *strategyComposerImpl) composeOutputStrategy(policies *policies.BusinessPolicies, composed *ComposedStrategies) error {
	outputStrategy := ComposedOutputStrategy{
		DefaultFormat:      "json",
		CompressionEnabled: false,
	}

	// Configure based on output policies
	if policies.OutputPolicy != nil {
		outputStrategy.DefaultFormat = policies.OutputPolicy.DefaultFormat
		outputStrategy.CompressionEnabled = policies.OutputPolicy.Compression
		outputStrategy.RoutingRules = policies.OutputPolicy.RoutingRules

		// Determine output strategy type
		if len(outputStrategy.RoutingRules) > 1 {
			// Multiple routing rules suggest conditional routing
			outputStrategy.Strategies = []OutputStrategyType{ConditionalRouting}
		} else {
			// Simple output
			outputStrategy.Strategies = []OutputStrategyType{SimpleOutput}
		}

		// Add multi-sink for high-throughput scenarios
		if composed.FetchingStrategy.Concurrency > 10 {
			outputStrategy.Strategies = append(outputStrategy.Strategies, MultiSinkOutput)
			outputStrategy.MultiSinkConfig = &MultiSinkConfiguration{
				SinkTypes: []string{"primary", "secondary"},
				FanOut:    false,
				Failover:  true,
			}
		}
	} else {
		// Default output strategy
		outputStrategy.Strategies = []OutputStrategyType{SimpleOutput}
		outputStrategy.RoutingRules = map[string]string{"default": "main-sink"}
	}

	composed.OutputStrategy = outputStrategy
	return nil
}

// ValidateComposition validates a composed strategy.
// Experimental.
func (sc *strategyComposerImpl) ValidateComposition(composed *ComposedStrategies) error {
	if composed == nil {
		return errors.New("composed strategies cannot be nil")
	}

	// Validate fetching strategy
	if err := sc.validateFetchingStrategy(&composed.FetchingStrategy); err != nil {
		return fmt.Errorf("invalid fetching strategy: %w", err)
	}

	// Validate processing strategy
	if err := sc.validateProcessingStrategy(&composed.ProcessingStrategy); err != nil {
		return fmt.Errorf("invalid processing strategy: %w", err)
	}

	// Validate output strategy
	if err := sc.validateOutputStrategy(&composed.OutputStrategy); err != nil {
		return fmt.Errorf("invalid output strategy: %w", err)
	}

	// Check for strategy conflicts
	if err := sc.checkStrategyConflicts(composed); err != nil {
		return fmt.Errorf("conflicting strategies: %w", err)
	}

	// Validate resource constraints
	if err := sc.validateResourceConstraints(composed); err != nil {
		return fmt.Errorf("resource constraints: %w", err)
	}

	return nil
}

func (sc *strategyComposerImpl) validateFetchingStrategy(strategy *ComposedFetchingStrategy) error {
	if strategy.Concurrency <= 0 {
		return errors.New("concurrency must be positive")
	}

	if strategy.Timeout <= 0 {
		return errors.New("timeout must be positive")
	}

	if strategy.RetryEnabled && strategy.RetryConfig.MaxRetries <= 0 {
		return errors.New("retry enabled but max retries is not positive")
	}

	return nil
}

func (sc *strategyComposerImpl) validateProcessingStrategy(strategy *ComposedProcessingStrategy) error {
	if strategy.QualityThreshold < 0 || strategy.QualityThreshold > 1 {
		return errors.New("quality threshold must be between 0 and 1")
	}

	if strategy.ParallelSteps && strategy.Concurrency <= 0 {
		return errors.New("parallel steps enabled but concurrency is not positive")
	}

	return nil
}

func (sc *strategyComposerImpl) validateOutputStrategy(strategy *ComposedOutputStrategy) error {
	if strategy.DefaultFormat == "" {
		return errors.New("default format cannot be empty")
	}

	return nil
}

func (sc *strategyComposerImpl) checkStrategyConflicts(composed *ComposedStrategies) error {
	// Check for conflicting fetching strategies
	fetchingStrategies := composed.FetchingStrategy.Strategies
	if containsConflictingStrategies(fetchingStrategies, []FetchingStrategyType{ParallelFetching, SequentialFetching}) {
		return errors.New("parallel and sequential fetching strategies conflict")
	}

	return nil
}

func (sc *strategyComposerImpl) validateResourceConstraints(composed *ComposedStrategies) error {
	maxConcurrency := 100 // Reasonable limit

	if composed.FetchingStrategy.Concurrency > maxConcurrency {
		return fmt.Errorf("fetching concurrency %d exceeds maximum %d", composed.FetchingStrategy.Concurrency, maxConcurrency)
	}

	if composed.ProcessingStrategy.Concurrency > maxConcurrency {
		return fmt.Errorf("processing concurrency %d exceeds maximum %d", composed.ProcessingStrategy.Concurrency, maxConcurrency)
	}

	return nil
}

// OptimizeComposition optimizes a composed strategy.
// Experimental.
func (sc *strategyComposerImpl) OptimizeComposition(composed *ComposedStrategies) (*ComposedStrategies, error) {
	if composed == nil {
		return nil, errors.New("composed strategies cannot be nil")
	}

	optimized := *composed // Create a copy

	// Optimize fetching strategy
	if optimized.FetchingStrategy.Concurrency > 50 {
		optimized.FetchingStrategy.Concurrency = 50 // Cap at reasonable limit
	}

	if optimized.FetchingStrategy.Timeout < 5*time.Second {
		optimized.FetchingStrategy.Timeout = 5 * time.Second // Minimum reasonable timeout
	}

	// Optimize processing strategy
	if optimized.ProcessingStrategy.QualityThreshold < 0.5 {
		optimized.ProcessingStrategy.QualityThreshold = 0.5 // Reasonable minimum
	}

	// Update metadata
	optimized.Metadata.OptimizedFor = "performance"
	optimized.Metadata.ComposedAt = time.Now()

	return &optimized, nil
}

// Helper functions

func containsConflictingStrategies[T comparable](strategies []T, conflictingPair []T) bool {
	if len(conflictingPair) != 2 {
		return false
	}

	hasFirst := false
	hasSecond := false

	for _, strategy := range strategies {
		if strategy == conflictingPair[0] {
			hasFirst = true
		}
		if strategy == conflictingPair[1] {
			hasSecond = true
		}
	}

	return hasFirst && hasSecond
}

// Strategy execution implementation

// NewStrategyExecutor creates a new strategy executor.
// Experimental.
func NewStrategyExecutor(strategies *ComposedStrategies) *StrategyExecutor {
	return &StrategyExecutor{
		strategies: strategies,
	}
}

// CreateExecutionPlan creates an execution plan for the given URLs.
// Experimental.
func (se *StrategyExecutor) CreateExecutionPlan(ctx context.Context, urls []string) (*ExecutionPlan, error) {
	if len(urls) == 0 {
		return nil, errors.New("no URLs provided")
	}

	se.mutex.RLock()
	defer se.mutex.RUnlock()

	plan := &ExecutionPlan{
		FetchingPlan: FetchingExecutionPlan{
			URLs:        urls,
			Concurrency: se.strategies.FetchingStrategy.Concurrency,
			Timeout:     se.strategies.FetchingStrategy.Timeout,
			BatchSize:   calculateBatchSize(len(urls), se.strategies.FetchingStrategy.Concurrency),
		},
		ProcessingPlan: ProcessingExecutionPlan{
			Steps:       se.strategies.ProcessingStrategy.Steps,
			Concurrency: se.strategies.ProcessingStrategy.Concurrency,
			BatchSize:   10, // Default batch size
		},
		OutputPlan: OutputExecutionPlan{
			Format:      se.strategies.OutputStrategy.DefaultFormat,
			Compression: se.strategies.OutputStrategy.CompressionEnabled,
			Routing:     se.strategies.OutputStrategy.RoutingRules,
		},
	}

	return plan, nil
}

func calculateBatchSize(urlCount, concurrency int) int {
	if concurrency <= 0 {
		return 1
	}

	batchSize := urlCount / concurrency
	if batchSize < 1 {
		return 1
	}

	return batchSize
}

// Performance monitoring implementation

// NewStrategyPerformanceMonitor creates a new performance monitor.
// Experimental.
func NewStrategyPerformanceMonitor() *StrategyPerformanceMonitor {
	return &StrategyPerformanceMonitor{
		metrics: make(map[string]*StrategyMetrics),
	}
}

// RecordFetchingPerformance records fetching strategy performance.
// Experimental.
func (spm *StrategyPerformanceMonitor) RecordFetchingPerformance(strategyType string, latency time.Duration, success bool) {
	spm.mutex.Lock()
	defer spm.mutex.Unlock()

	key := "fetching_" + strategyType
	if _, exists := spm.metrics[key]; !exists {
		spm.metrics[key] = &StrategyMetrics{}
	}

	metric := spm.metrics[key]
	metric.AverageLatency = (metric.AverageLatency + latency) / 2 // Simple running average
	if success {
		metric.SuccessRate = (metric.SuccessRate + 1.0) / 2
	} else {
		metric.SuccessRate = (metric.SuccessRate + 0.0) / 2
	}
	metric.LastUpdated = time.Now()
}

// RecordProcessingPerformance records processing strategy performance.
// Experimental.
func (spm *StrategyPerformanceMonitor) RecordProcessingPerformance(strategyType string, latency time.Duration, quality float64) {
	spm.mutex.Lock()
	defer spm.mutex.Unlock()

	key := "processing_" + strategyType
	if _, exists := spm.metrics[key]; !exists {
		spm.metrics[key] = &StrategyMetrics{}
	}

	metric := spm.metrics[key]
	metric.AverageLatency = (metric.AverageLatency + latency) / 2
	metric.SuccessRate = (metric.SuccessRate + quality) / 2 // Use quality as success indicator
	metric.LastUpdated = time.Now()
}

// RecordOutputPerformance records output strategy performance.
// Experimental.
func (spm *StrategyPerformanceMonitor) RecordOutputPerformance(strategyType string, latency time.Duration, success bool) {
	spm.mutex.Lock()
	defer spm.mutex.Unlock()

	key := "output_" + strategyType
	if _, exists := spm.metrics[key]; !exists {
		spm.metrics[key] = &StrategyMetrics{}
	}

	metric := spm.metrics[key]
	metric.AverageLatency = (metric.AverageLatency + latency) / 2
	if success {
		metric.SuccessRate = (metric.SuccessRate + 1.0) / 2
	} else {
		metric.SuccessRate = (metric.SuccessRate + 0.0) / 2
	}
	metric.LastUpdated = time.Now()
}

// GetMetrics returns current performance metrics.
// Experimental.
func (spm *StrategyPerformanceMonitor) GetMetrics() *PerformanceMetrics {
	spm.mutex.RLock()
	defer spm.mutex.RUnlock()

	metrics := &PerformanceMetrics{
		FetchingMetrics:   make(map[string]*StrategyMetrics),
		ProcessingMetrics: make(map[string]*StrategyMetrics),
		OutputMetrics:     make(map[string]*StrategyMetrics),
	}

	for key, metric := range spm.metrics {
		metricCopy := *metric // Copy the metric

		if len(key) > 9 && key[:9] == "fetching_" {
			strategyType := key[9:]
			metrics.FetchingMetrics[strategyType] = &metricCopy
		} else if len(key) > 11 && key[:11] == "processing_" {
			strategyType := key[11:]
			metrics.ProcessingMetrics[strategyType] = &metricCopy
		} else if len(key) > 7 && key[:7] == "output_" {
			strategyType := key[7:]
			metrics.OutputMetrics[strategyType] = &metricCopy
		}
	}

	return metrics
}

// AnalyzePerformance provides performance analysis and recommendations.
// Experimental.
func (spm *StrategyPerformanceMonitor) AnalyzePerformance() *PerformanceRecommendations {
	metrics := spm.GetMetrics()

	recommendations := &PerformanceRecommendations{
		Suggestions:    make([]string, 0),
		OptimalConfigs: make(map[string]interface{}),
		Warnings:       make([]string, 0),
	}

	// Analyze fetching performance
	for strategyType, metric := range metrics.FetchingMetrics {
		if metric.SuccessRate < 0.9 {
			recommendations.Suggestions = append(recommendations.Suggestions,
				fmt.Sprintf("Consider adding retry logic to %s fetching strategy", strategyType))
		}

		if metric.AverageLatency > 10*time.Second {
			recommendations.Warnings = append(recommendations.Warnings,
				fmt.Sprintf("High latency detected in %s fetching strategy", strategyType))
		}
	}

	return recommendations
}

// Optimization implementation

// NewStrategyOptimizer creates a new strategy optimizer.
// Experimental.
func NewStrategyOptimizer() *StrategyOptimizer {
	return &StrategyOptimizer{
		optimizationRules: make(map[string]OptimizationRule),
	}
}

// OptimizeBasedOnMetrics optimizes strategies based on performance metrics.
// Experimental.
func (so *StrategyOptimizer) OptimizeBasedOnMetrics(strategy *ComposedStrategies, metrics *PerformanceMetrics) (*ComposedStrategies, error) {
	if strategy == nil {
		return nil, errors.New("strategy cannot be nil")
	}

	optimized := *strategy // Create a copy

	// Optimize based on fetching metrics
	for strategyType, metric := range metrics.FetchingMetrics {
		if strategyType == "parallel" && metric.SuccessRate < 0.9 {
			// Reduce concurrency if success rate is low
			if optimized.FetchingStrategy.Concurrency > 5 {
				optimized.FetchingStrategy.Concurrency = optimized.FetchingStrategy.Concurrency / 2
			}
		}

		if metric.AverageLatency > 5*time.Second {
			// Increase timeout if latency is high
			optimized.FetchingStrategy.Timeout = metric.AverageLatency * 2
		}
	}

	return &optimized, nil
}

// Adaptive strategy management

// NewAdaptiveStrategyManager creates a new adaptive strategy manager.
// Experimental.
func NewAdaptiveStrategyManager() *AdaptiveStrategyManager {
	return &AdaptiveStrategyManager{
		adjustmentHistory: make([]StrategyAdjustment, 0),
	}
}

// AdjustStrategy adjusts strategy based on performance feedback.
// Experimental.
func (asm *AdaptiveStrategyManager) AdjustStrategy(strategy *ComposedStrategies, feedback *PerformanceFeedback) (*ComposedStrategies, error) {
	if strategy == nil {
		return nil, errors.New("strategy cannot be nil")
	}

	if feedback == nil {
		return strategy, nil // No feedback, no adjustment
	}

	adjusted := *strategy // Create a copy
	adjustmentMade := false

	// Adjust fetching concurrency based on success rate and resource usage
	if feedback.SuccessRate < 0.9 && adjusted.FetchingStrategy.Concurrency > 1 {
		previousConcurrency := adjusted.FetchingStrategy.Concurrency
		adjusted.FetchingStrategy.Concurrency = int(float64(adjusted.FetchingStrategy.Concurrency) * 0.8)

		asm.recordAdjustment(StrategyAdjustment{
			Timestamp:      time.Now(),
			PreviousConfig: previousConcurrency,
			NewConfig:      adjusted.FetchingStrategy.Concurrency,
			Reason:         "low success rate",
			Impact:         "reduced concurrency",
		})
		adjustmentMade = true
	} else if feedback.SuccessRate > 0.95 && feedback.CPUUsage < 0.7 && adjusted.FetchingStrategy.AdaptiveConfig != nil {
		// Increase concurrency if performance is good and resources are available
		previousConcurrency := adjusted.FetchingStrategy.Concurrency
		newConcurrency := int(float64(adjusted.FetchingStrategy.Concurrency) * 1.2)

		if newConcurrency <= adjusted.FetchingStrategy.AdaptiveConfig.MaxConcurrency {
			adjusted.FetchingStrategy.Concurrency = newConcurrency

			asm.recordAdjustment(StrategyAdjustment{
				Timestamp:      time.Now(),
				PreviousConfig: previousConcurrency,
				NewConfig:      adjusted.FetchingStrategy.Concurrency,
				Reason:         "good performance and available resources",
				Impact:         "increased concurrency",
			})
			adjustmentMade = true
		}
	}

	if !adjustmentMade {
		return strategy, nil // No adjustment needed
	}

	return &adjusted, nil
}

func (asm *AdaptiveStrategyManager) recordAdjustment(adjustment StrategyAdjustment) {
	asm.mutex.Lock()
	defer asm.mutex.Unlock()

	asm.adjustmentHistory = append(asm.adjustmentHistory, adjustment)

	// Keep only last 100 adjustments
	if len(asm.adjustmentHistory) > 100 {
		asm.adjustmentHistory = asm.adjustmentHistory[1:]
	}
}
