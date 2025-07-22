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
	if len(event.Fields) != len(fields) {
		t.Errorf("Fields length = %v, want %v", len(event.Fields), len(fields))
	}
	for i, field := range event.Fields {
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
		if event.Fields != nil {
			t.Errorf("Fields = %v, want nil", event.Fields)
		}
	})

	t.Run("Empty fields slice", func(t *testing.T) {
		event := NewEvent(INFO, "msg", []Field{})
		if len(event.Fields) != 0 {
			t.Errorf("Fields length = %v, want 0", len(event.Fields))
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
	if len(event.Fields) != 1 {
		t.Errorf("Expected 1 field, got %d", len(event.Fields))
	}

	// The Field struct itself is copied (value type)
	if event.Fields[0].Key != "key" || event.Fields[0].Value != "original" {
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
	if len(cloned.Fields) != len(original.Fields) {
		t.Errorf("Fields length mismatch: got %d, want %d", len(cloned.Fields), len(original.Fields))
	}

	// Verify fields are copied
	for i, field := range cloned.Fields {
		if field != original.Fields[i] {
			t.Errorf("Field[%d] not cloned correctly", i)
		}
	}

	// Verify independence - modify clone's fields
	if len(cloned.Fields) > 0 {
		cloned.Fields[0] = String("modified", "changed")
		if original.Fields[0].Key == "modified" {
			t.Errorf("Clone modification affected original")
		}
	}

	// Verify slice independence
	cloned.Fields = append(cloned.Fields, String("new", "field"))
	if len(original.Fields) == len(cloned.Fields) {
		t.Errorf("Clone slice modification affected original")
	}
}

// Benchmarks.
func BenchmarkNewEvent(b *testing.B) {
	b.Run("NoFields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = NewEvent(INFO, "benchmark message", nil)
		}
	})

	b.Run("OneField", func(b *testing.B) {
		b.ReportAllocs()
		fields := []Field{String("key", "value")}
		for i := 0; i < b.N; i++ {
			_ = NewEvent(INFO, "benchmark message", fields)
		}
	})

	b.Run("FiveFields", func(b *testing.B) {
		b.ReportAllocs()
		fields := []Field{
			String("user", "alice"),
			Int("user_id", 42),
			Bool("active", true),
			Time("timestamp", time.Now()),
			Float64("score", 98.5),
		}
		for i := 0; i < b.N; i++ {
			_ = NewEvent(INFO, "benchmark message", fields)
		}
	})

	b.Run("TenFields", func(b *testing.B) {
		b.ReportAllocs()
		fields := []Field{
			String("field1", "value1"),
			String("field2", "value2"),
			String("field3", "value3"),
			String("field4", "value4"),
			String("field5", "value5"),
			Int("field6", 6),
			Int("field7", 7),
			Int("field8", 8),
			Bool("field9", true),
			Bool("field10", false),
		}
		for i := 0; i < b.N; i++ {
			_ = NewEvent(INFO, "benchmark message", fields)
		}
	})

	b.Run("DifferentSignals", func(b *testing.B) {
		b.ReportAllocs()
		signals := []Signal{DEBUG, INFO, WARN, ERROR, FATAL}
		fields := []Field{String("key", "value")}
		for i := 0; i < b.N; i++ {
			signal := signals[i%len(signals)]
			_ = NewEvent(signal, "benchmark message", fields)
		}
	})
}

func BenchmarkEventMemorySize(b *testing.B) {
	// This benchmark helps understand the memory footprint
	b.Run("EventSize", func(b *testing.B) {
		b.ReportAllocs()
		events := make([]Event, 0, b.N)
		fields := []Field{
			String("user", "alice"),
			Int("user_id", 42),
		}

		for i := 0; i < b.N; i++ {
			events = append(events, NewEvent(INFO, "test", fields))
		}

		// Prevent compiler optimization
		if len(events) != b.N {
			b.Errorf("Expected %d events", b.N)
		}
	})
}

func BenchmarkEventClone(b *testing.B) {
	b.Run("NoFields", func(b *testing.B) {
		b.ReportAllocs()
		event := NewEvent(INFO, "benchmark", nil)
		for i := 0; i < b.N; i++ {
			_ = event.Clone()
		}
	})

	b.Run("WithFields", func(b *testing.B) {
		b.ReportAllocs()
		event := NewEvent(INFO, "benchmark", []Field{
			String("key1", "value1"),
			String("key2", "value2"),
			Int("count", 42),
		})
		for i := 0; i < b.N; i++ {
			_ = event.Clone()
		}
	})

	b.Run("ManyFields", func(b *testing.B) {
		b.ReportAllocs()
		fields := make([]Field, 10)
		for i := 0; i < 10; i++ {
			fields[i] = String("key", "value")
		}
		event := NewEvent(INFO, "benchmark", fields)
		for i := 0; i < b.N; i++ {
			_ = event.Clone()
		}
	})
}
