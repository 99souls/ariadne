package ratelimit

import (
	"context"
	"errors"
	"hash/fnv"
	"site-scraper/pkg/models"
	"sync"
	"time"
)

// ErrCircuitOpen indicates that the rate limiter's circuit breaker is open for the domain.
var ErrCircuitOpen = errors.New("ratelimit: circuit open")

// RateLimiter mediates request admission per domain.
type RateLimiter interface {
	Acquire(ctx context.Context, domain string) (Permit, error)
	Feedback(domain string, fb Feedback)
	Snapshot() LimiterSnapshot
}

// Permit represents an acquired slot for request execution.
type Permit interface {
	Release()
}

// Feedback provides response characteristics for adaptive algorithms.
type Feedback struct {
	StatusCode int
	Latency    time.Duration
	Err        error
	RetryAfter time.Duration
}

// LimiterSnapshot exposes summary statistics for observability.
type LimiterSnapshot struct {
	TotalRequests    int64
	Throttled        int64
	Denied           int64
	OpenCircuits     int64
	HalfOpenCircuits int64
}

// AdaptiveRateLimiter implements intelligent per-domain adaptive rate limiting.
type AdaptiveRateLimiter struct {
	cfg   models.RateLimitConfig
	clock Clock

	shards []*domainShard
	mask   uint64

	metricsMu sync.Mutex
	metrics   LimiterSnapshot

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

// NewAdaptiveRateLimiter constructs a limiter with the provided configuration.
func NewAdaptiveRateLimiter(cfg models.RateLimitConfig) *AdaptiveRateLimiter {
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

	limiter := &AdaptiveRateLimiter{
		cfg:           cfg,
		clock:         realClock{},
		shards:        shards,
		mask:          uint64(cfg.Shards - 1),
		stopCh:        make(chan struct{}),
		evictInterval: interval,
	}

	limiter.startEvictionLoop()

	return limiter
}

// WithClock injects a custom clock (primarily for testing).
func (l *AdaptiveRateLimiter) WithClock(clock Clock) *AdaptiveRateLimiter {
	if clock != nil {
		l.clock = clock
	}
	return l
}

// Acquire currently returns an immediate permit placeholder.
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

// Feedback updates adaptive state based on response characteristics.
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

// Snapshot returns aggregated limiter metrics.
func (l *AdaptiveRateLimiter) Snapshot() LimiterSnapshot {
	base := func() LimiterSnapshot {
		l.metricsMu.Lock()
		defer l.metricsMu.Unlock()
		return l.metrics
	}()

	var open int64
	var halfOpen int64

	for _, shard := range l.shards {
		shard.mu.RLock()
		for _, state := range shard.domains {
			state.mu.Lock()
			switch state.breaker.state {
			case circuitOpen:
				open++
			case circuitHalfOpen:
				halfOpen++
			}
			state.mu.Unlock()
		}
		shard.mu.RUnlock()
	}

	base.OpenCircuits = open
	base.HalfOpenCircuits = halfOpen
	return base
}

type immediatePermit struct{}

func (immediatePermit) Release() {}

func (l *AdaptiveRateLimiter) startEvictionLoop() {
	l.evictWG.Add(1)
	go l.evictLoop()
}

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

// Close stops background goroutines and releases resources.
func (l *AdaptiveRateLimiter) Close() error {
	l.stopOnce.Do(func() {
		close(l.stopCh)
		l.evictWG.Wait()
	})
	return nil
}

func sleepWithContext(ctx context.Context, clock Clock, d time.Duration) bool {
	if d <= 0 {
		return true
	}

	if ctx != nil {
		select {
		case <-ctx.Done():
			return false
		default:
		}
	}

	clock.Sleep(d)

	if ctx != nil {
		select {
		case <-ctx.Done():
			return false
		default:
		}
	}

	return true
}
