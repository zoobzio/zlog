# Field Constructors Reference

This document covers all the field constructor functions for adding structured data to log events.

## Basic Field Types

### String

```go
func String(key, value string) Field
```

Creates a string field.

**Example:**
```go
zlog.Info("User action",
    zlog.String("user_id", "user_123"),
    zlog.String("action", "login"),
    zlog.String("ip_address", "192.168.1.100"))
```

### Int

```go
func Int(key string, value int) Field
```

Creates an integer field.

**Example:**
```go
zlog.Info("API request",
    zlog.String("method", "GET"),
    zlog.String("path", "/api/users"),
    zlog.Int("status_code", 200),
    zlog.Int("response_size", 1024))
```

### Int64

```go
func Int64(key string, value int64) Field
```

Creates a 64-bit integer field.

**Example:**
```go
zlog.Info("File processed",
    zlog.String("file_name", "data.csv"),
    zlog.Int64("file_size", 1073741824), // 1GB
    zlog.Int64("records_processed", 1000000))
```

### Float64

```go
func Float64(key string, value float64) Field
```

Creates a floating-point field.

**Example:**
```go
zlog.Info("Payment processed",
    zlog.String("payment_id", "pay_123"),
    zlog.Float64("amount", 99.99),
    zlog.Float64("tax", 8.50),
    zlog.String("currency", "USD"))
```

### Bool

```go
func Bool(key string, value bool) Field
```

Creates a boolean field.

**Example:**
```go
zlog.Info("User registration",
    zlog.String("user_id", "user_123"),
    zlog.Bool("email_verified", false),
    zlog.Bool("newsletter_subscribed", true),
    zlog.Bool("terms_accepted", true))
```

## Time and Duration Fields

### Duration

```go
func Duration(key string, value time.Duration) Field
```

Creates a duration field. The duration is stored as nanoseconds but rendered appropriately.

**Example:**
```go
start := time.Now()
// ... do work ...
zlog.Info("Operation completed",
    zlog.String("operation", "database_backup"),
    zlog.Duration("elapsed_time", time.Since(start)),
    zlog.Duration("timeout", 30*time.Second))
```

### Time

```go
func Time(key string, value time.Time) Field
```

Creates a timestamp field.

**Example:**
```go
zlog.Info("Event scheduled",
    zlog.String("event_id", "evt_123"),
    zlog.Time("scheduled_at", scheduledTime),
    zlog.Time("created_at", time.Now()))
```

## Error Handling

### Err

```go
func Err(err error) Field
```

Creates an error field with the key "error". This is the standard way to include errors in log events.

**Example:**
```go
file, err := os.Open("config.yaml")
if err != nil {
    zlog.Error("Failed to open configuration file",
        zlog.Err(err),
        zlog.String("file_path", "config.yaml"),
        zlog.String("operation", "read_config"))
    return err
}
```

### NamedErr

```go
func NamedErr(key string, err error) Field
```

Creates an error field with a custom key name. Useful when logging multiple errors.

**Example:**
```go
primaryErr := connectToPrimary()
fallbackErr := connectToFallback()

if primaryErr != nil && fallbackErr != nil {
    zlog.Error("All database connections failed",
        zlog.NamedErr("primary_error", primaryErr),
        zlog.NamedErr("fallback_error", fallbackErr))
}
```

## Complex Data Types

### Any

```go
func Any(key string, value interface{}) Field
```

Creates a field that can hold any type. The value will be serialized appropriately (usually as JSON for complex types).

**Example:**
```go
type UserPreferences struct {
    Theme    string `json:"theme"`
    Language string `json:"language"`
    Timezone string `json:"timezone"`
}

prefs := UserPreferences{
    Theme:    "dark",
    Language: "en",
    Timezone: "UTC",
}

zlog.Info("User preferences updated",
    zlog.String("user_id", "user_123"),
    zlog.Any("preferences", prefs))
```

**  Performance Note:** `Any()` is more expensive than typed field constructors. Use specific types when possible.

### Data

```go
func Data(key string, value []byte) Field
```

Creates a field for binary data. Data is base64-encoded when serialized.

**Example:**
```go
fileContent, err := os.ReadFile("image.png")
if err != nil {
    zlog.Error("Failed to read file", zlog.Err(err))
    return
}

zlog.Info("File uploaded",
    zlog.String("file_name", "image.png"),
    zlog.String("content_type", "image/png"),
    zlog.Int("size", len(fileContent)),
    zlog.Data("checksum", computeSHA256(fileContent)))
```

