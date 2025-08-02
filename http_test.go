package zlog

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewHTTPSink(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("creates sink successfully", func(t *testing.T) {
		sink := NewHTTPSink(server.URL)
		if sink == nil {
			t.Error("expected non-nil sink")
		}
	})

	t.Run("handles empty URL", func(t *testing.T) {
		sink := NewHTTPSink("")

		event := NewEvent("ERROR", "test message", nil)
		_, err := sink.Process(context.Background(), event)

		if err == nil {
			t.Error("expected error for empty URL")
		}
		if !strings.Contains(err.Error(), "valid URL") {
			t.Errorf("expected URL validation error, got: %v", err)
		}
	})

	t.Run("processes basic event successfully", func(t *testing.T) {
		sink := NewHTTPSink(server.URL)

		event := NewEvent("INFO", "test message", []Field{String("key", "value")})
		_, err := sink.Process(context.Background(), event)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestHTTPSinkJSONPayload(t *testing.T) {
	var receivedPayload map[string]interface{}
	var payloadMutex sync.Mutex

	// Create test server that captures the JSON payload
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify content type
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			t.Errorf("expected Content-Type application/json, got: %s", contentType)
		}

		// Read and parse body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		payloadMutex.Lock()
		err = json.Unmarshal(body, &receivedPayload)
		payloadMutex.Unlock()

		if err != nil {
			t.Errorf("failed to parse JSON payload: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("sends correct JSON format", func(t *testing.T) {
		sink := NewHTTPSink(server.URL)

		event := NewEvent("WARN", "test warning", []Field{
			String("user_id", "user123"),
			Int("retry_count", 3),
			Bool("critical", true),
		})
		event.Caller.File = "test.go"
		event.Caller.Line = 456

		_, err := sink.Process(context.Background(), event)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Verify payload structure
		payloadMutex.Lock()
		payload := receivedPayload
		payloadMutex.Unlock()

		// Check required fields
		if payload["signal"] != "WARN" {
			t.Errorf("expected signal 'WARN', got: %v", payload["signal"])
		}
		if payload["message"] != "test warning" {
			t.Errorf("expected message 'test warning', got: %v", payload["message"])
		}
		if payload["caller"] != "test.go:456" {
			t.Errorf("expected caller 'test.go:456', got: %v", payload["caller"])
		}

		// Check structured fields
		if payload["user_id"] != "user123" {
			t.Errorf("expected user_id 'user123', got: %v", payload["user_id"])
		}
		if payload["retry_count"] != float64(3) { // JSON numbers are float64
			t.Errorf("expected retry_count 3, got: %v", payload["retry_count"])
		}
		if payload["critical"] != true {
			t.Errorf("expected critical true, got: %v", payload["critical"])
		}

		// Check timestamp field exists
		if _, exists := payload["time"]; !exists {
			t.Error("expected time field in payload")
		}
	})

	t.Run("handles empty fields", func(t *testing.T) {
		sink := NewHTTPSink(server.URL)

		event := NewEvent("DEBUG", "debug message", nil)
		_, err := sink.Process(context.Background(), event)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Should still have basic fields
		payloadMutex.Lock()
		payload := receivedPayload
		payloadMutex.Unlock()

		if payload["signal"] != "DEBUG" {
			t.Errorf("expected signal 'DEBUG', got: %v", payload["signal"])
		}
		if payload["message"] != "debug message" {
			t.Errorf("expected message 'debug message', got: %v", payload["message"])
		}
	})
}

func TestHTTPSinkOptions(t *testing.T) {
	var receivedHeaders http.Header
	var receivedMethod string
	var headerMutex sync.Mutex

	// Create test server that captures headers and method
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headerMutex.Lock()
		receivedHeaders = r.Header.Clone()
		receivedMethod = r.Method
		headerMutex.Unlock()

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("WithMethod option", func(t *testing.T) {
		sink := NewHTTPSink(server.URL, WithMethod("PUT"))

		event := NewEvent("INFO", "test", nil)
		_, err := sink.Process(context.Background(), event)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		headerMutex.Lock()
		method := receivedMethod
		headerMutex.Unlock()

		if method != "PUT" {
			t.Errorf("expected method PUT, got: %s", method)
		}
	})

	t.Run("WithHeaders option", func(t *testing.T) {
		customHeaders := map[string]string{
			"Authorization": "Bearer token123",
			"X-API-Key":     "key456",
		}

		sink := NewHTTPSink(server.URL, WithHeaders(customHeaders))

		event := NewEvent("INFO", "test", nil)
		_, err := sink.Process(context.Background(), event)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		headerMutex.Lock()
		headers := receivedHeaders
		headerMutex.Unlock()

		if headers.Get("Authorization") != "Bearer token123" {
			t.Errorf("expected Authorization header, got: %s", headers.Get("Authorization"))
		}
		if headers.Get("X-API-Key") != "key456" {
			t.Errorf("expected X-API-Key header, got: %s", headers.Get("X-API-Key"))
		}
		// Should still have default Content-Type
		if headers.Get("Content-Type") != "application/json" {
			t.Errorf("expected Content-Type application/json, got: %s", headers.Get("Content-Type"))
		}
	})

	t.Run("WithUserAgent option", func(t *testing.T) {
		sink := NewHTTPSink(server.URL, WithUserAgent("MyApp/1.0"))

		event := NewEvent("INFO", "test", nil)
		_, err := sink.Process(context.Background(), event)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		headerMutex.Lock()
		headers := receivedHeaders
		headerMutex.Unlock()

		if headers.Get("User-Agent") != "MyApp/1.0" {
			t.Errorf("expected User-Agent MyApp/1.0, got: %s", headers.Get("User-Agent"))
		}
	})

	t.Run("default headers", func(t *testing.T) {
		sink := NewHTTPSink(server.URL)

		event := NewEvent("INFO", "test", nil)
		_, err := sink.Process(context.Background(), event)

		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		headerMutex.Lock()
		headers := receivedHeaders
		headerMutex.Unlock()

		if headers.Get("Content-Type") != "application/json" {
			t.Errorf("expected default Content-Type application/json, got: %s", headers.Get("Content-Type"))
		}
		if headers.Get("User-Agent") != "zlog-http-sink/1.0" {
			t.Errorf("expected default User-Agent zlog-http-sink/1.0, got: %s", headers.Get("User-Agent"))
		}
	})
}

func TestHTTPSinkErrorHandling(t *testing.T) {
	t.Run("handles HTTP error status codes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error")) //nolint:errcheck // Test response write
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL)
		event := NewEvent("ERROR", "test error", nil)

		_, err := sink.Process(context.Background(), event)

		if err == nil {
			t.Error("expected error for HTTP 500 response")
		}
		if !strings.Contains(err.Error(), "500") {
			t.Errorf("expected status code 500 in error, got: %v", err)
		}
		if !strings.Contains(err.Error(), "Internal server error") {
			t.Errorf("expected response body in error, got: %v", err)
		}
	})

	t.Run("handles 4xx client errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Bad request")) //nolint:errcheck // Test response write
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL)
		event := NewEvent("INFO", "test", nil)

		_, err := sink.Process(context.Background(), event)

		if err == nil {
			t.Error("expected error for HTTP 400 response")
		}
		if !strings.Contains(err.Error(), "400") {
			t.Errorf("expected status code 400 in error, got: %v", err)
		}
	})

	t.Run("handles network connection errors", func(t *testing.T) {
		// Use invalid URL that will cause connection error
		sink := NewHTTPSink("http://localhost:99999/invalid")
		event := NewEvent("ERROR", "test", nil)

		_, err := sink.Process(context.Background(), event)

		if err == nil {
			t.Error("expected error for connection failure")
		}
		if !strings.Contains(err.Error(), "HTTP request failed") {
			t.Errorf("expected connection error, got: %v", err)
		}
	})

	t.Run("handles request timeout", func(t *testing.T) {
		// Create slow server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond) // Sleep longer than timeout
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL, WithTimeout(50*time.Millisecond))
		event := NewEvent("INFO", "test", nil)

		start := time.Now()
		_, err := sink.Process(context.Background(), event)
		elapsed := time.Since(start)

		if err == nil {
			t.Error("expected timeout error")
		}
		if elapsed > 150*time.Millisecond {
			t.Errorf("timeout took too long: %v", elapsed)
		}
	})

	t.Run("handles context cancellation", func(t *testing.T) {
		// Create slow server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL)
		event := NewEvent("INFO", "test", nil)

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		_, err := sink.Process(ctx, event)

		if err == nil {
			t.Error("expected context cancellation error")
		}
	})
}

