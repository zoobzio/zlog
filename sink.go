package zlog

import (
	"context"

	"github.com/zoobzio/pipz"
)

// Sink processes events routed by signal with composable capabilities.
//
// Sinks are the extensibility point of zlog - they determine what happens to
// events after they're emitted. Common sink patterns include:
//
//   - Writing to files or stdout/stderr
//   - Sending to external services (Elasticsearch, Datadog, etc.)
//   - Filtering or transforming events
//   - Aggregating metrics
//   - Triggering alerts
//
// Multiple sinks can process the same signal concurrently. Each sink receives
// its own copy of events, preventing interference between sinks.
//
// Sinks provide a fluent builder API for adding capabilities like retry,
// batching, filtering, and async processing. Each capability wraps the
// underlying processor with pipz primitives.
//
// Example with capabilities:
//
//	sink := zlog.NewSink("api", handler).
//	    WithRetry(3).
//	    WithTimeout(30 * time.Second)
type Sink struct {
	processor pipz.Chainable[Event]
}

// Process delegates to the underlying processor.
// This makes Sink implement pipz.Chainable[Event].
func (s Sink) Process(ctx context.Context, event Event) (Event, error) {
	return s.processor.Process(ctx, event)
}

// Name returns the name of the underlying processor.
func (s Sink) Name() pipz.Name {
	return s.processor.Name()
}

// NewSink creates a custom sink that processes events.
//
// The name parameter identifies the sink in error messages and debugging output.
// The handler function is called for each event routed to this sink.
//
// Example sink that writes to a file:
//
//	fileSink := zlog.NewSink("file-writer", func(ctx context.Context, event zlog.Event) error {
//	    _, err := fmt.Fprintf(file, "[%s] %s: %s\n",
//	        event.Time.Format(time.RFC3339),
//	        event.Signal,
//	        event.Message)
//	    return err
//	})
//
// Example sink that sends metrics:
//
//	metricSink := zlog.NewSink("metrics", func(ctx context.Context, event zlog.Event) error {
//	    for _, field := range event.Fields {
//	        if field.Key == "duration" {
//	            metrics.RecordDuration(event.Signal, field.Value.(time.Duration))
//	        }
//	    }
//	    return nil
//	})
//
// Sinks should handle errors gracefully - returning an error doesn't affect
// other sinks or the application. Sinks run asynchronously after Emit returns.
//
// The returned Sink can be enhanced with capabilities using the fluent API:
//
//	sink := zlog.NewSink("example", handler).WithRetry(3)
func NewSink(name string, handler func(context.Context, Event) error) *Sink {
	return &Sink{
		processor: pipz.Effect(name, handler),
	}
}
