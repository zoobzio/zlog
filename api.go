package zlog

import (
	"io"
	"os"
	"runtime"
	"time"
)

// Public API functions - Zero breaking changes

// Emit sends a log event with a custom signal type.
// This is the primary API for zlog's signal-based logging system.
// Standard signals (DEBUG, INFO, etc.) are provided, but any string can be used.
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

// Debug logs a debug message with structured fields.
func Debug(msg string, fields ...Field) {
	Emit(DEBUG, msg, fields...)
}

// Info logs an info message with structured fields.
func Info(msg string, fields ...Field) {
	Emit(INFO, msg, fields...)
}

// Warn logs a warning message with structured fields.
func Warn(msg string, fields ...Field) {
	Emit(WARN, msg, fields...)
}

// Error logs an error message with structured fields.
func Error(msg string, fields ...Field) {
	Emit(ERROR, msg, fields...)
}

// Fatal logs a fatal message with structured fields and exits.
func Fatal(msg string, fields ...Field) {
	Emit(FATAL, msg, fields...)
	// Give pipeline time to flush
	time.Sleep(100 * time.Millisecond)
	os.Exit(1)
}

// EnableStandardLogging enables JSON output for standard log signals (INFO, WARN, ERROR, FATAL).
// This is the most common configuration for applications that need traditional logging.
// Note: DEBUG is not included - use EnableDebugLogging() to enable debug logs.
func EnableStandardLogging(w io.Writer) {
	NewStandardLogSink(w)
}

// EnableDebugLogging enables JSON output for DEBUG signals only.
// Use this separately from standard logging to control debug output.
func EnableDebugLogging(w io.Writer) {
	NewDebugSink(w)
}

// EnableAuditLogging enables JSON output for audit and security signals (AUDIT, SECURITY).
// Use this for compliance and security tracking requirements.
func EnableAuditLogging(w io.Writer) {
	NewAuditSink(w)
}

// EnableMetricLogging enables JSON output for metric signals (METRIC).
// Use this for application metrics and monitoring data.
func EnableMetricLogging(w io.Writer) {
	NewMetricSink(w)
}
