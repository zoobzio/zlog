package zlog

import (
	"context"
	"sync"

	"github.com/zoobzio/pipz"
)

// Dispatch manages the signal-based event routing system.
//
// Dispatch is the core infrastructure that connects events to sinks based on
// their signals. It uses pipz.Switch for efficient routing and automatically
// handles sequential processing when multiple sinks are registered for the
// same signal.
//
// The dispatch system is designed for:
//   - Minimal overhead for event routing
//   - Dynamic route updates without stopping event flow
//   - Sequential processing for predictable performance
//
// This is an internal type - users interact through RouteSignal() and Emit().
type Dispatch struct {
	router    *pipz.Switch[Event, Signal]
	routes    map[Signal][]Sink                // Track sinks per signal
	sequences map[Signal]*pipz.Sequence[Event] // Track sequence processors for updates
	mu        sync.RWMutex
}

// Package-level singleton for global routing.
// This design allows simple API usage while maintaining testability.
var dispatch *Dispatch

// Initialize the default dispatch with signal-based routing.
func init() {
	dispatch = &Dispatch{
		routes:    make(map[Signal][]Sink),
		sequences: make(map[Signal]*pipz.Sequence[Event]),
	}

	// Create a Switch that routes based on the event's signal.
	// Events with unregistered signals pass through unchanged.
	dispatch.router = pipz.NewSwitch[Event, Signal]("signal-router", func(_ context.Context, e Event) Signal {
		return e.Signal
	})
}

// process sends an event through the routing pipeline.
//
// This is called by Emit() and runs asynchronously. Events are routed based
// on their signal, with unregistered signals being silently dropped.
// Errors from sinks are isolated and don't affect the caller or other sinks.
func (d *Dispatch) process(event Event) {
	// Use ephemeral context from current goroutine if available
	// This enables distributed tracing without changing the Emit() API
	ctx := getContext()
	// Process through the switch - unrouted signals pass through unchanged
	_, err := d.router.Process(ctx, event)
	if err != nil {
		// Errors from sinks are isolated - we don't propagate them
		// This maintains the fire-and-forget semantics of Emit()
		return
	}
}

// routeSignal adds a sink to process events with the specified signal.
//
// This method handles the complexity of optimizing for different sink counts:
//   - 1 sink: Direct routing (fastest path)
//   - 2 sinks: Create sequence for ordered processing
//   - 3+ sinks: Add to existing sequence
//
// Routes can be added dynamically without stopping event flow.
func (d *Dispatch) routeSignal(signal Signal, sink Sink) {
	d.mu.Lock()
	defer d.mu.Unlock()

	// Add sink to our tracking
	d.routes[signal] = append(d.routes[signal], sink)
	sinks := d.routes[signal]

	switch len(sinks) {
	case 1:
		// First sink - add directly to switch
		d.router.AddRoute(signal, sink)

	case 2:
		// Second sink - need to switch to sequence
		// Get the first sink from our routes
		firstSink := d.routes[signal][0]

		// Create sequence with both sinks
		sequence := pipz.NewSequence[Event](string(signal))
		sequence.Register(firstSink)
		sequence.Register(sink)

		// Replace route and store sequence for future updates
		d.router.AddRoute(signal, sequence)
		d.sequences[signal] = sequence

	default:
		// 3+ sinks - just add to existing sequence
		if sequence, ok := d.sequences[signal]; ok {
			sequence.Register(sink)
		}
	}
}

// RouteSignal registers one or more sinks to process events with the specified signal.
//
// Multiple sinks can process the same signal - they run sequentially in the
// order they were registered. This provides predictable performance and
// avoids the overhead of event cloning:
//
//	// Send errors to multiple destinations (processed in order)
//	zlog.RouteSignal(zlog.ERROR, fileSink, alertSink, metricsSink)
//
//	// Or add them separately - same effect
//	zlog.RouteSignal(zlog.ERROR, fileSink)      // 1. Permanent storage
//	zlog.RouteSignal(zlog.ERROR, alertSink)     // 2. Team notifications
//	zlog.RouteSignal(zlog.ERROR, metricsSink)   // 3. Error rate tracking
//
//	// Route business events
//	zlog.RouteSignal(PAYMENT_RECEIVED, auditSink, analyticsSink)
//
// Routes can be added at any time, even after events start flowing.
// There's no way to remove routes - design your signal strategy accordingly.
func RouteSignal(signal Signal, sinks ...*Sink) {
	for _, sink := range sinks {
		dispatch.routeSignal(signal, *sink)
	}
}
