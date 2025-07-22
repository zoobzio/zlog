# zlog

Signal-based logging for Go. Move beyond traditional log levels to a flexible signal system where any string can trigger custom processing pipelines.

## Quick Start

```go
package main

import (
    "os"
    "github.com/zoobzio/zlog"
)

func main() {
    // Enable standard logging (INFO, WARN, ERROR, FATAL)
    zlog.EnableStandardLogging(os.Stderr)
    
    // Your application logs
    zlog.Info("Server starting", zlog.Int("port", 8080))
    zlog.Debug("This won't show - debug not enabled")
    
    // Something goes wrong
    zlog.Error("Database connection failed", 
        zlog.String("host", "localhost"),
        zlog.Err(err),
    )
}
```

## Why zlog?

### The Problem

Traditional loggers force you into rigid level hierarchies (DEBUG < INFO < WARN < ERROR). But real applications have diverse logging needs:
- Security events need different handling than debug logs
- Audit trails require guaranteed delivery and special formatting
- Business metrics shouldn't mix with application errors
- Different environments need different log routing

### The Solution  

zlog treats logs as **signals** - any string can be a signal type that routes to specific handlers:
- **Signal-based routing**: Route AUDIT logs to compliance storage, METRIC to monitoring systems
- **Self-configuring sinks**: Handlers register for the signals they care about
- **Environment flexibility**: Different signal routes for development vs production
- **Concurrent processing**: Fast, thread-safe sink execution

## Installation

```bash
go get github.com/zoobzio/zlog
```

Requirements: Go 1.21+ (for generics)

## Real-World Scenarios

### 1. Audit Trail for Compliance

**Problem**: Financial services need tamper-proof audit logs separate from application logs, with guaranteed delivery to compliance systems.

```go
// Traditional approach - audit logs mixed with everything else
logger.Info("User logged in", "user", "alice")
logger.Info("Permission granted", "user", "alice", "permission", "admin")  // This is audit!
logger.Debug("Cache miss", "key", "user:alice")
logger.Info("Request completed", "duration", "45ms")

// How do you extract just audit events from this mess?
```

**Solution**: Route audit signals to dedicated append-only storage:

```go
// Setup separate audit trail
auditFile, _ := os.OpenFile("/secure/audit.log", 
    os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
zlog.EnableAuditLogging(auditFile)

// Application logs go to stderr
zlog.EnableStandardLogging(os.Stderr)

// Audit events are automatically separated
zlog.Info("User logged in", zlog.String("user", "alice"))     // → stderr
zlog.Audit("Permission granted",                               // → audit.log
    zlog.String("user", "alice"),
    zlog.String("permission", "admin"),
    zlog.String("granted_by", "system"),
)

// Security events also go to audit trail
zlog.Security("Failed login attempt",                          // → audit.log
    zlog.String("ip", request.RemoteAddr),
    zlog.Int("attempt", 3),
)
```

### 2. Development vs Production Logging

**Problem**: Developers need verbose debugging that would overwhelm production systems. Production needs structured logs for monitoring without the noise.

```go
// Traditional approach - complex configuration files
if config.LogLevel == "debug" && config.Environment == "development" {
    logger.SetLevel(DEBUG)
} else if config.Environment == "production" {
    logger.SetLevel(INFO) 
}

// But what about:
// - Debug logs for just one module?
// - Critical alerts that need different handling?
// - Performance metrics that shouldn't go to stderr?
```

**Solution**: Environment-specific signal routing:

```go
if os.Getenv("ENV") == "production" {
    // Production: JSON to stdout, no debug logs
    zlog.EnableStandardLogging(os.Stdout)
    
    // Critical errors to alerting system
    const ALERT zlog.Signal = "ALERT"
    alertFile, _ := os.OpenFile("/var/log/alerts.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    zlog.RouteSignal(ALERT, zlog.NewWriterSink(alertFile))
    
} else {
    // Development: Pretty print with debug enabled
    zlog.EnableStandardLogging(os.Stderr)
    zlog.EnableDebugLogging(os.Stderr)
    
    // Slow queries to separate file for analysis
    const SLOW_QUERY zlog.Signal = "SLOW_QUERY"
    slowFile, _ := os.Create("slow-queries.log")
    zlog.RouteSignal(SLOW_QUERY, zlog.NewWriterSink(slowFile))
}

// Same code works in both environments
zlog.Debug("Cache lookup", zlog.String("key", key))           // Only in dev
zlog.Info("Request handled", zlog.Duration("latency", took))  // Both environments

if took > 5*time.Second {
    zlog.Emit(ALERT, "Request too slow",                      // Pages in prod
        zlog.Duration("latency", took),
        zlog.String("endpoint", r.URL.Path),
    )
}
```

