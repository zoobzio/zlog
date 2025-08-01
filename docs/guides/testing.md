# Testing Guide

This guide covers testing strategies for applications using zlog, testing custom sinks, and debugging logging behavior.

## Testing Logging Code

### Basic Testing Strategy

The key to testing logging code is to verify that the right events are emitted with the correct data, not testing the output format:

```go
func TestUserRegistration(t *testing.T) {
    // Setup a test sink to capture events
    var capturedEvents []zlog.Event
    testSink := zlog.NewSink("test", func(ctx context.Context, event zlog.Event) error {
        capturedEvents = append(capturedEvents, event)
        return nil
    })
    
    // Route the signal we want to test
    zlog.RouteSignal(USER_REGISTERED, testSink)
    
    // Run the code under test
    user := &User{ID: "123", Email: "test@example.com"}
    err := registerUser(user)
    
    // Verify
    assert.NoError(t, err)
    assert.Len(t, capturedEvents, 1)
    
    event := capturedEvents[0]
    assert.Equal(t, USER_REGISTERED, event.Signal)
    assert.Equal(t, "User registered successfully", event.Message)
    
    // Check fields
    assert.Contains(t, event.Fields, zlog.String("user_id", "123"))
    assert.Contains(t, event.Fields, zlog.String("email", "test@example.com"))
}
```

### Test Helper Functions

Create helpers to make testing easier:

```go
// TestLogCapture captures events for testing
type TestLogCapture struct {
    Events []zlog.Event
    mutex  sync.Mutex
}

func NewTestLogCapture() *TestLogCapture {
    return &TestLogCapture{
        Events: make([]zlog.Event, 0),
    }
}

func (tc *TestLogCapture) Sink() zlog.Sink {
    return zlog.NewSink("test-capture", func(ctx context.Context, event zlog.Event) error {
        tc.mutex.Lock()
        defer tc.mutex.Unlock()
        tc.Events = append(tc.Events, event)
        return nil
    })
}

func (tc *TestLogCapture) GetEvents() []zlog.Event {
    tc.mutex.Lock()
    defer tc.mutex.Unlock()
    events := make([]zlog.Event, len(tc.Events))
    copy(events, tc.Events)
    return events
}

func (tc *TestLogCapture) GetEventsWithSignal(signal zlog.Signal) []zlog.Event {
    tc.mutex.Lock()
    defer tc.mutex.Unlock()
    
    var filtered []zlog.Event
    for _, event := range tc.Events {
        if event.Signal == signal {
            filtered = append(filtered, event)
        }
    }
    return filtered
}

func (tc *TestLogCapture) Reset() {
    tc.mutex.Lock()
    defer tc.mutex.Unlock()
    tc.Events = tc.Events[:0]
}
```

### Using the Test Helper

```go
func TestPaymentProcessing(t *testing.T) {
    // Setup
    capture := NewTestLogCapture()
    zlog.RouteSignal(PAYMENT_RECEIVED, capture.Sink())
    zlog.RouteSignal(PAYMENT_FAILED, capture.Sink())
    
    t.Run("successful payment", func(t *testing.T) {
        capture.Reset()
        
        result := processPayment("user_123", 100.00)
        
        assert.True(t, result.Success)
        
        events := capture.GetEventsWithSignal(PAYMENT_RECEIVED)
        assert.Len(t, events, 1)
        
        event := events[0]
        assertFieldValue(t, event, "user_id", "user_123")
        assertFieldValue(t, event, "amount", 100.00)
    })
    
    t.Run("failed payment", func(t *testing.T) {
        capture.Reset()
        
        // Trigger failure condition
        result := processPayment("user_invalid", -10.00)
        
        assert.False(t, result.Success)
        
        events := capture.GetEventsWithSignal(PAYMENT_FAILED)
        assert.Len(t, events, 1)
        
        event := events[0]
        assertFieldValue(t, event, "user_id", "user_invalid")
        assertFieldValue(t, event, "reason", "invalid_amount")
    })
}

// Helper to check field values
func assertFieldValue(t *testing.T, event zlog.Event, key string, expected any) {
    t.Helper()
    
    for _, field := range event.Fields {
        if field.Key == key {
            assert.Equal(t, expected, field.Value, "field %s", key)
            return
        }
    }
    t.Errorf("field %s not found in event", key)
}
```

## Testing Custom Sinks

### Unit Testing Sinks

Test sink behavior independently:

```go
func TestMetricsSink(t *testing.T) {
    // Setup test metrics registry
    registry := prometheus.NewRegistry()
    counter := prometheus.NewCounterVec(
        prometheus.CounterOpts{Name: "test_events_total"},
        []string{"signal"},
    )
    registry.MustRegister(counter)
    
    // Create sink under test
    sink := createMetricsSink(counter)
    
    // Test event processing
    event := zlog.NewEvent(USER_REGISTERED, "Test event", []zlog.Field{
        zlog.String("user_id", "123"),
    })
    
    err := sink.Process(context.Background(), event)
    assert.NoError(t, err)
    
    // Verify metrics were updated
    families, err := registry.Gather()
    assert.NoError(t, err)
    
    found := false
    for _, family := range families {
        if *family.Name == "test_events_total" {
            metrics := family.GetMetric()
            assert.Len(t, metrics, 1)
            assert.Equal(t, float64(1), *metrics[0].Counter.Value)
            found = true
            break
        }
    }
    assert.True(t, found, "metric not found")
}
```

