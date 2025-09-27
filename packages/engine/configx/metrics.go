package configx

import "sync/atomic"

// MetricsRecorder defines observability counters used by the configx subsystem.
type MetricsRecorder interface {
    IncApplySuccess()
    IncApplyFailure()
    IncRollback()
    SetActiveVersion(v int64)
}

// InMemoryMetrics is a thread-safe atomic implementation for tests.
type InMemoryMetrics struct {
    ApplySuccess int64
    ApplyFailure int64
    Rollbacks    int64
    ActiveVer    int64
}

func (m *InMemoryMetrics) IncApplySuccess() { atomic.AddInt64(&m.ApplySuccess, 1) }
func (m *InMemoryMetrics) IncApplyFailure() { atomic.AddInt64(&m.ApplyFailure, 1) }
func (m *InMemoryMetrics) IncRollback()     { atomic.AddInt64(&m.Rollbacks, 1) }
func (m *InMemoryMetrics) SetActiveVersion(v int64) { atomic.StoreInt64(&m.ActiveVer, v) }