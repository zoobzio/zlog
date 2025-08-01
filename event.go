package zlog

import (
	"time"
)

// CallerInfo contains the file, line, and function of the log call site.
type CallerInfo struct {
	File     string
	Function string
	Line     int
}

// Event represents an immutable signal event that flows through sinks.
type Event struct {
	Time    time.Time
	Caller  CallerInfo
	Signal  Signal
	Message string
	Fields  []Field
}

// NewEvent creates a new Event with the current timestamp.
//
// This is primarily used internally by Emit() and the convenience functions.
// Most users should use those higher-level functions instead of creating
// events directly.
//
// The fields parameter can be nil if no structured data is needed.
func NewEvent(signal Signal, msg string, fields []Field) Event {
	return Event{
		Time:    time.Now(),
		Signal:  signal,
		Message: msg,
		Fields:  fields,
	}
}

// Clone creates a deep copy of the event for safe concurrent processing.
//
// This method satisfies the pipz.Cloner interface, allowing events to be
// processed by multiple sinks concurrently. Each sink receives its own copy,
// preventing any interference between sinks.
//
// The clone includes a copy of the Fields slice to ensure complete isolation.
func (e Event) Clone() Event {
	// Copy the fields slice to ensure isolation
	fieldsCopy := make([]Field, len(e.Fields))
	copy(fieldsCopy, e.Fields)

	return Event{
		Time:    e.Time,    // time.Time is a value type
		Signal:  e.Signal,  // Signal (string) is immutable
		Message: e.Message, // strings are immutable
		Fields:  fieldsCopy,
	}
}
