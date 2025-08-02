package zlog

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zoobzio/pipz"
)

// TestOrder is a sample type for testing typed loggers.
type TestOrder struct {
	ID     string
	Amount float64
	Status string
}

// Clone implements pipz.Cloner for TestOrder.
func (o TestOrder) Clone() TestOrder {
	return TestOrder{ID: o.ID, Amount: o.Amount, Status: o.Status}
}

func TestNewLogger(t *testing.T) {
	logger := NewLogger[TestOrder]()

	if logger == nil {
		t.Fatal("NewLogger returned nil")
	}

	if logger.pipeline == nil {
		t.Fatal("pipeline is nil")
	}

	if logger.router == nil {
		t.Fatal("router is nil")
	}
}

func TestLoggerHook(t *testing.T) {
	logger := NewLogger[TestOrder]()

	var processedCount int64
	hook := pipz.Effect[Event[TestOrder]]("test-hook", func(_ context.Context, _ Event[TestOrder]) error {
		atomic.AddInt64(&processedCount, 1)
		return nil
	})

	// Test single hook
	logger.Hook("HIGH_VALUE", hook)

	// Verify hook was added
	logger.mu.RLock()
	hooks := logger.hooks["HIGH_VALUE"]
	logger.mu.RUnlock()

	if len(hooks) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(hooks))
	}
}

func TestLoggerHookMultiple(t *testing.T) {
	logger := NewLogger[TestOrder]()

	var hook1Count, hook2Count, hook3Count int64

	hook1 := pipz.Effect[Event[TestOrder]]("hook1", func(_ context.Context, _ Event[TestOrder]) error {
		atomic.AddInt64(&hook1Count, 1)
		return nil
	})

	hook2 := pipz.Effect[Event[TestOrder]]("hook2", func(_ context.Context, _ Event[TestOrder]) error {
		atomic.AddInt64(&hook2Count, 1)
		return nil
	})

	hook3 := pipz.Effect[Event[TestOrder]]("hook3", func(_ context.Context, _ Event[TestOrder]) error {
		atomic.AddInt64(&hook3Count, 1)
		return nil
	})

	// Add hooks one by one to test scaffold creation
	logger.Hook("MULTI_TEST", hook1)
	logger.Hook("MULTI_TEST", hook2)
	logger.Hook("MULTI_TEST", hook3)

	// Verify all hooks were added and scaffold was created
	logger.mu.RLock()
	hooks := logger.hooks["MULTI_TEST"]
	scaffold := logger.scaffolds["MULTI_TEST"]
	logger.mu.RUnlock()

	if len(hooks) != 3 {
		t.Errorf("Expected 3 hooks, got %d", len(hooks))
	}

	if scaffold == nil {
		t.Error("Expected scaffold to be created for multiple hooks")
	}
}

func TestLoggerHookAll(t *testing.T) {
	logger := NewLogger[TestOrder]()

	var globalProcessedCount int64
	globalHook := pipz.Effect[Event[TestOrder]]("global-hook", func(_ context.Context, _ Event[TestOrder]) error {
		atomic.AddInt64(&globalProcessedCount, 1)
		return nil
	})

	logger.HookAll(globalHook)

	// The hook should be added to the pipeline
	// We can't directly verify this without accessing internals,
	// but we can test the fluent API returns the logger
	result := logger.HookAll(globalHook)
	if result != logger {
		t.Error("HookAll should return the logger for chaining")
	}
}

func TestLoggerWithFilter(t *testing.T) {
	logger := NewLogger[TestOrder]()

	// Add filter for orders over $100
	filteredLogger := logger.WithFilter(func(event Event[TestOrder]) bool {
		return event.Data.Amount > 100.0
	})

	if filteredLogger != logger {
		t.Error("WithFilter should return the same logger for chaining")
	}
}

func TestLoggerWithTimeout(t *testing.T) {
	logger := NewLogger[TestOrder]()

	timedLogger := logger.WithTimeout(5 * time.Second)

	if timedLogger != logger {
		t.Error("WithTimeout should return the same logger for chaining")
	}
}

func TestLoggerWithRetry(t *testing.T) {
	logger := NewLogger[TestOrder]()

	retriedLogger := logger.WithRetry(3)

	if retriedLogger != logger {
		t.Error("WithRetry should return the same logger for chaining")
	}
}

func TestLoggerWithAsync(t *testing.T) {
	logger := NewLogger[TestOrder]()

	asyncLogger := logger.WithAsync()

	if asyncLogger != logger {
		t.Error("WithAsync should return the same logger for chaining")
	}
}

