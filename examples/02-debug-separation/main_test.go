package main

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/zoobzio/zlog"
)

func TestDebugSeparation(t *testing.T) {
	// Clean up before test
	os.Remove("debug.log")
	defer os.Remove("debug.log")

	// Capture standard output
	stdBuf := &bytes.Buffer{}
	zlog.EnableStandardLogging(stdBuf)

	// Create debug file
	debugFile, err := os.Create("debug.log")
	if err != nil {
		t.Fatalf("Failed to create debug file: %v", err)
	}
	defer debugFile.Close()

	zlog.EnableDebugLogging(debugFile)

	// Generate logs of different levels
	t.Run("LogSeparation", func(t *testing.T) {
		zlog.Debug("Debug message")
		zlog.Info("Info message")
		zlog.Warn("Warn message")
		zlog.Error("Error message")

		// Force flush
		debugFile.Sync()

		// Check standard output - should NOT have DEBUG
		stdOutput := stdBuf.String()
		if strings.Contains(stdOutput, "Debug message") {
			t.Error("Debug message should not appear in standard output")
		}
		if !strings.Contains(stdOutput, "Info message") {
			t.Error("Info message should appear in standard output")
		}
		if !strings.Contains(stdOutput, "Warn message") {
			t.Error("Warn message should appear in standard output")
		}
		if !strings.Contains(stdOutput, "Error message") {
			t.Error("Error message should appear in standard output")
		}

		// Check debug file - should ONLY have DEBUG
		debugContent, err := os.ReadFile("debug.log")
		if err != nil {
			t.Fatalf("Failed to read debug file: %v", err)
		}

		debugStr := string(debugContent)
		if !strings.Contains(debugStr, "Debug message") {
			t.Error("Debug message should appear in debug file")
		}
		if strings.Contains(debugStr, "Info message") {
			t.Error("Info message should not appear in debug file")
		}
	})
}

func TestDebugOnlyMode(t *testing.T) {
	// Test with ONLY debug logging enabled
	os.Remove("debug.log")
	defer os.Remove("debug.log")

	debugFile, err := os.Create("debug.log")
	if err != nil {
		t.Fatalf("Failed to create debug file: %v", err)
	}
	defer debugFile.Close()

	// Only enable debug logging
	zlog.EnableDebugLogging(debugFile)
	// Note: NOT calling EnableStandardLogging

	// These should only go to debug file
	zlog.Debug("Debug only message")

	// These should go nowhere (no standard logging enabled)
	zlog.Info("This goes nowhere")
	zlog.Error("This also goes nowhere")

	debugFile.Sync()

	// Check debug file
	debugContent, err := os.ReadFile("debug.log")
	if err != nil {
		t.Fatalf("Failed to read debug file: %v", err)
	}

	debugStr := string(debugContent)
	if !strings.Contains(debugStr, "Debug only message") {
		t.Error("Debug message should appear in debug file")
	}

	// Verify it's valid JSON
	lines := strings.Split(strings.TrimSpace(debugStr), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		var result map[string]interface{}
		if err := json.Unmarshal([]byte(line), &result); err != nil {
			t.Errorf("Invalid JSON in debug log: %v", err)
		}
		if result["signal"] != "DEBUG" {
			t.Errorf("Expected signal DEBUG, got %v", result["signal"])
		}
	}
}

func TestMultipleDebugSinks(t *testing.T) {
	// Test that we can have multiple debug sinks
	buf1 := &bytes.Buffer{}
	buf2 := &bytes.Buffer{}

	// Route DEBUG to multiple destinations
	zlog.RouteSignal(zlog.DEBUG, zlog.NewWriterSink(buf1))
	zlog.RouteSignal(zlog.DEBUG, zlog.NewWriterSink(buf2))

	zlog.Debug("Multi-sink debug")

	// Both should receive the message
	if !strings.Contains(buf1.String(), "Multi-sink debug") {
		t.Error("First debug sink should receive message")
	}
	if !strings.Contains(buf2.String(), "Multi-sink debug") {
		t.Error("Second debug sink should receive message")
	}
}
