package zlog

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSinkWithAsync(t *testing.T) {
	t.Run("processes events asynchronously", func(t *testing.T) {
		var wg sync.WaitGroup
		processed := make(chan Log, 1)

		handler := func(_ context.Context, event Log) error {
			defer wg.Done()
			processed <- event
			return nil
		}

		sink := NewSink("test", handler).WithAsync()
		event := NewEvent("TEST", "test message", nil)

		wg.Add(1)
		// Process should return immediately
		start := time.Now()
		_, err := sink.Process(context.Background(), event)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		// Should return almost immediately
		if elapsed > 10*time.Millisecond {
			t.Errorf("async processing took too long: %v", elapsed)
		}

		// Wait for async processing to complete
		wg.Wait()

		// Verify event was processed
		select {
		case processedEvent := <-processed:
			if processedEvent.Signal != event.Signal {
				t.Errorf("signal mismatch, got %s, want %s", processedEvent.Signal, event.Signal)
			}
		default:
			t.Error("event was not processed")
		}
	})

	t.Run("doesn't block on slow handler", func(t *testing.T) {
		var wg sync.WaitGroup
		started := make(chan bool, 1)

		handler := func(_ context.Context, _ Log) error {
			defer wg.Done()
			started <- true
			time.Sleep(100 * time.Millisecond) // Slow operation
			return nil
		}

		sink := NewSink("test", handler).WithAsync()
		event := NewEvent("TEST", "test message", nil)

		wg.Add(1)
		start := time.Now()
		_, err := sink.Process(context.Background(), event)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		// Should return immediately, not wait for slow handler
		if elapsed > 20*time.Millisecond {
			t.Errorf("async processing blocked caller for: %v", elapsed)
		}

		// Verify handler started
		select {
		case <-started:
			// Good, handler is running
		case <-time.After(50 * time.Millisecond):
			t.Error("handler didn't start within timeout")
		}

		// Clean up
		wg.Wait()
	})

	t.Run("errors don't propagate back", func(t *testing.T) {
		handlerCalled := make(chan bool, 1)

		handler := func(_ context.Context, _ Log) error {
			handlerCalled <- true
			return errors.New("handler error")
		}

		sink := NewSink("test", handler).WithAsync()
		event := NewEvent("TEST", "test message", nil)

		_, err := sink.Process(context.Background(), event)

		// Should not get error from async handler
		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		// Verify handler was called
		select {
		case <-handlerCalled:
			// Good, handler was called even though it errored
		case <-time.After(50 * time.Millisecond):
			t.Error("handler wasn't called")
		}
	})

	t.Run("multiple events spawn multiple goroutines", func(t *testing.T) {
		var activeCount int32
		var maxActive int32
		var wg sync.WaitGroup

		handler := func(_ context.Context, _ Log) error {
			defer wg.Done()

			// Increment active count
			current := atomic.AddInt32(&activeCount, 1)

			// Track max concurrent
			for {
				maxVal := atomic.LoadInt32(&maxActive)
				if current <= maxVal || atomic.CompareAndSwapInt32(&maxActive, maxVal, current) {
					break
				}
			}

			// Hold to allow concurrency
			time.Sleep(50 * time.Millisecond)

			// Decrement active count
			atomic.AddInt32(&activeCount, -1)
			return nil
		}

		sink := NewSink("test", handler).WithAsync()

		// Send multiple events rapidly
		eventCount := 5
		wg.Add(eventCount)

		for i := 0; i < eventCount; i++ {
			event := NewEvent("TEST", "test message", []Field{Int("index", i)})
			_, err := sink.Process(context.Background(), event)
			if err != nil {
				t.Errorf("event %d: unexpected error: %v", i, err)
			}
		}

		// Wait for all to complete
		wg.Wait()

		// Should have had multiple goroutines active concurrently
		if maxActive < 2 {
			t.Errorf("expected concurrent execution, but max active was %d", maxActive)
		}
	})

	t.Run("context cancellation doesn't affect async processing", func(t *testing.T) {
		processed := make(chan bool, 1)

		handler := func(ctx context.Context, _ Log) error {
			// Sleep to ensure parent context would be canceled
			time.Sleep(50 * time.Millisecond)

			// Check if this context is canceled (it shouldn't be)
			if ctx.Err() != nil {
				return ctx.Err()
			}

			processed <- true
			return nil
		}

		sink := NewSink("test", handler).WithAsync()
		event := NewEvent("TEST", "test message", nil)

		// Create a context that we'll cancel immediately
		ctx, cancel := context.WithCancel(context.Background())

		_, err := sink.Process(ctx, event)
		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		// Cancel the context immediately after processing
		cancel()

		// Async processing should still complete
		select {
		case <-processed:
			// Good, handler completed despite context cancellation
		case <-time.After(100 * time.Millisecond):
			t.Error("handler didn't complete after context cancellation")
		}
	})

	t.Run("works with other adapters", func(t *testing.T) {
		var callCount int32
		processed := make(chan bool, 1)

		handler := func(_ context.Context, _ Log) error {
			count := atomic.AddInt32(&callCount, 1)
			if count < 2 {
				return errors.New("simulated failure")
			}
			processed <- true
			return nil
		}

		// Retry THEN async - retry happens in background
		sink := NewSink("test", handler).
			WithRetry(3).
			WithAsync()

		event := NewEvent("TEST", "test message", nil)

		// Should return immediately
		start := time.Now()
		_, err := sink.Process(context.Background(), event)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		if elapsed > 20*time.Millisecond {
			t.Errorf("async processing blocked for: %v", elapsed)
		}

		// Wait for async retry to succeed
		select {
		case <-processed:
			// Good, retry worked in background
		case <-time.After(500 * time.Millisecond):
			t.Error("async retry didn't complete")
		}

		// Should have retried
		finalCount := atomic.LoadInt32(&callCount)
		if finalCount != 2 {
			t.Errorf("expected 2 calls (retry) but got %d", finalCount)
		}
	})

	t.Run("order of async matters", func(t *testing.T) {
		var handlerCompleted int32

		slowHandler := func(ctx context.Context, _ Log) error {
			select {
			case <-time.After(100 * time.Millisecond):
				atomic.StoreInt32(&handlerCompleted, 1)
				return nil
			case <-ctx.Done():
				// Context was canceled (timeout)
				return ctx.Err()
			}
		}

		// Timeout THEN async - timeout applies to handler in background
		handlerCompleted = 0
		timeoutThenAsync := NewSink("test", slowHandler).
			WithTimeout(50 * time.Millisecond).
			WithAsync()

		event := NewEvent("TEST", "test message", nil)

		// Process with timeout then async
		_, err := timeoutThenAsync.Process(context.Background(), event)
		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		// Wait to see if handler completes
		time.Sleep(150 * time.Millisecond)

		// Handler should have timed out in background
		if atomic.LoadInt32(&handlerCompleted) == 1 {
			t.Error("handler shouldn't complete when timeout is shorter than processing time")
		}

		// Async THEN timeout - timeout applies to the async Effect itself
		asyncThenTimeout := NewSink("test", slowHandler).
			WithAsync().
			WithTimeout(50 * time.Millisecond)

		// The timeout wraps the async Effect, which returns immediately
		// So timeout won't actually affect the background goroutine
		start := time.Now()
		_, err = asyncThenTimeout.Process(context.Background(), event)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		// Should still return immediately since async Effect completes instantly
		if elapsed > 20*time.Millisecond {
			t.Errorf("async with timeout took too long: %v", elapsed)
		}
	})
}
