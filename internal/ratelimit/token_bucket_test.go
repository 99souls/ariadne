package ratelimit

import (
	"math"
	"testing"
	"time"
)

type fakeClock struct {
	now time.Time
}

func newFakeClock(start time.Time) *fakeClock {
	return &fakeClock{now: start}
}

func (c *fakeClock) Now() time.Time {
	return c.now
}

func (c *fakeClock) Advance(d time.Duration) {
	c.now = c.now.Add(d)
}

func (c *fakeClock) Sleep(d time.Duration) {
	c.Advance(d)
}

func TestTokenBucketReserveImmediate(t *testing.T) {
	clock := newFakeClock(time.Unix(0, 0))
	tb := newTokenBucket(2, 2, clock.Now())

	if wait, ok := tb.Reserve(clock.Now(), 1); !ok || wait != 0 {
		t.Fatalf("expected immediate token availability, got wait=%v ok=%v", wait, ok)
	}
}

func TestTokenBucketReserveWait(t *testing.T) {
	clock := newFakeClock(time.Unix(0, 0))
	tb := newTokenBucket(1, 2, clock.Now())

	if _, ok := tb.Reserve(clock.Now(), 1); !ok {
		t.Fatalf("initial reserve should succeed")
	}

	if wait, ok := tb.Reserve(clock.Now(), 1); ok || math.Abs(wait.Seconds()-0.5) > 1e-9 {
		t.Fatalf("expected wait of 0.5s and no immediate tokens, got wait=%v ok=%v", wait, ok)
	}

	clock.Advance(250 * time.Millisecond)
	if wait, ok := tb.Reserve(clock.Now(), 1); ok || math.Abs(wait.Seconds()-0.25) > 1e-9 {
		t.Fatalf("after 0.25s advance expected wait 0.25s, got wait=%v ok=%v", wait, ok)
	}

	clock.Advance(250 * time.Millisecond)
	if wait, ok := tb.Reserve(clock.Now(), 1); !ok || wait != 0 {
		t.Fatalf("after refill expected immediate token, got wait=%v ok=%v", wait, ok)
	}
}

func TestTokenBucketCapacityCap(t *testing.T) {
	clock := newFakeClock(time.Unix(0, 0))
	tb := newTokenBucket(3, 10, clock.Now())

	// drain
	for i := 0; i < 3; i++ {
		if _, ok := tb.Reserve(clock.Now(), 1); !ok {
			t.Fatalf("expected tokens during drain iteration %d", i)
		}
	}

	clock.Advance(10 * time.Second)

	tb.refill(clock.Now())

	if tb.tokens > tb.capacity {
		t.Fatalf("tokens exceeded capacity: %v > %v", tb.tokens, tb.capacity)
	}

	if tb.tokens != tb.capacity {
		t.Fatalf("tokens should refill to capacity, got %v", tb.tokens)
	}
}
