package ratelimit

import (
	"math"
	"time"
)

type tokenBucket struct {
	capacity   float64
	fillRate   float64
	tokens     float64
	lastRefill time.Time
}

func newTokenBucket(capacity, fillRate float64, now time.Time) *tokenBucket {
	if capacity <= 0 {
		capacity = 1
	}
	if fillRate <= 0 {
		fillRate = capacity
	}

	return &tokenBucket{
		capacity:   capacity,
		fillRate:   fillRate,
		tokens:     capacity,
		lastRefill: now,
	}
}

func (tb *tokenBucket) refill(now time.Time) {
	if now.Before(tb.lastRefill) {
		return
	}

	elapsed := now.Sub(tb.lastRefill).Seconds()
	if elapsed <= 0 {
		return
	}

	refillAmount := elapsed * tb.fillRate
	if refillAmount <= 0 {
		return
	}

	tb.tokens = math.Min(tb.capacity, tb.tokens+refillAmount)
	tb.lastRefill = now
}

func (tb *tokenBucket) Reserve(now time.Time, amount float64) (time.Duration, bool) {
	if amount <= 0 {
		return 0, true
	}

	tb.refill(now)

	if tb.tokens >= amount {
		tb.tokens -= amount
		return 0, true
	}

	deficit := amount - tb.tokens
	if tb.fillRate <= 0 {
		return time.Duration(math.MaxInt64), false
	}

	waitSeconds := deficit / tb.fillRate
	if waitSeconds < 0 {
		waitSeconds = 0
	}

	return time.Duration(waitSeconds * float64(time.Second)), false
}

func (tb *tokenBucket) setFillRate(rate float64) {
	if rate <= 0 {
		return
	}
	tb.fillRate = rate
}
