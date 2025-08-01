package zlog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewRotatingFileSink(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "test.log")

	t.Run("creates sink successfully", func(t *testing.T) {
		sink := NewRotatingFileSink(logFile, 1024, 3)
		if sink == nil {
			t.Error("expected non-nil sink")
			return
		}

		// Test basic functionality
		event := NewEvent("TEST", "test message", []Field{String("key", "value")})
		_, err := sink.Process(context.Background(), event)
		if err != nil {
			t.Errorf("expected no error but got: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			t.Error("expected log file to be created")
		}
	})

	t.Run("handles invalid directory", func(t *testing.T) {
		invalidPath := filepath.Join(tempDir, "nonexistent", "dir", "test.log")
		sink := NewRotatingFileSink(invalidPath, 1024, 3)

		event := NewEvent("TEST", "test message", nil)
		_, err := sink.Process(context.Background(), event)
		if err == nil {
			t.Error("expected error when writing to invalid path")
		}
	})

	t.Run("uses default values for invalid parameters", func(t *testing.T) {
		logFile2 := filepath.Join(tempDir, "defaults.log")
		sink := NewRotatingFileSink(logFile2, 0, 0) // Invalid size and files

		event := NewEvent("TEST", "test message", nil)
		_, err := sink.Process(context.Background(), event)
		if err != nil {
			t.Errorf("expected no error with default values but got: %v", err)
		}
	})
}

func TestRotatingFileSinkJSONFormat(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "json_test.log")

	sink := NewRotatingFileSink(logFile, 1024*1024, 3)

	t.Run("writes correct JSON format", func(t *testing.T) {
		event := NewEvent("INFO", "test message", []Field{
			String("user_id", "12345"),
			Int("count", 42),
			Bool("active", true),
		})
		event.Caller.File = "test.go"
		event.Caller.Line = 123

		_, err := sink.Process(context.Background(), event)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Read and parse the JSON
		content, readErr := os.ReadFile(logFile)
		if readErr != nil {
			t.Fatalf("failed to read log file: %v", readErr)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) != 1 {
			t.Errorf("expected 1 log line but got %d", len(lines))
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
			t.Fatalf("failed to parse JSON: %v", err)
		}

		// Verify required fields
		if entry["signal"] != "INFO" {
			t.Errorf("expected signal 'INFO' but got %v", entry["signal"])
		}
		if entry["message"] != "test message" {
			t.Errorf("expected message 'test message' but got %v", entry["message"])
		}
		if entry["caller"] != "test.go:123" {
			t.Errorf("expected caller 'test.go:123' but got %v", entry["caller"])
		}

		// Verify structured fields
		if entry["user_id"] != "12345" {
			t.Errorf("expected user_id '12345' but got %v", entry["user_id"])
		}
		if entry["count"] != float64(42) { // JSON numbers are float64
			t.Errorf("expected count 42 but got %v", entry["count"])
		}
		if entry["active"] != true {
			t.Errorf("expected active true but got %v", entry["active"])
		}

		// Verify timestamp format
		if _, ok := entry["time"]; !ok {
			t.Error("expected time field")
		}
	})

	t.Run("handles empty fields", func(t *testing.T) {
		event := NewEvent("DEBUG", "empty fields test", nil)

		_, err := sink.Process(context.Background(), event)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestRotatingFileSinkRotation(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "rotation_test.log")

	// Use small max size to trigger rotation
	maxSize := int64(200)
	sink := NewRotatingFileSink(logFile, maxSize, 3)

	t.Run("rotates when size exceeded", func(t *testing.T) {
		// Write enough events to trigger rotation
		for i := 0; i < 10; i++ {
			event := NewEvent("INFO", fmt.Sprintf("Long message %d to exceed size limit and trigger rotation mechanism", i), nil)
			_, err := sink.Process(context.Background(), event)
			if err != nil {
				t.Errorf("unexpected error on event %d: %v", i, err)
			}
		}

		// Check that rotation occurred - should have backup files
		if _, err := os.Stat(logFile + ".1"); os.IsNotExist(err) {
			t.Error("expected rotated file .1 to exist")
		}

		// Original file should still exist and be smaller than maxSize
		info, err := os.Stat(logFile)
		if err != nil {
			t.Errorf("expected current log file to exist: %v", err)
		} else if info.Size() > maxSize {
			t.Errorf("current file size %d should be less than maxSize %d", info.Size(), maxSize)
		}
	})

	t.Run("maintains correct file naming order", func(t *testing.T) {
		// Write more events to create multiple rotated files
		for i := 0; i < 20; i++ {
			event := NewEvent("INFO", fmt.Sprintf("Another long message %d to create more rotated files for testing", i), nil)
			_, err := sink.Process(context.Background(), event)
			if err != nil {
				t.Errorf("unexpected error on event %d: %v", i, err)
			}
		}

		// Check file order (.1 should be newer than .2, etc.)
		files := []string{logFile + ".1", logFile + ".2", logFile + ".3"}
		var prevModTime time.Time

		for i, filename := range files {
			info, err := os.Stat(filename)
			if os.IsNotExist(err) {
				continue // It's OK if later files don't exist
			}
			if err != nil {
				t.Errorf("error stating file %s: %v", filename, err)
				continue
			}

			if i > 0 && info.ModTime().After(prevModTime) {
				t.Errorf("file %s should be older than previous file", filename)
			}
			prevModTime = info.ModTime()
		}
	})

	t.Run("respects maxFiles limit", func(t *testing.T) {
		// Write many more events to test file cleanup
		for i := 0; i < 50; i++ {
			event := NewEvent("INFO", fmt.Sprintf("Excessive logging message %d for cleanup testing", i), nil)
			_, err := sink.Process(context.Background(), event)
			if err != nil {
				t.Errorf("unexpected error on event %d: %v", i, err)
			}
		}

		// Should not have more than maxFiles (3) backup files
		if _, err := os.Stat(logFile + ".4"); !os.IsNotExist(err) {
			t.Error("expected file .4 to not exist (exceeds maxFiles limit)")
		}
	})
}

func TestRotatingFileSinkConcurrency(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "concurrent_test.log")

	sink := NewRotatingFileSink(logFile, 1024*1024, 5)

	t.Run("handles concurrent writes", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10
		eventsPerGoroutine := 100

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

		// Verify file exists
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			t.Error("expected log file to exist after concurrent writes")
		}

		// Read and count total lines
		content, err := os.ReadFile(logFile)
		if err != nil {
			t.Fatalf("failed to read log file: %v", err)
		}

		lines := strings.Split(strings.TrimSpace(string(content)), "\n")
		if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
			t.Error("expected log entries from concurrent writes")
		}

		// Verify each line is valid JSON
		for i, line := range lines {
			if line == "" {
				continue
			}
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(line), &entry); err != nil {
				t.Errorf("line %d is not valid JSON: %v", i, err)
			}
		}
	})
}

