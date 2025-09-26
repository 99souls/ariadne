package ratelimit

// (migrated from internal/ratelimit/limiter.go)
// NOTE: Original file kept as forwarding shim under internal/ratelimit during migration.

import (
    "context"
    "errors"
    "hash/fnv"
    engmodels "site-scraper/packages/engine/models"
    "sync"
    "time"
)

var ErrCircuitOpen = errors.New("ratelimit: circuit open")

type RateLimiter interface {
	Acquire(ctx context.Context, domain string) (Permit, error)
	Feedback(domain string, fb Feedback)
	Snapshot() LimiterSnapshot
}

type Permit interface { Release() }

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
    cfg   engmodels.RateLimitConfig
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
	mu sync.RWMutex
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
	shard.mu.RLock(); state := shard.domains[domain]; shard.mu.RUnlock()
	if state != nil { return state }
	shard.mu.Lock(); defer shard.mu.Unlock()
	if state = shard.domains[domain]; state == nil { state = newDomainState(l.cfg, l.clock.Now()); shard.domains[domain] = state }
	return state
}

func (l *AdaptiveRateLimiter) withMetrics(mutator func(*LimiterSnapshot)) { l.metricsMu.Lock(); mutator(&l.metrics); l.metricsMu.Unlock() }

func NewAdaptiveRateLimiter(cfg engmodels.RateLimitConfig) *AdaptiveRateLimiter {
	if cfg.Shards <= 0 || (cfg.Shards&(cfg.Shards-1)) != 0 { cfg.Shards = 16 }
	if cfg.DomainStateTTL <= 0 { cfg.DomainStateTTL = 2 * time.Minute }
	shards := make([]*domainShard, cfg.Shards)
	for i := range shards { shards[i] = &domainShard{domains: make(map[string]*domainState)} }
	interval := cfg.DomainStateTTL / 2
	if interval <= 0 { interval = cfg.DomainStateTTL }
	if interval <= 0 { interval = time.Minute }
	limiter := &AdaptiveRateLimiter{cfg: cfg, clock: realClock{}, shards: shards, mask: uint64(cfg.Shards - 1), stopCh: make(chan struct{}), evictInterval: interval}
	limiter.startEvictionLoop(); return limiter
}

func (l *AdaptiveRateLimiter) WithClock(clock Clock) *AdaptiveRateLimiter { if clock != nil { l.clock = clock }; return l }

func (l *AdaptiveRateLimiter) Acquire(ctx context.Context, domain string) (Permit, error) {
	if ctx == nil { ctx = context.Background() }
	if !l.cfg.Enabled { return immediatePermit{}, nil }
	normalized, err := normalizeDomain(domain); if err != nil { return nil, err }
	state := l.getOrCreateDomainState(normalized)
	for {
		select { case <-ctx.Done(): return nil, ctx.Err(); default: }
		now := l.clock.Now()
		wait, err := state.planRequest(l.cfg, now)
		if err != nil { if errors.Is(err, ErrCircuitOpen) { l.withMetrics(func(m *LimiterSnapshot){ m.Denied++ }) }; return nil, err }
		if wait <= 0 { l.withMetrics(func(m *LimiterSnapshot){ m.TotalRequests++ }); return immediatePermit{}, nil }
		l.withMetrics(func(m *LimiterSnapshot){ m.Throttled++ })
		if !sleepWithContext(ctx, l.clock, wait) { return nil, ctx.Err() }
	}
}

func (l *AdaptiveRateLimiter) Feedback(domain string, fb Feedback) {
	if !l.cfg.Enabled { return }
	normalized, err := normalizeDomain(domain); if err != nil { return }
	state := l.getOrCreateDomainState(normalized)
	state.applyFeedback(l.cfg, fb, l.clock.Now())
}

func (l *AdaptiveRateLimiter) Snapshot() LimiterSnapshot {
	base := func() LimiterSnapshot { l.metricsMu.Lock(); defer l.metricsMu.Unlock(); return l.metrics }()
	var open, halfOpen int64; var domains []DomainSummary
	for _, shard := range l.shards { shard.mu.RLock(); for _, state := range shard.domains { state.mu.Lock(); switch state.breaker.state { case circuitOpen: open++; case circuitHalfOpen: halfOpen++ }
		cs := "closed"; switch state.breaker.state { case circuitOpen: cs = "open"; case circuitHalfOpen: cs = "half-open" }
		domains = append(domains, DomainSummary{Domain: "", FillRate: state.fillRate, CircuitState: cs, LastActivity: state.lastActivity}); state.mu.Unlock() }; shard.mu.RUnlock() }
	if len(domains) > 0 { domains = domains[:0]; for _, shard := range l.shards { shard.mu.RLock(); for name, state := range shard.domains { state.mu.Lock(); cs := "closed"; switch state.breaker.state { case circuitOpen: cs = "open"; case circuitHalfOpen: cs = "half-open" }; domains = append(domains, DomainSummary{Domain: name, FillRate: state.fillRate, CircuitState: cs, LastActivity: state.lastActivity}); state.mu.Unlock() }; shard.mu.RUnlock() }
		if len(domains) > 1 { for i := 1; i < len(domains); i++ { j := i; for j > 0 && domains[j-1].LastActivity.Before(domains[j].LastActivity) { domains[j-1], domains[j] = domains[j], domains[j-1]; j-- } } }
		if len(domains) > 10 { domains = append([]DomainSummary(nil), domains[:10]...) }
		base.Domains = domains }
	base.OpenCircuits = open; base.HalfOpenCircuits = halfOpen; return base
}

type immediatePermit struct{}
func (immediatePermit) Release() {}

func (l *AdaptiveRateLimiter) startEvictionLoop(){ l.evictWG.Add(1); go l.evictLoop() }
func (l *AdaptiveRateLimiter) evictLoop(){ defer l.evictWG.Done(); ticker := time.NewTicker(l.evictInterval); defer ticker.Stop(); for { select { case <-ticker.C: l.evictIdleDomains(); case <-l.stopCh: return } } }
func (l *AdaptiveRateLimiter) evictIdleDomains(){ ttl := l.cfg.DomainStateTTL; if ttl <= 0 { return }; now := l.clock.Now(); for _, shard := range l.shards { shard.mu.Lock(); for domain, state := range shard.domains { state.mu.Lock(); idle := now.Sub(state.lastActivity); state.mu.Unlock(); if idle >= ttl { delete(shard.domains, domain) } }; shard.mu.Unlock() } }
func (l *AdaptiveRateLimiter) Close() error { l.stopOnce.Do(func(){ close(l.stopCh); l.evictWG.Wait() }); return nil }

func sleepWithContext(ctx context.Context, clock Clock, d time.Duration) bool { if d <= 0 { return true }; if ctx != nil { select { case <-ctx.Done(): return false; default: } }; clock.Sleep(d); if ctx != nil { select { case <-ctx.Done(): return false; default: } }; return true }
