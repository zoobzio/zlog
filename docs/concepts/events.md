# Understanding Events

Events are the core data structure in zlog. Every log entry - whether from `zlog.Info()` or `zlog.Emit()` - becomes an Event that flows through the routing system.

## Event Structure

An Event contains everything needed to understand what happened:

```go
type Event struct {
    Time    time.Time    // When the event occurred
    Signal  Signal       // What type of event this is
    Message string       // Human-readable description
    Fields  []Field      // Structured data
    Caller  CallerInfo   // Where in the code this came from
}
```

### Time

Automatically set to `time.Now()` when the event is created. This ensures consistent timing regardless of when sinks process the event.

```go
zlog.Info("User logged in")  // Time is captured immediately
```

### Signal

The event type that determines routing. Can be predefined constants or any string:

```go
zlog.Info("Server started")                    // Signal: "INFO"
zlog.Emit("PAYMENT_RECEIVED", "Payment processed")  // Signal: "PAYMENT_RECEIVED"
```

### Message

Human-readable description of what happened. Keep it concise but informative:

```go
// Good messages
zlog.Info("User authenticated successfully")
zlog.Error("Database connection timeout")
zlog.Emit(ORDER_SHIPPED, "Order shipped to customer")

// Avoid - too verbose or redundant
zlog.Info("INFO: User with ID user_123 has been successfully authenticated")
zlog.Error("ERROR: An error occurred while connecting to the database")
```

### Fields

Structured key-value data that provides context. Use the type-safe constructors:

```go
zlog.Info("API request completed",
    zlog.String("method", "POST"),
    zlog.String("path", "/api/users"),
    zlog.Int("status", 201),
    zlog.Duration("latency", 45*time.Millisecond),
    zlog.String("user_id", "user_123"))
```

### Caller Information

Automatically captured source code location:

```go
type CallerInfo struct {
    File     string  // Full path to source file
    Line     int     // Line number
    Function string  // Function name with package
}
```

Example caller info: `{"caller": "handlers/auth.go:42"}`

## Event Creation

Events are created automatically when you call logging functions:

```go
// These all create Event objects internally
zlog.Debug("Cache miss", zlog.String("key", cacheKey))
zlog.Info("Server starting", zlog.Int("port", 8080))
zlog.Error("Connection failed", zlog.Err(err))
zlog.Emit(USER_REGISTERED, "New user", zlog.String("email", email))
```

You can also create events directly (though this is rarely needed):

```go
event := zlog.NewEvent(zlog.INFO, "Direct event", []zlog.Field{
    zlog.String("source", "manual"),
})
```

## Event Immutability

Events are immutable after creation. When multiple sinks process the same event, each gets its own copy via the `Clone()` method:

```go
// Original event
event := zlog.NewEvent(zlog.INFO, "Original", nil)

// Each sink gets a clone - modifications don't affect other sinks
clone := event.Clone()
```

This prevents sinks from interfering with each other and enables safe concurrent processing.

## Working with Fields

Fields carry the structured data that makes events useful. Each field has a key, value, and type:

```go
type Field struct {
    Key   string
    Value any
    Type  FieldType
}
```

### Common Field Patterns

**User Context:**
```go
zlog.Info("Action performed",
    zlog.String("user_id", userID),
    zlog.String("username", username),
    zlog.String("role", user.Role))
```

**Request Context:**
```go
zlog.Info("Request processed",
    zlog.String("request_id", requestID),
    zlog.String("method", req.Method),
    zlog.String("path", req.URL.Path),
    zlog.String("remote_addr", req.RemoteAddr),
    zlog.Int("status", 200))
```

**Performance Metrics:**
```go
zlog.Info("Operation completed",
    zlog.Duration("duration", time.Since(start)),
    zlog.Int("items_processed", count),
    zlog.Float64("throughput", float64(count)/duration.Seconds()))
```

**Error Context:**
```go
zlog.Error("Database operation failed",
    zlog.Err(err),
    zlog.String("operation", "SELECT"),
    zlog.String("table", "users"),
    zlog.Int("retry_count", attempts))
```

### Field Best Practices

1. **Use consistent keys**: `user_id` not sometimes `userID` or `userId`
2. **Include units in names**: `timeout_seconds` not just `timeout`
3. **Avoid nested objects when possible**: Flat structures are easier to query
4. **Use appropriate types**: Don't stringify numbers or durations
5. **Include relevant context**: What would you want to know when debugging?

## Event Processing Flow

Here's how events flow through zlog:

1. **Creation**: User calls `zlog.Info()`, `zlog.Emit()`, etc.
2. **Event Construction**: `NewEvent()` creates immutable Event
3. **Caller Capture**: Runtime introspection adds caller info
4. **Routing**: Event sent to dispatch system
5. **Signal Matching**: Dispatch routes based on event signal
6. **Concurrent Processing**: Each registered sink gets a clone
7. **Sink Processing**: Sinks handle events (write, send, store, etc.)

```go
zlog.Info("User logged in", zlog.String("user_id", "123"))
    “
Event{Signal: "INFO", Message: "User logged in", Fields: [...]}
    “
Dispatch ’ Router ’ [stderrSink, metricsSink, auditSink]
    “                    “            “           “
   stderr            Prometheus    audit.log   ...
```

## Event Size Considerations

Events are copied when processed concurrently, so consider size:

```go
// Efficient - small, focused fields
zlog.Info("File uploaded",
    zlog.String("file_id", fileID),
    zlog.Int64("size_bytes", fileSize),
    zlog.String("content_type", contentType))

// Less efficient - large data in events
zlog.Info("File uploaded", 
    zlog.Data("file_content", largeFileBytes))  // Avoid large data

// Better - reference large data
zlog.Info("File uploaded",
    zlog.String("file_id", fileID),
    zlog.String("storage_path", storagePath))  // Reference, not content
```

## Custom Event Processing

You can access all event data in sinks:

```go
customSink := zlog.NewSink("custom", func(ctx context.Context, event zlog.Event) error {
    // Access all event properties
    fmt.Printf("Signal: %s\n", event.Signal)
    fmt.Printf("Time: %s\n", event.Time.Format(time.RFC3339))
    fmt.Printf("Message: %s\n", event.Message)
    fmt.Printf("Caller: %s:%d\n", event.Caller.File, event.Caller.Line)
    
    // Process fields
    for _, field := range event.Fields {
        fmt.Printf("Field %s (%s): %v\n", field.Key, field.Type, field.Value)
        
        // Type-specific handling
        switch field.Type {
        case zlog.DurationType:
            if duration, ok := field.Value.(time.Duration); ok {
                fmt.Printf("Duration in ms: %.2f\n", duration.Seconds()*1000)
            }
        case zlog.ErrorType:
            fmt.Printf("Error occurred: %s\n", field.Value)
        }
    }
    
    return nil
})
```

Events are the fundamental unit of information in zlog. Understanding their structure and lifecycle helps you design effective logging strategies and build powerful sinks that extract maximum value from your application's events.