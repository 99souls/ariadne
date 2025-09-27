package ratelimit

import "time"

// Clock abstracts time operations for deterministic testing.
type Clock interface {
	Now() time.Time
	Sleep(time.Duration)
}
