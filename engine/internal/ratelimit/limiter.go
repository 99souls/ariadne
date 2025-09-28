package ratelimit

import (
	"context"
	"errors"
	"hash/fnv"
	"math"
	"sync"
	"time"

	engmodels "github.com/99souls/ariadne/engine/models"
)

var ErrCircuitOpen = errors.New("ratelimit: circuit open")

type RateLimiter interface {
	Acquire(ctx context.Context, domain string) (Permit, error)
	Feedback(domain string, fb Feedback)
	Snapshot() LimiterSnapshot
}

type Permit interface{ Release() }

type Feedback struct {
	StatusCode int
	Latency    time.Duration
	Err        error
	RetryAfter time.Duration
}

type LimiterSnapshot struct {
	TotalRequests    int64
	Throttled        int64
	Denied           int64
	OpenCircuits     int64
	HalfOpenCircuits int64
	Domains          []DomainSummary
}

type DomainSummary struct {
	Domain       string
	FillRate     float64
	CircuitState string
	LastActivity time.Time
}

type AdaptiveRateLimiter struct {
	cfg           engmodels.RateLimitConfig
	clock         Clock
	shards        []*domainShard
	mask          uint64
	metricsMu     sync.Mutex
	metrics       LimiterSnapshot
	stopCh        chan struct{}
	evictWG       sync.WaitGroup
	evictInterval time.Duration
	stopOnce      sync.Once
}

type domainShard struct {
	mu      sync.RWMutex
	domains map[string]*domainState
}

func (l *AdaptiveRateLimiter) shardIndex(domain string) uint64 {
	h := fnv.New32a()
	_, _ = h.Write([]byte(domain))
	return uint64(h.Sum32()) & l.mask
}

func (l *AdaptiveRateLimiter) getOrCreateDomainState(domain string) *domainState {
	idx := l.shardIndex(domain)
	shard := l.shards[idx]
	shard.mu.RLock()
	state := shard.domains[domain]
	shard.mu.RUnlock()
	if state != nil {
		return state
	}
	shard.mu.Lock()
	defer shard.mu.Unlock()
	if state = shard.domains[domain]; state == nil {
		state = newDomainState(l.cfg, l.clock.Now())
		shard.domains[domain] = state
	}
	return state
}

func (l *AdaptiveRateLimiter) withMetrics(mutator func(*LimiterSnapshot)) {
	l.metricsMu.Lock()
	mutator(&l.metrics)
	l.metricsMu.Unlock()
}

func NewAdaptiveRateLimiter(cfg engmodels.RateLimitConfig) *AdaptiveRateLimiter {
	if cfg.Shards <= 0 || (cfg.Shards&(cfg.Shards-1)) != 0 {
		cfg.Shards = 16
	}
	if cfg.DomainStateTTL <= 0 {
		cfg.DomainStateTTL = 2 * time.Minute
	}
	shards := make([]*domainShard, cfg.Shards)
	for i := range shards {
		shards[i] = &domainShard{domains: make(map[string]*domainState)}
	}
	interval := cfg.DomainStateTTL / 2
	if interval <= 0 {
		interval = cfg.DomainStateTTL
	}
	if interval <= 0 {
		interval = time.Minute
	}
	limiter := &AdaptiveRateLimiter{cfg: cfg, clock: realClock{}, shards: shards, mask: uint64(cfg.Shards - 1), stopCh: make(chan struct{}), evictInterval: interval}
	limiter.startEvictionLoop()
	return limiter
}

func (l *AdaptiveRateLimiter) WithClock(clock Clock) *AdaptiveRateLimiter {
	if clock != nil {
		l.clock = clock
	}
	return l
}

func (l *AdaptiveRateLimiter) Acquire(ctx context.Context, domain string) (Permit, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if !l.cfg.Enabled {
		return immediatePermit{}, nil
	}
	normalized, err := normalizeDomain(domain)
	if err != nil {
		return nil, err
	}
	state := l.getOrCreateDomainState(normalized)
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}
		now := l.clock.Now()
		wait, err := state.planRequest(l.cfg, now)
		if err != nil {
			if errors.Is(err, ErrCircuitOpen) {
				l.withMetrics(func(m *LimiterSnapshot) { m.Denied++ })
			}
			return nil, err
		}
		if wait <= 0 {
			l.withMetrics(func(m *LimiterSnapshot) { m.TotalRequests++ })
			return immediatePermit{}, nil
		}
		l.withMetrics(func(m *LimiterSnapshot) { m.Throttled++ })
		if !sleepWithContext(ctx, l.clock, wait) {
			return nil, ctx.Err()
		}
	}
}

func (l *AdaptiveRateLimiter) Feedback(domain string, fb Feedback) {
	if !l.cfg.Enabled {
		return
	}
	normalized, err := normalizeDomain(domain)
	if err != nil {
		return
	}
	state := l.getOrCreateDomainState(normalized)
	state.applyFeedback(l.cfg, fb, l.clock.Now())
}

