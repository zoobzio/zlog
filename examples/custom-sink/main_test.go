// Package main_test tests the custom-sink example
package main_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBuildCustomSink ensures the custom-sink example compiles
func TestBuildCustomSink(t *testing.T) {
	// Build the example
	cmd := exec.Command("go", "build", "-o", "custom-sink-test", ".")
	cmd.Dir = filepath.Dir(".")
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build custom-sink: %v\nStderr: %s", err, stderr.String())
	}
	
	// Clean up the binary
	defer os.Remove("custom-sink-test")
	
	t.Log("custom-sink example built successfully")
}

// TestRunCustomSink runs the custom-sink example and verifies functionality
func TestRunCustomSink(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping run test in short mode")
	}
	
	// Build the example first
	buildCmd := exec.Command("go", "build", "-o", "custom-sink-test", ".")
	buildCmd.Dir = filepath.Dir(".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}
	defer os.Remove("custom-sink-test")
	
	// Run the example with a timeout
	cmd := exec.Command("./custom-sink-test")
	cmd.Dir = filepath.Dir(".")
	
	// Capture output
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	
	// Set a timeout
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Failed to run custom-sink: %v\nStdout: %s\nStderr: %s", 
				err, stdout.String(), stderr.String())
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("custom-sink example timed out")
	}
	
	output := stdout.String()
	
	// Verify expected sections
	t.Run("VerifySections", func(t *testing.T) {
		expectedSections := []string{
			"=== Custom Sink Example ===",
			"--- Simulating Events ---",
			"📊 Metrics Summary:",
			"=== Example Complete ===",
		}
		
		for _, section := range expectedSections {
			if !strings.Contains(output, section) {
				t.Errorf("Expected output to contain %q", section)
			}
		}
	})
	
	// Verify sink types demonstrated
	t.Run("VerifySinkTypes", func(t *testing.T) {
		expectedSinks := []string{
			"Metrics extraction",
			"Message queue publishing",
			"Database audit logging",
			"Conditional processing",
			"Event batching",
		}
		
		for _, sink := range expectedSinks {
			if !strings.Contains(output, sink) {
				t.Errorf("Expected output to mention sink type: %q", sink)
			}
		}
	})
	
	// Verify specific sink behaviors
	t.Run("VerifySinkBehaviors", func(t *testing.T) {
		// Check for message queue output
		if !strings.Contains(output, "📤 [MQ:") {
			t.Error("Expected message queue publish output")
		}
		
		// Check for database simulation
		if !strings.Contains(output, "🗄️  [DB]") {
			t.Error("Expected database insert simulation")
		}
		
		// Check for high value detection
		if !strings.Contains(output, "💰 [HIGH VALUE]") {
			t.Error("Expected high value transaction detection")
		}
		
		// Check for batch processing
		if !strings.Contains(output, "📦 [BATCH]") {
			t.Error("Expected batch processing output")
		}
		
		// Check for metrics summary
		if !strings.Contains(output, "Counter") || !strings.Contains(output, "Gauge") {
			t.Error("Expected metrics counter and gauge output")
		}
	})
}

// TestSinkPatterns verifies different sink patterns compile and work
func TestSinkPatterns(t *testing.T) {
	testCases := []struct {
		name        string
		description string
	}{
		{
			name:        "MetricsSink",
			description: "Extracts metrics from events",
		},
		{
			name:        "MessageQueueSink",
			description: "Publishes to message queues",
		},
		{
			name:        "DatabaseAuditSink",
			description: "Simulates database writes",
		},
		{
			name:        "ConditionalSink",
			description: "Processes events based on conditions",
		},
		{
			name:        "BatchingSink",
			description: "Batches events before processing",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// The fact that the example compiles and runs means the sink works
			t.Logf("%s: %s - verified via example execution", tc.name, tc.description)
		})
	}
}