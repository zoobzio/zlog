package zlog

import (
	"context"
	"sync"

	"github.com/zoobzio/pipz"
)

// Dispatch manages the processing pipeline for signal events.
type Dispatch struct {
	pipeline     pipz.Chainable[Event]
	signalRoutes map[Signal][]Sink
	switchRoutes sync.Map
	mu           sync.RWMutex
}

// Package-level singleton.
var (
	dispatch *Dispatch
	once     sync.Once
)

// Initialize the default dispatch.
func init() {
	dispatch = &Dispatch{
		signalRoutes: make(map[Signal][]Sink),
	}

	// Create a custom processor that reads from sync.Map
	dispatch.pipeline = pipz.ProcessorFunc[Event](func(ctx context.Context, e Event) (Event, error) {
		key := string(e.Signal)
		if chainable, ok := dispatch.switchRoutes.Load(key); ok {
			return chainable.(pipz.Chainable[Event]).Process(ctx, e)
		}
		// Unrouted signals return unchanged
		return e, nil
	})
}

// process sends an event through the pipeline.
func (d *Dispatch) process(event Event) {
	d.mu.RLock()
	pipeline := d.pipeline
	d.mu.RUnlock()

	ctx := context.Background()
	// Process through the pipeline - unrouted signals go nowhere
	_, _ = pipeline.Process(ctx, event)
}

// routeSignal adds a sink to listen for a specific signal.
func (d *Dispatch) routeSignal(signal Signal, sink Sink) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add sink to our tracking
	d.signalRoutes[signal] = append(d.signalRoutes[signal], sink)
	sinks := d.signalRoutes[signal]

	// Update the sync.Map with the new pipeline for this signal
	if len(sinks) == 1 {
		// Single sink - just an Effect
		effect := pipz.Effect(sink.Name(), func(ctx context.Context, e Event) error {
			return sink.Write(e)
		})
		d.switchRoutes.Store(string(signal), effect)
	} else {
		// Multiple sinks - rebuild Concurrent for this signal
		effects := make([]pipz.Chainable[Event], len(sinks))
		for i, s := range sinks {
			// capture for closure
			sink := s
			effects[i] = pipz.Effect(sink.Name(), func(ctx context.Context, e Event) error {
				return sink.Write(e)
			})
		}
		d.switchRoutes.Store(string(signal), pipz.Concurrent(effects...))
	}
}

// RouteSignal adds a sink to listen for a specific signal type.
// Multiple sinks can listen to the same signal and will process concurrently.
func RouteSignal(signal Signal, sink Sink) {
	dispatch.routeSignal(signal, sink)
}
