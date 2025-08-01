package zlog

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSinkWithTimeout(t *testing.T) {
	tests := []struct {
		name          string
		timeout       time.Duration
		handlerDelay  time.Duration
		expectTimeout bool
		expectError   bool
		maxDuration   time.Duration // Maximum acceptable test duration
	}{
		{
			name:          "completes within timeout",
			timeout:       100 * time.Millisecond,
			handlerDelay:  10 * time.Millisecond,
			expectTimeout: false,
			expectError:   false,
			maxDuration:   150 * time.Millisecond,
		},
		{
			name:          "times out with slow handler",
			timeout:       50 * time.Millisecond,
			handlerDelay:  200 * time.Millisecond,
			expectTimeout: true,
			expectError:   true,
			maxDuration:   100 * time.Millisecond,
		},
		{
			name:          "zero timeout gets default",
			timeout:       0,
			handlerDelay:  10 * time.Millisecond,
			expectTimeout: false,
			expectError:   false,
			maxDuration:   1 * time.Second, // Should use default timeout
		},
		{
			name:          "negative timeout gets default",
			timeout:       -5 * time.Second,
			handlerDelay:  10 * time.Millisecond,
			expectTimeout: false,
			expectError:   false,
			maxDuration:   1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCalled bool

			handler := func(ctx context.Context, _ Event) error {
				handlerCalled = true

				// Check for context cancellation during delay
				select {
				case <-time.After(tt.handlerDelay):
					return nil // Completed normally
				case <-ctx.Done():
					return ctx.Err() // Context was canceled
				}
			}

			sink := NewSink("test", handler).WithTimeout(tt.timeout)

			event := NewEvent("TEST", "test message", nil)

			start := time.Now()
			_, err := sink.Process(context.Background(), event)
			duration := time.Since(start)

			// Check that handler was called
			if !handlerCalled {
				t.Error("expected handler to be called")
			}

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			// Check timeout behavior
			if tt.expectTimeout {
				if duration >= tt.handlerDelay {
					t.Errorf("expected timeout before %v, but took %v", tt.handlerDelay, duration)
				}
			}

			// Check overall duration is reasonable
			if duration > tt.maxDuration {
				t.Errorf("operation took too long: %v (max: %v)", duration, tt.maxDuration)
			}
		})
	}
}

func TestSinkWithTimeoutContextCancellation(t *testing.T) {
	var handlerCalled bool

	handler := func(ctx context.Context, _ Event) error {
		handlerCalled = true

		// Simulate some work, but respect context cancellation
		select {
		case <-time.After(100 * time.Millisecond):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	sink := NewSink("test", handler).WithTimeout(1 * time.Second) // Long timeout

	// Create a context that we'll cancel quickly
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	event := NewEvent("TEST", "test message", nil)

	start := time.Now()
	_, err := sink.Process(ctx, event)
	duration := time.Since(start)

	// Should have been called
	if !handlerCalled {
		t.Error("expected handler to be called")
	}

	// Should get cancellation error
	if err == nil {
		t.Error("expected context cancellation error but got none")
	}

	// Should complete quickly due to cancellation, not timeout
	if duration > 50*time.Millisecond {
		t.Errorf("expected quick cancellation, but took %v", duration)
	}
}

func TestSinkWithTimeoutHandlerError(t *testing.T) {
	// Test that handler errors are propagated correctly

	handler := func(_ context.Context, _ Event) error {
		// Quick operation that returns an error
		return errors.New("handler error")
	}

	sink := NewSink("test", handler).WithTimeout(1 * time.Second)

	event := NewEvent("TEST", "test message", nil)

	_, err := sink.Process(context.Background(), event)

	// Should get the handler error, not a timeout
	if err == nil {
		t.Error("expected handler error but got none")
	}
}

func TestSinkWithTimeoutChaining(t *testing.T) {
	// Test chaining multiple timeout wrappers

	var callCount int

	handler := func(_ context.Context, _ Event) error {
		callCount++
		time.Sleep(10 * time.Millisecond) // Quick operation
		return nil
	}

	// Chain multiple timeouts - inner timeout should be more restrictive
	sink := NewSink("test", handler).
		WithTimeout(100 * time.Millisecond). // Outer timeout
		WithTimeout(50 * time.Millisecond)   // Inner timeout (more restrictive)

	event := NewEvent("TEST", "test message", nil)

	start := time.Now()
	_, err := sink.Process(context.Background(), event)
	duration := time.Since(start)

	// Should succeed since 10ms is within both timeouts
	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	if callCount != 1 {
		t.Errorf("expected 1 call but got %d", callCount)
	}

	// Should complete quickly
	if duration > 30*time.Millisecond {
		t.Errorf("operation took too long: %v", duration)
	}
}

func TestSinkWithTimeoutLongOperation(t *testing.T) {
	// Test with an operation that's definitely too slow

	handler := func(ctx context.Context, _ Event) error {
		// Very slow operation that should definitely timeout
		select {
		case <-time.After(1 * time.Second):
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	sink := NewSink("test", handler).WithTimeout(50 * time.Millisecond)

	event := NewEvent("TEST", "test message", nil)

	start := time.Now()
	_, err := sink.Process(context.Background(), event)
	duration := time.Since(start)

	// Should get timeout error
	if err == nil {
		t.Error("expected timeout error but got none")
	}

	// Should timeout quickly, not wait for the full 1 second
	if duration > 100*time.Millisecond {
		t.Errorf("timeout took too long: %v", duration)
	}

	// Should timeout in approximately the specified duration
	if duration < 40*time.Millisecond {
		t.Errorf("timeout happened too quickly: %v", duration)
	}
}

func TestSinkWithTimeoutPreservesEventData(t *testing.T) {
	var receivedEvent Event

	handler := func(_ context.Context, event Event) error {
		receivedEvent = event
		return nil
	}

	sink := NewSink("test", handler).WithTimeout(100 * time.Millisecond)

	originalEvent := NewEvent("TEST", "test message", []Field{
		String("key", "value"),
		Int("number", 42),
	})

	_, err := sink.Process(context.Background(), originalEvent)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	// Event should be preserved exactly
	if receivedEvent.Signal != originalEvent.Signal {
		t.Errorf("signal mismatch, got %s, want %s", receivedEvent.Signal, originalEvent.Signal)
	}
	if receivedEvent.Message != originalEvent.Message {
		t.Errorf("message mismatch, got %s, want %s", receivedEvent.Message, originalEvent.Message)
	}
	if len(receivedEvent.Fields) != len(originalEvent.Fields) {
		t.Errorf("field count mismatch, got %d, want %d", len(receivedEvent.Fields), len(originalEvent.Fields))
	}
}
