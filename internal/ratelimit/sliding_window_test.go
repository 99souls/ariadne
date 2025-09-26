package ratelimit

import (
	"testing"
	"time"
)

func TestSlidingWindowErrorRate(t *testing.T) {
	now := time.Unix(0, 0)
	sw := newSlidingWindow(10*time.Second, 1*time.Second)

	sw.record(now, 1, 0)                           // success
	sw.record(now.Add(500*time.Millisecond), 1, 1) // error
	sw.record(now.Add(1500*time.Millisecond), 1, 0)

	total, errors := sw.snapshot(now.Add(2 * time.Second))
	if total != 3 || errors != 1 {
		t.Fatalf("expected total=3 errors=1, got total=%d errors=%d", total, errors)
	}

	rate := sw.errorRate(now.Add(2 * time.Second))
	if rate < 0.32 || rate > 0.35 {
		t.Fatalf("expected error rate about 0.333, got %f", rate)
	}
}

func TestSlidingWindowEviction(t *testing.T) {
	now := time.Unix(0, 0)
	sw := newSlidingWindow(5*time.Second, 1*time.Second)

	sw.record(now, 1, 1)
	sw.record(now.Add(2*time.Second), 1, 0)
	sw.record(now.Add(4*time.Second), 1, 0)

	total, errors := sw.snapshot(now.Add(4 * time.Second))
	if total != 3 || errors != 1 {
		t.Fatalf("expected total=3 errors=1 before eviction, got %d/%d", total, errors)
	}

	total, errors = sw.snapshot(now.Add(6 * time.Second))
	if total != 2 || errors != 0 {
		t.Fatalf("expected old bucket evicted, got %d/%d", total, errors)
	}

	rate := sw.errorRate(now.Add(6 * time.Second))
	if rate != 0 {
		t.Fatalf("expected zero error rate after eviction, got %f", rate)
	}
}
