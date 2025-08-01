package zlog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// log.go - Standard logging module for terminal output.
//
// This file demonstrates the module pattern for zlog. A module is simply:
//   1. One or more sinks that process events
//   2. A function that routes signals to those sinks
//
// This module provides traditional logging to stderr with JSON formatting,
// suitable for development and production environments that collect logs
// from container output streams.

// stderrJSONSink outputs JSON-formatted logs to stderr.
//
// This sink demonstrates a typical JSON formatter that includes all event
// fields in a flat structure. The format is compatible with most log
// aggregation systems (ELK, Datadog, CloudWatch, etc.).
//
// Output format:
//
//	{"time":"2023-10-20T15:04:05Z","signal":"INFO","message":"User logged in","caller":"auth.go:42","user_id":"123"}
var stderrJSONSink = NewSink("stderr-json", func(_ context.Context, event Event) error {
	// Build JSON structure
	entry := map[string]interface{}{
		"time":    event.Time.Format(time.RFC3339Nano),
		"signal":  string(event.Signal),
		"message": event.Message,
	}

	// Add caller info if available.
	// We use basename:line format for readability in terminals.
	if event.Caller.File != "" {
		// Format as "file.go:42" for clean output
		entry["caller"] = fmt.Sprintf("%s:%d", event.Caller.File, event.Caller.Line)
	}

	// Add all structured fields as top-level JSON properties.
	// This flattens the structure but makes fields easily searchable.
	for _, field := range event.Fields {
		entry[field.Key] = field.Value
	}

	encoder := json.NewEncoder(os.Stderr)
	return encoder.Encode(entry)
})

// EnableStandardLogging enables JSON output to stderr for standard log signals.
// The level parameter determines the minimum signal level that will be logged:
//   - DEBUG: All signals (DEBUG, INFO, WARN, ERROR, FATAL)
//   - INFO: INFO and above (INFO, WARN, ERROR, FATAL)
//   - WARN: WARN and above (WARN, ERROR, FATAL)
//   - ERROR: ERROR and above (ERROR, FATAL)
//   - FATAL: Only FATAL
func EnableStandardLogging(level Signal) {
	// Route signals based on level
	switch level {
	case DEBUG:
		RouteSignal(DEBUG, stderrJSONSink)
		fallthrough
	case INFO:
		RouteSignal(INFO, stderrJSONSink)
		fallthrough
	case WARN:
		RouteSignal(WARN, stderrJSONSink)
		fallthrough
	case ERROR:
		RouteSignal(ERROR, stderrJSONSink)
		fallthrough
	case FATAL:
		RouteSignal(FATAL, stderrJSONSink)
	}
}
