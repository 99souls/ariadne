package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/99souls/ariadne/engine/strategies"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	// Use OTLP trace exporter if available in future; for tests we fall back to no-op provider
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// BusinessMetricsCollector collects and aggregates business-level metrics
type BusinessMetricsCollector struct {
	ruleMetrics     map[string]*RuleMetrics
	strategyMetrics map[string]*StrategyMetrics
	outcomeMetrics  map[string]*OutcomeMetrics
	mutex           sync.RWMutex
}

// RuleMetrics tracks metrics for business rule evaluations
type RuleMetrics struct {
	PolicyName       string
	RuleName         string
	TotalEvaluations int
	SuccessfulEvals  int
	FailedEvals      int
	TotalLatency     time.Duration
	AverageLatency   time.Duration
	SuccessRate      float64
	LastEvaluation   time.Time
}

// StrategyMetrics tracks metrics for strategy executions
type StrategyMetrics struct {
	StrategyName        string
	ExecutionCount      int
	TotalItemsProcessed int
	TotalSuccessful     int
	TotalLatency        time.Duration
	AverageLatency      time.Duration
	AverageSuccessRate  float64
	LastExecution       time.Time
}

// OutcomeMetrics tracks business outcome metrics
type OutcomeMetrics struct {
	OutcomeName  string
	Count        int
	Metadata     map[string]interface{}
	LastRecorded time.Time
}

// AggregatedMetrics provides a comprehensive view of all metrics
type AggregatedMetrics struct {
	PolicyMetrics    map[string]*PolicyAggregateMetrics `json:"policy_metrics"`
	StrategyMetrics  map[string]*StrategyMetrics        `json:"strategy_metrics"`
	BusinessOutcomes map[string]*OutcomeMetrics         `json:"business_outcomes"`
	CollectionTime   time.Time                          `json:"collection_time"`
}

// PolicyAggregateMetrics provides aggregated metrics per policy
type PolicyAggregateMetrics struct {
	PolicyName       string                  `json:"policy_name"`
	TotalEvaluations int                     `json:"total_evaluations"`
	SuccessfulEvals  int                     `json:"successful_evaluations"`
	FailedEvals      int                     `json:"failed_evaluations"`
	AverageLatency   time.Duration           `json:"average_latency"`
	SuccessRate      float64                 `json:"success_rate"`
	RuleBreakdown    map[string]*RuleMetrics `json:"rule_breakdown"`
}

// PrometheusExporter exports business metrics to Prometheus
type PrometheusExporter struct {
	collector          *BusinessMetricsCollector
	namespace          string
	registry           *prometheus.Registry
	ruleEvaluations    *prometheus.CounterVec
	strategyExecutions *prometheus.CounterVec
	businessOutcomes   *prometheus.CounterVec
	customMetrics      map[string]prometheus.Collector
}

// OpenTelemetryTracer provides distributed tracing for business operations
type OpenTelemetryTracer struct {
	tracer      oteltrace.Tracer
	serviceName string
	environment string
}

// StructuredLogger provides business-context-aware logging
type StructuredLogger struct {
	logger      *slog.Logger
	serviceName string
	level       slog.Level
}

// HealthCheckSystem manages health checks for all business components
type HealthCheckSystem struct {
	checks map[string]HealthCheckFunc
	mutex  sync.RWMutex
}

// HealthCheckFunc represents a function that performs a health check
type HealthCheckFunc func(ctx context.Context) HealthCheckResult

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"` // "healthy", "degraded", "unhealthy"
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	Issues    []string               `json:"issues,omitempty"`
}

// OverallHealthResult represents the overall health of the system
type OverallHealthResult struct {
	OverallStatus    string              `json:"overall_status"`
	ComponentResults []HealthCheckResult `json:"component_results"`
	CheckedAt        time.Time           `json:"checked_at"`
	Summary          HealthSummary       `json:"summary"`
}

// HealthSummary provides a summary of health check results
type HealthSummary struct {
	TotalComponents     int `json:"total_components"`
	HealthyComponents   int `json:"healthy_components"`
	DegradedComponents  int `json:"degraded_components"`
	UnhealthyComponents int `json:"unhealthy_components"`
}

