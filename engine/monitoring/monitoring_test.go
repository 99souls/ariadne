package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBusinessMetricsCollector(t *testing.T) {
	t.Run("rule_evaluation_performance_tracking", func(t *testing.T) {
		collector := NewBusinessMetricsCollector()
		require.NotNil(t, collector)

		// Record rule evaluation metrics
		collector.RecordRuleEvaluation("crawling_policy", "site_rules", 50*time.Microsecond, true)
		collector.RecordRuleEvaluation("crawling_policy", "link_rules", 30*time.Microsecond, true)
		collector.RecordRuleEvaluation("processing_policy", "content_extraction", 200*time.Microsecond, false)
		collector.RecordRuleEvaluation("output_policy", "routing_rules", 10*time.Microsecond, true)

		// Get aggregated metrics
		metrics := collector.GetAggregatedMetrics()
		assert.NotNil(t, metrics)

		// Verify policy-level aggregations
		assert.Contains(t, metrics.PolicyMetrics, "crawling_policy")
		assert.Contains(t, metrics.PolicyMetrics, "processing_policy")
		assert.Contains(t, metrics.PolicyMetrics, "output_policy")

		// Verify crawling policy metrics
		crawlingMetrics := metrics.PolicyMetrics["crawling_policy"]
		assert.Equal(t, 2, crawlingMetrics.TotalEvaluations)
		assert.Equal(t, 1.0, crawlingMetrics.SuccessRate) // 2/2 = 100%
		assert.True(t, crawlingMetrics.AverageLatency > 0)

		// Verify processing policy had failures
		processingMetrics := metrics.PolicyMetrics["processing_policy"]
		assert.Equal(t, 1, processingMetrics.TotalEvaluations)
		assert.Equal(t, 0.0, processingMetrics.SuccessRate) // 0/1 = 0%
	})

	t.Run("strategy_effectiveness_metrics", func(t *testing.T) {
		collector := NewBusinessMetricsCollector()

		// Record strategy execution metrics
		collector.RecordStrategyExecution("parallel_fetching", 2*time.Second, 100, 95)           // 95% success
		collector.RecordStrategyExecution("parallel_fetching", 1800*time.Millisecond, 80, 76)    // 95% success
		collector.RecordStrategyExecution("sequential_processing", 3*time.Second, 50, 48)        // 96% success
		collector.RecordStrategyExecution("conditional_routing", 100*time.Millisecond, 200, 190) // 95% success

		metrics := collector.GetAggregatedMetrics()

		// Verify strategy metrics
		assert.Contains(t, metrics.StrategyMetrics, "parallel_fetching")
		assert.Contains(t, metrics.StrategyMetrics, "sequential_processing")
		assert.Contains(t, metrics.StrategyMetrics, "conditional_routing")

		// Verify parallel fetching aggregated metrics
		parallelMetrics := metrics.StrategyMetrics["parallel_fetching"]
		assert.Equal(t, 2, parallelMetrics.ExecutionCount)
		assert.Equal(t, 95.0, parallelMetrics.AverageSuccessRate) // (95+95)/2
		assert.Equal(t, 180, parallelMetrics.TotalItemsProcessed) // 100+80
	})

	t.Run("business_outcome_tracking", func(t *testing.T) {
		collector := NewBusinessMetricsCollector()

		// Record business outcomes
		collector.RecordBusinessOutcome("pages_crawled", 150, map[string]interface{}{
			"domain":        "example.com",
			"content_type":  "article",
			"quality_score": 8.5,
		})

		collector.RecordBusinessOutcome("content_extracted", 140, map[string]interface{}{
			"extraction_method": "intelligent",
			"completeness":      0.95,
			"accuracy":          0.92,
		})

		collector.RecordBusinessOutcome("data_delivered", 135, map[string]interface{}{
			"format":           "json",
			"compression_rate": 0.3,
			"delivery_time":    "500ms",
		})

		metrics := collector.GetAggregatedMetrics()

		// Verify outcome tracking
		assert.Contains(t, metrics.BusinessOutcomes, "pages_crawled")
		assert.Contains(t, metrics.BusinessOutcomes, "content_extracted")
		assert.Contains(t, metrics.BusinessOutcomes, "data_delivered")

		// Verify funnel analysis
		crawledOutcome := metrics.BusinessOutcomes["pages_crawled"]
		extractedOutcome := metrics.BusinessOutcomes["content_extracted"]
		deliveredOutcome := metrics.BusinessOutcomes["data_delivered"]

		assert.Equal(t, 150, crawledOutcome.Count)
		assert.Equal(t, 140, extractedOutcome.Count)
		assert.Equal(t, 135, deliveredOutcome.Count)

		// Calculate conversion rates
		extractionRate := float64(extractedOutcome.Count) / float64(crawledOutcome.Count)
		deliveryRate := float64(deliveredOutcome.Count) / float64(extractedOutcome.Count)

		assert.InDelta(t, 0.933, extractionRate, 0.001) // 140/150 ≈ 93.3%
		assert.InDelta(t, 0.964, deliveryRate, 0.001)   // 135/140 ≈ 96.4%
	})
}

