package main

import (
	"errors"
	"os"

	"github.com/zoobzio/zlog"
)

func main() {
	// Enable standard logging to stderr
	// This includes INFO, WARN, ERROR, and FATAL signals
	zlog.EnableStandardLogging(os.Stderr)

	// Basic info log with structured field
	zlog.Info("Application starting",
		zlog.String("version", "1.0.0"),
		zlog.Int("pid", os.Getpid()),
	)

	// Simulate some application behavior
	connectDatabase()
	checkCache("user:123")
	sendEmail("user@example.com")

	zlog.Info("Application shutdown complete")
}

func connectDatabase() {
	// Log successful connection with details
	zlog.Info("Database connected",
		zlog.String("host", "localhost"),
		zlog.Int("port", 5432),
	)
}

func checkCache(key string) {
	// Log a warning - not an error, but worth noting
	zlog.Warn("Cache miss",
		zlog.String("key", key),
	)
}

func sendEmail(recipient string) {
	// Simulate an error
	err := errors.New("connection timeout")

	// Log error with context
	zlog.Error("Failed to send email",
		zlog.Err(err),
		zlog.String("recipient", recipient),
	)
}