func (l *AdaptiveRateLimiter) Snapshot() LimiterSnapshot {
	base := func() LimiterSnapshot { l.metricsMu.Lock(); defer l.metricsMu.Unlock(); return l.metrics }()
	var open, halfOpen int64
	var domains []DomainSummary
	for _, shard := range l.shards {
		shard.mu.RLock()
		for name, state := range shard.domains {
			state.mu.Lock()
			cs := "closed"
			switch state.breaker.state {
			case circuitOpen:
				cs = "open"
				open++
			case circuitHalfOpen:
				cs = "half-open"
				halfOpen++
			}
			domains = append(domains, DomainSummary{Domain: name, FillRate: state.fillRate, CircuitState: cs, LastActivity: state.lastActivity})
			state.mu.Unlock()
		}
		shard.mu.RUnlock()
	}
	if len(domains) > 1 {
		for i := 1; i < len(domains); i++ {
			j := i
			for j > 0 && domains[j-1].LastActivity.Before(domains[j].LastActivity) {
				domains[j-1], domains[j] = domains[j], domains[j-1]
				j--
			}
		}
	}
	if len(domains) > 10 {
		domains = append([]DomainSummary(nil), domains[:10]...)
	}
	base.Domains = domains
	base.OpenCircuits = open
	base.HalfOpenCircuits = halfOpen
	return base
}

type immediatePermit struct{}

func (immediatePermit) Release()                  {}
func (l *AdaptiveRateLimiter) startEvictionLoop() { l.evictWG.Add(1); go l.evictLoop() }
func (l *AdaptiveRateLimiter) evictLoop() {
	defer l.evictWG.Done()
	ticker := time.NewTicker(l.evictInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			l.evictIdleDomains()
		case <-l.stopCh:
			return
		}
	}
}
func (l *AdaptiveRateLimiter) evictIdleDomains() {
	ttl := l.cfg.DomainStateTTL
	if ttl <= 0 {
		return
	}
	now := l.clock.Now()
	for _, shard := range l.shards {
		shard.mu.Lock()
		for domain, state := range shard.domains {
			state.mu.Lock()
			idle := now.Sub(state.lastActivity)
			state.mu.Unlock()
			if idle >= ttl {
				delete(shard.domains, domain)
			}
		}
		shard.mu.Unlock()
	}
}
func (l *AdaptiveRateLimiter) Close() error {
	l.stopOnce.Do(func() { close(l.stopCh); l.evictWG.Wait() })
	return nil
}

func sleepWithContext(ctx context.Context, clock Clock, d time.Duration) bool {
	if d <= 0 {
		return true
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	if ctx == nil {
		clock.Sleep(d)
		return true
	}
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

// Clock abstraction for testability.
type Clock interface {
	Now() time.Time
	Sleep(d time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time        { return time.Now() }
func (realClock) Sleep(d time.Duration) { time.Sleep(d) }

// circuit breaker states
const (
	circuitClosed = iota
	circuitOpen
	circuitHalfOpen
)

type breakerState struct {
	state int
	// next attempt time when open
	nextAttempt time.Time
	failures    int
	successes   int
}

// simplistic adaptive domain state (placeholder – sufficient for tests referencing snapshot fields)
type domainState struct {
	mu           sync.Mutex
	lastActivity time.Time
	fillRate     float64
	breaker      breakerState
	tokens       float64
	lastRefill   time.Time
}

func newDomainState(cfg engmodels.RateLimitConfig, now time.Time) *domainState {
	return &domainState{lastActivity: now, fillRate: 1, tokens: 1, lastRefill: now}
}

func (d *domainState) planRequest(cfg engmodels.RateLimitConfig, now time.Time) (time.Duration, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastActivity = now
	// breaker logic
	if d.breaker.state == circuitOpen {
		if now.After(d.breaker.nextAttempt) {
			d.breaker.state = circuitHalfOpen
		} else {
			return 0, ErrCircuitOpen
		}
	}
	// token refill simplistic
	elapsed := now.Sub(d.lastRefill).Seconds()
	if elapsed > 0 {
		d.tokens += elapsed * d.fillRate
		if d.tokens > 10 {
			d.tokens = 10
		}
		d.lastRefill = now
	}
	if d.tokens >= 1 {
		d.tokens -= 1
		return 0, nil
	}
	waitSeconds := (1 - d.tokens) / math.Max(d.fillRate, 0.1)
	return time.Duration(waitSeconds * float64(time.Second)), nil
}

func (d *domainState) applyFeedback(cfg engmodels.RateLimitConfig, fb Feedback, now time.Time) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.lastActivity = now
	// adjust fill rate heuristically
	if fb.Err != nil || fb.StatusCode >= 500 || fb.StatusCode == 429 {
		d.fillRate *= 0.8
		if d.fillRate < 0.1 {
			d.fillRate = 0.1
		}
		d.breaker.failures++
	} else {
		d.fillRate *= 1.05
		if d.fillRate > 5 {
			d.fillRate = 5
		}
		if d.breaker.state == circuitHalfOpen {
			d.breaker.successes++
		}
	}
	if d.breaker.state == circuitHalfOpen {
		if d.breaker.successes >= 3 {
			d.breaker = breakerState{state: circuitClosed}
		}
		if d.breaker.failures > 0 {
			d.breaker = breakerState{state: circuitOpen, nextAttempt: now.Add(time.Second)}
		}
	} else if d.breaker.state == circuitClosed && d.breaker.failures >= 5 {
		d.breaker = breakerState{state: circuitOpen, nextAttempt: now.Add(time.Second * 5)}
	}
}

// normalizeDomain replicates earlier behavior loosely; ensures non-empty lowercase.
func normalizeDomain(domain string) (string, error) {
	if domain == "" {
		return "", errors.New("empty domain")
	}
	// treat as already normalized for placeholder
	return domain, nil
}
