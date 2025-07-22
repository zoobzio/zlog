package main

import (
	"os"
	"time"

	"github.com/zoobzio/zlog"
)

func main() {
	// Standard logs go to stderr (console)
	zlog.EnableStandardLogging(os.Stderr)

	// Debug logs go to a separate file
	debugFile, err := os.Create("debug.log")
	if err != nil {
		zlog.Fatal("Failed to create debug file", zlog.Err(err))
	}
	defer debugFile.Close()

	zlog.EnableDebugLogging(debugFile)

	// Now let's see the separation in action
	runApplication()
}

func runApplication() {
	// This goes to stderr (console)
	zlog.Info("Server starting", zlog.Int("port", 8080))

	// These go to debug.log file only
	zlog.Debug("Configuration loaded",
		zlog.String("config_path", "/etc/app/config.json"),
	)
	zlog.Debug("Database connection pool initialized",
		zlog.Int("pool_size", 10),
		zlog.Int("max_idle", 5),
	)

	// Simulate handling a request
	handleRequest("GET", "/api/users")

	// This goes to stderr
	zlog.Info("Server shutdown initiated")
}

func handleRequest(method, path string) {
	// Info logs go to console
	zlog.Info("Request received",
		zlog.String("method", method),
		zlog.String("path", path),
	)

	// Debug logs go to file - implementation details
	query := "SELECT * FROM users WHERE active = true"
	zlog.Debug("Query execution",
		zlog.String("sql", query),
		zlog.Int("rows", 42),
	)

	// Simulate slow query warning - goes to console
	duration := 1500 * time.Millisecond
	zlog.Warn("Slow query",
		zlog.Duration("duration", duration),
		zlog.Int("duration_ms", 1500),
	)
}

func init() {
	// Clean up any existing debug.log from previous runs
	os.Remove("debug.log")
}