// IntegratedMonitoringSystem combines all monitoring components
type IntegratedMonitoringSystem struct {
	metricsCollector *BusinessMetricsCollector
	prometheus       *PrometheusExporter
	tracer           *OpenTelemetryTracer
	logger           *StructuredLogger
	healthSystem     *HealthCheckSystem
}

// NewBusinessMetricsCollector creates a new business metrics collector
func NewBusinessMetricsCollector() *BusinessMetricsCollector {
	return &BusinessMetricsCollector{
		ruleMetrics:     make(map[string]*RuleMetrics),
		strategyMetrics: make(map[string]*StrategyMetrics),
		outcomeMetrics:  make(map[string]*OutcomeMetrics),
	}
}

// RecordRuleEvaluation records metrics for a business rule evaluation
func (bmc *BusinessMetricsCollector) RecordRuleEvaluation(policyName, ruleName string, latency time.Duration, success bool) {
	bmc.mutex.Lock()
	defer bmc.mutex.Unlock()

	key := fmt.Sprintf("%s.%s", policyName, ruleName)

	if _, exists := bmc.ruleMetrics[key]; !exists {
		bmc.ruleMetrics[key] = &RuleMetrics{
			PolicyName: policyName,
			RuleName:   ruleName,
		}
	}

	metrics := bmc.ruleMetrics[key]
	metrics.TotalEvaluations++
	metrics.TotalLatency += latency
	metrics.AverageLatency = metrics.TotalLatency / time.Duration(metrics.TotalEvaluations)
	metrics.LastEvaluation = time.Now()

	if success {
		metrics.SuccessfulEvals++
	} else {
		metrics.FailedEvals++
	}

	metrics.SuccessRate = float64(metrics.SuccessfulEvals) / float64(metrics.TotalEvaluations)
}

// RecordStrategyExecution records metrics for strategy execution
func (bmc *BusinessMetricsCollector) RecordStrategyExecution(strategyName string, latency time.Duration, itemsProcessed, successfulItems int) {
	bmc.mutex.Lock()
	defer bmc.mutex.Unlock()

	if _, exists := bmc.strategyMetrics[strategyName]; !exists {
		bmc.strategyMetrics[strategyName] = &StrategyMetrics{
			StrategyName: strategyName,
		}
	}

	metrics := bmc.strategyMetrics[strategyName]
	metrics.ExecutionCount++
	metrics.TotalItemsProcessed += itemsProcessed
	metrics.TotalSuccessful += successfulItems
	metrics.TotalLatency += latency
	metrics.AverageLatency = metrics.TotalLatency / time.Duration(metrics.ExecutionCount)
	metrics.LastExecution = time.Now()

	metrics.AverageSuccessRate = float64(metrics.TotalSuccessful) / float64(metrics.TotalItemsProcessed) * 100.0
}

// RecordBusinessOutcome records a business outcome metric
func (bmc *BusinessMetricsCollector) RecordBusinessOutcome(outcomeName string, count int, metadata map[string]interface{}) {
	bmc.mutex.Lock()
	defer bmc.mutex.Unlock()

	if _, exists := bmc.outcomeMetrics[outcomeName]; !exists {
		bmc.outcomeMetrics[outcomeName] = &OutcomeMetrics{
			OutcomeName: outcomeName,
			Metadata:    make(map[string]interface{}),
		}
	}

	metrics := bmc.outcomeMetrics[outcomeName]
	metrics.Count = count
	metrics.LastRecorded = time.Now()

	// Merge metadata
	for k, v := range metadata {
		metrics.Metadata[k] = v
	}
}

