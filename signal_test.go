package zlog

import (
	"testing"
)

// TestSignalCreation tests creating and using signals.
func TestSignalCreation(t *testing.T) {
	// Test creating custom signals
	customSignal := Signal("CUSTOM_EVENT")
	if customSignal != "CUSTOM_EVENT" {
		t.Errorf("Expected signal 'CUSTOM_EVENT', got '%s'", customSignal)
	}

	// Test that signals are just strings
	s := string(customSignal)
	if s != "CUSTOM_EVENT" {
		t.Errorf("Signal string conversion failed")
	}
}

// TestStandardSignals tests the predefined signal constants.
func TestStandardSignals(t *testing.T) {
	tests := []struct {
		name     string
		signal   Signal
		expected string
	}{
		{"DEBUG", DEBUG, "DEBUG"},
		{"INFO", INFO, "INFO"},
		{"WARN", WARN, "WARN"},
		{"ERROR", ERROR, "ERROR"},
		{"FATAL", FATAL, "FATAL"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.signal) != tt.expected {
				t.Errorf("Signal %s = %s, want %s", tt.name, tt.signal, tt.expected)
			}
		})
	}
}
