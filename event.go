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
// The generic type T allows for different data payloads:
//   - Log for the global logger with structured fields
//   - Event[Order] for typed loggers with domain objects
type Event[T any] struct {
	Time    time.Time
	Data    T
	Message string
	Signal  Signal
	Caller  CallerInfo
}

// Clone creates a copy of the Event.
// This implements the pipz.Cloner interface for use with pipz pipelines.
func (e Event[T]) Clone() Event[T] {
	newEvent := Event[T]{
		Time:    e.Time,
		Caller:  e.Caller,
		Signal:  e.Signal,
		Message: e.Message,
		Data:    e.Data,
	}

	// If T implements Clone() method, use it for deep copying
	if cloner, ok := any(e.Data).(interface{ Clone() T }); ok {
		newEvent.Data = cloner.Clone()
	}

	return newEvent
}

// Log is the standard event type used by the global logger.
// It's an alias for Event[Fields] to provide a cleaner API.
type Log = Event[Fields]

// NewEvent creates a new Event with the current timestamp.
//
// This is primarily used internally by Emit() and the convenience functions.
// Most users should use those higher-level functions instead of creating
// events directly.
//
// The fields parameter can be nil if no structured data is needed.
func NewEvent(signal Signal, msg string, fields []Field) Log {
	return Log{
		Time:    time.Now(),
		Signal:  signal,
		Message: msg,
		Data:    Fields(fields),
	}
}