// GetAggregatedMetrics returns aggregated metrics across all dimensions
func (bmc *BusinessMetricsCollector) GetAggregatedMetrics() *AggregatedMetrics {
	bmc.mutex.RLock()
	defer bmc.mutex.RUnlock()

	// Aggregate policy metrics
	policyMetrics := make(map[string]*PolicyAggregateMetrics)

	for _, ruleMetric := range bmc.ruleMetrics {
		policyName := ruleMetric.PolicyName

		if _, exists := policyMetrics[policyName]; !exists {
			policyMetrics[policyName] = &PolicyAggregateMetrics{
				PolicyName:    policyName,
				RuleBreakdown: make(map[string]*RuleMetrics),
			}
		}

		policyAgg := policyMetrics[policyName]
		policyAgg.TotalEvaluations += ruleMetric.TotalEvaluations
		policyAgg.SuccessfulEvals += ruleMetric.SuccessfulEvals
		policyAgg.FailedEvals += ruleMetric.FailedEvals

		// Store rule breakdown
		policyAgg.RuleBreakdown[ruleMetric.RuleName] = ruleMetric
	}

	// Calculate aggregated success rates and latencies
	for _, policyAgg := range policyMetrics {
		if policyAgg.TotalEvaluations > 0 {
			policyAgg.SuccessRate = float64(policyAgg.SuccessfulEvals) / float64(policyAgg.TotalEvaluations)

			// Calculate average latency across rules
			var totalLatency time.Duration
			var totalEvals int
			for _, ruleMetric := range policyAgg.RuleBreakdown {
				totalLatency += ruleMetric.TotalLatency
				totalEvals += ruleMetric.TotalEvaluations
			}
			if totalEvals > 0 {
				policyAgg.AverageLatency = totalLatency / time.Duration(totalEvals)
			}
		}
	}

	return &AggregatedMetrics{
		PolicyMetrics:    policyMetrics,
		StrategyMetrics:  bmc.strategyMetrics,
		BusinessOutcomes: bmc.outcomeMetrics,
		CollectionTime:   time.Now(),
	}
}

// NewPrometheusExporter creates a new Prometheus metrics exporter
func NewPrometheusExporter(collector *BusinessMetricsCollector, namespace string) (*PrometheusExporter, error) {
	registry := prometheus.NewRegistry()

	ruleEvaluations := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "rule_evaluations_total",
			Help:      "Total number of business rule evaluations",
		},
		[]string{"policy", "rule", "status"},
	)

	strategyExecutions := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "strategy_executions_total",
			Help:      "Total number of strategy executions",
		},
		[]string{"strategy", "status"},
	)

	businessOutcomes := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "business_outcomes_total",
			Help:      "Total number of business outcomes",
		},
		[]string{"outcome_type"},
	)

	// Register metrics
	registry.MustRegister(ruleEvaluations)
	registry.MustRegister(strategyExecutions)
	registry.MustRegister(businessOutcomes)

	return &PrometheusExporter{
		collector:          collector,
		namespace:          namespace,
		registry:           registry,
		ruleEvaluations:    ruleEvaluations,
		strategyExecutions: strategyExecutions,
		businessOutcomes:   businessOutcomes,
		customMetrics:      make(map[string]prometheus.Collector),
	}, nil
}

// GetMetricsHandler returns the HTTP handler for Prometheus metrics
func (pe *PrometheusExporter) GetMetricsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Sync metrics from collector before serving
		pe.syncMetrics()

		handler := promhttp.HandlerFor(pe.registry, promhttp.HandlerOpts{})
		handler.ServeHTTP(w, r)
	})
}

