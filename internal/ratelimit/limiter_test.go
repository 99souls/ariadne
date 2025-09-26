package ratelimit

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestAdaptiveLimiterAcquireSuccess(t *testing.T) {
	cfg := testRateLimitConfig()
	cfg.Enabled = true
	clock := newFakeClock(time.Unix(0, 0))
	limiter := NewAdaptiveRateLimiter(cfg)
	limiter = limiter.WithClock(clock)
	defer limiter.Close()

	permit, err := limiter.Acquire(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("expected immediate acquire success, got error: %v", err)
	}
	permit.Release()

	clock.Advance(50 * time.Millisecond)
	limiter.Feedback("example.com", Feedback{StatusCode: 200, Latency: 50 * time.Millisecond})
}

func TestAdaptiveLimiterCircuitOpenAfterFailures(t *testing.T) {
	cfg := testRateLimitConfig()
	cfg.Enabled = true
	cfg.ConsecutiveFailThreshold = 1
	cfg.OpenStateDuration = 2 * time.Second
	clock := newFakeClock(time.Unix(0, 0))
	limiter := NewAdaptiveRateLimiter(cfg)
	limiter = limiter.WithClock(clock)
	defer limiter.Close()

	permit, err := limiter.Acquire(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}
	permit.Release()

	clock.Advance(10 * time.Millisecond)
	limiter.Feedback("example.com", Feedback{StatusCode: 503, Latency: 100 * time.Millisecond})

	clock.Advance(10 * time.Millisecond)
	_, err = limiter.Acquire(context.Background(), "example.com")
	if !errors.Is(err, ErrCircuitOpen) {
		t.Fatalf("expected ErrCircuitOpen, got %v", err)
	}

	clock.Advance(cfg.OpenStateDuration)
	permit, err = limiter.Acquire(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("expected half-open probe after open duration, got %v", err)
	}
	permit.Release()
}

func TestAdaptiveLimiterRetryAfterDelay(t *testing.T) {
	cfg := testRateLimitConfig()
	cfg.Enabled = true
	cfg.InitialRPS = 1
	cfg.TokenBucketCapacity = 1
	clock := newFakeClock(time.Unix(0, 0))
	limiter := NewAdaptiveRateLimiter(cfg)
	limiter = limiter.WithClock(clock)
	defer limiter.Close()

	permit, err := limiter.Acquire(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}
	permit.Release()

	clock.Advance(10 * time.Millisecond)
	limiter.Feedback("example.com", Feedback{StatusCode: 200, Latency: 50 * time.Millisecond, RetryAfter: 2 * time.Second})

	before := clock.Now()
	permit, err = limiter.Acquire(context.Background(), "example.com")
	if err != nil {
		t.Fatalf("unexpected acquire error: %v", err)
	}
	permit.Release()

	waited := clock.Now().Sub(before)
	if waited < 2*time.Second {
		t.Fatalf("expected limiter to wait for retry-after duration, waited %v", waited)
	}
}
