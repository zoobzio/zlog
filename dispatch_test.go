package zlog

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/zoobzio/pipz"
)

// Test sink that tracks calls.
type testSink struct {
	name   string
	events []Event
	mu     sync.Mutex
}

func (t *testSink) Write(event Event) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = append(t.events, event)
	return nil
}

func (t *testSink) Name() string {
	return t.name
}

func (t *testSink) EventCount() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return len(t.events)
}

func (t *testSink) Clear() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.events = nil
}

func TestSignalRouting(t *testing.T) {
	// Save and restore dispatch state
	oldDispatch := dispatch
	defer func() { dispatch = oldDispatch }()

	// Create fresh dispatch
	dispatch = &Dispatch{
		signalRoutes: make(map[Signal][]Sink),
	}
	dispatch.pipeline = pipz.ProcessorFunc[Event](func(ctx context.Context, e Event) (Event, error) {
		key := string(e.Signal)
		if chainable, ok := dispatch.switchRoutes.Load(key); ok {
			return chainable.(pipz.Chainable[Event]).Process(ctx, e)
		}
		return e, nil
	})

	t.Run("Single sink routing", func(t *testing.T) {
		sink := &testSink{name: "test1"}
		RouteSignal(INFO, sink)

		Info("test message")

		// Give concurrent processing time
		time.Sleep(10 * time.Millisecond)

		if sink.EventCount() != 1 {
			t.Errorf("Expected 1 event, got %d", sink.EventCount())
		}
	})

	t.Run("Multiple signals to same sink", func(t *testing.T) {
		sink := &testSink{name: "test2"}
		RouteSignal(INFO, sink)
		RouteSignal(WARN, sink)
		RouteSignal(ERROR, sink)

		Info("info")
		Warn("warn")
		Error("error")

		time.Sleep(10 * time.Millisecond)

		if sink.EventCount() != 3 {
			t.Errorf("Expected 3 events, got %d", sink.EventCount())
		}
	})

	t.Run("Multiple sinks for same signal", func(t *testing.T) {
		sink1 := &testSink{name: "sink1"}
		sink2 := &testSink{name: "sink2"}
		sink3 := &testSink{name: "sink3"}

		RouteSignal(INFO, sink1)
		RouteSignal(INFO, sink2)
		RouteSignal(INFO, sink3)

		Info("broadcast")

		time.Sleep(10 * time.Millisecond)

		if sink1.EventCount() != 1 {
			t.Errorf("Sink1: expected 1 event, got %d", sink1.EventCount())
		}
		if sink2.EventCount() != 1 {
			t.Errorf("Sink2: expected 1 event, got %d", sink2.EventCount())
		}
		if sink3.EventCount() != 1 {
			t.Errorf("Sink3: expected 1 event, got %d", sink3.EventCount())
		}
	})

	t.Run("Unrouted signals go nowhere", func(t *testing.T) {
		sink := &testSink{name: "test3"}
		RouteSignal(INFO, sink)

		// Send to unrouted signal
		Emit("UNROUTED", "should not appear")
		Debug("also unrouted")

		time.Sleep(10 * time.Millisecond)

		if sink.EventCount() != 0 {
			t.Errorf("Expected 0 events, got %d", sink.EventCount())
		}
	})
}

