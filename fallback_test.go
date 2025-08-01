package zlog

import (
	"context"
	"errors"
	"testing"
)

func TestSinkWithFallback(t *testing.T) {
	tests := []struct {
		name           string
		primaryFails   bool
		fallbackFails  bool
		expectError    bool
		expectPrimary  bool
		expectFallback bool
	}{
		{
			name:           "primary succeeds, fallback not called",
			primaryFails:   false,
			fallbackFails:  false,
			expectError:    false,
			expectPrimary:  true,
			expectFallback: false,
		},
		{
			name:           "primary fails, fallback succeeds",
			primaryFails:   true,
			fallbackFails:  false,
			expectError:    false,
			expectPrimary:  true,
			expectFallback: true,
		},
		{
			name:           "both primary and fallback fail",
			primaryFails:   true,
			fallbackFails:  true,
			expectError:    true,
			expectPrimary:  true,
			expectFallback: true,
		},
		{
			name:           "primary succeeds, fallback would fail",
			primaryFails:   false,
			fallbackFails:  true, // Shouldn't matter
			expectError:    false,
			expectPrimary:  true,
			expectFallback: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var primaryCalled, fallbackCalled bool

			primaryHandler := func(_ context.Context, _ Event) error {
				primaryCalled = true
				if tt.primaryFails {
					return errors.New("primary failure")
				}
				return nil
			}

			fallbackHandler := func(_ context.Context, _ Event) error {
				fallbackCalled = true
				if tt.fallbackFails {
					return errors.New("fallback failure")
				}
				return nil
			}

			primarySink := NewSink("primary", primaryHandler)
			fallbackSink := NewSink("fallback", fallbackHandler)

			sink := primarySink.WithFallback(fallbackSink)

			event := NewEvent("TEST", "test message", nil)

			_, err := sink.Process(context.Background(), event)

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			// Check which handlers were called
			if tt.expectPrimary && !primaryCalled {
				t.Error("expected primary to be called")
			}
			if !tt.expectPrimary && primaryCalled {
				t.Error("expected primary not to be called")
			}

			if tt.expectFallback && !fallbackCalled {
				t.Error("expected fallback to be called")
			}
			if !tt.expectFallback && fallbackCalled {
				t.Error("expected fallback not to be called")
			}
		})
	}
}

func TestSinkWithFallbackContextCancellation(t *testing.T) {
	var primaryCalled, fallbackCalled bool

	primaryHandler := func(ctx context.Context, _ Event) error {
		primaryCalled = true
		// Check context and fail to trigger fallback
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("primary failed")
	}

	fallbackHandler := func(ctx context.Context, _ Event) error {
		fallbackCalled = true
		// Check context in fallback too
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return nil
	}

	primarySink := NewSink("primary", primaryHandler)
	fallbackSink := NewSink("fallback", fallbackHandler)

	sink := primarySink.WithFallback(fallbackSink)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event := NewEvent("TEST", "test message", nil)

	_, err := sink.Process(ctx, event)

	// Should get context cancellation error
	if err == nil {
		t.Error("expected context cancellation error but got none")
	}

	// Both should be called and both should respect context
	if !primaryCalled {
		t.Error("expected primary to be called")
	}
	if !fallbackCalled {
		t.Error("expected fallback to be called")
	}
}

