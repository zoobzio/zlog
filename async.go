package zlog

import (
	"context"

	"github.com/zoobzio/pipz"
)

// WithAsync adds asynchronous processing to the sink.
//
// The sink will process events in a background goroutine without blocking
// the caller. This is useful for slow sinks (external APIs, databases) that
// shouldn't block the main application flow.
//
// Important characteristics:
//   - Fire-and-forget: errors are not reported back to the caller
//   - No buffering: each event spawns a new goroutine immediately
//   - No backpressure: unlimited goroutines can be spawned
//   - Fresh context: background processing uses context.Background()
//
// Example usage:
//
//	// Prevent slow API calls from blocking
//	asyncSink := zlog.NewSink("api", slowApiHandler).WithAsync()
//	zlog.RouteSignal(zlog.INFO, asyncSink)
//
//	// Combine with other adapters for robust async processing
//	robustSink := zlog.NewSink("external", handler).
//	    WithAsync().                    // Don't block the application
//	    WithRetry(3).                   // Retry failures in background
//	    WithTimeout(30 * time.Second)   // Timeout long operations
//
// Warning: WithAsync provides no backpressure control. If events are
// produced faster than they can be processed, goroutines will accumulate.
// For high-volume scenarios, consider implementing a proper queuing system.
//
// The original context is not propagated to avoid issues with short-lived
// contexts (e.g., HTTP request contexts) canceling background work.
func (s *Sink) WithAsync() *Sink {
	// Capture the current processor
	innerProcessor := s.processor

	return &Sink{
		processor: pipz.Effect[Log]("async", func(_ context.Context, event Log) error {
			// Spawn goroutine for fire-and-forget processing
			go func() {
				// Use fresh context since parent might be canceled
				// This ensures background processing completes even if
				// the original request/operation has finished
				asyncCtx := context.Background()

				// Process in background, ignoring result
				// Errors are not propagated back to the caller
				_, _ = innerProcessor.Process(asyncCtx, event) //nolint:errcheck
			}()

			// Return immediately with no error
			// The caller doesn't wait for processing to complete
			return nil
		}),
	}
}
