package zlog

import (
	"context"
	"errors"
	"testing"
)

func TestNewSink(t *testing.T) {
	// Test creating a simple sink
	called := false
	sink := NewSink("test-sink", func(_ context.Context, _ Log) error {
		called = true
		return nil
	})

	// Process an event
	ctx := context.Background()
	event := NewEvent(INFO, "test message", nil)
	_, err := sink.Process(ctx, event)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !called {
		t.Error("sink handler was not called")
	}
}

func TestNewSinkError(t *testing.T) {
	// Test sink that returns an error
	expectedErr := errors.New("test error")
	sink := NewSink("error-sink", func(_ context.Context, _ Log) error {
		return expectedErr
	})

	// Process an event
	ctx := context.Background()
	event := NewEvent(ERROR, "error message", nil)
	_, err := sink.Process(ctx, event)

	if err == nil {
		t.Error("expected error but got nil")
	}
}