// syncMetrics synchronizes metrics from the business collector to Prometheus
func (pe *PrometheusExporter) syncMetrics() {
	if pe.collector == nil {
		return
	}

	aggregated := pe.collector.GetAggregatedMetrics()

	// Sync rule evaluation metrics
	for _, policyMetrics := range aggregated.PolicyMetrics {
		for _, ruleMetric := range policyMetrics.RuleBreakdown {
			successLabels := prometheus.Labels{
				"policy": ruleMetric.PolicyName,
				"rule":   ruleMetric.RuleName,
				"status": "success",
			}
			failLabels := prometheus.Labels{
				"policy": ruleMetric.PolicyName,
				"rule":   ruleMetric.RuleName,
				"status": "failed",
			}

			pe.ruleEvaluations.With(successLabels).Add(float64(ruleMetric.SuccessfulEvals))
			pe.ruleEvaluations.With(failLabels).Add(float64(ruleMetric.FailedEvals))
		}
	}

	// Sync strategy execution metrics
	for _, strategyMetric := range aggregated.StrategyMetrics {
		successLabels := prometheus.Labels{
			"strategy": strategyMetric.StrategyName,
			"status":   "success",
		}

		pe.strategyExecutions.With(successLabels).Add(float64(strategyMetric.ExecutionCount))
	}

	// Sync business outcomes
	for _, outcomeMetric := range aggregated.BusinessOutcomes {
		outcomeLabels := prometheus.Labels{
			"outcome_type": outcomeMetric.OutcomeName,
		}

		pe.businessOutcomes.With(outcomeLabels).Add(float64(outcomeMetric.Count))
	}
}

// RegisterCustomMetric registers a custom business metric
func (pe *PrometheusExporter) RegisterCustomMetric(name, help, metricType string) error {

	var collector prometheus.Collector

	switch metricType {
	case "counter":
		collector = prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: pe.namespace,
				Name:      name,
				Help:      help,
			},
			[]string{"source", "type"},
		)
	case "gauge":
		collector = prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: pe.namespace,
				Name:      name,
				Help:      help,
			},
			[]string{"source", "type"},
		)
	case "histogram":
		collector = prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: pe.namespace,
				Name:      name,
				Help:      help,
			},
			[]string{"source", "type"},
		)
	default:
		return fmt.Errorf("unsupported metric type: %s", metricType)
	}

	pe.registry.MustRegister(collector)
	pe.customMetrics[name] = collector

	return nil
}

// RecordCustomMetric records a value for a custom metric
func (pe *PrometheusExporter) RecordCustomMetric(name string, value float64, labels map[string]string) error {
	collector, exists := pe.customMetrics[name]

	if !exists {
		return fmt.Errorf("custom metric not found: %s", name)
	}

	switch c := collector.(type) {
	case *prometheus.CounterVec:
		c.With(labels).Add(value)
	case *prometheus.GaugeVec:
		c.With(labels).Set(value)
	case *prometheus.HistogramVec:
		c.With(labels).Observe(value)
	}

	return nil
}

// NewOpenTelemetryTracer creates a new OpenTelemetry tracer
func NewOpenTelemetryTracer(serviceName, environment string) (*OpenTelemetryTracer, error) {
	// Set up a basic tracer provider (no external exporter to avoid deprecated Jaeger usage)
	tp := trace.NewTracerProvider(
		trace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(serviceName),
			semconv.DeploymentEnvironmentKey.String(environment),
		)),
	)
	otel.SetTracerProvider(tp)
	tracer := otel.Tracer(serviceName)
	return &OpenTelemetryTracer{tracer: tracer, serviceName: serviceName, environment: environment}, nil
}

// StartBusinessOperation starts a new trace for a business operation
func (ott *OpenTelemetryTracer) StartBusinessOperation(ctx context.Context, operationName string, attributes map[string]interface{}) (context.Context, oteltrace.Span) {
	attrs := make([]attribute.KeyValue, 0, len(attributes))
	for k, v := range attributes {
		attrs = append(attrs, attribute.String(k, fmt.Sprintf("%v", v)))
	}

	return ott.tracer.Start(ctx, operationName, oteltrace.WithAttributes(attrs...))
}

// RecordRuleEvaluation records rule evaluation within a trace
func (ott *OpenTelemetryTracer) RecordRuleEvaluation(ctx context.Context, policyName, ruleName string, latency time.Duration, success bool, attributes map[string]interface{}) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent("rule_evaluation", oteltrace.WithAttributes(
			attribute.String("policy", policyName),
			attribute.String("rule", ruleName),
			attribute.Int64("latency_microseconds", latency.Microseconds()),
			attribute.Bool("success", success),
		))

		for k, v := range attributes {
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", v)))
		}
	}
}

