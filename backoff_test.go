package zlog

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSinkWithBackoff(t *testing.T) {
	tests := []struct {
		name        string
		maxAttempts int
		baseDelay   time.Duration
		failCount   int // How many times to fail before succeeding
		expectError bool
		expectCalls int
		minDuration time.Duration // Minimum expected duration due to backoff delays
		maxDuration time.Duration // Maximum acceptable duration
	}{
		{
			name:        "succeeds on first try",
			maxAttempts: 3,
			baseDelay:   50 * time.Millisecond,
			failCount:   0,
			expectError: false,
			expectCalls: 1,
			minDuration: 0,
			maxDuration: 50 * time.Millisecond,
		},
		{
			name:        "succeeds on second try after backoff",
			maxAttempts: 3,
			baseDelay:   50 * time.Millisecond,
			failCount:   1,
			expectError: false,
			expectCalls: 2,
			minDuration: 50 * time.Millisecond,  // At least one backoff delay
			maxDuration: 150 * time.Millisecond, // Some tolerance for timing
		},
		{
			name:        "succeeds on third try with exponential delays",
			maxAttempts: 3,
			baseDelay:   25 * time.Millisecond,
			failCount:   2,
			expectError: false,
			expectCalls: 3,
			minDuration: 75 * time.Millisecond,  // 25ms + 50ms delays
			maxDuration: 150 * time.Millisecond, // Some tolerance
		},
		{
			name:        "fails after all retries with backoff",
			maxAttempts: 3,
			baseDelay:   20 * time.Millisecond,
			failCount:   5, // Always fail
			expectError: true,
			expectCalls: 3,
			minDuration: 60 * time.Millisecond,  // 20ms + 40ms delays
			maxDuration: 120 * time.Millisecond, // Some tolerance
		},
		{
			name:        "zero attempts defaults to 1",
			maxAttempts: 0,
			baseDelay:   10 * time.Millisecond,
			failCount:   0,
			expectError: false,
			expectCalls: 1,
			minDuration: 0,
			maxDuration: 50 * time.Millisecond,
		},
		{
			name:        "negative attempts defaults to 1",
			maxAttempts: -1,
			baseDelay:   10 * time.Millisecond,
			failCount:   0,
			expectError: false,
			expectCalls: 1,
			minDuration: 0,
			maxDuration: 50 * time.Millisecond,
		},
		{
			name:        "zero delay gets default",
			maxAttempts: 2,
			baseDelay:   0,
			failCount:   1,
			expectError: false,
			expectCalls: 2,
			minDuration: 100 * time.Millisecond, // Default base delay
			maxDuration: 250 * time.Millisecond,
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

			sink := NewSink("test", handler).WithBackoff(tt.maxAttempts, tt.baseDelay)

			event := NewEvent("TEST", "test message", nil)

			start := time.Now()
			_, err := sink.Process(context.Background(), event)
			duration := time.Since(start)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			// Check call count
			if callCount != tt.expectCalls {
				t.Errorf("expected %d calls but got %d", tt.expectCalls, callCount)
			}

			// Check timing - backoff should introduce delays
			if duration < tt.minDuration {
				t.Errorf("operation completed too quickly: %v (expected at least %v)", duration, tt.minDuration)
			}
			if duration > tt.maxDuration {
				t.Errorf("operation took too long: %v (expected at most %v)", duration, tt.maxDuration)
			}
		})
	}
}

func TestSinkWithBackoffContextCancellation(t *testing.T) {
	var callCount int

	handler := func(ctx context.Context, _ Event) error {
		callCount++
		// Check if context is canceled
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("always fail")
	}

	sink := NewSink("test", handler).WithBackoff(5, 100*time.Millisecond)

	// Create a context that we'll cancel after a short delay
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel the context after allowing one attempt and partial backoff
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	event := NewEvent("TEST", "test message", nil)

	start := time.Now()
	_, err := sink.Process(ctx, event)
	duration := time.Since(start)

	// Should get cancellation error
	if err == nil {
		t.Error("expected context cancellation error but got none")
	}

	// Should have stopped early due to cancellation
	if duration > 200*time.Millisecond {
		t.Errorf("expected quick cancellation, but took %v", duration)
	}

	// Should have made at least one call
	if callCount < 1 {
		t.Errorf("expected at least 1 call but got %d", callCount)
	}
}

