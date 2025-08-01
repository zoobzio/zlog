package zlog

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewPrettyConsoleSink(t *testing.T) {
	t.Run("creates sink successfully", func(t *testing.T) {
		sink := NewPrettyConsoleSink()
		if sink == nil {
			t.Error("expected non-nil sink")
		}
	})

	t.Run("processes basic event", func(t *testing.T) {
		// Capture stderr
		oldStderr := os.Stderr
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatalf("failed to create pipe: %v", err)
		}
		os.Stderr = w

		sink := NewPrettyConsoleSink()
		event := NewEvent("INFO", "test message", nil)

		_, err = sink.Process(context.Background(), event)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Close writer and read output
		w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		if err != nil {
			t.Errorf("failed to read from pipe: %v", err)
		}
		output := buf.String()

		// Verify output contains expected elements
		if !strings.Contains(output, "INFO") {
			t.Error("expected output to contain INFO")
		}
		if !strings.Contains(output, "test message") {
			t.Error("expected output to contain test message")
		}
	})
}

func TestFormatSignalWithSymbol(t *testing.T) {
	tests := []struct {
		name      string
		signal    Signal
		useColors bool
		contains  []string
	}{
		{
			name:      "INFO with colors",
			signal:    INFO,
			useColors: true,
			contains:  []string{"[INFO]", "✓"},
		},
		{
			name:      "ERROR with colors",
			signal:    ERROR,
			useColors: true,
			contains:  []string{"[ERROR]", "✗"},
		},
		{
			name:      "WARN with colors",
			signal:    WARN,
			useColors: true,
			contains:  []string{"[WARN]", "⚠"},
		},
		{
			name:      "DEBUG with colors",
			signal:    DEBUG,
			useColors: true,
			contains:  []string{"[DEBUG]", "🔍"},
		},
		{
			name:      "FATAL with colors",
			signal:    FATAL,
			useColors: true,
			contains:  []string{"[FATAL]", "💀"},
		},
		{
			name:      "INFO without colors",
			signal:    INFO,
			useColors: false,
			contains:  []string{"[INFO]", "✓"},
		},
		{
			name:      "ERROR without colors",
			signal:    ERROR,
			useColors: false,
			contains:  []string{"[ERROR]", "✗"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSignalWithSymbol(tt.signal, tt.useColors)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("expected result to contain %q, got: %s", expected, result)
				}
			}

			// Verify color codes are present/absent as expected
			hasColorCodes := strings.Contains(result, "\033[")
			if tt.useColors && !hasColorCodes {
				t.Error("expected color codes when useColors=true")
			}
			if !tt.useColors && hasColorCodes {
				t.Error("expected no color codes when useColors=false")
			}
		})
	}
}

func TestFormatFields(t *testing.T) {
	t.Run("empty fields", func(t *testing.T) {
		result := formatFields(nil, true)
		if result != "" {
			t.Errorf("expected empty string for nil fields, got: %q", result)
		}

		result = formatFields([]Field{}, false)
		if result != "" {
			t.Errorf("expected empty string for empty fields, got: %q", result)
		}
	})

	t.Run("single field with colors", func(t *testing.T) {
		fields := []Field{String("user_id", "12345")}
		result := formatFields(fields, true)

		if !strings.Contains(result, "user_id") {
			t.Error("expected result to contain field key")
		}
		if !strings.Contains(result, "12345") {
			t.Error("expected result to contain field value")
		}
		if !strings.Contains(result, "└─") {
			t.Error("expected single field to use last branch symbol")
		}
	})

	t.Run("multiple fields with colors", func(t *testing.T) {
		fields := []Field{
			String("user_id", "12345"),
			Int("count", 42),
			Bool("active", true),
		}
		result := formatFields(fields, true)

		// Check all fields are present
		expectedContents := []string{"user_id", "12345", "count", "42", "active", "true"}
		for _, expected := range expectedContents {
			if !strings.Contains(result, expected) {
				t.Errorf("expected result to contain %q", expected)
			}
		}

		// Check tree structure
		if !strings.Contains(result, "├─") {
			t.Error("expected intermediate branch symbols")
		}
		if !strings.Contains(result, "└─") {
			t.Error("expected final branch symbol")
		}

		// Count lines (should be 3 fields + 1 leading newline = 4 lines total)
		lines := strings.Split(result, "\n")
		if len(lines) != 4 {
			t.Errorf("expected 4 lines (including leading newline), got %d", len(lines))
		}
	})

	t.Run("fields without colors", func(t *testing.T) {
		fields := []Field{String("key", "value")}
		result := formatFields(fields, false)

		if strings.Contains(result, "\033[") {
			t.Error("expected no color codes when useColors=false")
		}
		if !strings.Contains(result, "key") || !strings.Contains(result, "value") {
			t.Error("expected field key and value to be present")
		}
	})
}