### Testing Error Handling

Verify sinks handle errors correctly:

```go
func TestSinkErrorHandling(t *testing.T) {
    // Sink that fails on specific conditions
    failingSink := zlog.NewSink("failing", func(ctx context.Context, event zlog.Event) error {
        if event.Signal == "FAIL" {
            return errors.New("intentional failure")
        }
        return nil
    })
    
    // Test successful processing
    err := failingSink.Process(context.Background(), zlog.NewEvent("SUCCESS", "ok", nil))
    assert.NoError(t, err)
    
    // Test error condition
    err = failingSink.Process(context.Background(), zlog.NewEvent("FAIL", "fail", nil))
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "intentional failure")
}
```

### Testing Async Sinks

For sinks that process events asynchronously:

```go
func TestAsyncSink(t *testing.T) {
    processed := make(chan zlog.Event, 10)
    
    asyncSink := zlog.NewSink("async", func(ctx context.Context, event zlog.Event) error {
        // Simulate async processing
        go func() {
            time.Sleep(10 * time.Millisecond)  // Simulate work
            processed <- event
        }()
        return nil
    })
    
    // Send event
    event := zlog.NewEvent("TEST", "async test", nil)
    err := asyncSink.Process(context.Background(), event)
    assert.NoError(t, err)
    
    // Wait for async processing
    select {
    case processedEvent := <-processed:
        assert.Equal(t, event.Signal, processedEvent.Signal)
        assert.Equal(t, event.Message, processedEvent.Message)
    case <-time.After(100 * time.Millisecond):
        t.Fatal("async processing timed out")
    }
}
```

## Integration Testing

### Testing Multiple Sinks

Verify events are routed to all expected sinks:

```go
func TestMultipleSinkRouting(t *testing.T) {
    // Setup multiple captures
    auditCapture := NewTestLogCapture()
    metricsCapture := NewTestLogCapture()
    alertCapture := NewTestLogCapture()
    
    // Route to multiple sinks
    zlog.RouteSignal(PAYMENT_FAILED, auditCapture.Sink())
    zlog.RouteSignal(PAYMENT_FAILED, metricsCapture.Sink())
    zlog.RouteSignal(PAYMENT_FAILED, alertCapture.Sink())
    
    // Emit event
    zlog.Emit(PAYMENT_FAILED, "Payment failed", 
        zlog.String("user_id", "123"),
        zlog.String("reason", "insufficient_funds"))
    
    // Verify all sinks received the event
    assert.Len(t, auditCapture.GetEvents(), 1)
    assert.Len(t, metricsCapture.GetEvents(), 1)
    assert.Len(t, alertCapture.GetEvents(), 1)
    
    // Verify event content is identical
    auditEvent := auditCapture.GetEvents()[0]
    metricsEvent := metricsCapture.GetEvents()[0]
    alertEvent := alertCapture.GetEvents()[0]
    
    assert.Equal(t, auditEvent.Signal, metricsEvent.Signal)
    assert.Equal(t, auditEvent.Signal, alertEvent.Signal)
    assert.Equal(t, auditEvent.Message, metricsEvent.Message)
    assert.Equal(t, auditEvent.Message, alertEvent.Message)
}
```

### Testing Module Setup

Test that modules configure routing correctly:

```go
func TestStandardLoggingModule(t *testing.T) {
    // Clear any existing routing
    resetZlogState()
    
    capture := NewTestLogCapture()
    
    // Override the standard sink for testing
    originalSink := getStandardSink()  // Save original
    setStandardSink(capture.Sink())    // Use test sink
    defer setStandardSink(originalSink) // Restore
    
    // Enable standard logging
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Test that appropriate levels are routed
    zlog.Debug("debug message")  // Should not be captured (below INFO)
    zlog.Info("info message")    // Should be captured
    zlog.Error("error message")  // Should be captured
    
    events := capture.GetEvents()
    assert.Len(t, events, 2)  // INFO and ERROR only
    
    assert.Equal(t, zlog.INFO, events[0].Signal)
    assert.Equal(t, zlog.ERROR, events[1].Signal)
}
```

## Benchmarking

### Performance Regression Testing

Include benchmarks in your test suite:

```go
func BenchmarkHighVolumeLogging(b *testing.B) {
    // Setup
    capture := NewTestLogCapture()
    zlog.RouteSignal(zlog.INFO, capture.Sink())
    
    userID := "user_123"
    requestID := "req_456"
    
    b.ResetTimer()
    b.ReportAllocs()
    
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            zlog.Info("API request",
                zlog.String("user_id", userID),
                zlog.String("request_id", requestID),
                zlog.Int("status", 200),
                zlog.Duration("latency", 45*time.Millisecond))
        }
    })
    
    b.StopTimer()
    
    // Verify no events were lost
    events := capture.GetEvents()
    if len(events) != b.N {
        b.Errorf("Expected %d events, got %d", b.N, len(events))
    }
}
```