func TestSinkWithBackoffTimeout(t *testing.T) {
	var callCount int

	handler := func(ctx context.Context, _ Event) error {
		callCount++
		// Simulate work that respects context timeout
		select {
		case <-time.After(50 * time.Millisecond):
			return errors.New("handler failure")
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	sink := NewSink("test", handler).WithBackoff(5, 50*time.Millisecond)

	// Create a context with timeout that should prevent all retries
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	event := NewEvent("TEST", "test message", nil)

	start := time.Now()
	_, err := sink.Process(ctx, event)
	duration := time.Since(start)

	// Should get some kind of error (timeout or failure)
	if err == nil {
		t.Error("expected error but got none")
	}

	// Should respect timeout and not take too long
	if duration > 150*time.Millisecond {
		t.Errorf("operation took too long: %v", duration)
	}

	// Should have made fewer calls due to timeout
	if callCount > 3 {
		t.Errorf("expected fewer calls due to timeout, but got %d", callCount)
	}
}

func TestSinkWithBackoffExponentialDelays(t *testing.T) {
	// Test that backoff delays are actually exponential

	var callTimes []time.Time

	handler := func(_ context.Context, _ Event) error {
		callTimes = append(callTimes, time.Now())
		return errors.New("always fail")
	}

	sink := NewSink("test", handler).WithBackoff(4, 50*time.Millisecond)

	event := NewEvent("TEST", "test message", nil)

	start := time.Now()
	_, err := sink.Process(context.Background(), event)

	// Should have failed after all attempts
	if err == nil {
		t.Error("expected error but got none")
	}

	// Should have made 4 calls
	if len(callTimes) != 4 {
		t.Errorf("expected 4 calls but got %d", len(callTimes))
		return
	}

	// Check that delays are approximately exponential
	// First call is immediate
	firstCall := callTimes[0].Sub(start)
	if firstCall > 10*time.Millisecond {
		t.Errorf("first call should be immediate, but took %v", firstCall)
	}

	// Check delays between subsequent calls
	expectedDelays := []time.Duration{
		50 * time.Millisecond,  // After first failure
		100 * time.Millisecond, // After second failure (2x)
		200 * time.Millisecond, // After third failure (4x)
	}

	tolerance := 30 * time.Millisecond

	for i := 1; i < len(callTimes); i++ {
		actualDelay := callTimes[i].Sub(callTimes[i-1])
		expectedDelay := expectedDelays[i-1]

		if actualDelay < expectedDelay-tolerance {
			t.Errorf("delay %d too short: %v (expected ~%v)", i, actualDelay, expectedDelay)
		}
		if actualDelay > expectedDelay+tolerance {
			t.Errorf("delay %d too long: %v (expected ~%v)", i, actualDelay, expectedDelay)
		}
	}
}

func TestSinkWithBackoffChaining(t *testing.T) {
	// Test chaining backoff with other capabilities

	var callCount int

	handler := func(_ context.Context, _ Event) error {
		callCount++
		if callCount == 1 {
			return errors.New("fail once")
		}
		return nil
	}

	// Chain backoff with timeout
	sink := NewSink("test", handler).
		WithBackoff(3, 25*time.Millisecond).
		WithTimeout(500 * time.Millisecond)

	event := NewEvent("TEST", "test message", nil)

	start := time.Now()
	_, err := sink.Process(context.Background(), event)
	duration := time.Since(start)

	// Should succeed on second attempt
	if err != nil {
		t.Errorf("expected success but got: %v", err)
	}

	// Should have made 2 calls
	if callCount != 2 {
		t.Errorf("expected 2 calls but got %d", callCount)
	}

	// Should include at least one backoff delay
	if duration < 25*time.Millisecond {
		t.Errorf("expected backoff delay, but completed in %v", duration)
	}
}

func TestSinkWithBackoffPreservesEventData(t *testing.T) {
	var receivedEvents []Event

	handler := func(_ context.Context, event Event) error {
		receivedEvents = append(receivedEvents, event)
		if len(receivedEvents) < 3 {
			return errors.New("fail first two attempts")
		}
		return nil
	}

	sink := NewSink("test", handler).WithBackoff(3, 10*time.Millisecond)

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
