package zlog

import (
	"github.com/zoobzio/pipz"
)

// WithRetry adds retry capability to the sink.
//
// The sink will automatically retry failed operations up to the specified
// number of attempts. Retries are immediate without delay - for operations
// that need backoff between attempts, consider using pipz.NewBackoff directly.
//
// Each retry receives the same event data. Retries stop immediately if the
// context is canceled, allowing for early termination during application
// shutdown or timeout scenarios.
//
// Example usage:
//
//	// Basic retry - try up to 3 times total
//	reliableSink := zlog.NewSink("api", apiHandler).WithRetry(3)
//	zlog.RouteSignal(zlog.ERROR, reliableSink)
//
//	// Chaining with other capabilities (future)
//	complexSink := zlog.NewSink("complex", handler).
//	    WithRetry(3).
//	    WithTimeout(30 * time.Second)
//
// If all retry attempts fail, the last error is returned with attempt count
// information for debugging.
func (s *Sink) WithRetry(attempts int) *Sink {
	if attempts < 1 {
		attempts = 1
	}

	return &Sink{
		processor: pipz.NewRetry("retry", s.processor, attempts),
	}
}
