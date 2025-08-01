package zlog

import (
	"time"

	"github.com/zoobzio/pipz"
)

// WithTimeout adds timeout capability to the sink.
//
// The sink will enforce a hard timeout on event processing. If an operation
// takes longer than the specified duration, it will be canceled via context
// and a timeout error will be returned.
//
// This is critical for preventing hung operations, meeting SLA requirements,
// and protecting against slow external services. The wrapped sink handler
// should respect context cancellation for immediate termination.
//
// Example usage:
//
//	// Prevent slow API calls from hanging
//	apiSink := zlog.NewSink("api", apiHandler).WithTimeout(5 * time.Second)
//	zlog.RouteSignal(zlog.ERROR, apiSink)
//
//	// Combined with retry for robust error handling
//	resilientSink := zlog.NewSink("db", dbHandler).
//	    WithRetry(3).
//	    WithTimeout(10 * time.Second)
//
//	// Order matters - this retries the entire timeout operation
//	retryThenTimeout := sink.WithRetry(3).WithTimeout(30 * time.Second)
//
//	// This times out each retry attempt individually
//	timeoutThenRetry := sink.WithTimeout(10 * time.Second).WithRetry(3)
//
// If the timeout expires, the operation is canceled and a timeout error
// is returned. Operations that ignore context cancellation may continue
// running in the background even after timeout.
func (s *Sink) WithTimeout(duration time.Duration) *Sink {
	if duration <= 0 {
		duration = 30 * time.Second // Default timeout
	}

	return &Sink{
		processor: pipz.NewTimeout("timeout", s.processor, duration),
	}
}
