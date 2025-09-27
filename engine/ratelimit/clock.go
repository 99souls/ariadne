package ratelimit

import "time"

// Clock abstracts time operations for deterministic testing.
// Experimental: May move to internal if no external implementations appear.
type Clock interface {
    Now() time.Time
    Sleep(time.Duration)
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now() }
func (realClock) Sleep(d time.Duration) { time.Sleep(d) }
