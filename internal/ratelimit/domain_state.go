package ratelimit

import (
	"github.com/99souls/ariadne/engine/models"
	"math"
	"sync"
	"time"
)

const latencyEWMALambda = 0.2

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

type circuitBreaker struct {
	state             circuitState
	openedAt          time.Time
	halfOpenSuccesses int
	consecutiveFails  int
}

type domainState struct {
	mu sync.Mutex

	bucket   *tokenBucket
	fillRate float64

	latencyEWMA float64
	window      *slidingWindow

	breaker circuitBreaker

	nextEarliest time.Time
	lastActivity time.Time
}

func newDomainState(cfg models.RateLimitConfig, now time.Time) *domainState {
	fill := clampFloat(cfg.InitialRPS, cfg.MinRPS, cfg.MaxRPS)
	capacity := cfg.TokenBucketCapacity
	if capacity <= 0 {
		capacity = fill
	}

	bucket := newTokenBucket(capacity, fill, now)
	windowDur := cfg.StatsWindow
	if windowDur <= 0 {
		windowDur = 30 * time.Second
	}
	bucketDur := cfg.StatsBucket
	if bucketDur <= 0 {
		bucketDur = 2 * time.Second
	}
	window := newSlidingWindow(windowDur, bucketDur)

	state := &domainState{
		bucket:      bucket,
		fillRate:    fill,
		latencyEWMA: float64(cfg.LatencyTarget),
		window:      window,
		breaker: circuitBreaker{
			state: circuitClosed,
		},
		lastActivity: now,
	}

	return state
}

func (ds *domainState) applyFeedback(cfg models.RateLimitConfig, fb Feedback, now time.Time) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.lastActivity = now

	ds.bucket.refill(now)

	observed := fb.Latency
	if observed <= 0 {
		observed = cfg.LatencyTarget
	}

	ds.latencyEWMA = (1-latencyEWMALambda)*ds.latencyEWMA + latencyEWMALambda*float64(observed)

	shouldDecrease := isThrottleStatus(fb.StatusCode) || isServerErrorStatus(fb.StatusCode) || fb.Err != nil
	if !shouldDecrease {
		degradeThreshold := time.Duration(float64(cfg.LatencyTarget) * cfg.LatencyDegradeFactor)
		if degradeThreshold <= 0 {
			degradeThreshold = 2 * cfg.LatencyTarget
		}
		if observed >= degradeThreshold {
			shouldDecrease = true
		}
	}

	if shouldDecrease {
		ds.fillRate = math.Max(cfg.MinRPS, ds.fillRate*cfg.AIMDDecrease)
	} else if isSuccessfulStatus(fb.StatusCode) {
		ds.fillRate = math.Min(cfg.MaxRPS, ds.fillRate+cfg.AIMDIncrease)
	}

	ds.bucket.setFillRate(ds.fillRate)

	isError := isErrorFeedback(fb)
	if ds.window != nil {
		ds.window.record(now, 1, boolToInt(isError))
	}

	if isError {
		ds.breaker.consecutiveFails++
	} else if isSuccessfulStatus(fb.StatusCode) {
		ds.breaker.consecutiveFails = 0
	}

	if fb.RetryAfter > 0 {
		retryAt := now.Add(fb.RetryAfter)
		if retryAt.After(ds.nextEarliest) {
			ds.nextEarliest = retryAt
		}
	}

	var total int
	var errorRate float64
	if ds.window != nil {
		total, _ = ds.window.snapshot(now)
		errorRate = ds.window.errorRate(now)
	}

	ds.updateBreakerAfterFeedback(cfg, now, isError, isSuccessfulStatus(fb.StatusCode), errorRate, total)
}

func (ds *domainState) allowRequestLocked(cfg models.RateLimitConfig, now time.Time) bool {
	ds.lastActivity = now

	switch ds.breaker.state {
	case circuitClosed:
		return true
	case circuitOpen:
		if now.Sub(ds.breaker.openedAt) >= effectiveOpenDuration(cfg.OpenStateDuration) {
			ds.breaker.state = circuitHalfOpen
			ds.breaker.halfOpenSuccesses = 0
			return true
		}
		return false
	case circuitHalfOpen:
		return true
	default:
		return true
	}
}

func (ds *domainState) allowRequest(cfg models.RateLimitConfig, now time.Time) bool {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	return ds.allowRequestLocked(cfg, now)
}

func (ds *domainState) updateBreakerAfterFeedback(cfg models.RateLimitConfig, now time.Time, isError bool, success bool, errorRate float64, total int) {
	switch ds.breaker.state {
	case circuitClosed:
		minSamples := cfg.MinSamplesToTrip
		if minSamples <= 0 {
			minSamples = 1
		}
		if (cfg.ErrorRateThreshold > 0 && total >= minSamples && errorRate >= cfg.ErrorRateThreshold) ||
			(cfg.ConsecutiveFailThreshold > 0 && ds.breaker.consecutiveFails >= cfg.ConsecutiveFailThreshold) {
			ds.openBreaker(now)
		}
	case circuitOpen:
		if now.Sub(ds.breaker.openedAt) >= effectiveOpenDuration(cfg.OpenStateDuration) {
			ds.breaker.state = circuitHalfOpen
			ds.breaker.halfOpenSuccesses = 0
		}
	case circuitHalfOpen:
		if isError {
			ds.openBreaker(now)
			return
		}
		if success {
			probes := cfg.HalfOpenProbes
			if probes <= 0 {
				probes = 1
			}
			ds.breaker.halfOpenSuccesses++
			if ds.breaker.halfOpenSuccesses >= probes {
				ds.breaker.state = circuitClosed
				ds.breaker.consecutiveFails = 0
				ds.breaker.halfOpenSuccesses = 0
			}
		}
	}
}

func (ds *domainState) openBreaker(now time.Time) {
	ds.breaker.state = circuitOpen
	ds.breaker.openedAt = now
	ds.breaker.halfOpenSuccesses = 0
}

func effectiveOpenDuration(d time.Duration) time.Duration {
	if d <= 0 {
		return 10 * time.Second
	}
	return d
}

func clampFloat(value, min, max float64) float64 {
	if min > 0 && value < min {
		value = min
	}
	if max > 0 && value > max {
		value = max
	}
	if min > 0 && value < min {
		value = min
	}
	return value
}

func isSuccessfulStatus(code int) bool {
	return code >= 200 && code < 400
}

func isThrottleStatus(code int) bool {
	return code == 429 || code == 503
}

func isServerErrorStatus(code int) bool {
	return code >= 500 && code < 600
}

func isErrorFeedback(fb Feedback) bool {
	if fb.Err != nil {
		return true
	}
	if isThrottleStatus(fb.StatusCode) || isServerErrorStatus(fb.StatusCode) {
		return true
	}
	return false
}

func boolToInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
