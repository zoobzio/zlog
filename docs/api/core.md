# Core API Reference

This document covers the core functions of zlog's API for emitting events and managing logging.

## Event Emission Functions

### Emit

```go
func Emit(signal Signal, message string, fields ...Field)
```

Emit creates and dispatches an event with a custom signal.

**Parameters:**
- `signal`: The signal type for this event (e.g., "USER_REGISTERED", "PAYMENT_FAILED")
- `message`: Human-readable description of what happened
- `fields`: Zero or more structured fields to include with the event

**Example:**
```go
zlog.Emit("USER_REGISTERED", "New user account created",
    zlog.String("user_id", "user_123"),
    zlog.String("email", "alice@example.com"),
    zlog.String("registration_method", "email"))
```

### Debug

```go
func Debug(message string, fields ...Field)
```

Debug emits a DEBUG level event. Use for detailed diagnostic information.

**Parameters:**
- `message`: Debug message
- `fields`: Additional structured data

**Example:**
```go
zlog.Debug("Cache lookup performed",
    zlog.String("cache_key", "user:123"),
    zlog.Bool("cache_hit", false),
    zlog.Duration("lookup_time", 2*time.Millisecond))
```

### Info

```go
func Info(message string, fields ...Field)
```

Info emits an INFO level event for general operational information.

**Parameters:**
- `message`: Informational message
- `fields`: Additional structured data

**Example:**
```go
zlog.Info("Server started successfully",
    zlog.Int("port", 8080),
    zlog.String("version", "1.2.3"),
    zlog.Duration("startup_time", time.Since(start)))
```

### Warn

```go
func Warn(message string, fields ...Field)
```

Warn emits a WARN level event for potentially problematic situations.

**Parameters:**
- `message`: Warning message
- `fields`: Additional structured data

**Example:**
```go
zlog.Warn("High memory usage detected",
    zlog.Float64("memory_usage_percent", 85.2),
    zlog.String("component", "image_processor"))
```

### Error

```go
func Error(message string, fields ...Field)
```

Error emits an ERROR level event for error conditions that don't halt execution.

**Parameters:**
- `message`: Error description
- `fields`: Additional structured data (often including the error itself)

**Example:**
```go
zlog.Error("Database connection failed",
    zlog.Err(err),
    zlog.String("database", "users"),
    zlog.Int("retry_attempt", 3))
```

### Fatal

```go
func Fatal(message string, fields ...Field)
```

Fatal emits a FATAL level event and then calls `os.Exit(1)`. Use only for unrecoverable errors.

**Parameters:**
- `message`: Fatal error description
- `fields`: Additional structured data

**Example:**
```go
zlog.Fatal("Unable to load configuration",
    zlog.String("config_file", "/etc/app/config.yaml"),
    zlog.Err(err))
// Program exits here
```

**  Warning:** Fatal() calls `os.Exit(1)` and will terminate your program immediately. Use sparingly and only for truly unrecoverable situations.

## Signal Management

### RouteSignal

```go
func RouteSignal(signal Signal, sink Sink)
```

RouteSignal registers a sink to receive events with the specified signal.

**Parameters:**
- `signal`: The signal to route (e.g., "USER_REGISTERED", zlog.ERROR)
- `sink`: The sink that will process events with this signal

**Example:**
```go
// Create a custom sink
auditSink := zlog.NewSink("audit", func(ctx context.Context, event zlog.Event) error {
    return writeToAuditLog(event)
})

// Route custom business events to audit
zlog.RouteSignal("USER_REGISTERED", auditSink)
zlog.RouteSignal("PAYMENT_PROCESSED", auditSink)
zlog.RouteSignal("ADMIN_ACTION", auditSink)

// Route error events to alerting
alertSink := zlog.NewSink("alerts", func(ctx context.Context, event zlog.Event) error {
    return sendAlert(event)
})
zlog.RouteSignal(zlog.ERROR, alertSink)
zlog.RouteSignal(zlog.FATAL, alertSink)
```

**Notes:**
- Multiple sinks can be registered for the same signal
- When multiple sinks are registered, events are processed concurrently
- Registration is typically done during application startup

## Sink Creation

### NewSink

```go
func NewSink(name string, handler func(context.Context, Event) error) Sink
```

NewSink creates a new sink that processes events with the provided handler function.

**Parameters:**
- `name`: A descriptive name for the sink (used in debugging and metrics)
- `handler`: Function that processes events. Should return an error if processing fails.

**Example:**
```go
// File logging sink
fileSink := zlog.NewSink("file-logger", func(ctx context.Context, event zlog.Event) error {
    logEntry := fmt.Sprintf("%s [%s] %s\n", 
        event.Time.Format(time.RFC3339), event.Signal, event.Message)
    
    file, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer file.Close()
    
    _, err = file.WriteString(logEntry)
    return err
})

// HTTP sink for remote logging
httpSink := zlog.NewSink("http-logger", func(ctx context.Context, event zlog.Event) error {
    data, err := json.Marshal(event)
    if err != nil {
        return err
    }
    
    resp, err := http.Post("https://logs.example.com/events", 
        "application/json", bytes.NewReader(data))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
    }
    return nil
})
```

**Handler Function Guidelines:**
- Should be efficient as it may be called frequently
- Should handle context cancellation gracefully
- Should return meaningful errors for debugging
- Should not panic (will crash the application)
- Can perform I/O operations (network, file, database)

## Event Processing Flow

When you call any logging function, zlog follows this process:

1. **Event Creation**: Creates an `Event` struct with timestamp, signal, message, fields, and caller info
2. **Signal Lookup**: Finds all sinks registered for the event's signal
3. **Concurrent Dispatch**: If multiple sinks are registered, creates goroutines for concurrent processing
4. **Sink Processing**: Each sink processes its own copy of the event
5. **Error Handling**: Sink errors are logged but don't affect other sinks or your application

```go
// This call...
zlog.Error("Database error", zlog.Err(err), zlog.String("table", "users"))

// Results in this flow:
// 1. Create Event{Signal: "ERROR", Message: "Database error", Fields: [...], Time: now, Caller: {...}}
// 2. Look up sinks for "ERROR" signal ’ [stderrSink, alertSink, metricsSink]
// 3. Concurrently call:
//    - stderrSink.Process(ctx, event.Clone())
//    - alertSink.Process(ctx, event.Clone())
//    - metricsSink.Process(ctx, event.Clone())
// 4. Each sink processes independently
```

## Performance Characteristics

- **Single sink**: Direct function call (~100ns)
- **Multiple sinks**: Concurrent processing (~500ns + sink processing time)
- **Event creation**: Minimal allocations using object pools
- **Field creation**: Type-safe, no boxing for primitive types
- **Routing**: Lock-free map lookup on hot path

See the [Performance Guide](../guides/performance.md) for detailed performance information and optimization strategies.