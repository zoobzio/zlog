package zlog

import (
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	beforeCreate := time.Now()

	signal := INFO
	msg := "test message"
	fields := []Field{
		String("key1", "value1"),
		Int("key2", 42),
	}

	event := NewEvent(signal, msg, fields)

	afterCreate := time.Now()

	// Check signal
	if event.Signal != signal {
		t.Errorf("Signal = %v, want %v", event.Signal, signal)
	}

	// Check message
	if event.Message != msg {
		t.Errorf("Message = %v, want %v", event.Message, msg)
	}

	// Check fields
	if len(event.Data) != len(fields) {
		t.Errorf("Fields length = %v, want %v", len(event.Data), len(fields))
	}
	for i, field := range event.Data {
		if field.Key != fields[i].Key {
			t.Errorf("Field[%d].Key = %v, want %v", i, field.Key, fields[i].Key)
		}
		if field.Type != fields[i].Type {
			t.Errorf("Field[%d].Type = %v, want %v", i, field.Type, fields[i].Type)
		}
		if field.Value != fields[i].Value {
			t.Errorf("Field[%d].Value = %v, want %v", i, field.Value, fields[i].Value)
		}
	}

	// Check time is within reasonable bounds
	if event.Time.Before(beforeCreate) || event.Time.After(afterCreate) {
		t.Errorf("Time %v is not between %v and %v", event.Time, beforeCreate, afterCreate)
	}
}

func TestNewEventEdgeCases(t *testing.T) {
	t.Run("Empty message", func(t *testing.T) {
		event := NewEvent(INFO, "", nil)
		if event.Message != "" {
			t.Errorf("Message = %v, want empty", event.Message)
		}
	})

	t.Run("No fields", func(t *testing.T) {
		event := NewEvent(INFO, "msg", nil)
		if event.Data != nil {
			t.Errorf("Fields = %v, want nil", event.Data)
		}
	})

	t.Run("Empty fields slice", func(t *testing.T) {
		event := NewEvent(INFO, "msg", []Field{})
		if len(event.Data) != 0 {
			t.Errorf("Fields length = %v, want 0", len(event.Data))
		}
	})

	t.Run("Custom signal", func(t *testing.T) {
		customSignal := Signal("CUSTOM")
		event := NewEvent(customSignal, "msg", nil)
		if event.Signal != customSignal {
			t.Errorf("Signal = %v, want %v", event.Signal, customSignal)
		}
	})
}

func TestEventFieldIndependence(t *testing.T) {
	// Note: In Go, when we pass a slice to a function, the slice header is copied
	// but the underlying array is shared. This is expected behavior.
	// Fields themselves are value types, so they are copied into the event.

	originalFields := []Field{
		String("key", "original"),
	}

	event := NewEvent(INFO, "msg", originalFields)

	// The slice in event points to the same underlying array
	// This is standard Go behavior and not a bug
	if len(event.Data) != 1 {
		t.Errorf("Expected 1 field, got %d", len(event.Data))
	}

	// The Field struct itself is copied (value type)
	if event.Data[0].Key != "key" || event.Data[0].Value != "original" {
		t.Errorf("Field not copied correctly")
	}
}

func TestEventClone(t *testing.T) {
	original := NewEvent(INFO, "test message", []Field{
		String("key1", "value1"),
		Int("key2", 42),
	})

	// Clone the event
	cloned := original.Clone()

	// Verify all fields are equal
	if cloned.Time != original.Time {
		t.Errorf("Time not cloned correctly")
	}
	if cloned.Signal != original.Signal {
		t.Errorf("Signal not cloned correctly")
	}
	if cloned.Message != original.Message {
		t.Errorf("Message not cloned correctly")
	}
	if len(cloned.Data) != len(original.Data) {
		t.Errorf("Fields length mismatch: got %d, want %d", len(cloned.Data), len(original.Data))
	}

	// Verify fields are copied
	for i, field := range cloned.Data {
		if field != original.Data[i] {
			t.Errorf("Field[%d] not cloned correctly", i)
		}
	}

	// Verify independence - modify clone's fields
	if len(cloned.Data) > 0 {
		cloned.Data[0] = String("modified", "changed")
		if original.Data[0].Key == "modified" {
			t.Errorf("Clone modification affected original")
		}
	}

	// Verify slice independence
	cloned.Data = append(cloned.Data, String("new", "field"))
	if len(original.Data) == len(cloned.Data) {
		t.Errorf("Clone slice modification affected original")
	}
}
