package events

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"time"

	internaltracing "github.com/99souls/ariadne/engine/internal/telemetry/tracing"
	metrics "github.com/99souls/ariadne/engine/telemetry/metrics"
)

// Category enumerations (align with event-schema.md)
const (
	CategoryAssets    = "assets"
	CategoryPipeline  = "pipeline"
	CategoryRateLimit = "rate_limit"
	CategoryResources = "resources"
	CategoryConfig    = "config_change"
	CategoryError     = "error"
	CategoryHealth    = "health"
)

// Event is the structured envelope for telemetry events.
type Event struct {
	Time     time.Time              `json:"time"`
	Category string                 `json:"category"`
	Type     string                 `json:"type"`               // more specific subtype
	Severity string                 `json:"severity,omitempty"` // info|warn|error
	TraceID  string                 `json:"trace_id,omitempty"`
	SpanID   string                 `json:"span_id,omitempty"`
	Labels   map[string]string      `json:"labels,omitempty"`
	Fields   map[string]interface{} `json:"fields,omitempty"`
}

// Subscription is a handle representing a consumer of events.
type Subscription interface {
	C() <-chan Event
	Close() error
	ID() int64
}

// BusStats returns runtime counters for observability.
type BusStats struct {
	Subscribers        int64
	Published          uint64
	Dropped            uint64
	PerSubscriberDrops map[int64]uint64
}

// Bus defines the event bus interface.
type Bus interface {
	Publish(ev Event) error
	// PublishCtx enriches event with trace/span IDs from context (if any) then publishes.
	PublishCtx(ctx context.Context, ev Event) error
	Subscribe(buffer int) (Subscription, error)
	Unsubscribe(sub Subscription) error
	Stats() BusStats
}

// NewBus creates a bounded event bus.
func NewBus(provider metrics.Provider) Bus {
	b := &eventBus{subs: make(map[int64]*subscriber), provider: provider}
	b.initMetrics()
	return b
}

type eventBus struct {
	mu        sync.RWMutex
	subs      map[int64]*subscriber
	nextID    int64
	published atomic.Uint64
	dropped   atomic.Uint64

	provider   metrics.Provider
	mPublished metrics.Counter
	mDropped   metrics.Counter
}

func (b *eventBus) initMetrics() {
	if b.provider == nil {
		return
	}
	b.mPublished = b.provider.NewCounter(metrics.CounterOpts{CommonOpts: metrics.CommonOpts{Namespace: "ariadne", Subsystem: "events", Name: "published_total", Help: "Total events published"}})
	b.mDropped = b.provider.NewCounter(metrics.CounterOpts{CommonOpts: metrics.CommonOpts{Namespace: "ariadne", Subsystem: "events", Name: "dropped_total", Help: "Total events dropped due to backpressure", Labels: []string{"subscriber"}}})
}

func (b *eventBus) Publish(ev Event) error {
	if ev.Category == "" {
		return errors.New("event missing category")
	}
	if ev.Time.IsZero() {
		ev.Time = time.Now()
	}
	b.mu.RLock()
	subs := make([]*subscriber, 0, len(b.subs))
	for _, s := range b.subs {
		subs = append(subs, s)
	}
	b.mu.RUnlock()

	b.published.Add(1)
	if b.mPublished != nil {
		b.mPublished.Inc(1)
	}

	for _, s := range subs {
		select {
		case s.ch <- ev:
			// delivered
		default:
			// drop
			s.dropped.Add(1)
			b.dropped.Add(1)
			if b.mDropped != nil {
				b.mDropped.Inc(1, s.idLabel)
			}
		}
	}
	return nil
}

func (b *eventBus) PublishCtx(ctx context.Context, ev Event) error {
	if ev.TraceID == "" && ev.SpanID == "" { // only enrich if IDs absent
		if traceID, spanID := internaltracing.ExtractIDs(ctx); traceID != "" || spanID != "" {
			ev.TraceID = traceID
			ev.SpanID = spanID
		}
	}
	return b.Publish(ev)
}

func (b *eventBus) Subscribe(buffer int) (Subscription, error) {
	if buffer <= 0 {
		buffer = 64
	}
	ch := make(chan Event, buffer)
	id := atomic.AddInt64(&b.nextID, 1)
	sub := &subscriber{id: id, ch: ch, bus: b, idLabel: formatSubscriberID(id)}
	b.mu.Lock()
	b.subs[id] = sub
	b.mu.Unlock()
	return sub, nil
}

func (b *eventBus) Unsubscribe(sub Subscription) error {
	if sub == nil {
		return nil
	}
	id := sub.ID()
	b.mu.Lock()
	s := b.subs[id]
	delete(b.subs, id)
	b.mu.Unlock()
	if s != nil {
		close(s.ch)
	}
	return nil
}

func (b *eventBus) Stats() BusStats {
	b.mu.RLock()
	defer b.mu.RUnlock()
	stats := BusStats{Subscribers: int64(len(b.subs)), Published: b.published.Load(), Dropped: b.dropped.Load(), PerSubscriberDrops: make(map[int64]uint64)}
	for id, s := range b.subs {
		stats.PerSubscriberDrops[id] = s.dropped.Load()
	}
	return stats
}

type subscriber struct {
	id      int64
	ch      chan Event
	bus     *eventBus
	dropped atomic.Uint64
	idLabel string
}

func (s *subscriber) C() <-chan Event { return s.ch }
func (s *subscriber) ID() int64       { return s.id }
func (s *subscriber) Close() error    { return s.bus.Unsubscribe(s) }

func formatSubscriberID(id int64) string {
	// simple decimal string without fmt to avoid allocation in hot path (rarely executed though)
	// Quick implementation
	if id == 0 {
		return "0"
	}
	var digits [20]byte
	i := len(digits)
	for id > 0 {
		i--
		digits[i] = byte('0' + (id % 10))
		id /= 10
	}
	return string(digits[i:])
}
