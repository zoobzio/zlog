package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/zoobzio/zlog"
)

func TestRequestTracing(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.EnableStandardLogging(buf)

	t.Run("RequestCorrelation", func(t *testing.T) {
		buf.Reset()

		// Create a request context
		req := RequestContext{
			RequestID: "test_req_123",
			UserID:    "test_user",
			Method:    "GET",
			Path:      "/test",
			TraceID:   "test_trace_456",
		}

		ctx := context.WithValue(context.Background(), requestContextKey, req)

		// Log multiple events with the same request
		logWithRequest(getRequestContext(ctx), zlog.Info, "Event 1")
		logWithRequest(getRequestContext(ctx), zlog.Info, "Event 2")
		logWithRequest(getRequestContext(ctx), zlog.Info, "Event 3")

		// All logs should have the same request_id
		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		if len(lines) != 3 {
			t.Fatalf("Expected 3 log lines, got %d", len(lines))
		}

		for i, line := range lines {
			var log map[string]interface{}
			if err := json.Unmarshal([]byte(line), &log); err != nil {
				t.Fatalf("Failed to parse log line %d: %v", i, err)
			}

			if log["request_id"] != "test_req_123" {
				t.Errorf("Log %d: expected request_id=test_req_123, got %v", i, log["request_id"])
			}

			expectedMsg := fmt.Sprintf("Event %d", i+1)
			if log["message"] != expectedMsg {
				t.Errorf("Log %d: expected message=%s, got %v", i, expectedMsg, log["message"])
			}
		}
	})

	t.Run("RequestContextPropagation", func(t *testing.T) {
		buf.Reset()

		server := &Server{
			db:    &Database{},
			cache: &Cache{},
		}

		req := RequestContext{
			RequestID: "test_correlation",
			UserID:    "user789",
			Method:    "GET",
			Path:      "/api/users",
			TraceID:   "trace_correlation",
		}

		ctx := context.WithValue(context.Background(), requestContextKey, req)
		server.HandleRequest(ctx)

		// Check that all logs have the same request_id
		logs := parseLogs(t, buf.String())

		requestIDs := make(map[string]int)
		for _, log := range logs {
			if rid, ok := log["request_id"].(string); ok {
				requestIDs[rid]++
			}
		}

		// Should only have one request ID
		if len(requestIDs) != 1 {
			t.Errorf("Expected 1 unique request_id, found %d: %v", len(requestIDs), requestIDs)
		}

		// All logs should be for our test request
		if count := requestIDs["test_correlation"]; count == 0 {
			t.Error("Request ID was not propagated through all operations")
		}
	})
}

func TestLogWithRequest(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.EnableStandardLogging(buf)

	t.Run("FieldOrdering", func(t *testing.T) {
		buf.Reset()

		req := RequestContext{
			RequestID: "field_test",
			UserID:    "user123",
		}

		// Log with additional fields
		logWithRequest(req, zlog.Info, "Test message",
			zlog.String("extra1", "value1"),
			zlog.Int("extra2", 42),
		)

		var log map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
			t.Fatalf("Failed to parse log: %v", err)
		}

		// request_id should always be present
		if log["request_id"] != "field_test" {
			t.Errorf("Expected request_id=field_test, got %v", log["request_id"])
		}

		// Additional fields should be present
		if log["extra1"] != "value1" {
			t.Errorf("Expected extra1=value1, got %v", log["extra1"])
		}
		if log["extra2"] != float64(42) { // JSON numbers are float64
			t.Errorf("Expected extra2=42, got %v", log["extra2"])
		}
	})

	t.Run("EmptyContext", func(t *testing.T) {
		buf.Reset()

		// Test with no request context
		ctx := context.Background()
		req := getRequestContext(ctx)

		logWithRequest(req, zlog.Info, "No context")

		var log map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
			t.Fatalf("Failed to parse log: %v", err)
		}

		// Should have "unknown" request_id
		if log["request_id"] != "unknown" {
			t.Errorf("Expected request_id=unknown for missing context, got %v", log["request_id"])
		}
	})
}

func TestDatabaseOperations(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.EnableStandardLogging(buf)

	db := &Database{}

	t.Run("QueryLogging", func(t *testing.T) {
		buf.Reset()

		req := RequestContext{RequestID: "db_test"}
		ctx := context.WithValue(context.Background(), requestContextKey, req)

		// Execute a query
		db.QueryUsers(ctx)

		// Find the database query log
		logs := parseLogs(t, buf.String())

		var queryLog map[string]interface{}
		for _, log := range logs {
			if log["message"] == "Database query" {
				queryLog = log
				break
			}
		}

		if queryLog == nil {
			t.Fatal("Database query log not found")
		}

		// Check log contains query details
		if queryLog["request_id"] != "db_test" {
			t.Errorf("Expected request_id=db_test, got %v", queryLog["request_id"])
		}
		if queryLog["table"] != "users" {
			t.Errorf("Expected table=users, got %v", queryLog["table"])
		}
		if _, ok := queryLog["query"]; !ok {
			t.Error("Query log missing 'query' field")
		}
	})
}

func TestCacheOperations(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.EnableStandardLogging(buf)

	cache := &Cache{}

	t.Run("CacheSetLogging", func(t *testing.T) {
		buf.Reset()

		req := RequestContext{RequestID: "cache_test"}
		ctx := context.WithValue(context.Background(), requestContextKey, req)

		// Set a cache value
		cache.Set(ctx, "test_key", []byte("test_value"))

		var log map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &log); err != nil {
			t.Fatalf("Failed to parse log: %v", err)
		}

		// Check cache set log
		if log["message"] != "Cache set" {
			t.Errorf("Expected message='Cache set', got %v", log["message"])
		}
		if log["key"] != "test_key" {
			t.Errorf("Expected key=test_key, got %v", log["key"])
		}
		if log["ttl_seconds"] != float64(300) {
			t.Errorf("Expected ttl_seconds=300, got %v", log["ttl_seconds"])
		}
	})
}

func TestRequestIDGeneration(t *testing.T) {
	// Test that request IDs are unique
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateRequestID()
		if ids[id] {
			t.Errorf("Duplicate request ID generated: %s", id)
		}
		ids[id] = true

		// Check format
		if !strings.HasPrefix(id, "req_") {
			t.Errorf("Request ID should start with 'req_', got: %s", id)
		}
	}
}

func TestTraceIDGeneration(t *testing.T) {
	// Test that trace IDs are unique and properly formatted
	ids := make(map[string]bool)

	for i := 0; i < 100; i++ {
		id := generateTraceID()
		if ids[id] {
			t.Errorf("Duplicate trace ID generated: %s", id)
		}
		ids[id] = true

		// Check format
		if !strings.HasPrefix(id, "trace_") {
			t.Errorf("Trace ID should start with 'trace_', got: %s", id)
		}
	}
}

// Helper function to parse multiple log lines.
func parseLogs(t *testing.T, output string) []map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	logs := make([]map[string]interface{}, 0, len(lines))

	for _, line := range lines {
		if line == "" {
			continue
		}

		var log map[string]interface{}
		if err := json.Unmarshal([]byte(line), &log); err != nil {
			t.Fatalf("Failed to parse log line: %v\nLine: %s", err, line)
		}
		logs = append(logs, log)
	}

	return logs
}
