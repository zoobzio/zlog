package zlog

import (
	"context"
	"errors"
	"testing"
)

func TestDispatch(t *testing.T) {
	// Basic test that defaultLogger exists
	if defaultLogger == nil {
		t.Fatal("defaultLogger is nil")
	}
}

// TestDispatchProcessError tests error handling in dispatch.process.
//
//nolint:revive // t is unused but test confirms no panic occurs
func TestDispatchProcessError(t *testing.T) {
	// Create a sink that returns an error
	errorSink := NewSink("error-test", func(_ context.Context, _ Log) error {
		return errors.New("test error")
	})

	// Route to error sink
	signal := Signal("TEST_ERROR_HANDLING")
	RouteSignal(signal, errorSink)

	// Process should not panic even with error
	event := NewEvent(signal, "test", nil)
	defaultLogger.Process(event) // Should handle error gracefully

	// Test passes if no panic occurred
}
