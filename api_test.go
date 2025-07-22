package zlog

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/zoobzio/pipz"
)

func TestCallerInfo(t *testing.T) {
	buf := &bytes.Buffer{}
	EnableStandardLogging(buf)

	Info("test message") // LINE 21 - we'll check this exact line

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	caller, ok := result["caller"].(string)
	if !ok {
		t.Fatal("No caller field in output")
	}

	// Check it has the right file and line number
	if !strings.HasSuffix(caller, "api_test.go:21") {
		t.Errorf("Expected caller to end with api_test.go:21, got %s", caller)
	}
}

func TestLogLevelFunctions(t *testing.T) {
	// Set up a test sink
	buf := &bytes.Buffer{}
	EnableStandardLogging(buf)
	EnableDebugLogging(buf)

	tests := []struct {
		name    string
		logFunc func(string, ...Field)
		signal  Signal
		message string
		fields  []Field
	}{
		{
			name:    "Debug",
			logFunc: Debug,
			signal:  DEBUG,
			message: "debug message",
			fields:  []Field{String("level", "debug")},
		},
		{
			name:    "Info",
			logFunc: Info,
			signal:  INFO,
			message: "info message",
			fields:  []Field{String("level", "info")},
		},
		{
			name:    "Warn",
			logFunc: Warn,
			signal:  WARN,
			message: "warn message",
			fields:  []Field{String("level", "warn")},
		},
		{
			name:    "Error",
			logFunc: Error,
			signal:  ERROR,
			message: "error message",
			fields:  []Field{String("level", "error")},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf.Reset()
			tt.logFunc(tt.message, tt.fields...)

			// Give time for processing
			time.Sleep(10 * time.Millisecond)

			// Parse JSON output
			var result map[string]interface{}
			if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
				t.Fatalf("Invalid JSON output: %v", err)
			}

			if result["signal"] != string(tt.signal) {
				t.Errorf("signal = %v, want %v", result["signal"], tt.signal)
			}
			if result["message"] != tt.message {
				t.Errorf("message = %v, want %v", result["message"], tt.message)
			}
			if result["level"] != tt.fields[0].Value {
				t.Errorf("level field = %v, want %v", result["level"], tt.fields[0].Value)
			}
		})
	}
}

func TestEmitCustomSignal(t *testing.T) {
	buf := &bytes.Buffer{}
	customSignal := Signal("CUSTOM_SIGNAL")

	// Route custom signal
	RouteSignal(customSignal, NewWriterSink(buf))

	Emit(customSignal, "custom message", String("type", "custom"))

	time.Sleep(10 * time.Millisecond)

	var result map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("Invalid JSON output: %v", err)
	}

	if result["signal"] != "CUSTOM_SIGNAL" {
		t.Errorf("signal = %v, want CUSTOM_SIGNAL", result["signal"])
	}
	if result["message"] != "custom message" {
		t.Errorf("message = %v, want 'custom message'", result["message"])
	}
}

func TestFatal(t *testing.T) {
	if os.Getenv("BE_FATAL") == "1" {
		// This will be run in a subprocess
		// Need to enable logging to see output
		EnableStandardLogging(os.Stderr)
		Fatal("fatal error", String("test", "fatal"))
		return
	}

	// Run this test in a subprocess
	cmd := exec.Command(os.Args[0], "-test.run=TestFatal")
	cmd.Env = append(os.Environ(), "BE_FATAL=1")

	output, err := cmd.CombinedOutput()

	// The subprocess should have exited with status 1
	if e, ok := err.(*exec.ExitError); !ok || e.ExitCode() != 1 {
		t.Fatalf("Expected exit code 1, got %v", err)
	}

	// Check that the fatal message was logged (as JSON)
	if !strings.Contains(string(output), "fatal error") {
		t.Errorf("Fatal message not in output: %s", output)
	}
	if !strings.Contains(string(output), "FATAL") {
		t.Errorf("FATAL signal not in output: %s", output)
	}
}