func TestSinkWithFallbackEventData(t *testing.T) {
	var primaryEvent, fallbackEvent Event

	primaryHandler := func(_ context.Context, event Event) error {
		primaryEvent = event
		return errors.New("primary failed")
	}

	fallbackHandler := func(_ context.Context, event Event) error {
		fallbackEvent = event
		return nil
	}

	primarySink := NewSink("primary", primaryHandler)
	fallbackSink := NewSink("fallback", fallbackHandler)

	sink := primarySink.WithFallback(fallbackSink)

	originalEvent := NewEvent("TEST", "test message", []Field{
		String("key", "value"),
		Int("number", 42),
	})

	_, err := sink.Process(context.Background(), originalEvent)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	// Both handlers should receive identical event data
	if primaryEvent.Signal != originalEvent.Signal {
		t.Errorf("primary signal mismatch, got %s, want %s", primaryEvent.Signal, originalEvent.Signal)
	}
	if fallbackEvent.Signal != originalEvent.Signal {
		t.Errorf("fallback signal mismatch, got %s, want %s", fallbackEvent.Signal, originalEvent.Signal)
	}

	if primaryEvent.Message != originalEvent.Message {
		t.Errorf("primary message mismatch, got %s, want %s", primaryEvent.Message, originalEvent.Message)
	}
	if fallbackEvent.Message != originalEvent.Message {
		t.Errorf("fallback message mismatch, got %s, want %s", fallbackEvent.Message, originalEvent.Message)
	}

	if len(primaryEvent.Fields) != len(originalEvent.Fields) {
		t.Errorf("primary field count mismatch, got %d, want %d", len(primaryEvent.Fields), len(originalEvent.Fields))
	}
	if len(fallbackEvent.Fields) != len(originalEvent.Fields) {
		t.Errorf("fallback field count mismatch, got %d, want %d", len(fallbackEvent.Fields), len(originalEvent.Fields))
	}
}

func TestSinkWithFallbackChaining(t *testing.T) {
	// Test chaining fallbacks with other capabilities

	var calls []string

	primaryHandler := func(_ context.Context, _ Event) error {
		calls = append(calls, "primary")
		return errors.New("primary failed")
	}

	fallbackHandler := func(_ context.Context, _ Event) error {
		calls = append(calls, "fallback")
		return nil
	}

	primarySink := NewSink("primary", primaryHandler)
	fallbackSink := NewSink("fallback", fallbackHandler)

	// Chain with retry - retry the whole fallback operation
	sink := primarySink.
		WithFallback(fallbackSink).
		WithRetry(2)

	event := NewEvent("TEST", "test message", nil)

	_, err := sink.Process(context.Background(), event)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	// Should have tried primary, then fallback
	expectedCalls := []string{"primary", "fallback"}
	if len(calls) != len(expectedCalls) {
		t.Errorf("expected %d calls but got %d: %v", len(expectedCalls), len(calls), calls)
	}

	for i, expected := range expectedCalls {
		if i < len(calls) && calls[i] != expected {
			t.Errorf("call %d: expected %s but got %s", i, expected, calls[i])
		}
	}
}

func TestSinkWithFallbackMultipleLevels(t *testing.T) {
	// Test multiple levels of fallback (fallback of fallback)

	var calls []string

	primaryHandler := func(_ context.Context, _ Event) error {
		calls = append(calls, "primary")
		return errors.New("primary failed")
	}

	secondaryHandler := func(_ context.Context, _ Event) error {
		calls = append(calls, "secondary")
		return errors.New("secondary failed")
	}

	tertiaryHandler := func(_ context.Context, _ Event) error {
		calls = append(calls, "tertiary")
		return nil
	}

	primarySink := NewSink("primary", primaryHandler)
	secondarySink := NewSink("secondary", secondaryHandler)
	tertiarySink := NewSink("tertiary", tertiaryHandler)

	// Chain multiple fallbacks
	sink := primarySink.
		WithFallback(secondarySink.WithFallback(tertiarySink))

	event := NewEvent("TEST", "test message", nil)

	_, err := sink.Process(context.Background(), event)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	// Should have tried all three in order
	expectedCalls := []string{"primary", "secondary", "tertiary"}
	if len(calls) != len(expectedCalls) {
		t.Errorf("expected %d calls but got %d: %v", len(expectedCalls), len(calls), calls)
	}

	for i, expected := range expectedCalls {
		if i < len(calls) && calls[i] != expected {
			t.Errorf("call %d: expected %s but got %s", i, expected, calls[i])
		}
	}
}
