package output

import (
	"time"

	"github.com/99souls/ariadne/engine/models"
)

// SinkPolicy defines configuration for sink behavior
type SinkPolicy struct {
	// Retry behavior
	MaxRetries    int
	RetryDelay    time.Duration
	
	// Buffering behavior
	BufferSize    int
	FlushInterval time.Duration
	
	// Processing behavior
	EnableCompression bool
	FilterPattern     string
	TransformRules    []string
	
	// Routing behavior
	RoutePattern      string
	FailoverSinks     []string
	
	// Performance settings
	MaxConcurrency    int
	TimeoutDuration   time.Duration
}

// SinkStats provides metrics about sink operations
type SinkStats struct {
	WriteCount         int64
	WriteErrors        int64
	FlushCount         int64
	BytesProcessed     int64
	AverageLatency     time.Duration
	LastWrite          time.Time
	BufferUtilization  float64
	HealthStatus       string
}

// EnhancedOutputSink extends the basic OutputSink with enhanced capabilities
type EnhancedOutputSink interface {
	OutputSink // Embed the basic interface for backward compatibility
	
	// Enhanced configuration and monitoring
	Configure(policy SinkPolicy) error
	Stats() SinkStats
	IsHealthy() bool
	
	// Pipeline processing
	SetPreprocessor(fn func(*models.CrawlResult) (*models.CrawlResult, error))
	SetPostprocessor(fn func(*models.CrawlResult) error)
}

// RoutingCondition defines a condition for routing decisions
type RoutingCondition func(*models.CrawlResult) bool

// DefaultSinkPolicy returns a sensible default policy
func DefaultSinkPolicy() SinkPolicy {
	return SinkPolicy{
		MaxRetries:        3,
		RetryDelay:        100 * time.Millisecond,
		BufferSize:        1000,
		FlushInterval:     5 * time.Second,
		EnableCompression: false,
		FilterPattern:     "",
		TransformRules:    []string{},
		MaxConcurrency:    4,
		TimeoutDuration:   30 * time.Second,
	}
}