func TestConcurrentRouting(t *testing.T) {
	// Test that multiple goroutines can safely route signals
	const goroutines = 10
	const eventsPerGoroutine = 100

	sinks := make([]*testSink, goroutines)
	for i := 0; i < goroutines; i++ {
		sinks[i] = &testSink{name: string(rune(i))}
	}

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			signal := Signal("SIGNAL_" + string(rune(idx)))
			RouteSignal(signal, sinks[idx])

			for j := 0; j < eventsPerGoroutine; j++ {
				Emit(signal, "test")
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(50 * time.Millisecond) // Let events process

	for i, sink := range sinks {
		if sink.EventCount() != eventsPerGoroutine {
			t.Errorf("Sink %d: expected %d events, got %d", i, eventsPerGoroutine, sink.EventCount())
		}
	}
}

func TestRealWorldScenario(t *testing.T) {
	// Simulate a real application with different log types
	stdOut := &bytes.Buffer{}
	auditFile := &bytes.Buffer{}
	debugFile := &bytes.Buffer{}

	EnableStandardLogging(stdOut)
	EnableAuditLogging(auditFile)
	EnableDebugLogging(debugFile)

	// Generate various logs
	Debug("debug info")
	Info("application started")
	Warn("low memory")
	Error("connection failed")
	Emit(AUDIT, "user login", String("user", "alice"))
	Emit(SECURITY, "permission denied", String("resource", "admin"))

	// Let events process
	time.Sleep(20 * time.Millisecond)

	stdOutput := stdOut.String()
	auditOutput := auditFile.String()
	debugOutput := debugFile.String()

	// Check standard output has INFO, WARN, ERROR but not DEBUG
	if !strings.Contains(stdOutput, "application started") {
		t.Error("INFO not in standard output")
	}
	if !strings.Contains(stdOutput, "low memory") {
		t.Error("WARN not in standard output")
	}
	if !strings.Contains(stdOutput, "connection failed") {
		t.Error("ERROR not in standard output")
	}
	if strings.Contains(stdOutput, "debug info") {
		t.Error("DEBUG incorrectly in standard output")
	}

	// Check audit output has AUDIT and SECURITY
	if !strings.Contains(auditOutput, "user login") {
		t.Error("AUDIT not in audit output")
	}
	if !strings.Contains(auditOutput, "permission denied") {
		t.Error("SECURITY not in audit output")
	}

	// Check debug output only has DEBUG
	if !strings.Contains(debugOutput, "debug info") {
		t.Error("DEBUG not in debug output")
	}
	if strings.Contains(debugOutput, "application started") {
		t.Error("INFO incorrectly in debug output")
	}
}

// Benchmarks.
func BenchmarkDispatch(b *testing.B) {
	// Create a test sink
	sink := NewWriterSink(io.Discard)
	RouteSignal(INFO, sink)

	b.Run("SingleEvent", func(b *testing.B) {
		b.ReportAllocs()
		event := NewEvent(INFO, "benchmark", nil)
		for i := 0; i < b.N; i++ {
			dispatch.process(event)
		}
	})

	b.Run("EventWithFields", func(b *testing.B) {
		b.ReportAllocs()
		event := NewEvent(INFO, "benchmark", []Field{
			String("key1", "value1"),
			String("key2", "value2"),
			Int("count", 42),
		})
		for i := 0; i < b.N; i++ {
			dispatch.process(event)
		}
	})

	b.Run("UnroutedEvent", func(b *testing.B) {
		b.ReportAllocs()
		event := NewEvent("UNROUTED", "benchmark", nil)
		for i := 0; i < b.N; i++ {
			dispatch.process(event)
		}
	})

	b.Run("MultipleSinks", func(b *testing.B) {
		// Add more sinks to INFO
		RouteSignal(INFO, NewWriterSink(io.Discard))
		RouteSignal(INFO, NewWriterSink(io.Discard))

		b.ReportAllocs()
		event := NewEvent(INFO, "benchmark", nil)
		for i := 0; i < b.N; i++ {
			dispatch.process(event)
		}
	})
}

func BenchmarkConcurrentDispatch(b *testing.B) {
	sink := NewWriterSink(io.Discard)
	RouteSignal(INFO, sink)
	event := NewEvent(INFO, "concurrent", nil)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			dispatch.process(event)
		}
	})
}

func BenchmarkRouteSignal(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		signal := Signal("BENCH_" + string(rune(i%26)))
		sink := NewWriterSink(io.Discard)
		RouteSignal(signal, sink)
	}
}

// Stress test for high throughput.
func BenchmarkHighThroughput(b *testing.B) {
	// Set up multiple signals with multiple sinks each
	signals := []Signal{INFO, WARN, ERROR, DEBUG, AUDIT, METRIC}
	for _, sig := range signals {
		for i := 0; i < 3; i++ {
			RouteSignal(sig, NewWriterSink(io.Discard))
		}
	}

	events := make([]Event, len(signals))
	for i, sig := range signals {
		events[i] = NewEvent(sig, "stress test", []Field{
			String("index", string(rune(i))),
		})
	}

	b.ResetTimer()
	b.ReportAllocs()

	var counter int64
	b.RunParallel(func(pb *testing.PB) {
		localCounter := 0
		for pb.Next() {
			event := events[localCounter%len(events)]
			dispatch.process(event)
			localCounter++
		}
		atomic.AddInt64(&counter, int64(localCounter))
	})

	b.Logf("Processed %d events", counter)
}