// RecordStrategyExecution records strategy execution within a trace
func (ott *OpenTelemetryTracer) RecordStrategyExecution(ctx context.Context, strategyName string, latency time.Duration, attributes map[string]interface{}) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.AddEvent("strategy_execution", oteltrace.WithAttributes(
			attribute.String("strategy", strategyName),
			attribute.Int64("latency_microseconds", latency.Microseconds()),
		))

		for k, v := range attributes {
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", v)))
		}
	}
}

// RecordError records an error within a trace
func (ott *OpenTelemetryTracer) RecordError(ctx context.Context, errorType string, err error, attributes map[string]interface{}) {
	span := oteltrace.SpanFromContext(ctx)
	if span.IsRecording() {
		span.RecordError(err)
		span.SetAttributes(
			attribute.String("error.type", errorType),
			attribute.String("error.message", err.Error()),
		)

		for k, v := range attributes {
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", v)))
		}
	}
}

// FinishBusinessOperation finishes a business operation trace
func (ott *OpenTelemetryTracer) FinishBusinessOperation(span oteltrace.Span, success bool, attributes map[string]interface{}) {
	if span.IsRecording() {
		span.SetAttributes(attribute.Bool("operation.success", success))

		for k, v := range attributes {
			span.SetAttributes(attribute.String(k, fmt.Sprintf("%v", v)))
		}

		if success {
			span.SetStatus(codes.Ok, "operation completed successfully")
		} else {
			span.SetStatus(codes.Error, "operation failed")
		}
	}

	span.End()
}

// NewStructuredLogger creates a new structured logger for business operations
func NewStructuredLogger(level, format, serviceName string) (*StructuredLogger, error) {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "info":
		logLevel = slog.LevelInfo
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	var handler slog.Handler
	opts := &slog.HandlerOptions{
		Level: logLevel,
	}

	if format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	logger := slog.New(handler).With("service", serviceName)

	return &StructuredLogger{
		logger:      logger,
		serviceName: serviceName,
		level:       logLevel,
	}, nil
}

// LogRuleEvaluation logs business rule evaluation
func (sl *StructuredLogger) LogRuleEvaluation(policyName, ruleName string, context map[string]interface{}) {
	attrs := make([]any, 0, len(context)*2+4)
	attrs = append(attrs, "event_type", "rule_evaluation")
	attrs = append(attrs, "policy", policyName)
	attrs = append(attrs, "rule", ruleName)

	for k, v := range context {
		attrs = append(attrs, k, v)
	}

	sl.logger.Info("Rule evaluation", attrs...)
}

// LogStrategyExecution logs strategy execution
func (sl *StructuredLogger) LogStrategyExecution(strategyName string, context map[string]interface{}) {
	attrs := make([]any, 0, len(context)*2+2)
	attrs = append(attrs, "event_type", "strategy_execution")
	attrs = append(attrs, "strategy", strategyName)

	for k, v := range context {
		attrs = append(attrs, k, v)
	}

	sl.logger.Info("Strategy execution", attrs...)
}

// LogConfigurationChange logs configuration changes
func (sl *StructuredLogger) LogConfigurationChange(changeType string, context map[string]interface{}) {
	attrs := make([]any, 0, len(context)*2+2)
	attrs = append(attrs, "event_type", "configuration_change")
	attrs = append(attrs, "change_type", changeType)

	for k, v := range context {
		attrs = append(attrs, k, v)
	}

	sl.logger.Info("Configuration change", attrs...)
}

// LogBusinessOutcome logs business outcomes
func (sl *StructuredLogger) LogBusinessOutcome(outcomeName string, context map[string]interface{}) {
	attrs := make([]any, 0, len(context)*2+2)
	attrs = append(attrs, "event_type", "business_outcome")
	attrs = append(attrs, "outcome", outcomeName)

	for k, v := range context {
		attrs = append(attrs, k, v)
	}

	sl.logger.Info("Business outcome", attrs...)
}

