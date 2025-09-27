package configx

import (
    "sync"
    "time"
)

// ChangeEvent captures an emitted lifecycle occurrence for applies / rollbacks / failures.
type ChangeEvent struct {
    Type      string // apply|rollback|validation_error|simulation_reject|append_error
    Version   int64
    Hash      string
    Actor     string
    Error     error
    Timestamp time.Time
}

// Listener consumes change events.
type Listener interface { OnEvent(ChangeEvent) }

// Dispatcher fan-outs events to registered listeners in-process (synchronous, best-effort).
type Dispatcher struct {
    mu        sync.RWMutex
    listeners []Listener
}

func NewDispatcher() *Dispatcher { return &Dispatcher{} }

func (d *Dispatcher) Register(l Listener) {
    d.mu.Lock(); defer d.mu.Unlock()
    d.listeners = append(d.listeners, l)
}

func (d *Dispatcher) Emit(e ChangeEvent) {
    d.mu.RLock(); ls := append([]Listener(nil), d.listeners...); d.mu.RUnlock()
    for _, l := range ls { l.OnEvent(e) }
}

// InMemoryCollector is a simple listener for tests.
type InMemoryCollector struct { Events []ChangeEvent; mu sync.Mutex }
func (c *InMemoryCollector) OnEvent(e ChangeEvent) { c.mu.Lock(); c.Events = append(c.Events, e); c.mu.Unlock() }