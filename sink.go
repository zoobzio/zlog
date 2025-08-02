package zlog

import (
	"context"
	"time"

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
	processor pipz.Chainable[Log]
}

// Process delegates to the underlying processor.
// This makes Sink implement pipz.Chainable[Log].
func (s Sink) Process(ctx context.Context, event Log) (Log, error) {
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
//	fileSink := zlog.NewSink("file-writer", func(ctx context.Context, event zlog.Log) error {
//	    _, err := fmt.Fprintf(file, "[%s] %s: %s\n",
//	        event.Time.Format(time.RFC3339),
//	        event.Signal,
//	        event.Message)
//	    return err
//	})
//
// Example sink that sends metrics:
//
//	metricSink := zlog.NewSink("metrics", func(ctx context.Context, event zlog.Log) error {
//	    for _, field := range event.Data {
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
func NewSink(name string, handler func(context.Context, Log) error) *Sink {
	return &Sink{
		processor: pipz.Effect[Log](name, handler),
	}
}

// RateLimiterConfig configures rate limiting behavior.
type RateLimiterConfig struct {
	// RequestsPerSecond is the sustained rate limit.
	RequestsPerSecond float64
	// BurstSize allows temporary spikes above the rate.
	BurstSize int
	// WaitForSlot determines if Process should block or error when limited.
	WaitForSlot bool
}

// CircuitState represents the current state of a circuit breaker.
// This type is kept for backwards compatibility with tests and examples.
type CircuitState string

const (
	// CircuitClosed allows requests through (normal operation).
	CircuitClosed CircuitState = "closed"
	// CircuitOpen blocks all requests (failure mode).
	CircuitOpen CircuitState = "open"
	// CircuitHalfOpen allows limited requests for testing.
	CircuitHalfOpen CircuitState = "half-open"
)

// CircuitBreakerConfig configures circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold int
	// SuccessThreshold is the number of successes in half-open before closing.
	SuccessThreshold int
	// ResetTimeout is how long to wait before trying half-open.
	ResetTimeout time.Duration
}

// WithRateLimit adds token bucket rate limiting to a sink using pipz.NewRateLimiter.
//
// Token bucket algorithm provides:
//   - Sustained rate limiting (requests per second)
//   - Burst capacity for temporary spikes
//   - Optional blocking until tokens available
//
// Example:
//
//	httpSink := zlog.NewHTTPSink("https://api.example.com/logs").
//	    WithRateLimit(zlog.RateLimiterConfig{
//	        RequestsPerSecond: 100,  // Sustained rate
//	        BurstSize: 200,         // Allow bursts up to 200
//	        WaitForSlot: false,     // Don't block, fail fast
//	    })
func (s *Sink) WithRateLimit(config RateLimiterConfig) *Sink {
	// Apply defaults
	if config.RequestsPerSecond <= 0 {
		config.RequestsPerSecond = 100 // 100 RPS default
	}
	if config.BurstSize <= 0 {
		config.BurstSize = int(config.RequestsPerSecond) // Default burst = rate
	}

	// Create pipz rate limiter
	limiter := pipz.NewRateLimiter[Log](
		s.Name()+" [rate-limit]",
		config.RequestsPerSecond,
		config.BurstSize,
	)

	// Set mode based on WaitForSlot
	if config.WaitForSlot {
		limiter.SetMode("wait")
	} else {
		limiter.SetMode("drop")
	}

	// Wrap the sink's processor with rate limiting
	return &Sink{
		processor: pipz.NewSequence[Log]("rate-limited-sink", limiter, s.processor),
	}
}

// WithCircuitBreaker adds circuit breaker protection to a sink using pipz.NewCircuitBreaker.
//
// Circuit breaker prevents cascading failures by:
//   - Opening after consecutive failures reach threshold
//   - Blocking requests while open (fail-fast)
//   - Transitioning to half-open for recovery testing
//   - Closing after successful requests in half-open
//
// Example:
//
//	dbSink := zlog.NewSink("database", dbHandler).
//	    WithCircuitBreaker(zlog.CircuitBreakerConfig{
//	        FailureThreshold: 5,           // Open after 5 consecutive failures
//	        SuccessThreshold: 3,           // Close after 3 successes in half-open
//	        ResetTimeout: 30 * time.Second, // Try half-open after 30s
//	    })
func (s *Sink) WithCircuitBreaker(config CircuitBreakerConfig) *Sink {
	// Apply defaults
	if config.FailureThreshold <= 0 {
		config.FailureThreshold = 5
	}
	if config.SuccessThreshold <= 0 {
		config.SuccessThreshold = 2
	}
	if config.ResetTimeout <= 0 {
		config.ResetTimeout = 30 * time.Second
	}

	// Create pipz circuit breaker
	breaker := pipz.NewCircuitBreaker[Log](
		s.Name()+" [circuit-breaker]",
		s.processor,
		config.FailureThreshold,
		config.ResetTimeout,
	)

	// Set success threshold
	breaker.SetSuccessThreshold(config.SuccessThreshold)

	return &Sink{processor: breaker}
}

// RateLimitedSink creates a rate-limited sink with sensible defaults.
//
// This is a convenience function that creates a sink with:
//   - Specified requests per second sustained rate
//   - Burst capacity equal to 2x the rate
//   - Non-blocking mode (drops excess requests)
//
// Example:
//
//	sink := zlog.RateLimitedSink("api", 50, handler)
//	// Equivalent to:
//	// zlog.NewSink("api", handler).WithRateLimit(zlog.RateLimiterConfig{
//	//     RequestsPerSecond: 50,
//	//     BurstSize: 100,
//	//     WaitForSlot: false,
//	// })
func RateLimitedSink(name string, requestsPerSecond float64, handler func(context.Context, Log) error) *Sink {
	sink := NewSink(name, handler)
	return sink.WithRateLimit(RateLimiterConfig{
		RequestsPerSecond: requestsPerSecond,
		BurstSize:         int(requestsPerSecond * 2), // 2x rate for burst
		WaitForSlot:       false,
	})
}

// WithDefaultCircuitBreaker adds circuit breaker with sensible defaults.
//
// Default configuration:
//   - Opens after 5 consecutive failures
//   - Closes after 2 consecutive successes in half-open state
//   - Waits 30 seconds before attempting recovery
//
// Example:
//
//	sink := zlog.NewSink("fragile-api", handler).WithDefaultCircuitBreaker()
func (s *Sink) WithDefaultCircuitBreaker() *Sink {
	return s.WithCircuitBreaker(CircuitBreakerConfig{})
}
