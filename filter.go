package zlog

import (
	"context"

	"github.com/zoobzio/pipz"
)

// WithFilter adds conditional processing to the sink.
//
// The sink will only process events that pass the predicate function.
// Events that don't match are silently skipped without calling the
// underlying sink handler. This is useful for creating specialized
// sinks that only care about specific types of events.
//
// The predicate function receives the full event and should return
// true to process the event or false to skip it. This allows filtering
// on any aspect of the event: signal, message, fields, or metadata.
//
// Example usage:
//
//	// Only process ERROR events
//	errorSink := zlog.NewSink("errors", handler).
//	    WithFilter(func(ctx context.Context, e Log) bool {
//	        return e.Signal == zlog.ERROR
//	    })
//
//	// Only process high-value transactions
//	highValueSink := zlog.NewSink("big-money", handler).
//	    WithFilter(func(ctx context.Context, e Log) bool {
//	        for _, field := range e.Data {
//	            if field.Key == "amount" {
//	                if amount, ok := field.Value.(float64); ok {
//	                    return amount > 10000.0
//	                }
//	            }
//	        }
//	        return false
//	    })
//
//	// Only process events from specific source
//	internalSink := zlog.NewSink("internal", handler).
//	    WithFilter(func(ctx context.Context, e Log) bool {
//	        for _, field := range e.Data {
//	            if field.Key == "source" && field.Value == "internal" {
//	                return true
//	            }
//	        }
//	        return false
//	    })
//
//	// Chain with other capabilities
//	filteredRetrySink := zlog.NewSink("api", handler).
//	    WithFilter(func(ctx context.Context, e Log) bool {
//	        return e.Signal == zlog.ERROR
//	    }).
//	    WithRetry(3).
//	    WithTimeout(30 * time.Second)
//
// Filtering is transparent to the rest of the pipeline - other sinks
// in the same signal route will still receive all events. Only this
// specific sink becomes selective about what it processes.
//
// The predicate function should be fast since it's called for every
// event routed to this sink. Avoid expensive operations in the filter.
func (s *Sink) WithFilter(predicate func(context.Context, Log) bool) *Sink {
	return &Sink{
		processor: pipz.NewFilter[Log]("filter", predicate, s.processor),
	}
}