## Field Best Practices

### Consistent Naming

Use consistent field names across your application:

```go
// Good - consistent naming
const (
    FieldUserID    = "user_id"
    FieldRequestID = "request_id"
    FieldOperation = "operation"
    FieldDuration  = "duration"
)

zlog.Info("Operation completed",
    zlog.String(FieldUserID, userID),
    zlog.String(FieldRequestID, reqID),
    zlog.String(FieldOperation, "create_user"),
    zlog.Duration(FieldDuration, elapsed))

// Avoid - inconsistent naming
zlog.Info("Operation completed",
    zlog.String("userId", userID),        // camelCase
    zlog.String("request-id", reqID),     // kebab-case
    zlog.String("Operation", "create"),   // PascalCase
    zlog.Duration("time", elapsed))       // different meaning
```

### Appropriate Types

Use the most specific type available:

```go
// Good - specific types
zlog.Info("HTTP request",
    zlog.String("method", "POST"),           // String for text
    zlog.Int("status_code", 201),            // Int for numbers
    zlog.Duration("latency", 45*time.Millisecond), // Duration for time spans
    zlog.Bool("cached", false))              // Bool for true/false

// Avoid - using strings for everything
zlog.Info("HTTP request",
    zlog.String("method", "POST"),
    zlog.String("status_code", "201"),       // Should be Int
    zlog.String("latency", "45ms"),          // Should be Duration
    zlog.String("cached", "false"))         // Should be Bool
```

### Context Fields

Create helper functions for common field combinations:

```go
// Helper for request context
func requestFields(r *http.Request) []zlog.Field {
    return []zlog.Field{
        zlog.String("method", r.Method),
        zlog.String("path", r.URL.Path),
        zlog.String("remote_addr", r.RemoteAddr),
        zlog.String("user_agent", r.UserAgent()),
    }
}

// Helper for user context
func userFields(user *User) []zlog.Field {
    return []zlog.Field{
        zlog.String("user_id", user.ID),
        zlog.String("username", user.Username),
        zlog.String("role", user.Role),
    }
}

// Usage
zlog.Info("User action performed",
    append(requestFields(r),
        append(userFields(user),
            zlog.String("action", "update_profile"))...)...)
```

### Performance Considerations

1. **Reuse field objects**: Field constructors are fast, but you can reuse them in hot paths:

```go
// For very hot paths, you can pre-allocate
var serviceField = zlog.String("service", "payment-processor")
var versionField = zlog.String("version", "1.2.3")

func logPayment(amount float64) {
    zlog.Info("Payment processed",
        serviceField,  // Reused
        versionField,  // Reused  
        zlog.Float64("amount", amount)) // Created fresh
}
```

2. **Avoid expensive computations**: Don't compute expensive values unless the event will be logged:

```go
// Good - conditional expensive computation
if shouldLogDebug() {
    expensiveData := generateExpensiveDebugInfo()
    zlog.Debug("Debug information", zlog.Any("data", expensiveData))
}

// Avoid - always computing expensive data
expensiveData := generateExpensiveDebugInfo() // Always computed
zlog.Debug("Debug information", zlog.Any("data", expensiveData))
```

3. **Field count**: While there's no hard limit, keep field counts reasonable for performance:

```go
// Reasonable - focused, relevant fields
zlog.Info("API request",
    zlog.String("method", "POST"),
    zlog.String("path", "/api/users"),
    zlog.Int("status", 201),
    zlog.Duration("latency", elapsed),
    zlog.String("user_id", userID))

// Excessive - too many fields can hurt performance
zlog.Info("API request", /* 50+ fields... */)
```

## Field Serialization

Fields are serialized differently depending on the sink:

- **JSON sinks**: Native JSON types (string, number, boolean, null)
- **Text sinks**: String representation with appropriate formatting
- **Binary sinks**: Efficient binary encoding

Example JSON output:
```json
{
  "time": "2023-10-20T14:30:05.123Z",
  "signal": "INFO",
  "message": "Payment processed",
  "caller": "payment.go:123",
  "payment_id": "pay_123",
  "amount": 99.99,
  "currency": "USD",
  "success": true,
  "processing_time": "145ms"
}
```

For more information on field usage in practice, see the [Best Practices Guide](../guides/best-practices.md).