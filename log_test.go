package zlog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"testing"
)

//nolint:revive // t is unused but required for test function signature
func TestEnableStandardLogging(t *testing.T) {
	// Test that EnableStandardLogging with DEBUG level sets up all signals
	EnableStandardLogging(DEBUG)

	// Test with INFO level
	EnableStandardLogging(INFO)

	// Test with WARN level
	EnableStandardLogging(WARN)

	// Test with ERROR level
	EnableStandardLogging(ERROR)

	// Test with FATAL level
	EnableStandardLogging(FATAL)
}

// TestStderrJSONSinkOutput tests the actual JSON output format.
func TestStderrJSONSinkOutput(t *testing.T) {
	// Capture stderr
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w
	defer func() {
		os.Stderr = old
	}()

	// Process an event through the actual sink
	event := NewEvent(INFO, "test message", []Field{
		String("key", "value"),
		Int("count", 42),
	})
	event.Caller.File = "test.go"
	event.Caller.Line = 123

	ctx := context.Background()
	if _, err := stderrJSONSink.Process(ctx, event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Close write end and read output
	w.Close()
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	// Parse JSON output
	var output map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &output); err != nil {
		t.Fatalf("failed to parse JSON: %v", err)
	}

	// Verify fields
	if output["message"] != "test message" {
		t.Errorf("message = %v, want 'test message'", output["message"])
	}
	if output["signal"] != "INFO" {
		t.Errorf("signal = %v, want 'INFO'", output["signal"])
	}
	if output["key"] != "value" {
		t.Errorf("key = %v, want 'value'", output["key"])
	}
	if output["count"] != float64(42) { // JSON numbers are float64
		t.Errorf("count = %v, want 42", output["count"])
	}
	if output["caller"] != "test.go:123" {
		t.Errorf("caller = %v, want 'test.go:123'", output["caller"])
	}
	if _, ok := output["time"]; !ok {
		t.Error("missing time field")
	}
}