### 3. Distributed System Correlation

**Problem**: Microservices make debugging difficult - a single request touches multiple services and finding all related logs is painful.

```go
// Traditional approach - logs scattered everywhere
// api-gateway.log:
// 2024-01-20 10:30:45 INFO Request received /api/orders

// user-service.log:
// 2024-01-20 10:30:45 INFO Fetching user 12345
// 2024-01-20 10:30:46 ERROR Database timeout

// order-service.log:  
// 2024-01-20 10:30:47 INFO Creating order for user 12345

// Which logs belong to the same request? Good luck figuring that out!
```

**Solution**: Trace propagation with structured fields:

```go
// API Gateway starts the trace
traceID := generateTraceID()
ctx = context.WithValue(ctx, "trace_id", traceID)

zlog.Info("Request received",
    zlog.String("trace_id", traceID),
    zlog.String("service", "api-gateway"),
    zlog.String("endpoint", "/api/orders"),
)

// User Service (automatic trace propagation)
func GetUser(ctx context.Context, userID string) (*User, error) {
    traceID := ctx.Value("trace_id").(string)
    
    zlog.Info("Fetching user",
        zlog.String("trace_id", traceID),
        zlog.String("service", "user-service"),
        zlog.String("user_id", userID),
    )
    
    user, err := db.GetUser(userID)
    if err != nil {
        zlog.Error("User fetch failed",
            zlog.String("trace_id", traceID),
            zlog.String("service", "user-service"), 
            zlog.Err(err),
        )
        return nil, err
    }
    
    return user, nil
}

// Find all logs across services:
// grep "trace_123" *.log | jq -s 'sort_by(.time)'
```

## Core Concepts

### Signals, Not Levels

Traditional loggers use levels. zlog uses signals:

```go
// Built-in signals
zlog.Debug("Detailed trace")          // DEBUG signal
zlog.Info("Normal operation")         // INFO signal  
zlog.Warn("Potential issue")          // WARN signal
zlog.Error("Operation failed")        // ERROR signal

// Pre-defined domain signals
zlog.Audit("Compliance event")        // AUDIT signal
zlog.Security("Security event")       // SECURITY signal
zlog.Metric("Performance data")       // METRIC signal

// Custom signals
const PAYMENT zlog.Signal = "PAYMENT"
const FRAUD zlog.Signal = "FRAUD"
const SLOW_QUERY zlog.Signal = "SLOW_QUERY"
```

### Self-Registering Sinks

Sinks automatically register for their signals:

```go
type ErrorCountingSink struct {
    writer io.Writer
    errors int64
}

func (s *ErrorCountingSink) Write(event zlog.Event) error {
    atomic.AddInt64(&s.errors, 1)
    // Also write to file/stdout/etc
    return zlog.NewWriterSink(s.writer).Write(event)
}

func (s *ErrorCountingSink) Name() string {
    return "error_counter"
}

// Constructor self-registers for error signals
func NewErrorCountingSink(w io.Writer) *ErrorCountingSink {
    sink := &ErrorCountingSink{writer: w}
    
    // Self-register for signals we care about
    zlog.RouteSignal(zlog.ERROR, sink)
    zlog.RouteSignal(zlog.FATAL, sink)
    
    return sink
}

// Just create it - registration happens automatically
errorCounter := NewErrorCountingSink(os.Stderr)
```

### Structured Fields

Type-safe field construction:

```go
zlog.Info("Order processed",
    zlog.String("order_id", order.ID),
    zlog.String("customer_id", order.CustomerID),  
    zlog.Float64("amount", order.Total),
    zlog.Int("items", len(order.Items)),
    zlog.Duration("processing_time", elapsed),
    zlog.Time("completed_at", time.Now()),
    zlog.Err(err), // nil-safe
)
```

## Performance

- Zero-allocation field construction
- Lock-free signal routing via sync.Map
- Concurrent sink processing through pipz
- No reflection in the hot path

## Documentation

📚 **[Full Documentation](./docs/)**

- [Examples](./examples/) - 10 real-world scenarios with tests
- [API Reference](./docs/api.md) - Complete API documentation
- [Custom Sinks](./docs/sinks.md) - Building your own sinks
- [Best Practices](./docs/best-practices.md) - Production patterns


## License

MIT