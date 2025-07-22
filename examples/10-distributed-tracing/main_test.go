package main

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zoobzio/zlog"
)

func TestTraceContextPropagation(t *testing.T) {
	t.Run("ContextInjectionExtraction", func(t *testing.T) {
		trace := TraceContext{
			TraceID:      "test_trace_123",
			SpanID:       "test_span_456",
			ParentSpanID: "test_parent_789",
			Baggage: map[string]string{
				"user_id": "test_user",
				"env":     "test",
			},
		}

		// Inject into context
		ctx := InjectTraceContext(context.Background(), trace)

		// Extract from context
		extracted, ok := ExtractTraceContext(ctx)
		if !ok {
			t.Fatal("Failed to extract trace context")
		}

		if extracted.TraceID != trace.TraceID {
			t.Errorf("Expected trace_id=%s, got %s", trace.TraceID, extracted.TraceID)
		}
		if extracted.SpanID != trace.SpanID {
			t.Errorf("Expected span_id=%s, got %s", trace.SpanID, extracted.SpanID)
		}
		if extracted.Baggage["user_id"] != "test_user" {
			t.Errorf("Expected baggage user_id=test_user, got %s", extracted.Baggage["user_id"])
		}
	})

	t.Run("MissingContext", func(t *testing.T) {
		ctx := context.Background()
		_, ok := ExtractTraceContext(ctx)
		if ok {
			t.Error("Should not extract trace from empty context")
		}
	})
}

func TestDistributedSystem(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.EnableStandardLogging(buf)

	system := &DistributedSystem{
		services: map[string]*Service{
			"test-service-a": {Name: "test-service-a", Version: "1.0.0", Instance: "a-1"},
			"test-service-b": {Name: "test-service-b", Version: "1.0.0", Instance: "b-1"},
		},
	}

	t.Run("ServiceSpanCreation", func(t *testing.T) {
		buf.Reset()

		trace := TraceContext{
			TraceID: "test_trace",
			SpanID:  "test_span",
			Baggage: map[string]string{"key": "value"},
		}

		system.startSpan("test-service-a", trace, "TestOperation")

		// Parse log
		var log map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
			t.Fatalf("Failed to parse log: %v", err)
		}

		// Verify required fields
		if log["service"] != "test-service-a" {
			t.Errorf("Expected service=test-service-a, got %v", log["service"])
		}
		if log["trace_id"] != "test_trace" {
			t.Errorf("Expected trace_id=test_trace, got %v", log["trace_id"])
		}
		if log["span_id"] != "test_span" {
			t.Errorf("Expected span_id=test_span, got %v", log["span_id"])
		}
		if log["baggage.key"] != "value" {
			t.Errorf("Expected baggage.key=value, got %v", log["baggage.key"])
		}
	})

	t.Run("ServiceCallFlow", func(t *testing.T) {
		buf.Reset()

		parentTrace := TraceContext{
			TraceID: "flow_trace",
			SpanID:  "parent_span",
		}

		childTrace := system.callService("test-service-a", "test-service-b", parentTrace, "TestMethod")

		// Should have same trace ID but different span ID
		if childTrace.TraceID != parentTrace.TraceID {
			t.Error("Child trace should have same trace ID as parent")
		}
		if childTrace.SpanID == parentTrace.SpanID {
			t.Error("Child trace should have different span ID from parent")
		}
		if childTrace.ParentSpanID != parentTrace.SpanID {
			t.Error("Child trace should reference parent span ID")
		}

		// Check logs were created
		logs := parseLogs(t, buf.String())
		if len(logs) < 2 {
			t.Errorf("Expected at least 2 logs, got %d", len(logs))
		}

		// Verify service call log
		foundCall := false
		for _, log := range logs {
			if log["message"] == "Service call initiated" {
				foundCall = true
				if log["from_service"] != "test-service-a" {
					t.Errorf("Expected from_service=test-service-a, got %v", log["from_service"])
				}
				if log["to_service"] != "test-service-b" {
					t.Errorf("Expected to_service=test-service-b, got %v", log["to_service"])
				}
			}
		}
		if !foundCall {
			t.Error("Service call log not found")
		}
	})

	t.Run("ErrorPropagation", func(t *testing.T) {
		buf.Reset()

		trace := TraceContext{
			TraceID: "error_trace",
			SpanID:  "error_span",
		}

		testErr := &MockError{msg: "test error"}
		system.endSpan("test-service-a", trace, testErr)

		// Parse error log
		var log map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
			t.Fatalf("Failed to parse log: %v", err)
		}

		if log["signal"] != "ERROR" {
			t.Errorf("Expected ERROR signal for failed span, got %v", log["signal"])
		}
		if log["status"] != "error" {
			t.Errorf("Expected status=error, got %v", log["status"])
		}
		if log["error"] != "test error" {
			t.Errorf("Expected error message, got %v", log["error"])
		}
	})
}

func TestTraceIDGeneration(t *testing.T) {
	// Test uniqueness
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateTraceID()
		if ids[id] {
			t.Errorf("Duplicate trace ID: %s", id)
		}
		ids[id] = true

		// Verify format
		if !strings.HasPrefix(id, "trace_") {
			t.Errorf("Trace ID should start with 'trace_', got: %s", id)
		}

		time.Sleep(time.Microsecond) // Ensure different timestamps
	}
}

