package zlog

import (
	"context"
	"errors"
	"testing"
)

func TestSinkWithFilter(t *testing.T) {
	tests := []struct {
		name          string
		filterFunc    func(context.Context, Event) bool
		signal        Signal
		message       string
		fields        []Field
		expectCall    bool
		expectSuccess bool
	}{
		{
			name: "filter allows event through",
			filterFunc: func(_ context.Context, e Event) bool {
				return e.Signal == "TEST"
			},
			signal:        "TEST",
			message:       "test message",
			fields:        nil,
			expectCall:    true,
			expectSuccess: true,
		},
		{
			name: "filter blocks event",
			filterFunc: func(_ context.Context, e Event) bool {
				return e.Signal == "ALLOWED"
			},
			signal:        "BLOCKED",
			message:       "blocked message",
			fields:        nil,
			expectCall:    false,
			expectSuccess: true, // Still succeeds, just skipped
		},
		{
			name: "filter by field value allows",
			filterFunc: func(_ context.Context, e Event) bool {
				for _, field := range e.Fields {
					if field.Key == "level" && field.Value == "high" {
						return true
					}
				}
				return false
			},
			signal:        "PRIORITY",
			message:       "high priority event",
			fields:        []Field{String("level", "high")},
			expectCall:    true,
			expectSuccess: true,
		},
		{
			name: "filter by field value blocks",
			filterFunc: func(_ context.Context, e Event) bool {
				for _, field := range e.Fields {
					if field.Key == "level" && field.Value == "high" {
						return true
					}
				}
				return false
			},
			signal:        "PRIORITY",
			message:       "low priority event",
			fields:        []Field{String("level", "low")},
			expectCall:    false,
			expectSuccess: true,
		},
		{
			name: "filter by numeric field value",
			filterFunc: func(_ context.Context, e Event) bool {
				for _, field := range e.Fields {
					if field.Key == "amount" {
						if amount, ok := field.Value.(float64); ok {
							return amount > 1000.0
						}
					}
				}
				return false
			},
			signal:        "TRANSACTION",
			message:       "big transaction",
			fields:        []Field{Float64("amount", 5000.0)},
			expectCall:    true,
			expectSuccess: true,
		},
		{
			name: "filter by numeric field value blocks small amount",
			filterFunc: func(_ context.Context, e Event) bool {
				for _, field := range e.Fields {
					if field.Key == "amount" {
						if amount, ok := field.Value.(float64); ok {
							return amount > 1000.0
						}
					}
				}
				return false
			},
			signal:        "TRANSACTION",
			message:       "small transaction",
			fields:        []Field{Float64("amount", 50.0)},
			expectCall:    false,
			expectSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCalled bool
			var receivedEvent Event

			handler := func(_ context.Context, event Event) error {
				handlerCalled = true
				receivedEvent = event
				return nil
			}

			sink := NewSink("test", handler).WithFilter(tt.filterFunc)

			event := NewEvent(tt.signal, tt.message, tt.fields)

			_, err := sink.Process(context.Background(), event)

			// Check success expectation
			if tt.expectSuccess && err != nil {
				t.Errorf("expected success but got error: %v", err)
			}
			if !tt.expectSuccess && err == nil {
				t.Error("expected error but got success")
			}

			// Check if handler was called as expected
			if tt.expectCall && !handlerCalled {
				t.Error("expected handler to be called but it wasn't")
			}
			if !tt.expectCall && handlerCalled {
				t.Error("expected handler not to be called but it was")
			}

			// If handler was called, verify event data
			if handlerCalled {
				if receivedEvent.Signal != event.Signal {
					t.Errorf("signal mismatch, got %s, want %s", receivedEvent.Signal, event.Signal)
				}
				if receivedEvent.Message != event.Message {
					t.Errorf("message mismatch, got %s, want %s", receivedEvent.Message, event.Message)
				}
				if len(receivedEvent.Fields) != len(event.Fields) {
					t.Errorf("field count mismatch, got %d, want %d", len(receivedEvent.Fields), len(event.Fields))
				}
			}
		})
	}
}

func TestSinkWithFilterHandlerError(t *testing.T) {
	// Test that handler errors are still propagated when filter allows event

	handler := func(_ context.Context, _ Event) error {
		return errors.New("handler error")
	}

	sink := NewSink("test", handler).WithFilter(func(_ context.Context, _ Event) bool {
		return true // Allow all events
	})

	event := NewEvent("TEST", "test message", nil)

	_, err := sink.Process(context.Background(), event)

	// Should get the handler error
	if err == nil {
		t.Error("expected handler error but got none")
	}
}

func TestSinkWithFilterContextCancellation(t *testing.T) {
	var filterCalled, handlerCalled bool

	filterFunc := func(ctx context.Context, _ Event) bool {
		filterCalled = true
		// Check if context is canceled
		if ctx.Err() != nil {
			return false // Context canceled, reject event
		}
		return true
	}

	handler := func(_ context.Context, _ Event) error {
		handlerCalled = true
		return nil
	}

	sink := NewSink("test", handler).WithFilter(filterFunc)

	// Create a canceled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event := NewEvent("TEST", "test message", nil)

	_, err := sink.Process(ctx, event)

	// Filter should be called
	if !filterCalled {
		t.Error("expected filter to be called")
	}

	// Handler should not be called due to filter rejecting
	if handlerCalled {
		t.Error("expected handler not to be called due to context cancellation in filter")
	}

	// Should still succeed (filtered events succeed)
	if err != nil {
		t.Errorf("expected success (filtered) but got error: %v", err)
	}
}

