# Quick Start Guide

Get up and running with zlog in 5 minutes.

## Installation

```bash
go get github.com/zoobzio/zlog
```

Requires Go 1.21+ for generics support.

## Traditional Logging (Familiar Start)

If you're coming from other loggers, start here:

```go
package main

import (
    "errors"
    "time"
    
    "github.com/zoobzio/zlog"
)

func main() {
    // Enable JSON logging to stderr
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Use familiar log levels with structured fields
    zlog.Info("Server starting", 
        zlog.Int("port", 8080),
        zlog.String("env", "production"))
    
    zlog.Debug("This won't show with INFO level")
    
    // Rich structured logging
    err := errors.New("connection timeout")
    zlog.Error("Database connection failed",
        zlog.Err(err),
        zlog.String("host", "db.example.com"),
        zlog.Duration("timeout", 30*time.Second),
        zlog.Int("retry_count", 3))
        
    zlog.Info("Server ready")
}
```

Output:
```json
{"time":"2023-10-20T15:04:05Z","signal":"INFO","message":"Server starting","caller":"main.go:12","port":8080,"env":"production"}
{"time":"2023-10-20T15:04:05Z","signal":"ERROR","message":"Database connection failed","caller":"main.go:17","error":"connection timeout","host":"db.example.com","timeout":"30s","retry_count":3}
{"time":"2023-10-20T15:04:05Z","signal":"INFO","message":"Server ready","caller":"main.go:23"}
```

## Signal-Based Routing (The zlog Way)

Now let's see the real power - routing events by their meaning:

```go
package main

import (
    "context"
    "fmt"
    
    "github.com/zoobzio/zlog"
)

// Define signals that match your business domain
const (
    USER_REGISTERED  = zlog.Signal("USER_REGISTERED")
    PAYMENT_RECEIVED = zlog.Signal("PAYMENT_RECEIVED")
    PAYMENT_FAILED   = zlog.Signal("PAYMENT_FAILED")
    FRAUD_DETECTED   = zlog.Signal("FRAUD_DETECTED")
)

func main() {
    // Enable standard logging for operational events
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Create specialized sinks for business events
    auditSink := zlog.NewSink("audit", func(ctx context.Context, e zlog.Event) error {
        fmt.Printf("AUDIT: %s - %s\n", e.Signal, e.Message)
        return nil
    })
    
    alertSink := zlog.NewSink("alerts", func(ctx context.Context, e zlog.Event) error {
        fmt.Printf("ALERT: %s - %s\n", e.Signal, e.Message)
        return nil
    })
    
    metricsSink := zlog.NewSink("metrics", func(ctx context.Context, e zlog.Event) error {
        fmt.Printf("METRIC: %s event recorded\n", e.Signal)
        return nil
    })
    
    // Route signals to appropriate destinations
    zlog.RouteSignal(USER_REGISTERED, auditSink)
    zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)
    zlog.RouteSignal(PAYMENT_RECEIVED, metricsSink)
    zlog.RouteSignal(PAYMENT_FAILED, auditSink)
    zlog.RouteSignal(PAYMENT_FAILED, alertSink)
    zlog.RouteSignal(PAYMENT_FAILED, metricsSink)
    zlog.RouteSignal(FRAUD_DETECTED, auditSink)
    zlog.RouteSignal(FRAUD_DETECTED, alertSink)
    
    // Emit business events - they automatically go to the right places
    simulateUserFlow()
}

func simulateUserFlow() {
    // User registration
    zlog.Emit(USER_REGISTERED, "New user registered",
        zlog.String("user_id", "user_123"),
        zlog.String("email", "john@example.com"),
        zlog.String("plan", "premium"))
    
    // Successful payment
    zlog.Emit(PAYMENT_RECEIVED, "Payment processed successfully",
        zlog.String("user_id", "user_123"),
        zlog.String("payment_id", "pay_456"),
        zlog.Float64("amount", 99.99),
        zlog.String("currency", "USD"))
    
    // Failed payment
    zlog.Emit(PAYMENT_FAILED, "Payment declined",
        zlog.String("user_id", "user_789"),
        zlog.String("payment_id", "pay_789"),
        zlog.Float64("amount", 199.99),
        zlog.String("reason", "insufficient_funds"))
    
    // Security event
    zlog.Emit(FRAUD_DETECTED, "Suspicious payment pattern detected",
        zlog.String("user_id", "user_999"),
        zlog.Int("attempts", 5),
        zlog.String("pattern", "rapid_succession"))
}
```

## What Just Happened?

1. **Single Emit, Multiple Destinations**: When you emit `PAYMENT_FAILED`, it automatically goes to audit (compliance), alerts (team notification), and metrics (monitoring) - all from one function call.

2. **Clear Event Meaning**: `FRAUD_DETECTED` is much clearer than `log.Warn("fraud detected")`. The signal tells you exactly what happened.

3. **Centralized Routing**: You can see all your routing rules in one place. No more hunting through code to understand where events go.

4. **Structured by Default**: Every event includes rich context through type-safe field constructors.

## Field Types

zlog provides type-safe field constructors for common data types:

```go
zlog.Emit(MY_SIGNAL, "Event with rich context",
    zlog.String("user_id", "123"),           // String values
    zlog.Int("count", 42),                   // Integer values  
    zlog.Int64("timestamp", time.Now().Unix()), // 64-bit integers
    zlog.Float64("amount", 99.99),           // Floating point
    zlog.Bool("is_premium", true),           // Booleans
    zlog.Err(err),                          // Errors (key="error")
    zlog.Duration("latency", 125*time.Millisecond), // Durations
    zlog.Time("created_at", time.Now()),     // Time values
    zlog.Strings("tags", []string{"api", "v2"}),    // String slices
    zlog.Data("metadata", map[string]string{"key": "value"})) // Arbitrary data
```

## Log Levels with Signals

You can combine traditional log levels with signal routing:

```go
// Set up standard logging with DEBUG level
zlog.EnableStandardLogging(zlog.DEBUG)

// Also route some standard signals to special destinations
alertSink := zlog.NewSink("critical-alerts", alertHandler)
zlog.RouteSignal(zlog.ERROR, alertSink)  // All errors trigger alerts
zlog.RouteSignal(zlog.FATAL, alertSink)  // Fatal events trigger alerts

// Now you get both traditional logging AND signal routing
zlog.Error("Database connection lost", zlog.Err(err))  // → stderr + alerts
zlog.Debug("Cache hit", zlog.String("key", cacheKey))  // → stderr only
```

## Next Steps

- **[Concepts: Signals](./concepts/signals.md)** - Deep dive into signal-based thinking
- **[Examples: Web Service](./examples/web-service.md)** - Real HTTP API logging patterns  
- **[API Reference](./api/core.md)** - Complete function documentation