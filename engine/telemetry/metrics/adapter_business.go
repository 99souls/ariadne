package metrics

import (
    "sync/atomic"
    "time"

    legacy "ariadne/packages/engine/monitoring"
)

// BusinessCollectorAdapter exposes legacy BusinessMetricsCollector data via the new
// metrics Provider without re-registering the legacy PrometheusExporter metrics.
// It performs a periodic snapshot and updates counters/gauges accordingly.
type BusinessCollectorAdapter struct {
    legacy *legacy.BusinessMetricsCollector
    prov   Provider
    // instruments
    ruleEvalCounter     Counter   // labels: policy, rule, status (success|failed)
    strategyExecCounter Counter   // labels: strategy, status (only success for now)
    outcomeCounter      Counter   // labels: outcome_type

    lastSync atomic.Pointer[time.Time]
}

// NewBusinessCollectorAdapter constructs the adapter; if provider is nil returns nil.
func NewBusinessCollectorAdapter(legacy *legacy.BusinessMetricsCollector, p Provider) *BusinessCollectorAdapter {
    if legacy == nil || p == nil { return nil }
    adapter := &BusinessCollectorAdapter{legacy: legacy, prov: p}
    adapter.ruleEvalCounter = p.NewCounter(CounterOpts{ CommonOpts: CommonOpts{ Namespace: "ariadne", Subsystem: "business", Name: "rule_evaluations_total", Help: "Total number of business rule evaluations", Labels: []string{"policy", "rule", "status"}}})
    adapter.strategyExecCounter = p.NewCounter(CounterOpts{ CommonOpts: CommonOpts{ Namespace: "ariadne", Subsystem: "business", Name: "strategy_executions_total", Help: "Total number of strategy executions", Labels: []string{"strategy", "status"}}})
    adapter.outcomeCounter = p.NewCounter(CounterOpts{ CommonOpts: CommonOpts{ Namespace: "ariadne", Subsystem: "business", Name: "business_outcomes_total", Help: "Total number of business outcomes", Labels: []string{"outcome_type"}}})
    return adapter
}

// SyncOnce snapshots the legacy collector and updates counters with cumulative values.
// It assumes legacy metrics are monotonically increasing. We simply add the cumulative totals
// each sync, so callers should ensure SyncOnce is invoked only once (e.g. at export) or we
// would over-count. A future enhancement could store prior snapshot and compute deltas.
func (a *BusinessCollectorAdapter) SyncOnce() {
    if a == nil || a.legacy == nil { return }
    snap := a.legacy.GetAggregatedMetrics()
    if snap == nil { return }
    for _, policy := range snap.PolicyMetrics {
        for _, rule := range policy.RuleBreakdown {
            if rule.SuccessfulEvals > 0 {
                a.ruleEvalCounter.Inc(float64(rule.SuccessfulEvals), policy.PolicyName, rule.RuleName, "success")
            }
            if rule.FailedEvals > 0 {
                a.ruleEvalCounter.Inc(float64(rule.FailedEvals), policy.PolicyName, rule.RuleName, "failed")
            }
        }
    }
    for _, strat := range snap.StrategyMetrics {
        if strat.ExecutionCount > 0 {
            a.strategyExecCounter.Inc(float64(strat.ExecutionCount), strat.StrategyName, "success")
        }
    }
    for _, outcome := range snap.BusinessOutcomes {
        if outcome.Count > 0 {
            a.outcomeCounter.Inc(float64(outcome.Count), outcome.OutcomeName)
        }
    }
    now := time.Now(); a.lastSync.Store(&now)
}
