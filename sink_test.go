package zlog

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

// Mock writer for testing errors.
type errorWriter struct{}

func (e errorWriter) Write(p []byte) (n int, err error) {
	return 0, errors.New("write error")
}

func TestWriterSink(t *testing.T) {
	t.Run("Basic JSON output", func(t *testing.T) {
		buf := &bytes.Buffer{}
		sink := NewWriterSink(buf)

		event := NewEvent(INFO, "test message", []Field{
			String("key", "value"),
			Int("count", 42),
		})

		err := sink.Write(event)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// Parse JSON output
		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		// Check required fields
		if result["signal"] != "INFO" {
			t.Errorf("signal = %v, want INFO", result["signal"])
		}
		if result["message"] != "test message" {
			t.Errorf("message = %v, want 'test message'", result["message"])
		}
		if result["key"] != "value" {
			t.Errorf("key = %v, want 'value'", result["key"])
		}
		if result["count"] != float64(42) { // JSON numbers are float64
			t.Errorf("count = %v, want 42", result["count"])
		}

		// Check time format
		if _, ok := result["time"].(string); !ok {
			t.Errorf("time field is not a string")
		}
	})

	t.Run("All field types", func(t *testing.T) {
		buf := &bytes.Buffer{}
		sink := NewWriterSink(buf)

		now := time.Now()
		event := NewEvent(INFO, "test", []Field{
			String("string", "value"),
			Int("int", 42),
			Int64("int64", int64(9223372036854775807)),
			Float64("float", 3.14159),
			Bool("bool", true),
			Duration("duration", 5*time.Second),
			Time("time", now),
			ByteString("bytes", []byte("data")),
			Strings("strings", []string{"a", "b", "c"}),
			Data("data", map[string]int{"x": 1, "y": 2}),
		})

		err := sink.Write(event)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		// Check field presence
		expectedFields := []string{"string", "int", "int64", "float", "bool", "duration", "time", "bytes", "strings", "data"}
		for _, field := range expectedFields {
			if _, ok := result[field]; !ok {
				t.Errorf("Missing field: %s", field)
			}
		}
	})

	t.Run("Empty event", func(t *testing.T) {
		buf := &bytes.Buffer{}
		sink := NewWriterSink(buf)

		event := NewEvent(INFO, "", nil)

		err := sink.Write(event)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Invalid JSON output: %v", err)
		}

		if result["message"] != "" {
			t.Errorf("message = %v, want empty", result["message"])
		}
	})

	t.Run("Write error", func(t *testing.T) {
		sink := NewWriterSink(errorWriter{})
		event := NewEvent(INFO, "test", nil)

		err := sink.Write(event)
		if err == nil {
			t.Errorf("Expected write error, got nil")
		}
	})

	t.Run("Name", func(t *testing.T) {
		sink := NewWriterSink(&bytes.Buffer{})
		if sink.Name() != "writer" {
			t.Errorf("Name() = %v, want 'writer'", sink.Name())
		}
	})
}

