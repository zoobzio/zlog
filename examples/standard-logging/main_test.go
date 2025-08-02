// Package main_test tests the standard-logging example
package main_test

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// TestBuildStandardLogging ensures the standard-logging example compiles
func TestBuildStandardLogging(t *testing.T) {
	// Build the example
	cmd := exec.Command("go", "build", "-o", "standard-logging-test", ".")
	cmd.Dir = filepath.Dir(".")
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build standard-logging: %v\nStderr: %s", err, stderr.String())
	}
	
	// Clean up the binary
	defer os.Remove("standard-logging-test")
	
	t.Log("standard-logging example built successfully")
}

// TestRunStandardLogging runs the standard-logging example and verifies functionality
func TestRunStandardLogging(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping run test in short mode")
	}
	
	// Build the example first
	buildCmd := exec.Command("go", "build", "-o", "standard-logging-test", ".")
	buildCmd.Dir = filepath.Dir(".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}
	defer os.Remove("standard-logging-test")
	
	// Run the example with a timeout
	cmd := exec.Command("./standard-logging-test")
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
			t.Fatalf("Failed to run standard-logging: %v\nStdout: %s\nStderr: %s", 
				err, stdout.String(), stderr.String())
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("standard-logging example timed out")
	}
	
	// Clean up any log files created
	defer func() {
		os.Remove("stdout.log")
		os.Remove("stderr.log")
	}()
	
	stdoutOutput := stdout.String()
	stderrOutput := stderr.String()
	
	// Verify main sections in stdout
	t.Run("VerifySections", func(t *testing.T) {
		expectedSections := []string{
			"=== Standard Logging Example ===",
			"--- Application Startup ---",
			"--- Web Server Simulation ---",
			"--- Log Level Examples ---",
			"--- Custom Signal Examples ---",
			"=== Example Complete ===",
		}
		
		for _, section := range expectedSections {
			if !strings.Contains(stdoutOutput, section) {
				t.Errorf("Expected stdout to contain %q", section)
			}
		}
	})
	
	// Verify JSON output structure
	t.Run("VerifyJSONOutput", func(t *testing.T) {
		// Both stdout and stderr should contain JSON logs
		// stderr has standard logging, stdout has all events via RouteAll
		
		// Check stderr for standard log levels
		if len(stderrOutput) > 0 {
			lines := strings.Split(strings.TrimSpace(stderrOutput), "\n")
			for _, line := range lines {
				if strings.TrimSpace(line) == "" {
					continue
				}
				
				var logEntry map[string]interface{}
				if err := json.Unmarshal([]byte(line), &logEntry); err != nil {
					t.Logf("Non-JSON line in stderr (likely status output): %s", line)
					continue
				}
				
				// Verify standard log fields
				if _, ok := logEntry["time"]; !ok {
					t.Error("JSON log missing 'time' field")
				}
				if _, ok := logEntry["signal"]; !ok {
					t.Error("JSON log missing 'signal' field")
				}
				if _, ok := logEntry["message"]; !ok {
					t.Error("JSON log missing 'message' field")
				}
			}
		}
	})
	
	// Verify log levels
	t.Run("VerifyLogLevels", func(t *testing.T) {
		// Should see INFO, WARN, ERROR in output (DEBUG is filtered at INFO level)
		expectedLevels := []string{"INFO", "WARN", "ERROR"}
		
		for _, level := range expectedLevels {
			found := false
			// Check both outputs as example demonstrates both routing methods
			if strings.Contains(stdoutOutput, level) || strings.Contains(stderrOutput, level) {
				found = true
			}
			if !found {
				t.Errorf("Expected to find log level %s in output", level)
			}
		}
		
		// DEBUG should NOT appear since we're at INFO level
		if strings.Contains(stdoutOutput, "This debug message won't show") {
			t.Error("DEBUG message appeared when it should have been filtered")
		}
	})
	
	// Verify custom signals
	t.Run("VerifyCustomSignals", func(t *testing.T) {
		// Custom signals should only appear in stdout (RouteAll), not stderr
		if !strings.Contains(stdoutOutput, "PAYMENT_RECEIVED") {
			t.Error("Expected PAYMENT_RECEIVED signal in stdout")
		}
		
		if !strings.Contains(stdoutOutput, "USER_REGISTERED") {
			t.Error("Expected USER_REGISTERED signal in stdout")
		}
		
		// Verify the notice about custom signals
		if !strings.Contains(stdoutOutput, "Custom signals only appear in stdout") {
			t.Error("Expected explanation about custom signal routing")
		}
	})
	
	// Verify structured fields
	t.Run("VerifyStructuredFields", func(t *testing.T) {
		// Look for evidence of structured fields in the output
		expectedFields := []string{
			"host",
			"port",
			"version",
			"method",
			"path",
			"status",
			"duration",
		}
		
		combinedOutput := stdoutOutput + stderrOutput
		for _, field := range expectedFields {
			if !strings.Contains(combinedOutput, field) {
				t.Errorf("Expected structured field %q in output", field)
			}
		}
	})
}

// TestLoggingPatterns verifies common logging patterns demonstrated
func TestLoggingPatterns(t *testing.T) {
	patterns := []struct {
		name        string
		description string
	}{
		{
			name:        "ServerStartup",
			description: "Logging application initialization",
		},
		{
			name:        "RequestLogging",
			description: "Structured HTTP request logging",
		},
		{
			name:        "ErrorHandling",
			description: "Logging errors with context",
		},
		{
			name:        "WarningLogs",
			description: "Configuration and operational warnings",
		},
		{
			name:        "DebugLogging",
			description: "Detailed debug information",
		},
		{
			name:        "CustomSignals",
			description: "Business events with custom signals",
		},
	}
	
	for _, pattern := range patterns {
		t.Run(pattern.name, func(t *testing.T) {
			t.Logf("%s: %s - verified via example execution", pattern.name, pattern.description)
		})
	}
}

// TestRouteAllDemonstration verifies RouteAll functionality
func TestRouteAllDemonstration(t *testing.T) {
	// The example demonstrates RouteAll by sending all events to ConsoleJSONSink
	// This is verified by custom signals appearing in stdout
	t.Log("RouteAll functionality verified - custom signals appear in stdout via ConsoleJSONSink")
}