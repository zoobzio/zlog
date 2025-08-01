package zlog

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// HTTPOption configures HTTP sink behavior using the functional options pattern.
type HTTPOption func(*httpConfig)

// httpConfig holds configuration for HTTP sink.
type httpConfig struct {
	headers   map[string]string
	method    string
	userAgent string
	timeout   time.Duration
}

// WithMethod sets the HTTP method for requests (default: POST).
func WithMethod(method string) HTTPOption {
	return func(config *httpConfig) {
		if method != "" {
			config.method = method
		}
	}
}

// WithHeaders sets custom HTTP headers for requests.
// Common use cases:
//   - Authorization: "Bearer token123"
//   - Content-Type: "application/json" (set automatically)
//   - X-API-Key: "key123"
func WithHeaders(headers map[string]string) HTTPOption {
	return func(config *httpConfig) {
		if config.headers == nil {
			config.headers = make(map[string]string)
		}
		for k, v := range headers {
			config.headers[k] = v
		}
	}
}

// WithTimeout sets the HTTP request timeout (default: 30 seconds).
func WithTimeout(timeout time.Duration) HTTPOption {
	return func(config *httpConfig) {
		if timeout > 0 {
			config.timeout = timeout
		}
	}
}

// WithUserAgent sets a custom User-Agent header (default: "zlog-http-sink/1.0").
func WithUserAgent(userAgent string) HTTPOption {
	return func(config *httpConfig) {
		if userAgent != "" {
			config.userAgent = userAgent
		}
	}
}

// NewHTTPSink creates a sink that sends JSON-formatted events to an HTTP endpoint.
//
// This sink is designed for integration with webhooks, log aggregation APIs,
// and custom log collectors. It provides:
//   - Zero external dependencies (uses only Go stdlib)
//   - Same JSON format as other zlog sinks for consistency
//   - Configurable HTTP method, headers, and timeout
//   - Robust error handling for network failures
//   - Full compatibility with all sink adapters
//
// JSON payload format:
//
//	{"time":"2023-10-20T15:04:05Z","signal":"ERROR","message":"Database connection failed","caller":"db.go:123","error":"connection timeout"}
//
// Parameters:
//   - url: The HTTP endpoint to send events to
//   - options: Functional options for configuration
//
// Example usage:
//
//	// Basic webhook
//	webhook := zlog.NewHTTPSink("https://hooks.slack.com/services/...")
//
//	// API with authentication
//	apiSink := zlog.NewHTTPSink("https://api.example.com/logs",
//	    zlog.WithHeaders(map[string]string{
//	        "Authorization": "Bearer " + token,
//	        "X-API-Key": apiKey,
//	    }),
//	    zlog.WithTimeout(10 * time.Second),
//	)
//
//	// Custom method and user agent
//	customSink := zlog.NewHTTPSink("https://collector.example.com/events",
//	    zlog.WithMethod("PUT"),
//	    zlog.WithUserAgent("MyApp/1.0"),
//	)
//
//	// With adapters for reliability
//	productionSink := apiSink.
//	    WithRetry(3).
//	    WithTimeout(30 * time.Second).
//	    WithAsync()
//
// The sink handles HTTP error responses as processing errors, making them
// compatible with retry and fallback adapters. Network timeouts and connection
// failures are also handled gracefully.
//
// HTTP status codes 200-299 are considered successful. All other status codes
// result in an error that includes the response status and body (if available).
func NewHTTPSink(url string, options ...HTTPOption) *Sink {
	// Apply default configuration
	config := &httpConfig{
		method:    "POST",
		headers:   make(map[string]string),
		timeout:   30 * time.Second,
		userAgent: "zlog-http-sink/1.0",
	}

	// Apply functional options
	for _, option := range options {
		option(config)
	}

	// Validate URL
	if url == "" {
		return NewSink("http-failed", func(_ context.Context, _ Event) error {
			return fmt.Errorf("HTTP sink requires a valid URL")
		})
	}

	// Create HTTP client with configured timeout
	client := &http.Client{
		Timeout: config.timeout,
	}

	return NewSink("http", func(ctx context.Context, event Event) error {
		// Build JSON structure (same format as other zlog sinks)
		entry := map[string]interface{}{
			"time":    event.Time.Format(time.RFC3339Nano),
			"signal":  string(event.Signal),
			"message": event.Message,
		}

		// Add caller info if available
		if event.Caller.File != "" {
			entry["caller"] = fmt.Sprintf("%s:%d", event.Caller.File, event.Caller.Line)
		}

		// Add all structured fields as top-level JSON properties
		for _, field := range event.Fields {
			entry[field.Key] = field.Value
		}

		// Encode to JSON
		jsonData, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal event to JSON: %w", err)
		}

		// Create HTTP request
		req, err := http.NewRequestWithContext(ctx, config.method, url, bytes.NewReader(jsonData))
		if err != nil {
			return fmt.Errorf("failed to create HTTP request: %w", err)
		}

		// Set headers
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", config.userAgent)

		// Apply custom headers
		for key, value := range config.headers {
			req.Header.Set(key, value)
		}

		// Execute request
		resp, err := client.Do(req)
		if err != nil {
			return fmt.Errorf("HTTP request failed: %w", err)
		}
		defer resp.Body.Close()

		// Check status code (2xx = success)
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			// Read response body for error details (limit to 1KB to avoid memory issues)
			var bodyBytes []byte
			if resp.ContentLength >= 0 && resp.ContentLength <= 1024 {
				bodyBytes = make([]byte, resp.ContentLength)
				_, _ = resp.Body.Read(bodyBytes) //nolint:errcheck // Best effort error body read
			} else {
				// Read up to 1KB
				bodyBytes = make([]byte, 1024)
				n, _ := resp.Body.Read(bodyBytes) //nolint:errcheck // Best effort error body read
				bodyBytes = bodyBytes[:n]
			}

			return fmt.Errorf("HTTP request failed with status %d: %s", resp.StatusCode, string(bodyBytes))
		}

		return nil
	})
}