func TestRotatingFileSinkWithAdapters(t *testing.T) {
	tempDir := t.TempDir()
	logFile := filepath.Join(tempDir, "adapter_test.log")

	t.Run("works with retry adapter", func(t *testing.T) {
		// Test with a file sink that initially fails
		badFile := filepath.Join(tempDir, "nonexistent", "bad.log")
		sink := NewRotatingFileSink(badFile, 1024, 3).WithRetry(2)

		event := NewEvent("ERROR", "retry test", nil)
		_, err := sink.Process(context.Background(), event)

		// Should eventually fail even with retries
		if err == nil {
			t.Error("expected error even with retries for invalid path")
		}
	})

	t.Run("works with async adapter", func(t *testing.T) {
		sink := NewRotatingFileSink(logFile, 1024*1024, 3).WithAsync()

		event := NewEvent("INFO", "async test", []Field{String("test", "async")})

		// Should return immediately with async
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

		// Verify file was created
		if _, err := os.Stat(logFile); os.IsNotExist(err) {
			t.Error("expected log file to exist after async processing")
		}
	})

	t.Run("works with filter adapter", func(t *testing.T) {
		filterFile := filepath.Join(tempDir, "filter_test.log")
		sink := NewRotatingFileSink(filterFile, 1024*1024, 3).WithFilter(func(_ context.Context, e Event) bool {
			return e.Signal == "ERROR" // Only log errors
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

		// Check file contents
		if _, err := os.Stat(filterFile); os.IsNotExist(err) {
			t.Error("expected filtered log file to exist")
		} else {
			content, err := os.ReadFile(filterFile)
			if err != nil {
				t.Fatalf("failed to read filtered log file: %v", err)
			}

			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			if len(lines) != 1 || lines[0] == "" {
				t.Errorf("expected exactly 1 log entry (ERROR only), got %d lines", len(lines))
			}

			// Verify it's the error message
			var entry map[string]interface{}
			if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
				t.Fatalf("failed to parse JSON: %v", err)
			}

			if entry["signal"] != "ERROR" {
				t.Errorf("expected ERROR signal but got %v", entry["signal"])
			}
			if entry["message"] != "error message" {
				t.Errorf("expected 'error message' but got %v", entry["message"])
			}
		}
	})
}

func TestRotatingFileSinkEdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	t.Run("handles permission errors gracefully", func(t *testing.T) {
		// Create a read-only directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0o444); err != nil {
			t.Fatalf("failed to create read-only directory: %v", err)
		}
		defer func() {
			if err := os.Chmod(readOnlyDir, 0o755); err != nil {
				t.Errorf("failed to cleanup read-only directory: %v", err)
			}
		}() // Cleanup

		logFile := filepath.Join(readOnlyDir, "readonly.log")
		sink := NewRotatingFileSink(logFile, 1024, 3)

		event := NewEvent("ERROR", "permission test", nil)
		_, err := sink.Process(context.Background(), event)

		if err == nil {
			t.Error("expected permission error")
		}
	})

	t.Run("handles very small max size", func(t *testing.T) {
		logFile := filepath.Join(tempDir, "tiny.log")
		sink := NewRotatingFileSink(logFile, 10, 2) // Very small max size

		// Even one event should trigger rotation
		event := NewEvent("INFO", "This message is definitely longer than 10 bytes", nil)
		_, err := sink.Process(context.Background(), event)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		// Write another event
		event2 := NewEvent("INFO", "Another long message", nil)
		_, err2 := sink.Process(context.Background(), event2)
		if err2 != nil {
			t.Errorf("unexpected error: %v", err2)
		}

		// Should have rotated files
		if _, err := os.Stat(logFile + ".1"); os.IsNotExist(err) {
			t.Error("expected rotation with very small max size")
		}
	})

	t.Run("preserves event data integrity during rotation", func(t *testing.T) {
		logFile := filepath.Join(tempDir, "integrity.log")
		sink := NewRotatingFileSink(logFile, 300, 5)

		testEvents := []Event{
			NewEvent("INFO", "First event", []Field{String("id", "1")}),
			NewEvent("WARN", "Second event", []Field{String("id", "2")}),
			NewEvent("ERROR", "Third event", []Field{String("id", "3")}),
		}

		// Write events that should trigger rotation
		for _, event := range testEvents {
			_, err := sink.Process(context.Background(), event)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}

		// Collect all log entries from current and rotated files
		var allEntries []map[string]interface{}

		// Read current file
		if content, err := os.ReadFile(logFile); err == nil {
			lines := strings.Split(strings.TrimSpace(string(content)), "\n")
			for _, line := range lines {
				if line == "" {
					continue
				}
				var entry map[string]interface{}
				if err := json.Unmarshal([]byte(line), &entry); err == nil {
					allEntries = append(allEntries, entry)
				}
			}
		}

		// Read rotated files
		for i := 1; i <= 5; i++ {
			rotatedFile := fmt.Sprintf("%s.%d", logFile, i)
			if content, err := os.ReadFile(rotatedFile); err == nil {
				lines := strings.Split(strings.TrimSpace(string(content)), "\n")
				for _, line := range lines {
					if line == "" {
						continue
					}
					var entry map[string]interface{}
					if err := json.Unmarshal([]byte(line), &entry); err == nil {
						allEntries = append(allEntries, entry)
					}
				}
			}
		}

		// Verify we have the expected events
		if len(allEntries) < len(testEvents) {
			t.Errorf("expected at least %d entries but found %d", len(testEvents), len(allEntries))
		}

		// Verify event data integrity
		foundIDs := make(map[string]bool)
		for _, entry := range allEntries {
			if id, ok := entry["id"]; ok {
				if idStr, ok := id.(string); ok {
					foundIDs[idStr] = true
				}
			}
		}

		expectedIDs := []string{"1", "2", "3"}
		for _, expectedID := range expectedIDs {
			if !foundIDs[expectedID] {
				t.Errorf("missing expected event with id %s", expectedID)
			}
		}
	})
}
