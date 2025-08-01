package zlog

import (
	"github.com/zoobzio/pipz"
)

// WithFallback adds fallback capability to the sink.
//
// When the primary sink fails, the fallback sink will be tried automatically.
// This creates resilient processing chains that can recover from failures
// gracefully by switching to an alternative implementation.
//
// Unlike retry which attempts the same operation multiple times, fallback
// switches to a completely different sink. This is valuable when you have
// multiple ways to accomplish the same goal.
//
// Example usage:
//
//	// Primary/backup service failover
//	primarySink := zlog.NewSink("primary-api", primaryHandler)
//	backupSink := zlog.NewSink("backup-api", backupHandler)
//	resilientSink := primarySink.WithFallback(backupSink)
//
//	zlog.RouteSignal(zlog.ERROR, resilientSink)
//
//	// Graceful degradation - try database, fall back to cache
//	dbSink := zlog.NewSink("database", dbHandler)
//	cacheSink := zlog.NewSink("cache", cacheHandler)
//	storageSink := dbSink.WithFallback(cacheSink)
//
//	// Can be chained with other capabilities
//	robustSink := primarySink.
//	    WithRetry(2).
//	    WithFallback(backupSink).
//	    WithTimeout(10 * time.Second)
//
// If the primary sink succeeds, the fallback is never called. If the primary
// fails, the same event data is passed to the fallback sink. Both sinks
// receive identical event data for consistent processing.
func (s *Sink) WithFallback(fallbackSink *Sink) *Sink {
	return &Sink{
		processor: pipz.NewFallback("fallback", s.processor, fallbackSink.processor),
	}
}
