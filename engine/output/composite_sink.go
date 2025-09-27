package output

import (
	"fmt"
	"sync"

	"ariadne/packages/engine/models"
)

// CompositeSink writes to multiple sinks simultaneously
type CompositeSink struct {
	mutex sync.RWMutex
	sinks []EnhancedOutputSink
	stats SinkStats
}

// NewCompositeSink creates a new composite sink
func NewCompositeSink(sinks ...EnhancedOutputSink) *CompositeSink {
	return &CompositeSink{
		sinks: sinks,
		stats: SinkStats{HealthStatus: "healthy"},
	}
}

// Write implements OutputSink interface - writes to all sinks
func (c *CompositeSink) Write(result *models.CrawlResult) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var firstError error
	successCount := 0

	for _, sink := range c.sinks {
		err := sink.Write(result)
		if err != nil {
			if firstError == nil {
				firstError = err
			}
			c.stats.WriteErrors++
		} else {
			successCount++
		}
	}

	c.stats.WriteCount++

	// Return error if all sinks failed
	if successCount == 0 && len(c.sinks) > 0 {
		return fmt.Errorf("all sinks failed, first error: %w", firstError)
	}

	return nil
}

// Flush implements OutputSink interface
func (c *CompositeSink) Flush() error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	var firstError error

	for _, sink := range c.sinks {
		err := sink.Flush()
		if err != nil && firstError == nil {
			firstError = err
		}
	}

	c.stats.FlushCount++
	return firstError
}

// Close implements OutputSink interface
func (c *CompositeSink) Close() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	var firstError error

	for _, sink := range c.sinks {
		err := sink.Close()
		if err != nil && firstError == nil {
			firstError = err
		}
	}

	c.stats.HealthStatus = "closed"
	return firstError
}

// Name implements OutputSink interface
func (c *CompositeSink) Name() string {
	return "composite-sink"
}

// Configure implements EnhancedOutputSink interface
func (c *CompositeSink) Configure(policy SinkPolicy) error {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, sink := range c.sinks {
		err := sink.Configure(policy)
		if err != nil {
			return fmt.Errorf("failed to configure sink %s: %w", sink.Name(), err)
		}
	}

	return nil
}

// Stats implements EnhancedOutputSink interface
func (c *CompositeSink) Stats() SinkStats {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Aggregate stats from all sinks
	aggregated := c.stats

	for _, sink := range c.sinks {
		sinkStats := sink.Stats()
		aggregated.WriteCount += sinkStats.WriteCount
		aggregated.WriteErrors += sinkStats.WriteErrors
		aggregated.FlushCount += sinkStats.FlushCount
		aggregated.BytesProcessed += sinkStats.BytesProcessed
	}

	return aggregated
}

// IsHealthy implements EnhancedOutputSink interface
func (c *CompositeSink) IsHealthy() bool {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, sink := range c.sinks {
		if !sink.IsHealthy() {
			return false
		}
	}

	return true
}

// SetPreprocessor implements EnhancedOutputSink interface
func (c *CompositeSink) SetPreprocessor(fn func(*models.CrawlResult) (*models.CrawlResult, error)) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, sink := range c.sinks {
		sink.SetPreprocessor(fn)
	}
}

// SetPostprocessor implements EnhancedOutputSink interface
func (c *CompositeSink) SetPostprocessor(fn func(*models.CrawlResult) error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	for _, sink := range c.sinks {
		sink.SetPostprocessor(fn)
	}
}

// RoutingSink routes results to different sinks based on conditions
type RoutingSink struct {
	mutex  sync.RWMutex
	routes []routeEntry
	stats  SinkStats
}

type routeEntry struct {
	condition RoutingCondition
	sink      EnhancedOutputSink
}

// NewRoutingSink creates a new routing sink
func NewRoutingSink() *RoutingSink {
	return &RoutingSink{
		routes: make([]routeEntry, 0),
		stats:  SinkStats{HealthStatus: "healthy"},
	}
}

// AddRoute adds a routing rule
func (r *RoutingSink) AddRoute(condition RoutingCondition, sink EnhancedOutputSink) {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.routes = append(r.routes, routeEntry{
		condition: condition,
		sink:      sink,
	})
}

// Write implements OutputSink interface - routes to matching sinks
func (r *RoutingSink) Write(result *models.CrawlResult) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	if result == nil {
		return nil
	}

	routed := false
	var firstError error

	for _, route := range r.routes {
		if route.condition(result) {
			err := route.sink.Write(result)
			if err != nil {
				if firstError == nil {
					firstError = err
				}
				r.stats.WriteErrors++
			}
			routed = true
		}
	}

	r.stats.WriteCount++

	if !routed {
		r.stats.WriteErrors++
		return fmt.Errorf("no route matched for result: %s", result.URL)
	}

	return firstError
}

// Flush implements OutputSink interface
func (r *RoutingSink) Flush() error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	var firstError error

	for _, route := range r.routes {
		err := route.sink.Flush()
		if err != nil && firstError == nil {
			firstError = err
		}
	}

	r.stats.FlushCount++
	return firstError
}

// Close implements OutputSink interface
func (r *RoutingSink) Close() error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	var firstError error

	for _, route := range r.routes {
		err := route.sink.Close()
		if err != nil && firstError == nil {
			firstError = err
		}
	}

	r.stats.HealthStatus = "closed"
	return firstError
}

// Name implements OutputSink interface
func (r *RoutingSink) Name() string {
	return "routing-sink"
}

// Configure implements EnhancedOutputSink interface
func (r *RoutingSink) Configure(policy SinkPolicy) error {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, route := range r.routes {
		err := route.sink.Configure(policy)
		if err != nil {
			return fmt.Errorf("failed to configure routed sink %s: %w", route.sink.Name(), err)
		}
	}

	return nil
}

// Stats implements EnhancedOutputSink interface
func (r *RoutingSink) Stats() SinkStats {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.stats
}

// IsHealthy implements EnhancedOutputSink interface
func (r *RoutingSink) IsHealthy() bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return r.stats.HealthStatus == "healthy"
}

// SetPreprocessor implements EnhancedOutputSink interface
func (r *RoutingSink) SetPreprocessor(fn func(*models.CrawlResult) (*models.CrawlResult, error)) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, route := range r.routes {
		route.sink.SetPreprocessor(fn)
	}
}

// SetPostprocessor implements EnhancedOutputSink interface
func (r *RoutingSink) SetPostprocessor(fn func(*models.CrawlResult) error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	for _, route := range r.routes {
		route.sink.SetPostprocessor(fn)
	}
}

// Compile-time interface checks
var _ OutputSink = (*CompositeSink)(nil)
var _ EnhancedOutputSink = (*CompositeSink)(nil)
var _ OutputSink = (*RoutingSink)(nil)
var _ EnhancedOutputSink = (*RoutingSink)(nil)
