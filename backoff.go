package zlog

import (
	"time"

	"github.com/zoobzio/pipz"
)

// WithBackoff adds retry with exponential backoff capability to the sink.
//
// The sink will automatically retry failed operations with increasing delays
// between attempts. The delay starts at baseDelay and doubles after each
// failure, creating an exponential backoff pattern that prevents overwhelming
// failed services and allows time for transient issues to resolve.
//
// This is more sophisticated than basic retry as it includes delays between
// attempts, making it ideal for external services that may be temporarily
// overloaded or rate-limited.
//
// Example usage:
//
//	// Retry API calls with exponential backoff
//	apiSink := zlog.NewSink("api", apiHandler).
//	    WithBackoff(5, 100*time.Millisecond)
//	zlog.RouteSignal(zlog.ERROR, apiSink)
//
//	// Combined with timeout for robust error handling
//	resilientSink := zlog.NewSink("external", handler).
//	    WithBackoff(3, time.Second).
//	    WithTimeout(30 * time.Second)
//
//	// Backoff delays: 1s, 2s, 4s (total wait: 7s plus processing time)
//	dbSink := zlog.NewSink("database", dbHandler).
//	    WithBackoff(4, time.Second)
//
// The exponential backoff pattern (delay, 2*delay, 4*delay, ...) is widely
// used for handling rate limits, temporary service overload, and network
// congestion. The operation can be canceled via context during waits.
//
// Total time can be significant with multiple retries. Plan accordingly
// when setting maxAttempts and baseDelay values.
func (s *Sink) WithBackoff(maxAttempts int, baseDelay time.Duration) *Sink {
	if maxAttempts < 1 {
		maxAttempts = 1
	}
	if baseDelay <= 0 {
		baseDelay = 100 * time.Millisecond // Default base delay
	}

	return &Sink{
		processor: pipz.NewBackoff("backoff", s.processor, maxAttempts, baseDelay),
	}
}
