package health

import (
	"context"
	"sync"
	"time"
)

// Status enumerates health states.
type Status string

const (
	StatusUnknown   Status = "unknown"
	StatusHealthy   Status = "healthy"
	StatusDegraded  Status = "degraded"
	StatusUnhealthy Status = "unhealthy"
)

// ProbeResult represents one subsystem evaluation.
type ProbeResult struct {
	Name      string    `json:"name"`
	Status    Status    `json:"status"`
	Detail    string    `json:"detail,omitempty"`
	CheckedAt time.Time `json:"checked_at"`
}

// Snapshot aggregates probe results and overall rollup.
type Snapshot struct {
	Overall   Status        `json:"overall"`
	Probes    []ProbeResult `json:"probes"`
	Generated time.Time     `json:"generated"`
	TTL       time.Duration `json:"ttl"`
}

// Probe defines a callable returning a ProbeResult.
type Probe interface {
	Check(ctx context.Context) ProbeResult
}

type ProbeFunc func(ctx context.Context) ProbeResult

func (f ProbeFunc) Check(ctx context.Context) ProbeResult { return f(ctx) }

type Evaluator struct {
	probes []Probe
	ttl    time.Duration
	mu     sync.RWMutex
	cached Snapshot
}

// NewEvaluator creates an evaluator with the provided TTL for caching results.
func NewEvaluator(ttl time.Duration, probes ...Probe) *Evaluator {
	if ttl <= 0 {
		ttl = 2 * time.Second
	}
	return &Evaluator{probes: probes, ttl: ttl}
}

// Register adds more probes.
func (e *Evaluator) Register(p Probe) {
	if p == nil {
		return
	}
	e.mu.Lock()
	e.probes = append(e.probes, p)
	e.mu.Unlock()
}

// Evaluate returns a cached snapshot if within TTL else recomputes.
func (e *Evaluator) Evaluate(ctx context.Context) Snapshot {
	e.mu.RLock()
	cached := e.cached
	if cached.Generated.Add(e.ttl).After(time.Now()) {
		e.mu.RUnlock()
		return cached
	}
	e.mu.RUnlock()
	e.mu.Lock()
	defer e.mu.Unlock()
	// double-check within lock
	if e.cached.Generated.Add(e.ttl).After(time.Now()) {
		return e.cached
	}
	results := make([]ProbeResult, 0, len(e.probes))
	overall := StatusHealthy
	now := time.Now()
	for _, p := range e.probes {
		if p == nil {
			continue
		}
		pr := p.Check(ctx)
		if pr.CheckedAt.IsZero() {
			pr.CheckedAt = now
		}
		results = append(results, pr)
		switch pr.Status {
		case StatusUnhealthy:
			overall = StatusUnhealthy
		case StatusDegraded:
			if overall != StatusUnhealthy {
				overall = StatusDegraded
			}
		}
	}
	if len(results) == 0 {
		overall = StatusUnknown
	}
	snap := Snapshot{Overall: overall, Probes: results, Generated: now, TTL: e.ttl}
	e.cached = snap
	return snap
}

// ForceInvalidate clears the cached snapshot forcing next Evaluate to recompute.
// Intended for tests.
func (e *Evaluator) ForceInvalidate() {
	e.mu.Lock()
	e.cached.Generated = time.Time{}
	e.mu.Unlock()
}

// Helper constructors for common statuses.
func Healthy(name string) ProbeResult {
	return ProbeResult{Name: name, Status: StatusHealthy, CheckedAt: time.Now()}
}
func Degraded(name, detail string) ProbeResult {
	return ProbeResult{Name: name, Status: StatusDegraded, Detail: detail, CheckedAt: time.Now()}
}
func Unhealthy(name, detail string) ProbeResult {
	return ProbeResult{Name: name, Status: StatusUnhealthy, Detail: detail, CheckedAt: time.Now()}
}
func Unknown(name, detail string) ProbeResult {
	return ProbeResult{Name: name, Status: StatusUnknown, Detail: detail, CheckedAt: time.Now()}
}