func TestLoggerFluentChaining(t *testing.T) {
	var processedCount int64
	hook := pipz.Effect[Event[TestOrder]]("test-hook", func(_ context.Context, _ Event[TestOrder]) error {
		atomic.AddInt64(&processedCount, 1)
		return nil
	})

	logger := NewLogger[TestOrder]().
		WithFilter(func(event Event[TestOrder]) bool { return event.Data.Amount > 50.0 }).
		WithTimeout(5*time.Second).
		WithRetry(2).
		Hook("HIGH_VALUE", hook)

	if logger == nil {
		t.Fatal("Fluent chaining should return valid logger")
	}

	// Verify hook was added
	logger.mu.RLock()
	hooks := logger.hooks["HIGH_VALUE"]
	logger.mu.RUnlock()

	if len(hooks) != 1 {
		t.Errorf("Expected 1 hook after fluent chaining, got %d", len(hooks))
	}
}

func TestLoggerEmit(t *testing.T) {
	// Capture events emitted to global system
	var capturedEvents []Log
	captureHook := NewSink("capture", func(_ context.Context, event Log) error {
		capturedEvents = append(capturedEvents, event)
		return nil
	})

	// Set up global hook to capture all events
	HookAll(captureHook)
	defer func() {
		// Clean up by resetting dispatch
		defaultLogger = NewLogger[Fields]()
	}()

	logger := NewLogger[TestOrder]().Watch()

	testOrder := TestOrder{
		ID:     "ORD-123",
		Amount: 99.99,
		Status: "completed",
	}

	logger.Emit("ORDER_PROCESSED", "Order ORD-123 processed for $99.99", testOrder)

	// Give a moment for async processing
	time.Sleep(50 * time.Millisecond)

	// Verify event was captured
	if len(capturedEvents) == 0 {
		t.Fatal("No events were captured")
	}

	event := capturedEvents[0]
	if event.Signal != "ORDER_PROCESSED" {
		t.Errorf("Expected signal ORDER_PROCESSED, got %s", event.Signal)
	}

	expectedMsg := "Order ORD-123 processed for $99.99"
	if event.Message != expectedMsg {
		t.Errorf("Expected message %q, got %q", expectedMsg, event.Message)
	}

	// Verify the original typed event is included in Data field
	if len(event.Data) == 0 {
		t.Fatal("Expected at least one field")
	}

	dataField := event.Data[0]
	if dataField.Key != "event" {
		t.Errorf("Expected field key 'event', got %q", dataField.Key)
	}

	if dataField.Type != DataType {
		t.Errorf("Expected field type DataType, got %s", dataField.Type)
	}
}

func TestLoggerConcurrentEmit(t *testing.T) {
	var capturedCount int64
	captureHook := NewSink("capture", func(_ context.Context, _ Log) error {
		atomic.AddInt64(&capturedCount, 1)
		return nil
	})

	HookAll(captureHook)
	defer func() {
		defaultLogger = NewLogger[Fields]()
	}()

	logger := NewLogger[TestOrder]().Watch()

	const numGoroutines = 10
	const eventsPerGoroutine = 5

	var wg sync.WaitGroup
	// Emit events concurrently
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				order := TestOrder{
					ID:     fmt.Sprintf("ORD-%d-%d", goroutineID, j),
					Amount: float64(goroutineID*j + 1), // Avoid zero amounts
					Status: "test",
				}
				logger.Emit("CONCURRENT_TEST", fmt.Sprintf("Order %s", order.ID), order)
			}
		}(i)
	}

	// Wait for all goroutines to finish emitting
	wg.Wait()
	// Give additional time for async processing
	time.Sleep(200 * time.Millisecond)

	expectedCount := int64(numGoroutines * eventsPerGoroutine)
	actualCount := atomic.LoadInt64(&capturedCount)
	if actualCount != expectedCount {
		t.Errorf("Expected %d events, got %d", expectedCount, actualCount)
	}
}

func TestLoggerTypedHookProcessing(t *testing.T) {
	var processedOrders []TestOrder
	var mu sync.Mutex

	typedHook := pipz.Effect[Event[TestOrder]]("typed-processor", func(_ context.Context, event Event[TestOrder]) error {
		mu.Lock()
		processedOrders = append(processedOrders, event.Data)
		mu.Unlock()
		return nil
	})

	logger := NewLogger[TestOrder]()

	// Add typed hook - this should process the Order before conversion to Log
	logger.Hook("TYPED_TEST", typedHook)

	testOrder := TestOrder{
		ID:     "ORD-TYPED",
		Amount: 150.00,
		Status: "pending",
	}

	logger.Emit("TYPED_TEST", "Order ORD-TYPED", testOrder)

	// Wait for processing
	time.Sleep(50 * time.Millisecond)

	// Verify the typed hook processed the original Order
	mu.Lock()
	count := len(processedOrders)
	mu.Unlock()

	if count != 1 {
		t.Errorf("Expected 1 processed order, got %d", count)
	}

	if count > 0 {
		processed := processedOrders[0]
		if processed.ID != testOrder.ID {
			t.Errorf("Expected order ID %s, got %s", testOrder.ID, processed.ID)
		}
		if processed.Amount != testOrder.Amount {
			t.Errorf("Expected order amount %.2f, got %.2f", testOrder.Amount, processed.Amount)
		}
	}
}
