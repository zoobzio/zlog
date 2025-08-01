// Package zlog provides signal-based structured logging for Go applications.
//
// Traditional logging forces you into severity levels (debug, info, warn, error),
// but real applications have diverse event types that need different handling:
// payment events need audit trails, security events need alerting, metrics need
// aggregation, and debug logs need filtering. zlog solves this with signals.
//
// # Core Concepts
//
// Signals are simple strings that categorize events by their meaning, not severity.
// Instead of deciding if something is "info" or "warn", you emit events with
// meaningful signals like "PAYMENT_PROCESSED" or "CACHE_MISS".
//
// Events flow through a routing system that delivers them to appropriate sinks
// based on their signal. Multiple sinks can process the same signal concurrently,
// enabling patterns like storing errors in files while also sending alerts.
//
// # Basic Usage
//
// For traditional logging to stderr:
//
//	zlog.EnableStandardLogging(zlog.INFO)
//	zlog.Info("Application started", zlog.String("version", "1.0.0"))
//	zlog.Error("Database connection failed", zlog.Err(err))
//
// # Signal-Based Routing
//
// Define domain-specific signals and route them appropriately:
//
//	const (
//	    PAYMENT_RECEIVED = zlog.Signal("PAYMENT_RECEIVED")
//	    FRAUD_DETECTED   = zlog.Signal("FRAUD_DETECTED")
//	)
//
//	// Route payment events to audit sink
//	auditSink := zlog.NewSink("audit", handleAuditEvent)
//	zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)
//
//	// Route fraud to multiple destinations
//	zlog.RouteSignal(FRAUD_DETECTED, auditSink)
//	zlog.RouteSignal(FRAUD_DETECTED, alertSink)
//	zlog.RouteSignal(FRAUD_DETECTED, metricsSink)
//
//	// Emit domain events
//	zlog.Emit(PAYMENT_RECEIVED, "Payment processed",
//	    zlog.String("user_id", "123"),
//	    zlog.Float64("amount", 99.99),
//	)
//
// # Creating Modules
//
// Modules are functions that configure routing for specific use cases.
// See log.go for the standard logging module example:
//
//	var jsonSink = zlog.NewSink("json", formatJSON)
//
//	func EnableMyModule(config Config) {
//	    zlog.RouteSignal(SIGNAL1, jsonSink)
//	    zlog.RouteSignal(SIGNAL2, customSink)
//	}
//
// # Performance
//
// zlog is designed for high-throughput applications:
// - Efficient field creation with minimal allocations
// - Lock-free event routing on the hot path
// - Concurrent sink processing with event cloning for isolation
// - Immutable events prevent data races between sinks
//
// Built on github.com/zoobzio/pipz for advanced pipeline capabilities.
package zlog

import (
	"os"
	"runtime"
	"time"
)

// Emit sends an event with the specified signal, message, and optional fields.
//
// This is the primary logging function in zlog. Unlike traditional loggers that
// force you to choose a severity level, Emit lets you specify exactly what type
// of event this is through the signal parameter.
//
// The signal determines how the event is routed - different sinks can be
// registered to handle different signals. Any string can be used as a signal,
// though constants are provided for common cases (INFO, ERROR, etc.).
//
// Fields provide structured context using type-safe constructors:
//
//	zlog.Emit(zlog.INFO, "User logged in",
//	    zlog.String("user_id", "123"),
//	    zlog.String("ip", request.RemoteAddr),
//	    zlog.Duration("session_duration", 30*time.Minute),
//	)
//
// Emit automatically captures caller information (file, line, function) for
// debugging. Events are processed asynchronously - Emit returns immediately
// after routing the event to the appropriate sinks.
func Emit(signal Signal, msg string, fields ...Field) {
	event := NewEvent(signal, msg, fields)

	// Capture caller information
	// Skip 2 frames: Emit -> wrapper function (Info/Debug/etc) -> user code
	// For direct Emit calls, we only skip 1 frame
	skip := 1
	if signal == DEBUG || signal == INFO || signal == WARN || signal == ERROR || signal == FATAL {
		skip = 2
	}

	if pc, file, line, ok := runtime.Caller(skip); ok {
		fn := runtime.FuncForPC(pc)
		if fn != nil {
			event.Caller = CallerInfo{
				File:     file,
				Line:     line,
				Function: fn.Name(),
			}
		}
	}

	dispatch.process(event)
}

// Debug emits a debug-level event for development and troubleshooting.
// Debug events are typically filtered out in production.
//
//	zlog.Debug("Cache lookup", zlog.String("key", cacheKey))
func Debug(msg string, fields ...Field) {
	Emit(DEBUG, msg, fields...)
}

// Info emits an informational event for normal operational messages.
// Use this for events that confirm normal operation.
//
//	zlog.Info("Server started", zlog.Int("port", 8080))
func Info(msg string, fields ...Field) {
	Emit(INFO, msg, fields...)
}

// Warn emits a warning event for concerning but recoverable situations.
// Use this when something is wrong but the application can continue.
//
//	zlog.Warn("API rate limit approaching", zlog.Int("remaining", 100))
func Warn(msg string, fields ...Field) {
	Emit(WARN, msg, fields...)
}

// Error emits an error event for failures that need attention.
// The application continues running but something failed.
//
//	zlog.Error("Failed to send email", zlog.Err(err), zlog.String("to", email))
func Error(msg string, fields ...Field) {
	Emit(ERROR, msg, fields...)
}

// Fatal emits a fatal event and terminates the application with os.Exit(1).
// Use this for unrecoverable errors that prevent the application from continuing.
// Fatal includes a 100ms delay before exiting to allow sinks to flush.
//
//	zlog.Fatal("Failed to connect to database", zlog.Err(err))
func Fatal(msg string, fields ...Field) {
	Emit(FATAL, msg, fields...)
	// Give pipeline time to flush
	time.Sleep(100 * time.Millisecond)
	os.Exit(1)
}
