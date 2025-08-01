package zlog

import (
	"testing"
)

//nolint:revive // t is unused but required for test function signature
func TestEmit(t *testing.T) {
	// Test basic emit functionality
	Emit(INFO, "test message")
	Emit(INFO, "test with fields", String("key", "value"))
}

//nolint:revive // t is unused but required for test function signature
func TestLogLevelFunctions(t *testing.T) {
	// Test convenience functions
	Debug("debug message")
	Info("info message")
	Warn("warn message")
	Error("error message")
	// Skip Fatal as it calls os.Exit
}