func TestFormatCaller(t *testing.T) {
	t.Run("empty caller", func(t *testing.T) {
		caller := CallerInfo{}
		result := formatCaller(caller, true)
		if result != "" {
			t.Errorf("expected empty string for empty caller, got: %q", result)
		}
	})

	t.Run("caller with colors", func(t *testing.T) {
		caller := CallerInfo{File: "main.go", Line: 42}
		result := formatCaller(caller, true)

		if !strings.Contains(result, "main.go:42") {
			t.Error("expected caller to contain file:line")
		}
		if !strings.Contains(result, "\033[") {
			t.Error("expected color codes when useColors=true")
		}
	})

	t.Run("caller without colors", func(t *testing.T) {
		caller := CallerInfo{File: "test.go", Line: 123}
		result := formatCaller(caller, false)

		if !strings.Contains(result, "test.go:123") {
			t.Error("expected caller to contain file:line")
		}
		if strings.Contains(result, "\033[") {
			t.Error("expected no color codes when useColors=false")
		}
	})
}

func TestPrettyConsoleSinkIntegration(t *testing.T) {
	// Capture stderr
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	sink := NewPrettyConsoleSink()

	t.Run("complete event formatting", func(t *testing.T) {
		event := NewEvent("WARN", "Something went wrong", []Field{
			String("user_id", "user123"),
			Int("retry_count", 3),
			Bool("critical", false),
		})
		event.Caller = CallerInfo{File: "service.go", Line: 456}
		event.Time = time.Date(2023, 10, 20, 15, 4, 5, 0, time.UTC)

		_, err := sink.Process(context.Background(), event)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	// Close writer and read output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	if err != nil {
		t.Errorf("failed to read from pipe: %v", err)
	}
	output := buf.String()

	// Verify all components are present
	expectedContents := []string{
		"WARN", "⚠", "15:04:05", "Something went wrong",
		"service.go:456", "user_id", "user123", "retry_count", "3", "critical", "false",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(output, expected) {
			t.Errorf("expected output to contain %q, got output: %s", expected, output)
		}
	}

	// Verify tree structure for fields
	if !strings.Contains(output, "├─") || !strings.Contains(output, "└─") {
		t.Error("expected tree structure in field display")
	}
}

func TestPrettyConsoleSinkWithAdapters(t *testing.T) {
	// Capture stderr for all tests
	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stderr = w

	t.Run("works with filter adapter", func(t *testing.T) {
		sink := NewPrettyConsoleSink().WithFilter(func(_ context.Context, e Event) bool {
			return e.Signal == "ERROR" // Only show errors
		})

		// This should be filtered out
		infoEvent := NewEvent("INFO", "info message", nil)
		_, err := sink.Process(context.Background(), infoEvent)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// This should pass through
		errorEvent := NewEvent("ERROR", "error message", nil)
		_, err2 := sink.Process(context.Background(), errorEvent)
		if err2 != nil {
			t.Errorf("unexpected error: %v", err2)
		}
	})

	t.Run("works with async adapter", func(t *testing.T) {
		sink := NewPrettyConsoleSink().WithAsync()

		event := NewEvent("INFO", "async test", []Field{String("async", "true")})

		start := time.Now()
		_, err := sink.Process(context.Background(), event)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("unexpected error with async: %v", err)
		}

		if elapsed > 50*time.Millisecond {
			t.Errorf("async processing took too long: %v", elapsed)
		}

		// Give async processing time to complete
		time.Sleep(100 * time.Millisecond)
	})

	// Close writer and read output
	w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	if err != nil {
		t.Errorf("failed to read from pipe: %v", err)
	}
	output := buf.String()

	// Should contain the error message but not the info message (due to filter)
	if !strings.Contains(output, "error message") {
		t.Error("expected filtered error message in output")
	}
	if strings.Contains(output, "info message") {
		t.Error("expected info message to be filtered out")
	}

	// Should contain async test message
	if !strings.Contains(output, "async test") {
		t.Error("expected async test message in output")
	}
}

func TestIsTerminal(t *testing.T) {
	t.Run("terminal detection", func(_ *testing.T) {
		// This test is environment-dependent, so we just verify it returns a boolean
		result := isTerminal()

		// Should return either true or false
		_ = result // Just verify it doesn't panic
	})

	t.Run("respects TERM environment variable", func(t *testing.T) {
		// Save original TERM
		originalTerm := os.Getenv("TERM")
		defer os.Setenv("TERM", originalTerm)

		// Set TERM to empty (should disable colors)
		os.Setenv("TERM", "")
		result := isTerminal()
		if result {
			t.Error("expected isTerminal() to return false when TERM is empty")
		}

		// Set TERM to "dumb" (should disable colors)
		os.Setenv("TERM", "dumb")
		result = isTerminal()
		if result {
			t.Error("expected isTerminal() to return false when TERM is 'dumb'")
		}
	})
}