// LogDebug logs debug information
func (sl *StructuredLogger) LogDebug(message string, context map[string]interface{}) {
	if sl.level <= slog.LevelDebug {
		attrs := make([]any, 0, len(context)*2)
		for k, v := range context {
			attrs = append(attrs, k, v)
		}
		sl.logger.Debug(message, attrs...)
	}
}

// LogWarn logs warning information
func (sl *StructuredLogger) LogWarn(message string, context map[string]interface{}) {
	attrs := make([]any, 0, len(context)*2)
	for k, v := range context {
		attrs = append(attrs, k, v)
	}
	sl.logger.Warn(message, attrs...)
}

// LogError logs error information
func (sl *StructuredLogger) LogError(message string, err error, context map[string]interface{}) {
	attrs := make([]any, 0, len(context)*2+2)
	attrs = append(attrs, "error", err)

	for k, v := range context {
		attrs = append(attrs, k, v)
	}

	sl.logger.Error(message, attrs...)
}

// NewHealthCheckSystem creates a new health check system
func NewHealthCheckSystem() *HealthCheckSystem {
	return &HealthCheckSystem{
		checks: make(map[string]HealthCheckFunc),
	}
}

// RegisterCheck registers a health check function
func (hcs *HealthCheckSystem) RegisterCheck(name string, checkFunc HealthCheckFunc) error {
	hcs.mutex.Lock()
	defer hcs.mutex.Unlock()

	hcs.checks[name] = checkFunc
	return nil
}

// CheckHealth performs all registered health checks
func (hcs *HealthCheckSystem) CheckHealth(ctx context.Context) *OverallHealthResult {
	hcs.mutex.RLock()
	checks := make(map[string]HealthCheckFunc)
	for name, checkFunc := range hcs.checks {
		checks[name] = checkFunc
	}
	hcs.mutex.RUnlock()

	results := make([]HealthCheckResult, 0, len(checks))
	summary := HealthSummary{TotalComponents: len(checks)}

	for name, checkFunc := range checks {
		result := checkFunc(ctx)
		result.Name = name // Ensure name is used
		results = append(results, result)

		switch result.Status {
		case "healthy":
			summary.HealthyComponents++
		case "degraded":
			summary.DegradedComponents++
		case "unhealthy":
			summary.UnhealthyComponents++
		}
	}

	// Determine overall status
	overallStatus := "healthy"
	if summary.UnhealthyComponents > 0 {
		overallStatus = "unhealthy"
	} else if summary.DegradedComponents > 0 {
		overallStatus = "degraded"
	}

	return &OverallHealthResult{
		OverallStatus:    overallStatus,
		ComponentResults: results,
		CheckedAt:        time.Now(),
		Summary:          summary,
	}
}

// GetHealthHandler returns an HTTP handler for health checks
func (hcs *HealthCheckSystem) GetHealthHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		health := hcs.CheckHealth(ctx)

		w.Header().Set("Content-Type", "application/json")

		var statusCode int
		switch health.OverallStatus {
		case "unhealthy":
			statusCode = http.StatusServiceUnavailable
		case "degraded":
			statusCode = http.StatusOK
		default:
			statusCode = http.StatusOK
		}

		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(health); err != nil {
			// Best-effort logging to stderr; avoid dependency on StructuredLogger here
			fmt.Fprintf(os.Stderr, "health encode error: %v\n", err)
		}
	})
}

// NewIntegratedMonitoringSystem creates a comprehensive monitoring system
func NewIntegratedMonitoringSystem(
	metricsCollector *BusinessMetricsCollector,
	prometheus *PrometheusExporter,
	tracer *OpenTelemetryTracer,
	logger *StructuredLogger,
	healthSystem *HealthCheckSystem,
) (*IntegratedMonitoringSystem, error) {
	return &IntegratedMonitoringSystem{
		metricsCollector: metricsCollector,
		prometheus:       prometheus,
		tracer:           tracer,
		logger:           logger,
		healthSystem:     healthSystem,
	}, nil
}

