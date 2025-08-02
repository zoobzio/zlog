package zlog

// Package-level private logger for the global logging system.
// This replaces the old Dispatch struct with a Logger[Fields] instance.
var defaultLogger *Logger[Fields]

// Initialize the default logger.
func init() {
	defaultLogger = NewLogger[Fields]()
}

// Hook registers one or more sinks to process events with the specified signal.
//
// Multiple sinks can process the same signal - they run in parallel using
// fire-and-forget semantics. This provides optimal performance with automatic
// event cloning for safe concurrent processing:
//
//	// Send errors to multiple destinations (processed in parallel)
//	zlog.Hook(zlog.ERROR, fileSink, alertSink, metricsSink)
//
//	// Or add them separately - same effect (all run in parallel)
//	zlog.Hook(zlog.ERROR, fileSink)      // Permanent storage
//	zlog.Hook(zlog.ERROR, alertSink)     // Team notifications
//	zlog.Hook(zlog.ERROR, metricsSink)   // Error rate tracking
//
//	// Hook business events
//	zlog.Hook(PAYMENT_RECEIVED, auditSink, analyticsSink)
//
// Routes can be added at any time, even after events start flowing.
// There's no way to remove routes - design your signal strategy accordingly.
func Hook(signal Signal, sinks ...*Sink) {
	// Convert sinks to processors for the typed logger
	for _, sink := range sinks {
		defaultLogger.Hook(signal, *sink)
	}
}

// RouteSignal is a backward-compatible alias for Hook.
// Deprecated: Use Hook instead.
func RouteSignal(signal Signal, sinks ...*Sink) {
	Hook(signal, sinks...)
}

// HookAll registers one or more sinks to process ALL events before signal routing.
//
// These sinks run before the signal-based routing, allowing you to implement
// cross-cutting concerns like development logging, metrics collection, or
// audit trails that need to see every event:
//
//	// Log everything to console in development
//	if isDev {
//	    consoleSink := zlog.NewConsoleSink(os.Stderr)
//	    zlog.HookAll(consoleSink)
//	}
//
//	// Collect metrics for all events
//	zlog.HookAll(metricsSink)
//
// Global sinks run in the order they were registered, before any signal-specific
// routing occurs. They see every event emitted to the system.
func HookAll(sinks ...*Sink) {
	for _, sink := range sinks {
		defaultLogger.HookAll(*sink)
	}
}

// RouteAll is a backward-compatible alias for HookAll.
// Deprecated: Use HookAll instead.
func RouteAll(sinks ...*Sink) {
	HookAll(sinks...)
}
