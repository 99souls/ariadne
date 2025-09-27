package pipeline

import (
	"context"
	"sync"
	"testing"
	"time"

	engratelimit "ariadne/packages/engine/ratelimit"
)

// stubLimiter simulates a circuit that opens on first attempt then closes.
// Mirrors original internal test intent.
type stubLimiter struct { mu sync.Mutex; attempts map[string]int }
func newStubLimiter() *stubLimiter { return &stubLimiter{attempts: make(map[string]int)} }
func (s *stubLimiter) Acquire(ctx context.Context, domain string) (engratelimit.Permit, error) {
	if err := ctx.Err(); err != nil { return nil, err }
	s.mu.Lock(); defer s.mu.Unlock(); cnt := s.attempts[domain]+1; s.attempts[domain] = cnt
	if cnt == 1 { return nil, engratelimit.ErrCircuitOpen }
	return permitStub{}, nil
}
func (s *stubLimiter) Feedback(domain string, fb engratelimit.Feedback) {}
func (s *stubLimiter) Snapshot() engratelimit.LimiterSnapshot { return engratelimit.LimiterSnapshot{} }
func (s *stubLimiter) Attempts(domain string) int { s.mu.Lock(); defer s.mu.Unlock(); return s.attempts[domain] }

type permitStub struct{}
func (permitStub) Release() {}

func TestPipelineRateLimiterIntegration(t *testing.T){
	lim := newStubLimiter()
	cfg := &PipelineConfig{DiscoveryWorkers:1,ExtractionWorkers:1,ProcessingWorkers:1,OutputWorkers:1,BufferSize:4,RateLimiter: lim,RetryBaseDelay:1*time.Millisecond,RetryMaxDelay:2*time.Millisecond,RetryMaxAttempts:3}
	pl := NewPipeline(cfg); defer pl.Stop()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second); defer cancel()
	urls := []string{"https://example.com/limited"}
	results := pl.ProcessURLs(ctx, urls)
	processed := 0
	for r := range results { processed++; if r.Error != nil { t.Fatalf("unexpected error: %v", r.Error) } }
	if processed != len(urls) { t.Fatalf("expected %d results got %d", len(urls), processed) }
	if lim.Attempts("example.com") < 2 { t.Fatalf("expected limiter to retry after open circuit") }
}
