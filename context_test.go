package zlog

import (
	"context"
	"sync"
	"testing"
)

// Define custom types for context keys to avoid lint warnings.
type contextKey string

func TestContextPropagation(t *testing.T) {
	// Create a test context with a value
	testKey := contextKey("test-key")
	testValue := "test-value"
	ctx := context.WithValue(context.Background(), testKey, testValue)

	var capturedContext context.Context
	var mu sync.Mutex

	// Create a sink that captures the context it receives
	testSink := NewSink("context-test", func(receivedCtx context.Context, _ Event) error {
		mu.Lock()
		capturedContext = receivedCtx
		mu.Unlock()
		return nil
	})

	// Route test signal to our sink
	RouteSignal("TEST_CONTEXT", testSink)

	// Set the context for this goroutine
	SetContext(ctx)
	defer ClearContext()

	// Emit an event - it should use our context
	Emit("TEST_CONTEXT", "test message")

	// Give the async processing time to complete
	// In a real implementation, you might want a more robust synchronization mechanism
	for i := 0; i < 100; i++ {
		mu.Lock()
		captured := capturedContext
		mu.Unlock()

		if captured != nil {
			break
		}
	}

	// Verify the context was propagated
	mu.Lock()
	captured := capturedContext
	mu.Unlock()

	if captured == nil {
		t.Fatal("No context was captured by the sink")
	}

	// Verify the context contains our test value
	if value := captured.Value(testKey); value != testValue {
		t.Errorf("Expected context value %q, got %v", testValue, value)
	}
}

func TestClearContext(t *testing.T) {
	// Set a context
	testKey := contextKey("test-key")
	testValue := "test-value"
	ctx := context.WithValue(context.Background(), testKey, testValue)
	SetContext(ctx)

	// Verify it's set
	retrieved := getContext()
	if retrieved.Value(testKey) != testValue {
		t.Error("Context was not set correctly")
	}

	// Clear it
	ClearContext()

	// Verify it's cleared (should return background context)
	retrieved = getContext()
	if retrieved.Value(testKey) != nil {
		t.Error("Context was not cleared correctly")
	}
}

func TestWithContext(t *testing.T) {
	// Set an initial context
	initialKey := contextKey("initial-key")
	initialValue := "initial-value"
	initialCtx := context.WithValue(context.Background(), initialKey, initialValue)
	SetContext(initialCtx)

	// Use WithContext to temporarily override
	tempKey := contextKey("temp-key")
	tempValue := "temp-value"
	tempCtx := context.WithValue(context.Background(), tempKey, tempValue)

	restore := WithContext(tempCtx)

	// Verify the temporary context is active
	retrieved := getContext()
	if retrieved.Value(tempKey) != tempValue {
		t.Error("Temporary context was not set correctly")
	}
	if retrieved.Value(initialKey) != nil {
		t.Error("Old context should not be accessible during temporary override")
	}

	// Restore the original context
	restore()

	// Verify the original context is restored
	retrieved = getContext()
	if retrieved.Value(initialKey) != initialValue {
		t.Error("Original context was not restored correctly")
	}
	if retrieved.Value(tempKey) != nil {
		t.Error("Temporary context should be cleared after restore")
	}
}

func TestContextIsolationBetweenGoroutines(t *testing.T) {
	var wg sync.WaitGroup
	var mu sync.Mutex
	results := make(map[int]int)

	// Start multiple goroutines, each with different contexts
	for i := 0; i < 3; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			// Each goroutine sets its own context
			ctx := context.WithValue(context.Background(), contextKey("goroutine-id"), id)
			SetContext(ctx)
			defer ClearContext()

			// Get the context and verify it's isolated
			retrieved := getContext()
			goroutineID := retrieved.Value(contextKey("goroutine-id"))

			mu.Lock()
			if goroutineID != nil {
				if gid, ok := goroutineID.(int); ok {
					results[id] = gid
				}
			}
			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify each goroutine had its own context
	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	for i := 0; i < 3; i++ {
		if results[i] != i {
			t.Errorf("Goroutine %d had wrong context value: expected %d, got %v", i, i, results[i])
		}
	}
}

func TestNoContextSetUsesBackground(t *testing.T) {
	// Make sure no context is set
	ClearContext()

	// Getting context should return background
	ctx := getContext()
	if ctx != context.Background() {
		t.Error("Expected context.Background() when no context is set")
	}
}

// Benchmark the overhead of context storage.
func BenchmarkContextOperations(b *testing.B) {
	ctx := context.WithValue(context.Background(), contextKey("test"), "value")

	b.Run("SetContext", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			SetContext(ctx)
		}
	})

	b.Run("GetContext", func(b *testing.B) {
		SetContext(ctx)
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = getContext()
		}
	})

	b.Run("ClearContext", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			SetContext(ctx)
			ClearContext()
		}
	})
}
