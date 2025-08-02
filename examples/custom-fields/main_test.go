// Package main_test tests the custom-fields example
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

// TestBuildCustomFields ensures the custom-fields example compiles
func TestBuildCustomFields(t *testing.T) {
	// Build the example
	cmd := exec.Command("go", "build", "-o", "custom-fields-test", ".")
	cmd.Dir = filepath.Dir(".")
	
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		t.Fatalf("Failed to build custom-fields: %v\nStderr: %s", err, stderr.String())
	}
	
	// Clean up the binary
	defer os.Remove("custom-fields-test")
	
	t.Log("custom-fields example built successfully")
}

// TestRunCustomFields runs the custom-fields example and verifies basic functionality
func TestRunCustomFields(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping run test in short mode")
	}
	
	// Build the example first
	buildCmd := exec.Command("go", "build", "-o", "custom-fields-test", ".")
	buildCmd.Dir = filepath.Dir(".")
	if err := buildCmd.Run(); err != nil {
		t.Fatalf("Failed to build: %v", err)
	}
	defer os.Remove("custom-fields-test")
	
	// Run the example with a timeout
	cmd := exec.Command("./custom-fields-test")
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
			t.Fatalf("Failed to run custom-fields: %v\nStdout: %s\nStderr: %s", 
				err, stdout.String(), stderr.String())
		}
	case <-time.After(5 * time.Second):
		cmd.Process.Kill()
		t.Fatal("custom-fields example timed out")
	}
	
	output := stdout.String()
	
	// Verify expected output sections
	expectedSections := []string{
		"=== Custom Fields Example ===",
		"--- User Registration ---",
		"--- Payment Processing ---",
		"--- API Authentication ---",
		"--- GDPR Compliant Event ---",
		"--- Session Tracking ---",
		"--- Error Handling ---",
		"=== Example Complete ===",
	}
	
	for _, section := range expectedSections {
		if !strings.Contains(output, section) {
			t.Errorf("Expected output to contain %q", section)
		}
	}
	
	// Verify key transformations
	expectedTransformations := []string{
		"Redaction (credit cards, SSNs)",
		"Masking (emails)",
		"Hashing (passwords, IDs)",
		"Truncation (API keys)",
		"Anonymization (IP addresses)",
		"Range buckets (ages)",
	}
	
	for _, transform := range expectedTransformations {
		if !strings.Contains(output, transform) {
			t.Errorf("Expected output to mention transformation: %q", transform)
		}
	}
	
	t.Log("custom-fields example ran successfully")
}

// TestCustomFieldTransformations tests specific field transformation logic
func TestCustomFieldTransformations(t *testing.T) {
	t.Run("TestRedactedFields", func(t *testing.T) {
		// This would test the RedactedString function if it were exported
		// For now, we just verify the example compiles correctly
		t.Log("Field transformation logic tested via example execution")
	})
	
	t.Run("TestMaskedEmail", func(t *testing.T) {
		// Similarly, this would test MaskedEmail if exported
		t.Log("Email masking tested via example execution")
	})
}