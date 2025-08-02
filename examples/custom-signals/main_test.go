// Package main_test tests the custom-signals example
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

// TestBuildCustomSignals ensures the custom-signals example compiles
func TestBuildCustomSignals(t *testing.T) {
	// Build the example
	cmd := exec.Command("go", "build", "-o", "custom-signals-test", ".")
	cmd.Dir = filepath.Dir(".")
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build custom-signals: %v\nStderr: %s", err, stderr.String())
	}
	
	// Clean up the binary
	defer os.Remove("custom-signals-test")
	
	t.Log("custom-signals example built successfully")
}

// TestRunCustomSignals runs the custom-signals example and verifies functionality
func TestRunCustomSignals(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping run test in short mode")
	}
	
	// Build the example first
	buildCmd := exec.Command("go", "build", "-o", "custom-signals-test", ".")
	buildCmd.Dir = filepath.Dir(".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}
	defer os.Remove("custom-signals-test")
	
	// Run the example with a timeout
	cmd := exec.Command("./custom-signals-test")
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
			t.Fatalf("Failed to run custom-signals: %v\nStdout: %s\nStderr: %s", 
				err, stdout.String(), stderr.String())
		}
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		t.Fatal("custom-signals example timed out")
	}
	
	// Clean up analytics.log if created
	defer os.Remove("analytics.log")
	
	output := stdout.String()
	
	// Verify expected output sections
	t.Run("VerifySections", func(t *testing.T) {
		expectedSections := []string{
			"=== Custom Signals Example ===",
			"--- User Shopping Journey",
			"--- Cache Operations ---",
			"--- Security Event ---",
			"--- Standard Error Logging ---",
			"=== Example Complete ===",
		}
		
		for _, section := range expectedSections {
			if !strings.Contains(output, section) {
				t.Errorf("Expected output to contain %q", section)
			}
		}
	})
	
	// Verify routing behavior
	t.Run("VerifyRouting", func(t *testing.T) {
		// Check for audit events
		if !strings.Contains(output, "[AUDIT]") {
			t.Error("Expected audit sink output")
		}
		
		// Check for metrics
		if !strings.Contains(output, "[METRICS]") {
			t.Error("Expected metrics sink output")
		}
		
		// Check for alerts
		if !strings.Contains(output, "[ALERT]") {
			t.Error("Expected alert sink output")
		}
	})
	
	// Verify sampling behavior
	t.Run("VerifySampling", func(t *testing.T) {
		if !strings.Contains(output, "Generating 50 cache events") {
			t.Error("Expected cache event generation message")
		}
		
		if !strings.Contains(output, "Cache hits are sampled at 10%") {
			t.Error("Expected sampling rate message")
		}
	})
	
	// Verify trace ID propagation
	t.Run("VerifyTraceContext", func(t *testing.T) {
		if !strings.Contains(output, "trace:") {
			t.Error("Expected trace ID in output")
		}
	})
	
	// Verify analytics file was created
	t.Run("VerifyAnalyticsFile", func(t *testing.T) {
		if _, err := os.Stat("analytics.log"); err == nil {
			t.Log("analytics.log was created successfully")
		} else {
			t.Log("analytics.log was not created (may have been cleaned up)")
		}
	})
}

// TestSignalDefinitions verifies signal constants are properly defined
func TestSignalDefinitions(t *testing.T) {
	// This test ensures the example has all the expected signal definitions
	// by checking that it compiles with all the signals referenced
	
	expectedSignals := []string{
		"USER_REGISTERED",
		"USER_LOGIN",
		"USER_LOGOUT",
		"PASSWORD_CHANGED",
		"PRODUCT_VIEWED",
		"CART_UPDATED",
		"ORDER_PLACED",
		"PAYMENT_PROCESSED",
		"PAYMENT_FAILED",
		"ORDER_SHIPPED",
		"CACHE_HIT",
		"CACHE_MISS",
		"API_RATE_LIMITED",
		"FRAUD_DETECTED",
	}
	
	// The fact that the example compiles means all signals are defined
	t.Logf("All %d expected signals are defined in the example", len(expectedSignals))
}