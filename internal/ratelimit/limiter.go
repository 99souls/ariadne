// Package ratelimit (legacy path) now forwards to engine implementation.
// Deprecated: import "ariadne/packages/engine/ratelimit" instead.
package ratelimit

import (
    engmodels "ariadne/packages/engine/models"
    englimit "ariadne/packages/engine/ratelimit"
)

// Type aliases for backward compatibility
type (
    RateLimiter = englimit.RateLimiter
    Permit = englimit.Permit
    Feedback = englimit.Feedback
    LimiterSnapshot = englimit.LimiterSnapshot
    DomainSummary = englimit.DomainSummary
    AdaptiveRateLimiter = englimit.AdaptiveRateLimiter
)

var (
    ErrCircuitOpen = englimit.ErrCircuitOpen
    NewAdaptiveRateLimiter = func(cfg engmodels.RateLimitConfig) *AdaptiveRateLimiter { return englimit.NewAdaptiveRateLimiter(cfg) }
)

// Forwarders
// (no additional methods needed; underlying type methods are preserved through alias)