// MonitorRuleEvaluation monitors a rule evaluation with full observability
func (ims *IntegratedMonitoringSystem) MonitorRuleEvaluation(ctx context.Context, policyName, ruleName string, evalFunc func() error) error {
	start := time.Now()

	// Start tracing if available
	if ims.tracer != nil {
		var span oteltrace.Span
		ctx, span = ims.tracer.StartBusinessOperation(ctx, "rule_evaluation", map[string]interface{}{
			"policy": policyName,
			"rule":   ruleName,
		})
		defer span.End()
	}

	// Execute the rule evaluation
	err := evalFunc()
	latency := time.Since(start)
	success := err == nil

	// Record metrics
	if ims.metricsCollector != nil {
		ims.metricsCollector.RecordRuleEvaluation(policyName, ruleName, latency, success)
	}

	// Log the evaluation
	if ims.logger != nil {
		ims.logger.LogRuleEvaluation(policyName, ruleName, map[string]interface{}{
			"latency_ms": latency.Milliseconds(),
			"success":    success,
		})
	}

	// Record error if present
	if err != nil && ims.tracer != nil {
		ims.tracer.RecordError(ctx, "rule_evaluation_error", err, map[string]interface{}{
			"policy": policyName,
			"rule":   ruleName,
		})
	}

	return err
}

// MonitorStrategyExecution monitors strategy execution with full observability
func (ims *IntegratedMonitoringSystem) MonitorStrategyExecution(ctx context.Context, strategyName string, execFunc func() (int, int, error)) error {
	start := time.Now()

	// Start tracing if available
	if ims.tracer != nil {
		var span oteltrace.Span
		// ignore derived ctx to avoid lint ineffassign warning as we don't propagate further here
		_, span = ims.tracer.StartBusinessOperation(ctx, "strategy_execution", map[string]interface{}{
			"strategy": strategyName,
		})
		defer span.End()
	}

	// Execute the strategy
	itemsProcessed, successfulItems, err := execFunc()
	latency := time.Since(start)

	// Record metrics
	if ims.metricsCollector != nil {
		ims.metricsCollector.RecordStrategyExecution(strategyName, latency, itemsProcessed, successfulItems)
	}

	// Log the execution
	if ims.logger != nil {
		ims.logger.LogStrategyExecution(strategyName, map[string]interface{}{
			"latency_ms":       latency.Milliseconds(),
			"items_processed":  itemsProcessed,
			"successful_items": successfulItems,
			"success_rate":     float64(successfulItems) / float64(itemsProcessed),
		})
	}

	return err
}

// MonitorConfigurationChange monitors configuration changes
// MonitorConfigurationChange logs a configuration change event. Runtime config types were internalized; we accept generic version strings.
func (ims *IntegratedMonitoringSystem) MonitorConfigurationChange(changeType, oldVersion, newVersion string) {
	if ims.logger != nil {
		ims.logger.LogConfigurationChange(changeType, map[string]interface{}{
			"old_version": oldVersion,
			"new_version": newVersion,
			"change_type": changeType,
		})
	}
}

// MonitorComposedStrategyExecution monitors composed strategy execution
func (ims *IntegratedMonitoringSystem) MonitorComposedStrategyExecution(ctx context.Context, composedStrategies *strategies.ComposedStrategies, execFunc func() (int, int, error)) error {
	// Monitor as a combined strategy execution
	return ims.MonitorStrategyExecution(ctx, "composed_strategy", execFunc)
}

// GetAggregatedMetrics returns aggregated metrics from the collector
func (ims *IntegratedMonitoringSystem) GetAggregatedMetrics() *AggregatedMetrics {
	if ims.metricsCollector != nil {
		return ims.metricsCollector.GetAggregatedMetrics()
	}
	return &AggregatedMetrics{
		CollectionTime: time.Now(),
	}
}

// GetOverallHealth returns the overall system health
func (ims *IntegratedMonitoringSystem) GetOverallHealth(ctx context.Context) *OverallHealthResult {
	if ims.healthSystem != nil {
		return ims.healthSystem.CheckHealth(ctx)
	}
	return &OverallHealthResult{
		OverallStatus: "unknown",
		CheckedAt:     time.Now(),
	}
}
