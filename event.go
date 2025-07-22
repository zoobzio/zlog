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

// NewEvent creates a new event with the given parameters.
func NewEvent(signal Signal, msg string, fields []Field) Event {
	return Event{
		Time:    time.Now(),
		Signal:  signal,
		Message: msg,
		Fields:  fields,
	}
}

// Clone creates a deep copy of the Event.
// This implements the pipz.Cloner interface for efficient concurrent processing.
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
