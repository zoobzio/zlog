// Package main demonstrates using zlog as a traditional logger.
//
// This example shows how to migrate from standard logging patterns to zlog,
// using familiar log levels while gaining the benefits of structured logging
// and flexible routing.
package main

import (
	"errors"
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/zoobzio/zlog"
)

// simulateWebServer demonstrates common logging patterns in a web application.
func simulateWebServer() {
	// Log server startup
	zlog.Info("Server starting",
		zlog.String("host", "localhost"),
		zlog.Int("port", 8080),
		zlog.String("version", "1.0.0"),
	)

	// Simulate handling requests
	for i := 0; i < 5; i++ {
		// Generate fake request data
		method := []string{"GET", "POST", "PUT", "DELETE"}[rand.Intn(4)]
		path := []string{"/api/users", "/api/products", "/api/orders", "/health"}[rand.Intn(4)]
		status := []int{200, 201, 400, 404, 500}[rand.Intn(5)]
		duration := time.Duration(rand.Intn(100)) * time.Millisecond

		// Log the request with structured fields
		if status >= 500 {
			zlog.Error("Request failed",
				zlog.String("method", method),
				zlog.String("path", path),
				zlog.Int("status", status),
				zlog.Duration("duration", duration),
				zlog.Err(errors.New("internal server error")),
			)
		} else if status >= 400 {
			zlog.Warn("Client error",
				zlog.String("method", method),
				zlog.String("path", path),
				zlog.Int("status", status),
				zlog.Duration("duration", duration),
			)
		} else {
			zlog.Info("Request handled",
				zlog.String("method", method),
				zlog.String("path", path),
				zlog.Int("status", status),
				zlog.Duration("duration", duration),
			)
		}

		time.Sleep(200 * time.Millisecond)
	}

	// Simulate debug logging (only shows in DEBUG mode)
	zlog.Debug("Cache statistics",
		zlog.Int("hits", 150),
		zlog.Int("misses", 23),
		zlog.Float64("hit_rate", 0.867),
	)
}

// simulateStartupSequence shows logging during application initialization.
func simulateStartupSequence() {
	zlog.Info("Loading configuration",
		zlog.String("config_file", "/etc/app/config.yaml"),
	)

	// Simulate database connection
	zlog.Info("Connecting to database",
		zlog.String("host", "localhost"),
		zlog.Int("port", 5432),
		zlog.String("database", "myapp"),
	)

	// Simulate a warning during startup
	zlog.Warn("Using default cache size",
		zlog.String("reason", "CACHE_SIZE environment variable not set"),
		zlog.Int("default_size_mb", 100),
	)

	// Simulate loading modules
	modules := []string{"auth", "api", "metrics", "scheduler"}
	for _, module := range modules {
		zlog.Info("Module loaded",
			zlog.String("module", module),
			zlog.Duration("load_time", time.Duration(rand.Intn(50))*time.Millisecond),
		)
	}
}

func main() {
	// Traditional logging setup - choose your log level
	// In production, you might use INFO or WARN
	// In development, use DEBUG to see everything
	fmt.Println("=== Standard Logging Example ===")
	fmt.Println("Demonstrating traditional logging patterns with zlog")
	fmt.Println()

	// Enable standard logging with INFO level
	// This routes DEBUG, INFO, WARN, ERROR, and FATAL to stderr as JSON
	zlog.EnableStandardLogging(zlog.INFO)

	// Example: In development, also route ALL events to console
	// This demonstrates how RouteAll can be used for cross-cutting concerns
	fmt.Println("--- Demonstrating RouteAll with ConsoleJSONSink ---")
	fmt.Println("Adding ConsoleJSONSink to stdout that will see ALL events")
	fmt.Println("(Standard logging still goes to stderr, dev logging to stdout)")

	// Use the new ConsoleJSONSink that outputs to stdout
	// This way we can distinguish between standard logging (stderr) and dev logging (stdout)
	zlog.RouteAll(zlog.ConsoleJSONSink(true))

	// Show application startup sequence
	fmt.Println("--- Application Startup ---")
	simulateStartupSequence()

	fmt.Println("\n--- Web Server Simulation ---")
	simulateWebServer()

	// Demonstrate different log levels
	fmt.Println("\n--- Log Level Examples ---")

	// Debug - detailed information for troubleshooting
	zlog.Debug("This debug message won't show at INFO level")

	// Info - general operational messages
	zlog.Info("Application initialized successfully",
		zlog.Duration("startup_time", 250*time.Millisecond),
	)

	// Warn - something concerning but recoverable
	zlog.Warn("API rate limit approaching",
		zlog.Int("requests_remaining", 100),
		zlog.Duration("reset_in", 15*time.Minute),
	)

	// Error - something failed but app continues
	err := http.ErrHandlerTimeout
	zlog.Error("Failed to process webhook",
		zlog.Err(err),
		zlog.String("webhook_url", "https://example.com/hook"),
		zlog.Int("retry_count", 3),
	)

	// Fatal would exit the application - commented out for demo
	// zlog.Fatal("Unrecoverable error", zlog.Err(errors.New("database connection lost")))

	// Demonstrate custom signals that ONLY appear via RouteAll
	fmt.Println("\n--- Custom Signal Examples ---")
	fmt.Println("These custom signals will ONLY appear in stdout (ConsoleJSONSink)")

	// Custom business signals
	const PAYMENT_RECEIVED = zlog.Signal("PAYMENT_RECEIVED")
	const USER_REGISTERED = zlog.Signal("USER_REGISTERED")

	zlog.Emit(PAYMENT_RECEIVED, "Payment processed successfully",
		zlog.String("user_id", "usr_123"),
		zlog.Float64("amount", 99.99),
		zlog.String("currency", "USD"),
	)

	zlog.Emit(USER_REGISTERED, "New user registration",
		zlog.String("user_id", "usr_456"),
		zlog.String("email", "newuser@example.com"),
		zlog.String("plan", "premium"),
	)

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("Check the JSON output above to see structured logging in action!")
	fmt.Println("Notice: Custom signals only appear in stdout (RouteAll), not stderr (StandardLogging)")
}
