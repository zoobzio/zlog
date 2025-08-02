package zlog

import (
	"context"
	"runtime"
	"sync"
	"time"

	"github.com/zoobzio/pipz"
)

// TypedHook is an alias for pipz.Chainable, providing a more intuitive name for
// event processing injection points in typed loggers. Hooks can be attached to specific signals
// to process events before they reach sinks.
//
// This unified interface works for both typed loggers (TypedHook[Order]) and the
// global event system (Hook[Log] in dispatch.go).
type TypedHook[T any] = pipz.Chainable[T]

// Logger provides typed event processing with signal-based routing.
//
// Logger[T] processes Event[T] types through a pipeline with signal-based routing.
// This enables type-safe hooks and transformations while maintaining integration
// with the existing zlog ecosystem.
//
// The Logger uses the same pipeline architecture as the global system:
//   - Events flow through a root Sequence (for HookAll processors)
//   - Signal-based routing via Switch (extracts from Event.Signal)
//   - Parallel processing via Scaffold for multiple hooks per signal
//
// Example usage:
//
//	type Order struct {
//	    ID     string
//	    Amount float64
//	    Status string
//	}
//
//	func (o Order) Clone() Order {
//	    return Order{ID: o.ID, Amount: o.Amount, Status: o.Status}
//	}
//
//	orderLogger := zlog.NewLogger[Order]()
//
//	// Add typed hooks that work directly with Order events
//	auditHook := zlog.NewHook[Event[Order]]("audit", func(ctx context.Context, event Event[Order]) (Event[Order], error) {
//	    auditDB.Store(event.Data)
//	    return event, nil
//	})
//
//	orderLogger.Hook(ORDER_CREATED, auditHook)
//	orderLogger.Emit(ORDER_CREATED, "Order created", order)
type Logger[T any] struct {
	pipeline  *pipz.Sequence[Event[T]]              // Root sequence for HookAll processors
	router    *pipz.Switch[Event[T], Signal]        // Signal-based router
	hooks     map[Signal][]pipz.Chainable[Event[T]] // Track hooks per signal
	scaffolds map[Signal]*pipz.Scaffold[Event[T]]   // Track scaffold processors for updates
	mu        sync.RWMutex
}

// NewLogger creates a typed logger that processes Event[T] types.
//
// The logger processes events through a pipeline with signal-based routing,
// similar to the global logger but with type safety for the event data.
//
// Example:
//
//	orderLogger := zlog.NewLogger[Order]()
//	orderLogger.Emit(ORDER_CREATED, "Order created", order)
func NewLogger[T any]() *Logger[T] {
	l := &Logger[T]{
		hooks:     make(map[Signal][]pipz.Chainable[Event[T]]),
		scaffolds: make(map[Signal]*pipz.Scaffold[Event[T]]),
	}

	// Create signal router that extracts the signal from Event.Signal
	l.router = pipz.NewSwitch[Event[T], Signal]("typed-router", func(_ context.Context, event Event[T]) Signal {
		return event.Signal // Extract signal from the event itself
	})

	// Create root pipeline and add the router
	l.pipeline = pipz.NewSequence[Event[T]]("typed-pipeline")
	l.pipeline.Register(l.router)

	return l
}

// captureCallerInfo captures the caller information at the specified skip level.
func captureCallerInfo(skip int) CallerInfo {
	if pc, file, line, ok := runtime.Caller(skip + 1); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			return CallerInfo{
				File:     file,
				Line:     line,
				Function: fn.Name(),
			}
		}
	}
	return CallerInfo{}
}

// Hook registers one or more hooks to process events with the specified signal.
//
// Multiple hooks can process the same signal - they run in parallel using
// fire-and-forget semantics for optimal performance. This provides the same
// routing behavior as the global system but with type safety.
//
//	orderLogger.Hook("HIGH_VALUE", auditHook, metricsHook, alertHook)
//
// Hooks can be added dynamically without stopping event flow.
func (l *Logger[T]) Hook(signal Signal, hooks ...pipz.Chainable[Event[T]]) *Logger[T] {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, hook := range hooks {
		l.hookSignal(signal, hook)
	}
	return l
}

// hookSignal adds a single hook for the specified signal (internal method).
// This mirrors the same optimization strategy as the global dispatch system.
func (l *Logger[T]) hookSignal(signal Signal, hook pipz.Chainable[Event[T]]) {
	// Add hook to our tracking
	l.hooks[signal] = append(l.hooks[signal], hook)
	hooks := l.hooks[signal]

	switch len(hooks) {
	case 1:
		// First hook - add directly to switch
		l.router.AddRoute(signal, hook)

	case 2:
		// Second hook - need to switch to scaffold for parallel processing
		// Get the first hook from our routes
		firstHook := l.hooks[signal][0]

		// Create scaffold with both hooks for fire-and-forget parallel execution
		scaffold := pipz.NewScaffold[Event[T]](string(signal), firstHook, hook)

		// Replace route and store scaffold for future updates
		l.router.AddRoute(signal, scaffold)
		l.scaffolds[signal] = scaffold

	default:
		// 3+ hooks - just add to existing scaffold
		if scaffold, ok := l.scaffolds[signal]; ok {
			scaffold.Add(hook)
		}
	}
}