func TestHTTPSinkConcurrency(t *testing.T) {
	var requestCount int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		atomic.AddInt64(&requestCount, 1)
		// Small delay to test concurrent handling
		time.Sleep(10 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("handles concurrent requests", func(t *testing.T) {
		sink := NewHTTPSink(server.URL)

		var wg sync.WaitGroup
		numGoroutines := 10
		eventsPerGoroutine := 5

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()
				for j := 0; j < eventsPerGoroutine; j++ {
					event := NewEvent("INFO", fmt.Sprintf("Concurrent message from goroutine %d event %d", goroutineID, j), []Field{
						Int("goroutine_id", goroutineID),
						Int("event_num", j),
					})

					_, err := sink.Process(context.Background(), event)
					if err != nil {
						t.Errorf("goroutine %d event %d error: %v", goroutineID, j, err)
					}
				}
			}(i)
		}

		wg.Wait()

		expectedRequests := int64(numGoroutines * eventsPerGoroutine)
		finalCount := atomic.LoadInt64(&requestCount)

		if finalCount != expectedRequests {
			t.Errorf("expected %d requests, got %d", expectedRequests, finalCount)
		}
	})
}

func TestHTTPSinkWithAdapters(t *testing.T) {
	t.Run("works with retry adapter on failures", func(t *testing.T) {
		attemptCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			attemptCount++
			if attemptCount < 3 {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL).WithRetry(3)
		event := NewEvent("ERROR", "retry test", nil)

		_, err := sink.Process(context.Background(), event)

		if err != nil {
			t.Errorf("expected success after retries, got: %v", err)
		}
		if attemptCount != 3 {
			t.Errorf("expected 3 attempts, got: %d", attemptCount)
		}
	})

	t.Run("works with async adapter", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL).WithAsync()
		event := NewEvent("INFO", "async test", []Field{String("async", "true")})

		start := time.Now()
		_, err := sink.Process(context.Background(), event)
		elapsed := time.Since(start)

		if err != nil {
			t.Errorf("unexpected error with async: %v", err)
		}
		if elapsed > 50*time.Millisecond {
			t.Errorf("async processing took too long: %v", elapsed)
		}

		// Give async processing time to complete
		time.Sleep(100 * time.Millisecond)
	})

	t.Run("works with filter adapter", func(t *testing.T) {
		var receivedRequests int64

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			atomic.AddInt64(&receivedRequests, 1)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL).WithFilter(func(_ context.Context, e Log) bool {
			return e.Signal == "ERROR" // Only send errors
		})

		// This should be filtered out
		infoEvent := NewEvent("INFO", "info message", nil)
		_, err := sink.Process(context.Background(), infoEvent)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// This should pass through
		errorEvent := NewEvent("ERROR", "error message", nil)
		_, err2 := sink.Process(context.Background(), errorEvent)
		if err2 != nil {
			t.Errorf("unexpected error: %v", err2)
		}

		// Give some time for any async processing
		time.Sleep(50 * time.Millisecond)

		finalCount := atomic.LoadInt64(&receivedRequests)
		if finalCount != 1 {
			t.Errorf("expected 1 HTTP request (ERROR only), got %d", finalCount)
		}
	})

	t.Run("works with timeout adapter", func(t *testing.T) {
		// Create slow server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			time.Sleep(200 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL).WithTimeout(50 * time.Millisecond)
		event := NewEvent("INFO", "timeout test", nil)

		_, err := sink.Process(context.Background(), event)

		if err == nil {
			t.Error("expected timeout error")
		}
	})
}

func TestHTTPSinkResponseBodyLimiting(t *testing.T) {
	t.Run("limits large error response bodies", func(t *testing.T) {
		// Create server that returns large error response
		largeBody := strings.Repeat("error ", 500) // > 1KB
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte(largeBody)) //nolint:errcheck // Test response write
		}))
		defer server.Close()

		sink := NewHTTPSink(server.URL)
		event := NewEvent("ERROR", "test", nil)

		_, err := sink.Process(context.Background(), event)

		if err == nil {
			t.Error("expected error for HTTP 500 response")
		}

		// Error message should be truncated to reasonable length
		errorMsg := err.Error()
		if len(errorMsg) > 2000 { // Should be much shorter than original
			t.Errorf("error message too long: %d characters", len(errorMsg))
		}
	})
}
