package ratelimit

import (
	"math"
	"testing"
	"time"

	"github.com/99souls/ariadne/engine/models"
)

func testRateLimitConfig() models.RateLimitConfig {
	return models.RateLimitConfig{
		Enabled:                  true,
		InitialRPS:               2,
		MinRPS:                   0.5,
		MaxRPS:                   8,
		TokenBucketCapacity:      4,
		AIMDIncrease:             0.5,
		AIMDDecrease:             0.5,
		LatencyTarget:            100 * time.Millisecond,
		LatencyDegradeFactor:     2.0,
		ErrorRateThreshold:       0.4,
		MinSamplesToTrip:         5,
		ConsecutiveFailThreshold: 3,
		OpenStateDuration:        5 * time.Second,
		HalfOpenProbes:           1,
		RetryBaseDelay:           100 * time.Millisecond,
		RetryMaxDelay:            1 * time.Second,
		RetryMaxAttempts:         3,
		StatsWindow:              10 * time.Second,
		StatsBucket:              1 * time.Second,
		DomainStateTTL:           1 * time.Minute,
		Shards:                   4,
	}
}

func TestDomainStateAIMDIncreaseOnFastSuccess(t *testing.T) {
	cfg := testRateLimitConfig()
	now := time.Unix(0, 0)
	ds := newDomainState(cfg, now)

	initial := ds.fillRate
	fb := Feedback{StatusCode: 200, Latency: cfg.LatencyTarget / 2}
	ds.applyFeedback(cfg, fb, now.Add(50*time.Millisecond))

	expected := math.Min(cfg.MaxRPS, initial+cfg.AIMDIncrease)
	if !almostEqual(ds.fillRate, expected) {
		t.Fatalf("expected fill rate %v, got %v", expected, ds.fillRate)
	}

	if !almostEqual(ds.bucket.fillRate, ds.fillRate) {
		t.Fatalf("bucket fill rate mismatch: %v vs %v", ds.bucket.fillRate, ds.fillRate)
	}
}

func TestDomainStateAIMDDecreaseOnSlowSuccess(t *testing.T) {
	cfg := testRateLimitConfig()
	now := time.Unix(0, 0)
	ds := newDomainState(cfg, now)

	initial := ds.fillRate
	fb := Feedback{StatusCode: 200, Latency: time.Duration(float64(cfg.LatencyTarget) * cfg.LatencyDegradeFactor * 1.1)}
	ds.applyFeedback(cfg, fb, now.Add(200*time.Millisecond))

	expected := math.Max(cfg.MinRPS, initial*cfg.AIMDDecrease)
	if !almostEqual(ds.fillRate, expected) {
		t.Fatalf("expected fill rate %v, got %v", expected, ds.fillRate)
	}
}

func TestDomainStateAIMDDecreaseOnThrottleStatus(t *testing.T) {
	cfg := testRateLimitConfig()
	now := time.Unix(0, 0)
	ds := newDomainState(cfg, now)

	initial := ds.fillRate
	fb := Feedback{StatusCode: 429, Latency: cfg.LatencyTarget / 2}
	ds.applyFeedback(cfg, fb, now.Add(100*time.Millisecond))

	expected := math.Max(cfg.MinRPS, initial*cfg.AIMDDecrease)
	if !almostEqual(ds.fillRate, expected) {
		t.Fatalf("expected fill rate %v, got %v", expected, ds.fillRate)
	}
}

func TestCircuitBreakerOpensOnConsecutiveFailures(t *testing.T) {
	cfg := testRateLimitConfig()
	cfg.ConsecutiveFailThreshold = 2
	if cfg.OpenStateDuration == 0 {
		cfg.OpenStateDuration = 2 * time.Second
	}
	now := time.Unix(0, 0)
	ds := newDomainState(cfg, now)

	fail := Feedback{StatusCode: 503, Latency: cfg.LatencyTarget}
	ds.applyFeedback(cfg, fail, now.Add(500*time.Millisecond))
	if ds.breaker.state != circuitClosed {
		t.Fatalf("breaker should remain closed after first failure")
	}

	ds.applyFeedback(cfg, fail, now.Add(1*time.Second))
	if ds.breaker.state != circuitOpen {
		t.Fatalf("breaker should open after threshold failures")
	}

	if ds.allowRequest(cfg, now.Add(1500*time.Millisecond)) {
		t.Fatalf("request should be denied while breaker open")
	}
}

func TestCircuitBreakerHalfOpenAndRecovery(t *testing.T) {
	cfg := testRateLimitConfig()
	cfg.ConsecutiveFailThreshold = 1
	cfg.HalfOpenProbes = 2
	if cfg.OpenStateDuration == 0 {
		cfg.OpenStateDuration = 2 * time.Second
	}
	now := time.Unix(0, 0)
	ds := newDomainState(cfg, now)

	fail := Feedback{StatusCode: 503, Latency: cfg.LatencyTarget}
	ds.applyFeedback(cfg, fail, now.Add(100*time.Millisecond))
	if ds.breaker.state != circuitOpen {
		t.Fatalf("breaker should open immediately due to threshold 1")
	}

	allow := ds.allowRequest(cfg, now.Add(cfg.OpenStateDuration+100*time.Millisecond))
	if !allow {
		t.Fatalf("breaker should transition to half-open after open duration")
	}
	if ds.breaker.state != circuitHalfOpen {
		t.Fatalf("breaker state should be half-open")
	}

	success := Feedback{StatusCode: 200, Latency: cfg.LatencyTarget / 2}
	ds.applyFeedback(cfg, success, now.Add(cfg.OpenStateDuration+200*time.Millisecond))
	if ds.breaker.state != circuitHalfOpen {
		t.Fatalf("breaker should remain half-open until required probes satisfied")
	}

	ds.applyFeedback(cfg, success, now.Add(cfg.OpenStateDuration+300*time.Millisecond))
	if ds.breaker.state != circuitClosed {
		t.Fatalf("breaker should close after successful probes")
	}

	fb := Feedback{StatusCode: 503, Latency: cfg.LatencyTarget}
	ds.applyFeedback(cfg, fb, now.Add(cfg.OpenStateDuration+400*time.Millisecond))
	if ds.breaker.state != circuitOpen {
		t.Fatalf("breaker should reopen on failure in closed state")
	}
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-9
}