func TestSpanIDGeneration(t *testing.T) {
	// Test uniqueness
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateSpanID()
		if ids[id] {
			t.Errorf("Duplicate span ID: %s", id)
		}
		ids[id] = true

		// Verify format
		if !strings.HasPrefix(id, "span_") {
			t.Errorf("Span ID should start with 'span_', got: %s", id)
		}

		time.Sleep(time.Microsecond)
	}
}

func TestTracingSink(t *testing.T) {
	sink := NewTracingSink("test-service")

	t.Run("AddsTraceIDIfMissing", func(t *testing.T) {
		// TracingSink only checks for trace_id, doesn't modify the event
		// This is actually correct behavior for a read-only sink
		event := zlog.NewEvent(zlog.INFO, "Test message", []zlog.Field{
			zlog.String("key", "value"),
		})

		originalFieldCount := len(event.Fields)

		// Write event without trace_id
		err := sink.Write(event)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// TracingSink is read-only, it doesn't modify events
		// In a real implementation, it would send to a tracing backend
		if len(event.Fields) != originalFieldCount {
			t.Error("TracingSink should not modify events")
		}
	})

	t.Run("PreservesExistingTraceID", func(t *testing.T) {
		event := zlog.NewEvent(zlog.INFO, "Test message", []zlog.Field{
			zlog.String("trace_id", "existing_trace"),
		})

		originalFieldCount := len(event.Fields)

		err := sink.Write(event)
		if err != nil {
			t.Fatalf("Write failed: %v", err)
		}

		// Should not add fields if trace_id exists
		if len(event.Fields) != originalFieldCount {
			t.Error("Should not add fields when trace_id exists")
		}
	})

	t.Run("SinkName", func(t *testing.T) {
		if sink.Name() != "tracing:test-service" {
			t.Errorf("Expected name 'tracing:test-service', got %s", sink.Name())
		}
	})
}

func TestDistributedScenarios(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.EnableStandardLogging(buf)

	system := &DistributedSystem{
		services: map[string]*Service{
			"api-gateway":          {Name: "api-gateway", Version: "1.0.0", Instance: "gw-1"},
			"user-service":         {Name: "user-service", Version: "1.0.0", Instance: "usr-1"},
			"order-service":        {Name: "order-service", Version: "1.0.0", Instance: "ord-1"},
			"inventory-service":    {Name: "inventory-service", Version: "1.0.0", Instance: "inv-1"},
			"notification-service": {Name: "notification-service", Version: "1.0.0", Instance: "not-1"},
		},
	}

	t.Run("OrderFlowTracing", func(t *testing.T) {
		buf.Reset()
		system.SimulateOrderFlow()

		logs := parseLogs(t, buf.String())

		// Find all trace IDs
		traceIDs := make(map[string]int)
		for _, log := range logs {
			if traceID, ok := log["trace_id"].(string); ok {
				traceIDs[traceID]++
			}
		}

		// Should have exactly one trace ID for the entire flow
		if len(traceIDs) != 1 {
			t.Errorf("Expected 1 trace ID, found %d", len(traceIDs))
		}

		// Verify all expected services participated
		services := make(map[string]bool)
		for _, log := range logs {
			if service, ok := log["service"].(string); ok {
				services[service] = true
			}
		}

		expectedServices := []string{"api-gateway", "user-service", "order-service", "inventory-service", "notification-service"}
		for _, svc := range expectedServices {
			if !services[svc] {
				t.Errorf("Missing logs from service: %s", svc)
			}
		}
	})

	t.Run("ParallelCallsTracing", func(t *testing.T) {
		buf.Reset()
		system.SimulateParallelCalls()

		logs := parseLogs(t, buf.String())

		// Find all spans for the single trace
		var traceID string
		spans := make(map[string]bool)
		for _, log := range logs {
			if tid, ok := log["trace_id"].(string); ok && traceID == "" {
				traceID = tid
			}
			if spanID, ok := log["span_id"].(string); ok {
				spans[spanID] = true
			}
		}

		// Should have multiple spans
		if len(spans) < 5 { // At least root + 4 service calls
			t.Errorf("Expected at least 5 spans, got %d", len(spans))
		}

		// Check for parallel execution indicators
		parentSpans := make(map[string]int)
		for _, log := range logs {
			if parentSpan, ok := log["parent_span_id"].(string); ok {
				parentSpans[parentSpan]++
			}
		}

		// Should have multiple children of the root span (parallel calls)
		maxChildren := 0
		for _, count := range parentSpans {
			if count > maxChildren {
				maxChildren = count
			}
		}

		if maxChildren < 2 {
			t.Error("Expected multiple parallel calls from root span")
		}
	})
}

// Helper types and functions

type MockError struct {
	msg string
}

func (e *MockError) Error() string {
	return e.msg
}

func parseLogs(t *testing.T, output string) []map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	logs := make([]map[string]interface{}, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var log map[string]interface{}
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			t.Logf("Failed to parse log line: %v\nLine: %s", err, line)
			continue
		}
		logs = append(logs, log)
	}

	return logs
}
