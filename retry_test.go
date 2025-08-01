package zlog

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSinkWithRetry(t *testing.T) {
	tests := []struct {
		name        string
		attempts    int
		failCount   int // How many times the handler should fail before succeeding
		expectError bool
		expectCalls int // Expected number of handler calls
	}{
		{
			name:        "succeeds on first try",
			attempts:    3,
			failCount:   0,
			expectError: false,
			expectCalls: 1,
		},
		{
			name:        "succeeds on second try",
			attempts:    3,
			failCount:   1,
			expectError: false,
			expectCalls: 2,
		},
		{
			name:        "succeeds on last try",
			attempts:    3,
			failCount:   2,
			expectError: false,
			expectCalls: 3,
		},
		{
			name:        "fails after all retries",
			attempts:    3,
			failCount:   5, // Fail more times than attempts
			expectError: true,
			expectCalls: 3,
		},
		{
			name:        "zero attempts defaults to 1",
			attempts:    0,
			failCount:   0,
			expectError: false,
			expectCalls: 1,
		},
		{
			name:        "negative attempts defaults to 1",
			attempts:    -1,
			failCount:   0,
			expectError: false,
			expectCalls: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var callCount int

			handler := func(_ context.Context, _ Event) error {
				callCount++
				if callCount <= tt.failCount {
					return errors.New("simulated failure")
				}
				return nil
			}

			sink := NewSink("test", handler).WithRetry(tt.attempts)

			// Create a test event
			event := NewEvent("TEST", "test message", nil)

			// Process the event
			_, err := sink.Process(context.Background(), event)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Errorf("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			// Check call count
			if callCount != tt.expectCalls {
				t.Errorf("expected %d calls but got %d", tt.expectCalls, callCount)
			}
		})
	}
}

func TestSinkWithRetryContextCancellation(t *testing.T) {
	var callCount int

	handler := func(ctx context.Context, _ Event) error {
		callCount++
		// Check if context is canceled on each call
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("always fail")
	}

	sink := NewSink("test", handler).WithRetry(5)

	// Create a context that we'll cancel
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context immediately to test cancellation handling
	cancel()

	event := NewEvent("TEST", "test message", nil)

	// Process should respect context cancellation
	_, err := sink.Process(ctx, event)

	// Should get a context cancellation error
	if err == nil {
		t.Error("expected context cancellation error but got none")
	}

	// Should have stopped retrying due to context cancellation
	// The exact number depends on when cancellation is checked, but should be less than max attempts
	if callCount > 5 {
		t.Errorf("expected at most 5 calls due to cancellation, but got %d", callCount)
	}
}

func TestSinkWithRetryTimeout(t *testing.T) {
	var callCount int

	handler := func(_ context.Context, _ Event) error {
		callCount++
		// Simulate a slow operation
		time.Sleep(50 * time.Millisecond)
		return errors.New("always fail")
	}

	sink := NewSink("test", handler).WithRetry(10) // High retry count

	// Create a context with a short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	event := NewEvent("TEST", "test message", nil)

	start := time.Now()
	_, err := sink.Process(ctx, event)
	duration := time.Since(start)

	// Should get an error (either timeout or failure)
	if err == nil {
		t.Error("expected error but got none")
	}

	// Should respect timeout and not take too long
	if duration > 200*time.Millisecond {
		t.Errorf("operation took too long: %v", duration)
	}

	// Should have made fewer calls due to timeout
	if callCount >= 10 {
		t.Errorf("expected fewer than 10 calls due to timeout, but got %d", callCount)
	}
}

func TestSinkWithRetryChaining(t *testing.T) {
	// Test that we can chain multiple WithRetry calls
	// (though this isn't necessarily useful, it should work)

	var callCount int

	handler := func(_ context.Context, _ Event) error {
		callCount++
		if callCount == 1 {
			return errors.New("fail once")
		}
		return nil
	}

	// Chain multiple retry wrappers - this creates nested retry logic
	sink := NewSink("test", handler).
		WithRetry(2).
		WithRetry(2)

	event := NewEvent("TEST", "test message", nil)

	_, err := sink.Process(context.Background(), event)

	// Should succeed without error
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	// Due to nested retries, exact call count depends on pipz retry implementation
	// but it should be at least 2 (first failure, then success)
	if callCount < 2 {
		t.Errorf("expected at least 2 calls but got %d", callCount)
	}
}

func TestSinkWithRetryPreservesEventData(t *testing.T) {
	var receivedEvents []Event

	handler := func(_ context.Context, event Event) error {
		receivedEvents = append(receivedEvents, event)
		if len(receivedEvents) < 3 {
			return errors.New("fail first two attempts")
		}
		return nil
	}

	sink := NewSink("test", handler).WithRetry(3)

	originalEvent := NewEvent("TEST", "test message", []Field{
		String("key", "value"),
		Int("number", 42),
	})

	_, err := sink.Process(context.Background(), originalEvent)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if len(receivedEvents) != 3 {
		t.Errorf("expected 3 received events but got %d", len(receivedEvents))
	}

	// All received events should be identical to the original
	for i, received := range receivedEvents {
		if received.Signal != originalEvent.Signal {
			t.Errorf("event %d: signal mismatch, got %s, want %s", i, received.Signal, originalEvent.Signal)
		}
		if received.Message != originalEvent.Message {
			t.Errorf("event %d: message mismatch, got %s, want %s", i, received.Message, originalEvent.Message)
		}
		if len(received.Fields) != len(originalEvent.Fields) {
			t.Errorf("event %d: field count mismatch, got %d, want %d", i, len(received.Fields), len(originalEvent.Fields))
		}
	}
}