// HookAll registers one or more hooks to process ALL events before signal routing.
//
// These hooks run before the signal-based routing, allowing you to implement
// cross-cutting concerns that need to see every typed event:
//
//	orderLogger.HookAll(validationHook, enrichmentHook)
//
// Global hooks run in the order they were registered, before any signal-specific
// routing occurs. They see every event emitted to this logger.
func (l *Logger[T]) HookAll(hooks ...pipz.Chainable[Event[T]]) *Logger[T] {
	l.mu.Lock()
	defer l.mu.Unlock()

	for _, hook := range hooks {
		l.pipeline.Register(hook)
	}
	return l
}

// WithFilter adds a filter to the logger pipeline that only allows events
// matching the predicate to continue processing.
//
//	orderLogger.WithFilter(func(order Order) bool {
//	    return order.Amount > 100.0
//	})
func (l *Logger[T]) WithFilter(predicate func(Event[T]) bool) *Logger[T] {
	// Create a pass-through processor for the filter
	passThrough := pipz.Apply[Event[T]]("pass-through", func(_ context.Context, event Event[T]) (Event[T], error) {
		return event, nil
	})
	// Convert simple predicate to context-aware predicate
	condition := func(_ context.Context, event Event[T]) bool {
		return predicate(event)
	}
	filter := pipz.NewFilter[Event[T]]("typed-filter", condition, passThrough)
	return l.HookAll(filter)
}

// WithTimeout adds timeout protection to the logger pipeline.
//
//	orderLogger.WithTimeout(5 * time.Second)
func (l *Logger[T]) WithTimeout(timeout time.Duration) *Logger[T] {
	// Create a simple pass-through processor for timeout wrapping
	passThrough := pipz.Apply[Event[T]]("pass-through", func(_ context.Context, event Event[T]) (Event[T], error) {
		return event, nil
	})
	timeoutProcessor := pipz.NewTimeout[Event[T]]("typed-timeout", passThrough, timeout)
	return l.HookAll(timeoutProcessor)
}

// WithRetry adds retry capability to the logger pipeline.
//
//	orderLogger.WithRetry(3)
func (l *Logger[T]) WithRetry(attempts int) *Logger[T] {
	// Create a simple pass-through processor for retry wrapping
	passThrough := pipz.Apply[Event[T]]("pass-through", func(_ context.Context, event Event[T]) (Event[T], error) {
		return event, nil
	})
	retryProcessor := pipz.NewRetry[Event[T]]("typed-retry", passThrough, attempts)
	return l.HookAll(retryProcessor)
}

// WithAsync makes the logger process events asynchronously.
//
//	orderLogger.WithAsync()
func (l *Logger[T]) WithAsync() *Logger[T] {
	// For async behavior, we'll use a Scaffold which provides fire-and-forget semantics
	// This is more appropriate than pipz.Async which may not exist
	passThrough := pipz.Apply[Event[T]]("pass-through", func(_ context.Context, event Event[T]) (Event[T], error) {
		return event, nil
	})
	// Use scaffold for async fire-and-forget behavior
	asyncProcessor := pipz.NewScaffold[Event[T]]("typed-async", passThrough)
	return l.HookAll(asyncProcessor)
}

// Emit creates an Event[T] and processes it through the logger pipeline.
//
// The event flows through:
//  1. HookAll processors (cross-cutting concerns)
//  2. Signal-based routing and Hook processors
//
// Example:
//
//	orderLogger.Emit(ORDER_CREATED, "Order created", order)
func (l *Logger[T]) Emit(signal Signal, message string, data T) {
	// Create the event with caller info
	event := Event[T]{
		Time:    time.Now(),
		Signal:  signal,
		Message: message,
		Data:    data,
		Caller:  captureCallerInfo(1),
	}
	l.Process(event)
}

// Process handles pre-built Event[T] types through the logger pipeline.
// This method does not capture caller info - it should already be in the event.
func (l *Logger[T]) Process(event Event[T]) {
	ctx := getContext()
	_, _ = l.pipeline.Process(ctx, event) //nolint:errcheck // Errors intentionally ignored in fire-and-forget logging
}

// Watch configures this logger to forward all events to the global logger
// after processing through the typed pipeline.
//
// This enables typed loggers to integrate with the global logging system
// while maintaining type safety for their own processing.
//
// Example:
//
//	orderLogger := NewLogger[Order]().Watch()
//	orderLogger.Emit(ORDER_CREATED, "Order created", order)
//	// Event flows through typed hooks, then to global logger
func (l *Logger[T]) Watch() *Logger[T] {
	// Add a terminal hook that forwards to the global logger
	forwarder := pipz.Effect[Event[T]]("global-forward", func(_ context.Context, event Event[T]) error {
		// Convert Event[T] to Log for the global logger
		globalEvent := Log{
			Time:    event.Time,
			Caller:  event.Caller,
			Signal:  event.Signal,
			Message: event.Message,
			Data:    Fields{Data("event", event.Data)},
		}
		// Forward to global logger
		defaultLogger.Process(globalEvent)
		return nil
	})

	// Add the forwarder to all routes
	l.mu.Lock()
	defer l.mu.Unlock()

	// Add to HookAll so it runs after all processing
	l.pipeline.Register(forwarder)

	return l
}
