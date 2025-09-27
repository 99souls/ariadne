package output

import (
	"fmt"
	"sync"
	"time"

	"ariadne/packages/engine/models"
)

// EnhancedSink is a reference implementation of EnhancedOutputSink
type EnhancedSink struct {
	mutex         sync.RWMutex
	policy        SinkPolicy
	stats         SinkStats
	preprocessor  func(*models.CrawlResult) (*models.CrawlResult, error)
	postprocessor func(*models.CrawlResult) error
	buffer        []*models.CrawlResult
	lastFlush     time.Time
}

// NewEnhancedSink creates a new enhanced sink with default settings
func NewEnhancedSink() *EnhancedSink {
	return &EnhancedSink{
		policy:    DefaultSinkPolicy(),
		stats:     SinkStats{HealthStatus: "healthy"},
		buffer:    make([]*models.CrawlResult, 0, 1000),
		lastFlush: time.Now(),
	}
}

// Write implements OutputSink interface
func (s *EnhancedSink) Write(result *models.CrawlResult) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	startTime := time.Now()

	if result == nil {
		return nil
	}

	// Apply preprocessing if configured
	processedResult := result
	if s.preprocessor != nil {
		var err error
		processedResult, err = s.preprocessor(result)
		if err != nil {
			s.stats.WriteErrors++
			return fmt.Errorf("preprocessing failed: %w", err)
		}
	}

	// Add to buffer
	s.buffer = append(s.buffer, processedResult)
	s.stats.WriteCount++
	s.stats.LastWrite = time.Now()

	// Update buffer utilization
	s.stats.BufferUtilization = float64(len(s.buffer)) / float64(s.policy.BufferSize)

	// Apply postprocessing if configured
	if s.postprocessor != nil {
		err := s.postprocessor(processedResult)
		if err != nil {
			s.stats.WriteErrors++
			return fmt.Errorf("postprocessing failed: %w", err)
		}
	}

	// Update latency
	latency := time.Since(startTime)
	s.updateAverageLatency(latency)

	return nil
}

// Flush implements OutputSink interface
func (s *EnhancedSink) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.stats.FlushCount++
	s.lastFlush = time.Now()

	// Process buffer (in real implementation would write to destination)
	for _, result := range s.buffer {
		if result != nil {
			s.stats.BytesProcessed += int64(len(result.URL))
		}
	}

	// Clear buffer
	s.buffer = s.buffer[:0]
	s.stats.BufferUtilization = 0

	return nil
}

// Close implements OutputSink interface
func (s *EnhancedSink) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Flush any remaining data
	if err := s.Flush(); err != nil {
		s.stats.HealthStatus = "error"
		return fmt.Errorf("failed to flush during close: %w", err)
	}

	s.stats.HealthStatus = "closed"
	return nil
}

// Name implements OutputSink interface
func (s *EnhancedSink) Name() string {
	return "enhanced-sink"
}

// Configure implements EnhancedOutputSink interface
func (s *EnhancedSink) Configure(policy SinkPolicy) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	// Validate policy
	if policy.MaxRetries < 0 {
		return fmt.Errorf("MaxRetries cannot be negative: %d", policy.MaxRetries)
	}
	if policy.BufferSize <= 0 {
		return fmt.Errorf("BufferSize must be positive: %d", policy.BufferSize)
	}

	s.policy = policy

	// Resize buffer if needed
	if cap(s.buffer) != policy.BufferSize {
		newBuffer := make([]*models.CrawlResult, len(s.buffer), policy.BufferSize)
		copy(newBuffer, s.buffer)
		s.buffer = newBuffer
	}

	return nil
}

// Stats implements EnhancedOutputSink interface
func (s *EnhancedSink) Stats() SinkStats {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.stats
}

// IsHealthy implements EnhancedOutputSink interface
func (s *EnhancedSink) IsHealthy() bool {
	s.mutex.RLock()
	defer s.mutex.RUnlock()

	return s.stats.HealthStatus == "healthy"
}

// SetPreprocessor implements EnhancedOutputSink interface
func (s *EnhancedSink) SetPreprocessor(fn func(*models.CrawlResult) (*models.CrawlResult, error)) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.preprocessor = fn
}

// SetPostprocessor implements EnhancedOutputSink interface
func (s *EnhancedSink) SetPostprocessor(fn func(*models.CrawlResult) error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.postprocessor = fn
}

// updateAverageLatency updates the running average latency
func (s *EnhancedSink) updateAverageLatency(latency time.Duration) {
	if s.stats.WriteCount == 1 {
		s.stats.AverageLatency = latency
	} else {
		// Simple moving average
		s.stats.AverageLatency = (s.stats.AverageLatency*time.Duration(s.stats.WriteCount-1) + latency) / time.Duration(s.stats.WriteCount)
	}
}

// Compile-time interface checks
var _ OutputSink = (*EnhancedSink)(nil)
var _ EnhancedOutputSink = (*EnhancedSink)(nil)