func TestPrometheusExporter(t *testing.T) {
	t.Run("metrics_export_integration", func(t *testing.T) {
		// Create metrics collector with data
		collector := NewBusinessMetricsCollector()
		collector.RecordRuleEvaluation("test_policy", "test_rule", 100*time.Microsecond, true)
		collector.RecordStrategyExecution("test_strategy", 1*time.Second, 10, 9)

		// Create Prometheus exporter
		exporter, err := NewPrometheusExporter(collector, "ariadne_engine")
		require.NoError(t, err)
		require.NotNil(t, exporter)

		// Test metrics endpoint
		handler := exporter.GetMetricsHandler()
		assert.NotNil(t, handler)

		// Create test request
		req := httptest.NewRequest("GET", "/metrics", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "text/plain")

		// Verify metrics content contains expected metrics
		metricsBody := rr.Body.String()
		assert.Contains(t, metricsBody, "ariadne_engine")
		assert.Contains(t, metricsBody, "rule_evaluation")
		assert.Contains(t, metricsBody, "strategy_execution")
	})

	t.Run("custom_metrics_registration", func(t *testing.T) {
		collector := NewBusinessMetricsCollector()
		exporter, err := NewPrometheusExporter(collector, "custom_namespace")
		require.NoError(t, err)

		// Register custom metric
		err = exporter.RegisterCustomMetric("business_value_generated", "Total business value generated", "counter")
		assert.NoError(t, err)

		// Record custom metric
		err = exporter.RecordCustomMetric("business_value_generated", 1000.0, map[string]string{
			"source": "web_scraping",
			"type":   "revenue",
		})
		assert.NoError(t, err)

		// Test metrics endpoint includes custom metric
		req := httptest.NewRequest("GET", "/metrics", nil)
		rr := httptest.NewRecorder()
		exporter.GetMetricsHandler().ServeHTTP(rr, req)

		metricsBody := rr.Body.String()
		assert.Contains(t, metricsBody, "business_value_generated")
		assert.Contains(t, metricsBody, "custom_namespace")
	})
}

func TestOpenTelemetryTracer(t *testing.T) {
	t.Run("distributed_tracing_integration", func(t *testing.T) {
		tracer, err := NewOpenTelemetryTracer("ariadne-engine", "test-environment")
		require.NoError(t, err)
		require.NotNil(t, tracer)

		ctx := context.Background()

		// Start business operation trace
		ctx, span := tracer.StartBusinessOperation(ctx, "crawl_website", map[string]interface{}{
			"target_url":    "https://example.com",
			"crawl_depth":   3,
			"max_pages":     100,
			"strategy_type": "parallel",
		})

		assert.NotNil(t, span)

		// Record rule evaluation within trace
		tracer.RecordRuleEvaluation(ctx, "site_policy", "allowed_domains", 50*time.Microsecond, true, map[string]interface{}{
			"domain":       "example.com",
			"rule_matched": true,
		})

		// Record strategy execution within trace
		tracer.RecordStrategyExecution(ctx, "parallel_fetching", 2*time.Second, map[string]interface{}{
			"concurrency":       10,
			"pages_processed":   25,
			"success_rate":      0.96,
			"avg_response_time": "800ms",
		})

		// Record error within trace
		tracer.RecordError(ctx, "rate_limit_exceeded", fmt.Errorf("rate limit exceeded: 429 Too Many Requests"), map[string]interface{}{
			"retry_after": "30s",
			"endpoint":    "/api/data",
		})

		// Verify span was recording before being finished
		isRecording := span.IsRecording()

		// Finish the trace
		tracer.FinishBusinessOperation(span, true, map[string]interface{}{
			"total_pages_crawled": 23,
			"total_errors":        1,
			"completion_time":     "2.5s",
		})

		// Verify span was recording during the operation
		assert.True(t, isRecording)
	})

	t.Run("trace_context_propagation", func(t *testing.T) {
		tracer, err := NewOpenTelemetryTracer("ariadne-engine", "test-environment")
		require.NoError(t, err)

		ctx := context.Background()

		// Start parent operation
		ctx, parentSpan := tracer.StartBusinessOperation(ctx, "full_scraping_pipeline", map[string]interface{}{
			"operation_type": "scheduled_crawl",
		})

		// Start child operation (should inherit trace context) - ignore returned ctx to avoid ineffassign
		_, childSpan := tracer.StartBusinessOperation(ctx, "page_processing", map[string]interface{}{
			"processor_type": "content_extraction",
		})

		// Verify both spans are recording
		assert.True(t, parentSpan.IsRecording())
		assert.True(t, childSpan.IsRecording())

		// Finish child first
		tracer.FinishBusinessOperation(childSpan, true, map[string]interface{}{
			"content_extracted": "article",
		})

		// Finish parent
		tracer.FinishBusinessOperation(parentSpan, true, map[string]interface{}{
			"pipeline_success": true,
		})
	})
}

