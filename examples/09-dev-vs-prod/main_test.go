package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/zoobzio/zlog"
)

func TestEnvironmentConfiguration(t *testing.T) {
	// Save original stdout/stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	defer func() {
		os.Stdout = oldStdout
		os.Stderr = oldStderr
	}()

	t.Run("DevelopmentConfig", func(t *testing.T) {
		// Clean up before test
		cleanup()

		// Set development environment
		os.Setenv("ENV", "development")
		defer os.Unsetenv("ENV")

		// Capture output
		r, w, _ := os.Pipe()
		os.Stdout = w
		os.Stderr = w

		// Configure
		configureDevelopment()
		w.Close()

		// Read output
		buf := &bytes.Buffer{}
		buf.ReadFrom(r)
		output := buf.String()

		// Should see development indicators
		if !strings.Contains(output, "DEVELOPMENT mode") {
			t.Error("Should indicate development mode")
		}
		if !strings.Contains(output, "Debug logs:") {
			t.Error("Should show debug log configuration")
		}

		// Check files were created
		if _, err := os.Stat("debug.log"); os.IsNotExist(err) {
			t.Error("debug.log should be created in development")
		}
		if _, err := os.Stat("metrics.log"); os.IsNotExist(err) {
			t.Error("metrics.log should be created in development")
		}

		cleanup()
	})

	t.Run("ProductionConfig", func(t *testing.T) {
		// Set production environment
		os.Setenv("ENV", "production")
		defer os.Unsetenv("ENV")

		// In production, debug.log should NOT be created
		configureProduction()

		if _, err := os.Stat("debug.log"); !os.IsNotExist(err) {
			t.Error("debug.log should NOT be created in production")
		}
	})
}

func TestDebugLoggingByEnvironment(t *testing.T) {
	devBuf := &bytes.Buffer{}
	prodBuf := &bytes.Buffer{}

	t.Run("DebugInDevelopment", func(t *testing.T) {
		// Configure for development
		zlog.EnableStandardLogging(devBuf)
		zlog.EnableDebugLogging(devBuf)

		// Emit debug log
		zlog.Debug("Debug message", zlog.String("env", "dev"))

		// Should appear in development
		if !strings.Contains(devBuf.String(), "Debug message") {
			t.Error("Debug logs should appear in development")
		}
	})

	t.Run("NoDebugInProduction", func(t *testing.T) {
		// Configure for production (no debug sink)
		zlog.EnableStandardLogging(prodBuf)
		// Intentionally NOT calling EnableDebugLogging

		// Emit debug log
		zlog.Debug("Debug message", zlog.String("env", "prod"))

		// Should NOT appear in production
		if strings.Contains(prodBuf.String(), "Debug message") {
			t.Error("Debug logs should NOT appear in production")
		}
	})
}

func TestCloudSink(t *testing.T) {
	sink := NewCloudSink("test-bucket")

	t.Run("Buffering", func(t *testing.T) {
		// Add events
		for i := 0; i < 50; i++ {
			event := zlog.NewEvent(zlog.INFO, "Test event", nil)
			sink.Write(event)
		}

		// Should be buffered
		if len(sink.buffer) != 50 {
			t.Errorf("Expected 50 buffered events, got %d", len(sink.buffer))
		}

		// Add 50 more to trigger flush
		for i := 0; i < 50; i++ {
			event := zlog.NewEvent(zlog.INFO, "Test event", nil)
			sink.Write(event)
		}

		// Buffer should be cleared after flush
		if len(sink.buffer) != 0 {
			t.Errorf("Buffer should be empty after flush, got %d", len(sink.buffer))
		}
	})

	t.Run("Name", func(t *testing.T) {
		if sink.Name() != "cloud:test-bucket" {
			t.Errorf("Expected name 'cloud:test-bucket', got %s", sink.Name())
		}
	})
}