### Sink Performance Benchmarks

```go
func BenchmarkSinkPerformance(b *testing.B) {
    sinks := map[string]zlog.Sink{
        "noop": zlog.NewSink("noop", func(ctx context.Context, event zlog.Event) error {
            return nil
        }),
        "json": zlog.NewSink("json", func(ctx context.Context, event zlog.Event) error {
            _, err := json.Marshal(event)
            return err
        }),
        "capture": NewTestLogCapture().Sink(),
    }
    
    event := zlog.NewEvent(zlog.INFO, "Benchmark event", []zlog.Field{
        zlog.String("key1", "value1"),
        zlog.String("key2", "value2"),
        zlog.Int("number", 42),
    })
    
    for name, sink := range sinks {
        b.Run(name, func(b *testing.B) {
            b.ReportAllocs()
            for i := 0; i < b.N; i++ {
                sink.Process(context.Background(), event)
            }
        })
    }
}
```

## Test Isolation

### Preventing Test Interference

Ensure tests don't interfere with each other:

```go
func TestWithIsolation(t *testing.T) {
    // Save current state
    originalRoutes := saveZlogRoutes()
    defer restoreZlogRoutes(originalRoutes)
    
    // Clear routing for this test
    clearAllRoutes()
    
    // Your test code here
    capture := NewTestLogCapture()
    zlog.RouteSignal(TEST_SIGNAL, capture.Sink())
    
    // Test will automatically restore state when done
}

// Helper functions for state management
func saveZlogRoutes() map[zlog.Signal][]zlog.Sink {
    // Implementation depends on zlog internals
    // This is pseudocode - actual implementation would
    // need access to internal routing state
    return getCurrentRoutes()
}

func restoreZlogRoutes(routes map[zlog.Signal][]zlog.Sink) {
    clearAllRoutes()
    for signal, sinks := range routes {
        for _, sink := range sinks {
            zlog.RouteSignal(signal, sink)
        }
    }
}
```

### Table-Driven Tests

Test multiple scenarios efficiently:

```go
func TestEventValidation(t *testing.T) {
    tests := []struct {
        name          string
        signal        zlog.Signal
        message       string
        fields        []zlog.Field
        expectEvent   bool
        expectFields  map[string]any
    }{
        {
            name:        "valid user event",
            signal:      USER_REGISTERED,
            message:     "User registered",
            fields:      []zlog.Field{zlog.String("user_id", "123")},
            expectEvent: true,
            expectFields: map[string]any{"user_id": "123"},
        },
        {
            name:        "payment event with amount",
            signal:      PAYMENT_RECEIVED,
            message:     "Payment processed",
            fields:      []zlog.Field{zlog.Float64("amount", 99.99)},
            expectEvent: true,
            expectFields: map[string]any{"amount": 99.99},
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            capture := NewTestLogCapture()
            zlog.RouteSignal(tt.signal, capture.Sink())
            
            zlog.Emit(tt.signal, tt.message, tt.fields...)
            
            events := capture.GetEvents()
            if tt.expectEvent {
                assert.Len(t, events, 1)
                event := events[0]
                assert.Equal(t, tt.signal, event.Signal)
                assert.Equal(t, tt.message, event.Message)
                
                for key, expectedValue := range tt.expectFields {
                    assertFieldValue(t, event, key, expectedValue)
                }
            } else {
                assert.Len(t, events, 0)
            }
        })
    }
}
```

## Debugging Tests

### Debug Output

Add debug sinks during test development:

```go
func TestWithDebug(t *testing.T) {
    if testing.Verbose() {
        debugSink := zlog.NewSink("debug", func(ctx context.Context, event zlog.Event) error {
            t.Logf("Event: %s - %s", event.Signal, event.Message)
            for _, field := range event.Fields {
                t.Logf("  %s: %v", field.Key, field.Value)
            }
            return nil
        })
        
        // Route all signals to debug during testing
        zlog.RouteSignal(zlog.DEBUG, debugSink)
        zlog.RouteSignal(zlog.INFO, debugSink)
        zlog.RouteSignal(zlog.ERROR, debugSink)
    }
    
    // Your test code here
}
```

Run with: `go test -v`

### Event Inspection

Create helpers to inspect events during tests:

```go
func dumpEvent(t *testing.T, event zlog.Event) {
    t.Helper()
    t.Logf("Event Details:")
    t.Logf("  Signal: %s", event.Signal)
    t.Logf("  Message: %s", event.Message)
    t.Logf("  Time: %s", event.Time.Format(time.RFC3339))
    t.Logf("  Caller: %s:%d", event.Caller.File, event.Caller.Line)
    t.Logf("  Fields (%d):", len(event.Fields))
    for i, field := range event.Fields {
        t.Logf("    [%d] %s (%s): %v", i, field.Key, field.Type, field.Value)
    }
}
```

Following these testing patterns will help you build confidence in your logging implementation and catch regressions early.