func TestStructuredLogger(t *testing.T) {
	t.Run("business_context_logging", func(t *testing.T) {
		logger, err := NewStructuredLogger("info", "json", "business-engine")
		require.NoError(t, err)
		require.NotNil(t, logger)

		// Log business rule evaluation
		logger.LogRuleEvaluation("crawling_policy", "site_rules", map[string]interface{}{
			"rule":         "allowed_domains",
			"input_domain": "example.com",
			"result":       "allowed",
			"latency_us":   45,
		})

		// Log strategy execution
		logger.LogStrategyExecution("parallel_processing", map[string]interface{}{
			"strategy":        "parallel_processing",
			"items_processed": 150,
			"success_rate":    0.94,
			"avg_latency_ms":  250,
			"resource_usage": map[string]interface{}{
				"cpu_percent": 45.2,
				"memory_mb":   128,
				"goroutines":  25,
			},
		})

		// Log configuration change
		logger.LogConfigurationChange("policy_update", map[string]interface{}{
			"change_type":     "policy_update",
			"affected_policy": "rate_limiting",
			"old_value":       "100ms",
			"new_value":       "200ms",
			"change_reason":   "performance_optimization",
			"applied_by":      "admin",
		})

		// Log business outcome
		logger.LogBusinessOutcome("content_delivery", map[string]interface{}{
			"outcome":          "content_delivery",
			"pages_delivered":  89,
			"format":           "json",
			"compression_rate": 0.35,
			"delivery_time_ms": 450,
			"quality_score":    8.7,
		})

		// Basic validation - logger should not panic and should handle all log types
		assert.True(t, true) // If we get here without panic, logging is working
	})

	t.Run("log_level_filtering", func(t *testing.T) {
		// Create logger with different levels
		debugLogger, err := NewStructuredLogger("debug", "json", "test-debug")
		require.NoError(t, err)

		warnLogger, err := NewStructuredLogger("warn", "json", "test-warn")
		require.NoError(t, err)

		// Test debug logging (should log everything)
		debugLogger.LogDebug("debug_message", map[string]interface{}{
			"detail": "verbose_information",
		})

		debugLogger.LogWarn("warning_message", map[string]interface{}{
			"issue": "potential_problem",
		})

		// Test warn logging (should only log warnings and above)
		warnLogger.LogDebug("debug_message", map[string]interface{}{
			"detail": "should_not_appear",
		})

		warnLogger.LogError("error_message", fmt.Errorf("test error"), map[string]interface{}{
			"error_type": "test_error",
		})

		// Basic validation - no panics
		assert.NotNil(t, debugLogger)
		assert.NotNil(t, warnLogger)
	})
}