func TestMetricsAggregatorSink(t *testing.T) {
	sink := NewMetricsAggregatorSink()

	t.Run("MetricAggregation", func(t *testing.T) {
		// Send some metrics
		events := []struct {
			name  string
			value float64
		}{
			{"cpu.usage", 45.5},
			{"cpu.usage", 50.2},
			{"memory.usage", 1024.0},
		}

		for _, e := range events {
			event := zlog.NewEvent(zlog.METRIC, e.name, []zlog.Field{
				zlog.Float64("value", e.value),
			})
			sink.Write(event)
		}

		// Check aggregation
		if len(sink.metrics) != 2 {
			t.Errorf("Expected 2 metric types, got %d", len(sink.metrics))
		}

		if len(sink.metrics["cpu.usage"]) != 2 {
			t.Errorf("Expected 2 cpu.usage values, got %d", len(sink.metrics["cpu.usage"]))
		}
	})
}

func TestSampledSink(t *testing.T) {
	buf := &bytes.Buffer{}
	baseSink := zlog.NewWriterSink(buf)

	// Create 10% sampling sink
	sampledSink := NewSampledSink(0.1, baseSink)

	t.Run("Sampling", func(t *testing.T) {
		// Send 1000 events
		for i := 0; i < 1000; i++ {
			event := zlog.NewEvent(zlog.INFO, "Test", nil)
			sampledSink.Write(event)
		}

		// Count actual events written
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		actualCount := 0
		for _, line := range lines {
			if line != "" {
				actualCount++
			}
		}

		// Should be roughly 10% (allow for randomness)
		if actualCount < 50 || actualCount > 150 {
			t.Errorf("Expected ~100 events (10%%), got %d", actualCount)
		}
	})
}

func TestApplicationBehavior(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.EnableStandardLogging(buf)
	zlog.EnableDebugLogging(buf)

	app := &Application{env: "test"}

	t.Run("RequestHandling", func(t *testing.T) {
		buf.Reset()
		app.HandleRequest("test_req_1")

		output := buf.String()

		// Should have both debug and info logs
		if !strings.Contains(output, "Request started") {
			t.Error("Missing debug log for request start")
		}
		if !strings.Contains(output, "Request completed") {
			t.Error("Missing info log for request completion")
		}

		// Parse the completion log
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Request completed") {
				var log map[string]interface{}
				if err := json.Unmarshal([]byte(line), &log); err == nil {
					if log["request_id"] != "test_req_1" {
						t.Errorf("Expected request_id=test_req_1, got %v", log["request_id"])
					}
					if log["status"] != float64(200) {
						t.Errorf("Expected status=200, got %v", log["status"])
					}
				}
			}
		}
	})

	t.Run("ErrorSimulation", func(t *testing.T) {
		buf.Reset()
		app.SimulateErrors()

		output := buf.String()

		// Should have warning and error
		if !strings.Contains(output, "Cache miss rate high") {
			t.Error("Missing warning log")
		}
		if !strings.Contains(output, "Database connection failed") {
			t.Error("Missing error log")
		}
		if !strings.Contains(output, "Error recovery initiated") {
			t.Error("Missing debug log for error recovery")
		}
	})
}

func TestDevSink(t *testing.T) {
	buf := &bytes.Buffer{}
	sink := NewDevSink(buf)

	t.Run("ColoredOutput", func(t *testing.T) {
		// Test different signal colors
		signals := []struct {
			signal zlog.Signal
			color  string
		}{
			{zlog.DEBUG, "\033[36m"}, // Cyan
			{zlog.INFO, "\033[32m"},  // Green
			{zlog.WARN, "\033[33m"},  // Yellow
			{zlog.ERROR, "\033[31m"}, // Red
		}

		for _, s := range signals {
			buf.Reset()
			event := zlog.NewEvent(s.signal, "Test message", nil)
			sink.Write(event)

			output := buf.String()
			if !strings.Contains(output, s.color) {
				t.Errorf("Expected color code %s for signal %s", s.color, s.signal)
			}
			if !strings.Contains(output, string(s.signal)) {
				t.Errorf("Expected signal %s in output", s.signal)
			}
		}
	})
}

func cleanup() {
	os.Remove("debug.log")
	os.Remove("metrics.log")
	os.Remove("security.log")
}