func TestSinkWithFilterChaining(t *testing.T) {
	// Test chaining multiple filters

	var calls []string

	handler := func(_ context.Context, _ Event) error {
		calls = append(calls, "handler")
		return nil
	}

	// Chain multiple filters
	sink := NewSink("test", handler).
		WithFilter(func(_ context.Context, e Event) bool {
			calls = append(calls, "filter1")
			return e.Signal == "ALLOWED"
		}).
		WithFilter(func(_ context.Context, e Event) bool {
			calls = append(calls, "filter2")
			for _, field := range e.Fields {
				if field.Key == "pass" && field.Value == true {
					return true
				}
			}
			return false
		})

	event := NewEvent("ALLOWED", "test message", []Field{Bool("pass", true)})

	_, err := sink.Process(context.Background(), event)

	if err != nil {
		t.Errorf("expected no error but got: %v", err)
	}

	// Should have called both filters and handler
	expectedCalls := []string{"filter2", "filter1", "handler"}
	if len(calls) != len(expectedCalls) {
		t.Errorf("expected %d calls but got %d: %v", len(expectedCalls), len(calls), calls)
	}

	for i, expected := range expectedCalls {
		if i < len(calls) && calls[i] != expected {
			t.Errorf("call %d: expected %s but got %s", i, expected, calls[i])
		}
	}
}

func TestSinkWithFilterMultipleConditions(t *testing.T) {
	// Test complex filter with multiple conditions

	handler := func(_ context.Context, _ Event) error {
		return nil
	}

	// Complex filter: ERROR signal AND amount > 100
	complexFilter := func(_ context.Context, e Event) bool {
		if e.Signal != "ERROR" {
			return false
		}

		for _, field := range e.Fields {
			if field.Key == "amount" {
				if amount, ok := field.Value.(float64); ok {
					return amount > 100.0
				}
			}
		}
		return false
	}

	_ = NewSink("test", handler).WithFilter(complexFilter)

	tests := []struct {
		name       string
		signal     Signal
		fields     []Field
		expectCall bool
	}{
		{
			name:       "ERROR with high amount - should pass",
			signal:     "ERROR",
			fields:     []Field{Float64("amount", 500.0)},
			expectCall: true,
		},
		{
			name:       "ERROR with low amount - should block",
			signal:     "ERROR",
			fields:     []Field{Float64("amount", 50.0)},
			expectCall: false,
		},
		{
			name:       "INFO with high amount - should block",
			signal:     "INFO",
			fields:     []Field{Float64("amount", 500.0)},
			expectCall: false,
		},
		{
			name:       "ERROR with no amount field - should block",
			signal:     "ERROR",
			fields:     []Field{String("message", "error")},
			expectCall: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var handlerCalled bool

			// Reset handler for each test
			testSink := NewSink("test", func(_ context.Context, _ Event) error {
				handlerCalled = true
				return nil
			}).WithFilter(complexFilter)

			event := NewEvent(tt.signal, "test message", tt.fields)

			_, err := testSink.Process(context.Background(), event)

			if err != nil {
				t.Errorf("expected no error but got: %v", err)
			}

			if tt.expectCall && !handlerCalled {
				t.Error("expected handler to be called but it wasn't")
			}
			if !tt.expectCall && handlerCalled {
				t.Error("expected handler not to be called but it was")
			}
		})
	}
}

func TestSinkWithFilterCombinedWithOtherAdapters(t *testing.T) {
	// Test filter combined with retry and timeout

	var callCount int

	handler := func(_ context.Context, _ Event) error {
		callCount++
		if callCount == 1 {
			return errors.New("first attempt fails")
		}
		return nil
	}

	// Create filter function for reuse
	errorOnlyFilter := func(_ context.Context, e Event) bool {
		return e.Signal == "ERROR"
	}

	// Test with ERROR event (should pass filter and retry)
	errorEvent := NewEvent("ERROR", "error message", nil)
	callCount = 0 // Reset counter

	errorSink := NewSink("test", handler).WithFilter(errorOnlyFilter).WithRetry(2)
	_, err := errorSink.Process(context.Background(), errorEvent)

	if err != nil {
		t.Errorf("expected success after retry but got: %v", err)
	}

	if callCount != 2 {
		t.Errorf("expected 2 calls (retry) but got %d", callCount)
	}

	// Test with INFO event (should be filtered out, no retry)
	infoEvent := NewEvent("INFO", "info message", nil)
	callCount = 0 // Reset counter

	infoSink := NewSink("test", handler).WithFilter(errorOnlyFilter).WithRetry(2)
	_, err = infoSink.Process(context.Background(), infoEvent)

	if err != nil {
		t.Errorf("expected success (filtered) but got error: %v", err)
	}

	if callCount != 0 {
		t.Errorf("expected 0 calls (filtered) but got %d", callCount)
	}
}