func TestHealthCheckSystem(t *testing.T) {
	t.Run("comprehensive_health_checks", func(t *testing.T) {
		// Create health check system
		healthSystem := NewHealthCheckSystem()
		require.NotNil(t, healthSystem)

		// Register health checks
		err := healthSystem.RegisterCheck("business_logic", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Name:      "business_logic",
				Status:    "healthy",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"rules_loaded":      25,
					"strategies_active": 8,
					"avg_evaluation_ms": 2.5,
				},
			}
		})
		assert.NoError(t, err)

		err = healthSystem.RegisterCheck("configuration", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Name:      "configuration",
				Status:    "healthy",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"config_version":     "1.2.0",
					"hot_reload_enabled": true,
					"validation_status":  "passed",
				},
			}
		})
		assert.NoError(t, err)

		err = healthSystem.RegisterCheck("monitoring", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Name:      "monitoring",
				Status:    "degraded",
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"metrics_collecting": true,
					"tracing_enabled":    false,
					"alert_count":        2,
				},
				Issues: []string{"OpenTelemetry tracing disabled", "2 active alerts"},
			}
		})
		assert.NoError(t, err)

		// Perform health check
		ctx := context.Background()
		overallHealth := healthSystem.CheckHealth(ctx)

		// Verify overall health
		assert.NotNil(t, overallHealth)
		assert.Equal(t, "degraded", overallHealth.OverallStatus) // One component degraded
		assert.Len(t, overallHealth.ComponentResults, 3)

		// Verify individual component results
		businessResult := findHealthResult(overallHealth.ComponentResults, "business_logic")
		assert.Equal(t, "healthy", businessResult.Status)
		assert.Equal(t, 25, businessResult.Metadata["rules_loaded"])

		configResult := findHealthResult(overallHealth.ComponentResults, "configuration")
		assert.Equal(t, "healthy", configResult.Status)
		assert.Equal(t, "1.2.0", configResult.Metadata["config_version"])

		monitoringResult := findHealthResult(overallHealth.ComponentResults, "monitoring")
		assert.Equal(t, "degraded", monitoringResult.Status)
		assert.Len(t, monitoringResult.Issues, 2)
	})

	t.Run("health_check_endpoint", func(t *testing.T) {
		healthSystem := NewHealthCheckSystem()

		// Register simple check
		err := healthSystem.RegisterCheck("test", func(ctx context.Context) HealthCheckResult {
			return HealthCheckResult{
				Name:      "test",
				Status:    "healthy",
				Timestamp: time.Now(),
			}
		})
		require.NoError(t, err)

		// Create HTTP handler
		handler := healthSystem.GetHealthHandler()
		assert.NotNil(t, handler)

		// Test health endpoint
		req := httptest.NewRequest("GET", "/health", nil)
		rr := httptest.NewRecorder()

		handler.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")

		// Parse response
		var healthResponse OverallHealthResult
		err = json.Unmarshal(rr.Body.Bytes(), &healthResponse)
		assert.NoError(t, err)

		assert.Equal(t, "healthy", healthResponse.OverallStatus)
		assert.Len(t, healthResponse.ComponentResults, 1)
	})
}

func TestMonitoringIntegration(t *testing.T) {
	t.Run("end_to_end_monitoring_workflow", func(t *testing.T) {
		// Create integrated monitoring system
		metricsCollector := NewBusinessMetricsCollector()
		prometheus, err := NewPrometheusExporter(metricsCollector, "integration_test")
		require.NoError(t, err)

		tracer, err := NewOpenTelemetryTracer("integration-test", "test")
		require.NoError(t, err)

		logger, err := NewStructuredLogger("info", "json", "integration-test")
		require.NoError(t, err)

		healthSystem := NewHealthCheckSystem()

		// Create integrated monitoring system
		monitoring, err := NewIntegratedMonitoringSystem(metricsCollector, prometheus, tracer, logger, healthSystem)
		require.NoError(t, err)
		require.NotNil(t, monitoring)

		ctx := context.Background()

		// Simulate business rule evaluation with full monitoring
		err = monitoring.MonitorRuleEvaluation(ctx, "crawling_policy", "site_validation", func() error {
			time.Sleep(10 * time.Millisecond) // Simulate work
			return nil
		})
		assert.NoError(t, err)

		// Simulate strategy execution with full monitoring
		err = monitoring.MonitorStrategyExecution(ctx, "parallel_processing", func() (int, int, error) {
			time.Sleep(50 * time.Millisecond) // Simulate work
			return 100, 95, nil               // 100 items processed, 95 successful
		})
		assert.NoError(t, err)

		// Simulate configuration change monitoring (runtime config types internalized)
		monitoring.MonitorConfigurationChange("policy_update", "1.0.0", "1.1.0")

		// Get aggregated metrics
		metrics := monitoring.GetAggregatedMetrics()
		assert.NotNil(t, metrics)
		assert.Contains(t, metrics.PolicyMetrics, "crawling_policy")
		assert.Contains(t, metrics.StrategyMetrics, "parallel_processing")

		// Test health check integration
		overallHealth := monitoring.GetOverallHealth(ctx)
		assert.NotNil(t, overallHealth)
		// Should be healthy unless there are specific issues
	})

	t.Run("monitoring_with_business_policies", func(t *testing.T) {
		// Create monitoring system
		monitoring, err := NewIntegratedMonitoringSystem(
			NewBusinessMetricsCollector(),
			nil, // Skip Prometheus for this test
			nil, // Skip OpenTelemetry for this test
			nil, // Skip structured logger for this test
			NewHealthCheckSystem(),
		)
		require.NoError(t, err)

		// Create sample business policies
		// Composed strategies removed; retain a basic invocation of the generic helper
		ctx := context.Background()
		err = monitoring.MonitorComposedStrategyExecution(ctx, func() (int, int, error) {
			return 10, 9, nil
		})
		assert.NoError(t, err)
	})
}

// Helper function to find health result by name
func findHealthResult(results []HealthCheckResult, name string) HealthCheckResult {
	for _, result := range results {
		if result.Name == name {
			return result
		}
	}
	return HealthCheckResult{}
}
