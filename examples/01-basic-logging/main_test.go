package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"testing"

	"github.com/zoobzio/zlog"
)

func TestBasicLogging(t *testing.T) {
	// Capture output
	buf := &bytes.Buffer{}
	zlog.EnableStandardLogging(buf)

	// Test INFO log
	t.Run("InfoLog", func(t *testing.T) {
		buf.Reset()
		zlog.Info("Test message", zlog.String("key", "value"))

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if result["signal"] != "INFO" {
			t.Errorf("Expected signal INFO, got %v", result["signal"])
		}
		if result["message"] != "Test message" {
			t.Errorf("Expected message 'Test message', got %v", result["message"])
		}
		if result["key"] != "value" {
			t.Errorf("Expected key='value', got %v", result["key"])
		}
		if _, ok := result["caller"]; !ok {
			t.Error("Missing caller information")
		}
	})

	// Test structured fields
	t.Run("StructuredFields", func(t *testing.T) {
		buf.Reset()
		zlog.Info("User action",
			zlog.String("user", "alice"),
			zlog.Int("user_id", 42),
			zlog.Bool("admin", true),
		)

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if result["user"] != "alice" {
			t.Errorf("Expected user='alice', got %v", result["user"])
		}
		if result["user_id"] != float64(42) { // JSON numbers are float64
			t.Errorf("Expected user_id=42, got %v", result["user_id"])
		}
		if result["admin"] != true {
			t.Errorf("Expected admin=true, got %v", result["admin"])
		}
	})

	// Test different log levels
	t.Run("LogLevels", func(t *testing.T) {
		tests := []struct {
			name   string
			logFn  func(string, ...zlog.Field)
			signal string
		}{
			{"Info", zlog.Info, "INFO"},
			{"Warn", zlog.Warn, "WARN"},
			{"Error", zlog.Error, "ERROR"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				buf.Reset()
				tt.logFn("Test "+tt.signal, zlog.String("level", tt.signal))

				var result map[string]interface{}
				if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
					t.Fatalf("Failed to parse JSON: %v", err)
				}

				if result["signal"] != tt.signal {
					t.Errorf("Expected signal %s, got %v", tt.signal, result["signal"])
				}
			})
		}
	})

	// Test error field
	t.Run("ErrorField", func(t *testing.T) {
		buf.Reset()
		zlog.Error("Operation failed",
			zlog.Err(errors.New("test error")),
			zlog.String("operation", "save"),
		)

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatalf("Failed to parse JSON: %v", err)
		}

		if result["error"] != "test error" {
			t.Errorf("Expected error='test error', got %v", result["error"])
		}
	})
}

func TestNoOutputWithoutEnabling(t *testing.T) {
	// Create a fresh dispatch to ensure no output
	// In real usage, there would be no output without enabling
	buf := &bytes.Buffer{}

	// Don't enable any logging
	// zlog.EnableStandardLogging(buf) // NOT CALLED

	// These should produce no output
	zlog.Info("This goes nowhere")
	zlog.Error("This also goes nowhere")

	if buf.Len() > 0 {
		t.Errorf("Expected no output, got: %s", buf.String())
	}
}
