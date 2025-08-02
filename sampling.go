package zlog

import (
	"context"
	"math/rand"
	"sync/atomic"
)

// WithSampling returns a sink adapter that only processes a percentage of events.
//
// This is useful for high-volume signals where you want to reduce load while
// still getting a representative sample. The sampling is deterministic based
// on a counter to ensure consistent sampling rates.
//
// The rate parameter should be between 0.0 and 1.0:
//   - 0.0 = no events pass through (why would you do this?)
//   - 0.1 = 10% of events pass through
//   - 0.5 = 50% of events pass through
//   - 1.0 = all events pass through (no sampling)
//
// Example usage:
//
//	// Only process 10% of cache hit events to reduce metrics load
//	cacheSink := metricsSink.WithSampling(0.1)
//	zlog.RouteSignal(CACHE_HIT, cacheSink)
//
//	// Sample 1% of high-volume API logs
//	apiSink := fileSink.WithSampling(0.01).WithAsync()
//	zlog.RouteSignal(API_REQUEST, apiSink)
//
// The sampling decision is made before the event reaches the sink, so
// filtered events have minimal performance impact.
func (s *Sink) WithSampling(rate float64) *Sink {
	// Clamp rate to valid range
	if rate <= 0 {
		// Return a sink that drops everything
		return NewSink("sampling-drop-all", func(_ context.Context, _ Log) error {
			return nil
		})
	}
	if rate >= 1 {
		// No sampling needed
		return s
	}

	// Use a counter for deterministic sampling
	var counter uint64

	return s.WithFilter(func(_ context.Context, _ Log) bool {
		// Increment counter atomically
		count := atomic.AddUint64(&counter, 1)

		// Use modulo for deterministic sampling
		// For 10% sampling (0.1), we want every 10th event
		// So we accept when count % 10 == 0
		interval := uint64(1.0 / rate)
		return count%interval == 0
	})
}

// WithProbabilisticSampling returns a sink adapter that randomly samples events.
//
// Unlike WithSampling which uses deterministic sampling, this uses random
// sampling. Each event has an independent probability of being processed.
//
// This can be more appropriate when:
//   - Events arrive in bursts (deterministic might miss entire bursts)
//   - You need true statistical sampling
//   - Event order is unpredictable
//
// Example usage:
//
//	// Randomly sample 25% of events
//	randomSink := debugSink.WithProbabilisticSampling(0.25)
func (s *Sink) WithProbabilisticSampling(rate float64) *Sink {
	// Clamp rate to valid range
	if rate <= 0 {
		return NewSink("probabilistic-drop-all", func(_ context.Context, _ Log) error {
			return nil
		})
	}
	if rate >= 1 {
		return s
	}

	return s.WithFilter(func(_ context.Context, _ Log) bool {
		return rand.Float64() < rate //nolint:gosec // Weak random is acceptable for sampling
	})
}
