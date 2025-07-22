package zlog

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

// Sink defines the interface for event outputs.
// Implementations should be thread-safe as they may be called concurrently.
type Sink interface {
	// Write processes an event. The implementation should not modify the event.
	Write(event Event) error

	// Name returns a descriptive name for debugging and error messages.
	Name() string
}

// WriterSink writes JSON events to any io.Writer.
type WriterSink struct {
	writer io.Writer
}

// NewWriterSink creates a sink that writes JSON to the provided writer.
func NewWriterSink(w io.Writer) Sink {
	return &WriterSink{
		writer: w,
	}
}

// Write encodes the event as JSON and writes it to the writer.
func (s *WriterSink) Write(event Event) error {
	// Build JSON structure
	entry := map[string]interface{}{
		"time":    event.Time.Format(time.RFC3339Nano),
		"signal":  string(event.Signal),
		"message": event.Message,
	}

	// Add caller info if available
	if event.Caller.File != "" {
		// Format as "file.go:42" for clean output
		entry["caller"] = fmt.Sprintf("%s:%d", event.Caller.File, event.Caller.Line)
	}

	// Add fields
	for _, field := range event.Fields {
		entry[field.Key] = field.Value
	}

	encoder := json.NewEncoder(s.writer)
	return encoder.Encode(entry)
}

// Name returns the sink name.
func (s *WriterSink) Name() string {
	return "writer"
}

// NewStandardLogSink creates a JSON sink that listens to standard log signals.
// It automatically registers itself for INFO, WARN, ERROR, and FATAL signals.
// Note: DEBUG is intentionally excluded - use NewDebugSink() for debug logging.
func NewStandardLogSink(w io.Writer) Sink {
	sink := NewWriterSink(w)

	// Self-register for standard log signals (excluding DEBUG)
	RouteSignal(INFO, sink)
	RouteSignal(WARN, sink)
	RouteSignal(ERROR, sink)
	RouteSignal(FATAL, sink)

	return sink
}

// NewDebugSink creates a JSON sink that listens only to DEBUG signals.
// This allows debug logging to be enabled separately from standard logging.
func NewDebugSink(w io.Writer) Sink {
	sink := NewWriterSink(w)

	// Self-register for debug signals only
	RouteSignal(DEBUG, sink)

	return sink
}

// NewAuditSink creates a JSON sink that listens to audit-related signals.
// It automatically registers itself for AUDIT and SECURITY signals.
func NewAuditSink(w io.Writer) Sink {
	sink := NewWriterSink(w)

	// Self-register for audit signals
	RouteSignal(AUDIT, sink)
	RouteSignal(SECURITY, sink)

	return sink
}

// NewMetricSink creates a JSON sink that listens to metric signals.
// It automatically registers itself for METRIC signals.
func NewMetricSink(w io.Writer) Sink {
	sink := NewWriterSink(w)

	// Self-register for metric signals
	RouteSignal(METRIC, sink)

	return sink
}