func TestEnableFunctions(t *testing.T) {
	// Save dispatch state
	oldDispatch := dispatch
	defer func() { dispatch = oldDispatch }()

	// Create fresh dispatch
	dispatch = &Dispatch{
		signalRoutes: make(map[Signal][]Sink),
	}
	dispatch.pipeline = pipz.ProcessorFunc[Event](func(ctx context.Context, e Event) (Event, error) {
		key := string(e.Signal)
		if chainable, ok := dispatch.switchRoutes.Load(key); ok {
			return chainable.(pipz.Chainable[Event]).Process(ctx, e)
		}
		return e, nil
	})

	t.Run("EnableStandardLogging", func(t *testing.T) {
		buf := &bytes.Buffer{}
		EnableStandardLogging(buf)

		// Test that standard signals work
		Info("info test")
		Warn("warn test")
		Error("error test")
		Debug("debug test") // Should NOT work

		time.Sleep(20 * time.Millisecond)

		output := buf.String()
		if !strings.Contains(output, "info test") {
			t.Error("INFO not working after EnableStandardLogging")
		}
		if !strings.Contains(output, "warn test") {
			t.Error("WARN not working after EnableStandardLogging")
		}
		if !strings.Contains(output, "error test") {
			t.Error("ERROR not working after EnableStandardLogging")
		}
		if strings.Contains(output, "debug test") {
			t.Error("DEBUG incorrectly working after EnableStandardLogging")
		}
	})

	t.Run("EnableDebugLogging", func(t *testing.T) {
		buf := &bytes.Buffer{}
		EnableDebugLogging(buf)

		Debug("debug test")
		Info("info test") // Should NOT work (unless standard is also enabled)

		time.Sleep(20 * time.Millisecond)

		output := buf.String()
		if !strings.Contains(output, "debug test") {
			t.Error("DEBUG not working after EnableDebugLogging")
		}
	})

	t.Run("EnableAuditLogging", func(t *testing.T) {
		buf := &bytes.Buffer{}
		EnableAuditLogging(buf)

		Emit(AUDIT, "audit test")
		Emit(SECURITY, "security test")
		Info("info test") // Should NOT work

		time.Sleep(20 * time.Millisecond)

		output := buf.String()
		if !strings.Contains(output, "audit test") {
			t.Error("AUDIT not working after EnableAuditLogging")
		}
		if !strings.Contains(output, "security test") {
			t.Error("SECURITY not working after EnableAuditLogging")
		}
		if strings.Contains(output, "info test") {
			t.Error("INFO incorrectly working after EnableAuditLogging")
		}
	})

	t.Run("EnableMetricLogging", func(t *testing.T) {
		buf := &bytes.Buffer{}
		EnableMetricLogging(buf)

		Emit(METRIC, "metric test")
		Info("info test") // Should NOT work

		time.Sleep(20 * time.Millisecond)

		output := buf.String()
		if !strings.Contains(output, "metric test") {
			t.Error("METRIC not working after EnableMetricLogging")
		}
		if strings.Contains(output, "info test") {
			t.Error("INFO incorrectly working after EnableMetricLogging")
		}
	})
}

func TestMultipleEnables(t *testing.T) {
	// Test enabling multiple logging types
	stdBuf := &bytes.Buffer{}
	auditBuf := &bytes.Buffer{}

	EnableStandardLogging(stdBuf)
	EnableAuditLogging(auditBuf)

	// Send various signals
	Info("standard info")
	Emit(AUDIT, "audit event")

	time.Sleep(20 * time.Millisecond)

	// Check outputs are separated
	if !strings.Contains(stdBuf.String(), "standard info") {
		t.Error("Standard logging not working")
	}
	if strings.Contains(stdBuf.String(), "audit event") {
		t.Error("Audit event incorrectly in standard output")
	}

	if !strings.Contains(auditBuf.String(), "audit event") {
		t.Error("Audit logging not working")
	}
	if strings.Contains(auditBuf.String(), "standard info") {
		t.Error("Standard event incorrectly in audit output")
	}
}

// Benchmarks.
func BenchmarkAPIFunctions(b *testing.B) {
	// Set up sink
	EnableStandardLogging(io.Discard)
	EnableDebugLogging(io.Discard)

	b.Run("Debug", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Debug("benchmark message")
		}
	})

	b.Run("Info", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Info("benchmark message")
		}
	})

	b.Run("InfoWithFields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Info("benchmark message",
				String("user", "alice"),
				Int("count", 42),
				Bool("active", true),
			)
		}
	})

	b.Run("Emit", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Emit("CUSTOM", "benchmark message")
		}
	})

	b.Run("EmitWithFields", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			Emit("CUSTOM", "benchmark message",
				String("key1", "value1"),
				String("key2", "value2"),
			)
		}
	})
}

func BenchmarkEnableFunctions(b *testing.B) {
	b.Run("EnableStandardLogging", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			EnableStandardLogging(io.Discard)
		}
	})

	b.Run("EnableAllTypes", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			EnableStandardLogging(io.Discard)
			EnableDebugLogging(io.Discard)
			EnableAuditLogging(io.Discard)
			EnableMetricLogging(io.Discard)
		}
	})
}

func BenchmarkEndToEnd(b *testing.B) {
	// Full end-to-end benchmark
	EnableStandardLogging(io.Discard)

	fields := []Field{
		String("user", "alice"),
		Int("user_id", 42),
		Bool("active", true),
		Time("timestamp", time.Now()),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		Info("benchmark message", fields...)
	}
}