func TestWriterSinkFieldConflicts(t *testing.T) {
	buf := &bytes.Buffer{}
	sink := NewWriterSink(buf)

	// Test that user fields can override built-in fields
	event := NewEvent(INFO, "test", []Field{
		String("time", "user-time"),
		String("signal", "user-signal"),
		String("message", "user-message"),
	})

	err := sink.Write(event)
	if err != nil {
		t.Fatalf("Write failed: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	// User fields should override built-ins
	if result["time"] == "user-time" {
		t.Logf("User 'time' field overrides built-in (this may or may not be desired)")
	}
}

func TestSelfRegisteringSinks(t *testing.T) {
	// Since these interact with global state, we need to be careful
	// We'll test by checking that events flow to the correct sinks

	t.Run("StandardLogSink registration", func(t *testing.T) {
		buf := &bytes.Buffer{}
		_ = NewStandardLogSink(buf)

		// Send various signals
		Info("info test")
		Warn("warn test")
		Error("error test")
		Debug("debug test") // Should NOT appear

		output := buf.String()
		if !strings.Contains(output, "info test") {
			t.Errorf("INFO signal not routed to StandardLogSink")
		}
		if !strings.Contains(output, "warn test") {
			t.Errorf("WARN signal not routed to StandardLogSink")
		}
		if !strings.Contains(output, "error test") {
			t.Errorf("ERROR signal not routed to StandardLogSink")
		}
		if strings.Contains(output, "debug test") {
			t.Errorf("DEBUG signal incorrectly routed to StandardLogSink")
		}
	})

	t.Run("DebugSink registration", func(t *testing.T) {
		buf := &bytes.Buffer{}
		_ = NewDebugSink(buf)

		Debug("debug test")
		Info("info test") // Should NOT appear

		output := buf.String()
		if !strings.Contains(output, "debug test") {
			t.Errorf("DEBUG signal not routed to DebugSink")
		}
		if strings.Contains(output, "info test") {
			t.Errorf("INFO signal incorrectly routed to DebugSink")
		}
	})

	t.Run("AuditSink registration", func(t *testing.T) {
		buf := &bytes.Buffer{}
		_ = NewAuditSink(buf)

		Emit(AUDIT, "audit test")
		Emit(SECURITY, "security test")
		Emit(INFO, "info test") // Should NOT appear

		output := buf.String()
		if !strings.Contains(output, "audit test") {
			t.Errorf("AUDIT signal not routed to AuditSink")
		}
		if !strings.Contains(output, "security test") {
			t.Errorf("SECURITY signal not routed to AuditSink")
		}
		if strings.Contains(output, "info test") {
			t.Errorf("INFO signal incorrectly routed to AuditSink")
		}
	})

	t.Run("MetricSink registration", func(t *testing.T) {
		buf := &bytes.Buffer{}
		_ = NewMetricSink(buf)

		Emit(METRIC, "metric test")
		Emit(INFO, "info test") // Should NOT appear

		output := buf.String()
		if !strings.Contains(output, "metric test") {
			t.Errorf("METRIC signal not routed to MetricSink")
		}
		if strings.Contains(output, "info test") {
			t.Errorf("INFO signal incorrectly routed to MetricSink")
		}
	})
}

// Benchmarks.
func BenchmarkWriterSink(b *testing.B) {
	sink := NewWriterSink(io.Discard)

	b.Run("SimpleEvent", func(b *testing.B) {
		b.ReportAllocs()
		event := NewEvent(INFO, "benchmark message", nil)
		for i := 0; i < b.N; i++ {
			_ = sink.Write(event)
		}
	})

	b.Run("EventWithFields", func(b *testing.B) {
		b.ReportAllocs()
		event := NewEvent(INFO, "benchmark message", []Field{
			String("user", "alice"),
			Int("user_id", 42),
			Bool("active", true),
			Time("timestamp", time.Now()),
		})
		for i := 0; i < b.N; i++ {
			_ = sink.Write(event)
		}
	})

	b.Run("LargeEvent", func(b *testing.B) {
		b.ReportAllocs()
		fields := make([]Field, 20)
		for i := 0; i < 20; i++ {
			fields[i] = String("field"+string(rune(i)), "value"+string(rune(i)))
		}
		event := NewEvent(INFO, "benchmark message", fields)
		for i := 0; i < b.N; i++ {
			_ = sink.Write(event)
		}
	})

	b.Run("ConcurrentWrites", func(b *testing.B) {
		b.ReportAllocs()
		event := NewEvent(INFO, "concurrent test", []Field{
			String("key", "value"),
		})

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = sink.Write(event)
			}
		})
	})
}

func BenchmarkJSONEncoding(b *testing.B) {
	// Compare manual map building vs struct encoding
	event := NewEvent(INFO, "test", []Field{
		String("key1", "value1"),
		String("key2", "value2"),
		Int("count", 42),
	})

	b.Run("MapEncoding", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			entry := map[string]interface{}{
				"time":    event.Time.Format(time.RFC3339Nano),
				"signal":  string(event.Signal),
				"message": event.Message,
			}
			for _, field := range event.Fields {
				entry[field.Key] = field.Value
			}
			_ = json.NewEncoder(io.Discard).Encode(entry)
		}
	})
}
