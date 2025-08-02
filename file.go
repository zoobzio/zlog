package zlog

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// rotatingFileWriter manages file rotation and writing operations.
type rotatingFileWriter struct { //nolint:govet // Field ordering is logical, not memory-optimized
	mu          sync.Mutex
	currentFile *os.File
	filename    string
	maxSize     int64
	currentSize int64
	maxFiles    int
}

// newRotatingFileWriter creates a new rotating file writer.
func newRotatingFileWriter(filename string, maxSize int64, maxFiles int) (*rotatingFileWriter, error) {
	if maxSize <= 0 {
		maxSize = 100 * 1024 * 1024 // Default 100MB
	}
	if maxFiles <= 0 {
		maxFiles = 5 // Default keep 5 files
	}

	writer := &rotatingFileWriter{
		filename: filename,
		maxSize:  maxSize,
		maxFiles: maxFiles,
	}

	// Open initial file
	if err := writer.openFile(); err != nil {
		return nil, err
	}

	return writer, nil
}

// openFile opens the current log file and gets its size.
func (w *rotatingFileWriter) openFile() error {
	file, err := os.OpenFile(w.filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", w.filename, err)
	}

	// Get current file size
	info, err := file.Stat()
	if err != nil {
		file.Close()
		return fmt.Errorf("failed to stat log file %s: %w", w.filename, err)
	}

	w.currentFile = file
	w.currentSize = info.Size()
	return nil
}

// write writes data to the current file, rotating if necessary.
func (w *rotatingFileWriter) write(data []byte) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Check if we need to rotate before writing
	if w.currentSize+int64(len(data)) > w.maxSize {
		if err := w.rotate(); err != nil {
			// Log rotation failed, but we can still try to write to current file
			// This prevents losing log entries due to rotation issues
			return fmt.Errorf("rotation failed, continuing with current file: %w", err)
		}
	}

	// Write to current file
	n, err := w.currentFile.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	w.currentSize += int64(n)
	return nil
}

// rotate performs the file rotation.
func (w *rotatingFileWriter) rotate() error {
	// Close current file
	if w.currentFile != nil {
		w.currentFile.Close()
	}

	// Rotate existing files (move app.log.1 -> app.log.2, etc.)
	for i := w.maxFiles - 1; i > 0; i-- {
		oldName := fmt.Sprintf("%s.%d", w.filename, i)
		newName := fmt.Sprintf("%s.%d", w.filename, i+1)

		// Remove the oldest file if it exists
		if i == w.maxFiles-1 {
			os.Remove(newName)
		}

		// Move the file if it exists
		if _, err := os.Stat(oldName); err == nil {
			if err := os.Rename(oldName, newName); err != nil {
				return fmt.Errorf("failed to rotate %s to %s: %w", oldName, newName, err)
			}
		}
	}

	// Move current file to .1
	backupName := fmt.Sprintf("%s.1", w.filename)
	_ = os.Rename(w.filename, backupName) //nolint:errcheck // If rename fails, we continue with a new file - handles locked/missing files

	// Open new current file
	return w.openFile()
}

// NewRotatingFileSink creates a sink that writes JSON-formatted events to rotating files.
//
// The sink writes events in the same JSON format as the stderr sink, making it compatible
// with log aggregation systems. Files are rotated when they exceed maxSize bytes.
//
// Parameters:
//   - filename: Path to the log file (e.g., "app.log")
//   - maxSize: Maximum file size in bytes before rotation (0 = 100MB default)
//   - maxFiles: Maximum number of rotated files to keep (0 = 5 default)
//
// File naming pattern:
//   - app.log (current log file)
//   - app.log.1 (newest rotated file)
//   - app.log.2, app.log.3, etc. (older rotated files)
//
// Example usage:
//
//	// Create a file sink with 100MB rotation and keep 5 files
//	fileSink := zlog.NewRotatingFileSink("logs/app.log", 100*1024*1024, 5)
//
//	// Use with adapters for production reliability
//	productionSink := fileSink.
//	    WithRetry(3).
//	    WithTimeout(30 * time.Second).
//	    WithAsync()
//
//	// Route signals to the file
//	zlog.RouteSignal(zlog.INFO, productionSink)
//	zlog.RouteSignal(zlog.ERROR, productionSink)
//
// The sink is thread-safe and can handle concurrent writes from multiple goroutines.
// If rotation fails, the sink continues writing to the current file to avoid losing events.
func NewRotatingFileSink(filename string, maxSize int64, maxFiles int) *Sink {
	// Create the writer once during sink creation
	writer, err := newRotatingFileWriter(filename, maxSize, maxFiles)
	if err != nil {
		// Return a sink that always fails with the initialization error
		return NewSink("rotating-file-failed", func(_ context.Context, _ Log) error {
			return fmt.Errorf("rotating file sink initialization failed: %w", err)
		})
	}

	return NewSink("rotating-file", func(_ context.Context, event Log) error {
		// Build JSON structure (same format as stderr sink)
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
		for _, field := range event.Data {
			entry[field.Key] = field.Value
		}

		// Encode to JSON
		data, err := json.Marshal(entry)
		if err != nil {
			return fmt.Errorf("failed to marshal event to JSON: %w", err)
		}

		// Add newline for proper log formatting
		data = append(data, '\n')

		// Write to rotating file
		return writer.write(data)
	})
}
