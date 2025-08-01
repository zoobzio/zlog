# Best Practices

This guide covers recommended patterns and approaches for using zlog effectively in your applications.

## Signal Design

### Signal Naming Convention

Use consistent, descriptive signal names:

```go
// Good - consistent, descriptive
const (
    USER_REGISTERED   = "USER_REGISTERED"
    USER_DEACTIVATED  = "USER_DEACTIVATED"
    PAYMENT_RECEIVED  = "PAYMENT_RECEIVED"
    PAYMENT_FAILED    = "PAYMENT_FAILED"
    ORDER_SHIPPED     = "ORDER_SHIPPED"
    ORDER_CANCELLED   = "ORDER_CANCELLED"
)

// Avoid - inconsistent, unclear
const (
    user_login        = "userLogin"
    payment_ok        = "payment-ok"
    OrderDone         = "order_done"
    ERR               = "err"
)
```

### Signal Hierarchies

Organize signals by domain and action:

```go
// Domain-based organization
const (
    // User domain
    USER_REGISTERED    = "USER_REGISTERED"
    USER_AUTHENTICATED = "USER_AUTHENTICATED"
    USER_UPDATED       = "USER_UPDATED"
    USER_DELETED       = "USER_DELETED"
    
    // Payment domain
    PAYMENT_INITIATED  = "PAYMENT_INITIATED"
    PAYMENT_AUTHORIZED = "PAYMENT_AUTHORIZED"
    PAYMENT_CAPTURED   = "PAYMENT_CAPTURED"
    PAYMENT_FAILED     = "PAYMENT_FAILED"
    PAYMENT_REFUNDED   = "PAYMENT_REFUNDED"
    
    // Security domain
    SECURITY_LOGIN_FAILED     = "SECURITY_LOGIN_FAILED"
    SECURITY_ACCESS_DENIED    = "SECURITY_ACCESS_DENIED"
    SECURITY_PRIVILEGE_CHANGE = "SECURITY_PRIVILEGE_CHANGE"
)
```

### Semantic Events vs Log Levels

Design signals around business meaning, not severity:

```go
// Good - semantic business events
zlog.Emit(PAYMENT_FAILED, "Payment declined by processor",
    zlog.String("payment_id", paymentID),
    zlog.String("reason", "insufficient_funds"),
    zlog.Float64("amount", amount))

zlog.Emit(USER_SUSPICIOUS_ACTIVITY, "Multiple failed login attempts",
    zlog.String("user_id", userID),
    zlog.Int("attempt_count", attempts),
    zlog.Duration("time_window", timeWindow))

// Less ideal - severity-focused
zlog.Error("Payment error",
    zlog.String("error", "declined"))

zlog.Warn("Suspicious activity",
    zlog.String("type", "login_failures"))
```

## Event Structure

### Field Consistency

Use consistent field names across your application:

```go
// Establish field conventions
const (
    FieldUserID      = "user_id"
    FieldRequestID   = "request_id"
    FieldSessionID   = "session_id"
    FieldOperation   = "operation"
    FieldDuration    = "duration_ms"
    FieldAmount      = "amount"
    FieldCurrency    = "currency"
    FieldIPAddress   = "ip_address"
    FieldUserAgent   = "user_agent"
)

// Use consistently throughout codebase
zlog.Emit(USER_REGISTERED, "User registration completed",
    zlog.String(FieldUserID, user.ID),
    zlog.String(FieldRequestID, requestID),
    zlog.String(FieldIPAddress, clientIP))

zlog.Emit(PAYMENT_RECEIVED, "Payment processed successfully",
    zlog.String(FieldUserID, user.ID),
    zlog.String(FieldRequestID, requestID),
    zlog.Float64(FieldAmount, payment.Amount),
    zlog.String(FieldCurrency, payment.Currency))
```

### Context Propagation

Propagate important context through your request flow:

```go
// Extract context fields helper
func contextFields(ctx context.Context) []zlog.Field {
    var fields []zlog.Field
    
    if requestID := getRequestID(ctx); requestID != "" {
        fields = append(fields, zlog.String(FieldRequestID, requestID))
    }
    
    if userID := getUserID(ctx); userID != "" {
        fields = append(fields, zlog.String(FieldUserID, userID))
    }
    
    if sessionID := getSessionID(ctx); sessionID != "" {
        fields = append(fields, zlog.String(FieldSessionID, sessionID))
    }
    
    return fields
}

// Use context fields consistently
func processPayment(ctx context.Context, payment *Payment) error {
    start := time.Now()
    
    zlog.Emit(PAYMENT_INITIATED, "Payment processing started",
        append(contextFields(ctx),
            zlog.String("payment_id", payment.ID),
            zlog.Float64(FieldAmount, payment.Amount))...)
    
    // Process payment...
    
    if err != nil {
        zlog.Emit(PAYMENT_FAILED, "Payment processing failed",
            append(contextFields(ctx),
                zlog.String("payment_id", payment.ID),
                zlog.Err(err),
                zlog.Duration(FieldDuration, time.Since(start)))...)
        return err
    }
    
    zlog.Emit(PAYMENT_RECEIVED, "Payment processed successfully",
        append(contextFields(ctx),
            zlog.String("payment_id", payment.ID),
            zlog.Duration(FieldDuration, time.Since(start)))...)
    
    return nil
}
```

### Structured Error Information

Provide rich context for errors:

```go
// Good - rich error context
func connectToDatabase(config DatabaseConfig) error {
    start := time.Now()
    
    conn, err := sql.Open(config.Driver, config.URL)
    if err != nil {
        zlog.Emit(DATABASE_CONNECTION_FAILED, "Failed to open database connection",
            zlog.Err(err),
            zlog.String("driver", config.Driver),
            zlog.String("host", config.Host),
            zlog.Int("port", config.Port),
            zlog.Duration("attempt_duration", time.Since(start)),
            zlog.String("database", config.Database))
        return err
    }
    
    if err := conn.Ping(); err != nil {
        zlog.Emit(DATABASE_PING_FAILED, "Database ping failed",
            zlog.Err(err),
            zlog.String("driver", config.Driver),
            zlog.String("host", config.Host),
            zlog.Duration("ping_duration", time.Since(start)))
        return err
    }
    
    zlog.Emit(DATABASE_CONNECTED, "Database connection established",
        zlog.String("driver", config.Driver),
        zlog.String("host", config.Host),
        zlog.Duration("connection_time", time.Since(start)))
    
    return nil
}

// Avoid - minimal context
func connectToDatabase(config DatabaseConfig) error {
    conn, err := sql.Open(config.Driver, config.URL)
    if err != nil {
        zlog.Error("Database connection failed", zlog.Err(err))
        return err
    }
    
    zlog.Info("Connected to database")
    return nil
}
```

## Routing Architecture

### Layered Routing Strategy

Design routing in layers based on purpose:

```go
func setupLogging() {
    // Layer 1: Operational logging (for developers/ops)
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Layer 2: Business monitoring (for product/business)
    setupBusinessMetrics()
    
    // Layer 3: Security monitoring (for security team)
    setupSecurityMonitoring()
    
    // Layer 4: Compliance/audit (for compliance)
    setupAuditLogging()
    
    // Layer 5: Analytics (for data team)
    setupAnalyticsEvents()
}

func setupBusinessMetrics() {
    metricsSink := createMetricsSink()
    
    // Route business events to metrics
    zlog.RouteSignal(USER_REGISTERED, metricsSink)
    zlog.RouteSignal(PAYMENT_RECEIVED, metricsSink)
    zlog.RouteSignal(ORDER_COMPLETED, metricsSink)
    zlog.RouteSignal(SUBSCRIPTION_STARTED, metricsSink)
}

func setupSecurityMonitoring() {
    securitySink := createSecuritySink()
    
    // Route security events
    zlog.RouteSignal(SECURITY_LOGIN_FAILED, securitySink)
    zlog.RouteSignal(SECURITY_ACCESS_DENIED, securitySink)
    zlog.RouteSignal(SECURITY_PRIVILEGE_CHANGE, securitySink)
    zlog.RouteSignal(SECURITY_SUSPICIOUS_ACTIVITY, securitySink)
}
```

### Environment-Specific Routing

Configure routing based on environment:

```go
func setupLogging(env string) {
    switch env {
    case "development":
        setupDevelopmentLogging()
    case "staging":
        setupStagingLogging()
    case "production":
        setupProductionLogging()
    default:
        setupDevelopmentLogging()
    }
}

func setupDevelopmentLogging() {
    // Development: Pretty console output, all levels
    zlog.EnableStandardLogging(zlog.DEBUG)
    
    // Debug sink for immediate feedback
    debugSink := createConsoleSink()
    routeAllCustomSignals(debugSink)
}

func setupProductionLogging() {
    // Production: Conservative level, multiple destinations
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Business monitoring
    if err := setupMetrics(); err != nil {
        log.Fatalf("Failed to setup metrics: %v", err)
    }
    
    // Security monitoring
    if err := setupSecuritySIEM(); err != nil {
        log.Fatalf("Failed to setup security monitoring: %v", err)
    }
    
    // Audit trail
    if err := setupAuditDatabase(); err != nil {
        log.Fatalf("Failed to setup audit logging: %v", err)
    }
    
    // Error alerting
    if err := setupErrorAlerts(); err != nil {
        log.Fatalf("Failed to setup error alerts: %v", err)
    }
}
```

## Performance Considerations

### Hot Path Optimization

Optimize frequently called logging:

```go
// Pre-allocate common fields to reuse
var (
    commonFields = []zlog.Field{
        zlog.String("service", "payment-service"),
        zlog.String("version", version),
    }
)

// Fast path for high-frequency events
func logAPIRequest(method, path string, status int, duration time.Duration) {
    fields := make([]zlog.Field, 0, len(commonFields)+4)
    fields = append(fields, commonFields...)
    fields = append(fields,
        zlog.String("method", method),
        zlog.String("path", path),
        zlog.Int("status", status),
        zlog.Duration("duration", duration))
    
    zlog.Emit(API_REQUEST_COMPLETED, "API request completed", fields...)
}

// Conditional expensive operations
func logWithExpensiveData(ctx context.Context, includeDebugData bool) {
    fields := contextFields(ctx)
    
    if includeDebugData {
        // Only compute expensive debug data when needed
        debugData := computeExpensiveDebugInfo()
        fields = append(fields, zlog.Any("debug_data", debugData))
    }
    
    zlog.Emit(DEBUG_EVENT, "Debug information", fields...)
}
```

### Async Processing

Use async sinks for expensive operations:

```go
func createAsyncAuditSink(bufferSize int) zlog.Sink {
    eventChan := make(chan zlog.Event, bufferSize)
    
    // Background processor
    go func() {
        for event := range eventChan {
            if err := writeToAuditDatabase(event); err != nil {
                // Log error but don't block
                fmt.Printf("Audit write failed: %v\n", err)
            }
        }
    }()
    
    return zlog.NewSink("async-audit", func(ctx context.Context, event zlog.Event) error {
        select {
        case eventChan <- event:
            return nil
        default:
            // Buffer full - could log this as a metric
            return fmt.Errorf("audit buffer full")
        }
    })
}
```

## Error Handling

### Graceful Degradation

Design sinks to degrade gracefully:

```go
func createResilientSink(primary, fallback zlog.Sink) zlog.Sink {
    return zlog.NewSink("resilient", func(ctx context.Context, event zlog.Event) error {
        // Try primary first
        if err := primary.Process(ctx, event); err != nil {
            // Log the failure (to a different destination to avoid loops)
            fmt.Printf("Primary sink failed: %v\n", err)
            
            // Try fallback
            if fallbackErr := fallback.Process(ctx, event); fallbackErr != nil {
                return fmt.Errorf("both sinks failed - primary: %v, fallback: %v", 
                    err, fallbackErr)
            }
        }
        return nil
    })
}

// Usage
func setupResilientLogging() {
    primarySink := createHTTPSink("https://logs.company.com")
    fallbackSink := createFileSink("/var/log/app-fallback.log")
    resilientSink := createResilientSink(primarySink, fallbackSink)
    
    zlog.RouteSignal(zlog.ERROR, resilientSink)
    zlog.RouteSignal(zlog.FATAL, resilientSink)
}
```

### Circuit Breaker Pattern

Prevent cascade failures:

```go
type CircuitBreakerSink struct {
    underlying   zlog.Sink
    failures     int64
    lastFailTime time.Time
    threshold    int
    timeout      time.Duration
    mutex        sync.RWMutex
}

func (cb *CircuitBreakerSink) Process(ctx context.Context, event zlog.Event) error {
    cb.mutex.RLock()
    failures := cb.failures
    lastFail := cb.lastFailTime
    cb.mutex.RUnlock()
    
    // Check if circuit is open
    if failures >= int64(cb.threshold) {
        if time.Since(lastFail) < cb.timeout {
            return fmt.Errorf("circuit breaker open")
        }
        
        // Reset circuit breaker after timeout
        cb.mutex.Lock()
        cb.failures = 0
        cb.mutex.Unlock()
    }
    
    // Try the underlying sink
    err := cb.underlying.Process(ctx, event)
    if err != nil {
        cb.mutex.Lock()
        cb.failures++
        cb.lastFailTime = time.Now()
        cb.mutex.Unlock()
    } else if failures > 0 {
        // Reset on success
        cb.mutex.Lock()
        cb.failures = 0
        cb.mutex.Unlock()
    }
    
    return err
}
```

## Security and Privacy

### Sensitive Data Handling

Never log sensitive information:

```go
// Sensitive field list
var sensitiveFields = map[string]bool{
    "password":          true,
    "secret":           true,
    "token":            true,
    "api_key":          true,
    "credit_card":      true,
    "ssn":              true,
    "social_security":  true,
    "authorization":    true,
}

func sanitizeFields(fields []zlog.Field) []zlog.Field {
    sanitized := make([]zlog.Field, len(fields))
    
    for i, field := range fields {
        if sensitiveFields[strings.ToLower(field.Key)] {
            sanitized[i] = zlog.String(field.Key, "[REDACTED]")
        } else {
            sanitized[i] = field
        }
    }
    
    return sanitized
}

// Use in sensitive contexts
func logUserAction(action string, userID string, rawFields []zlog.Field) {
    safeFields := sanitizeFields(rawFields)
    safeFields = append(safeFields, zlog.String("user_id", userID))
    
    zlog.Emit(USER_ACTION, action, safeFields...)
}
```

### Data Minimization

Log only what's necessary:

```go
// Good - minimal necessary data
func logPaymentAttempt(payment *Payment, result *PaymentResult) {
    fields := []zlog.Field{
        zlog.String("payment_id", payment.ID),
        zlog.Float64("amount", payment.Amount),
        zlog.String("currency", payment.Currency),
        zlog.String("status", result.Status),
    }
    
    if result.Error != nil {
        fields = append(fields, zlog.String("error_code", result.ErrorCode))
        // Don't log the full error message - might contain sensitive info
    }
    
    zlog.Emit(PAYMENT_ATTEMPTED, "Payment attempt completed", fields...)
}

// Avoid - excessive data logging
func logPaymentAttemptBad(payment *Payment, user *User, result *PaymentResult) {
    zlog.Emit(PAYMENT_ATTEMPTED, "Payment attempt completed",
        zlog.Any("payment", payment),        // Entire payment object
        zlog.Any("user", user),              // Entire user object
        zlog.Any("result", result),          // Entire result object
        zlog.String("user_ip", user.LastIP), // PII
        zlog.String("user_email", user.Email)) // PII
}
```

## Testing and Debugging

### Structured Testing

Create reusable test patterns:

```go
// Test event expectations
type EventExpectation struct {
    Signal   zlog.Signal
    Message  string
    Fields   map[string]any
    Contains []string  // Message should contain these strings
}

func AssertEvents(t *testing.T, capture *TestLogCapture, expectations []EventExpectation) {
    t.Helper()
    
    events := capture.GetEvents()
    assert.Len(t, events, len(expectations))
    
    for i, expected := range expectations {
        if i >= len(events) {
            t.Errorf("Expected event %d not found", i)
            continue
        }
        
        event := events[i]
        assert.Equal(t, expected.Signal, event.Signal)
        
        if expected.Message != "" {
            assert.Equal(t, expected.Message, event.Message)
        }
        
        for _, substr := range expected.Contains {
            assert.Contains(t, event.Message, substr)
        }
        
        for key, expectedValue := range expected.Fields {
            assertFieldValue(t, event, key, expectedValue)
        }
    }
}
```

### Debug Helpers

Create development helpers:

```go
// Development debugging
func EnableDebugMode() {
    if os.Getenv("DEBUG_LOGGING") != "true" {
        return
    }
    
    debugSink := zlog.NewSink("debug", func(ctx context.Context, event zlog.Event) error {
        fmt.Printf("= [%s] %s\n", event.Signal, event.Message)
        for _, field := range event.Fields {
            fmt.Printf("   %s: %v\n", field.Key, field.Value)
        }
        fmt.Println()
        return nil
    })
    
    // Route everything to debug
    routeAllSignals(debugSink)
}

// Call during development setup
func init() {
    EnableDebugMode()
}
```

## Documentation and Maintenance

### Signal Documentation

Document your signal vocabulary:

```go
// signals.go - centralized signal definitions
package signals

// User lifecycle events
const (
    // USER_REGISTERED fires when a new user completes registration
    // Fields: user_id, email, registration_method
    USER_REGISTERED = "USER_REGISTERED"
    
    // USER_AUTHENTICATED fires on successful login
    // Fields: user_id, method (password/oauth/etc), ip_address
    USER_AUTHENTICATED = "USER_AUTHENTICATED"
    
    // USER_DEACTIVATED fires when user account is deactivated
    // Fields: user_id, reason, deactivated_by
    USER_DEACTIVATED = "USER_DEACTIVATED"
)

// Payment events
const (
    // PAYMENT_RECEIVED fires when payment is successfully processed
    // Fields: payment_id, user_id, amount, currency, method
    PAYMENT_RECEIVED = "PAYMENT_RECEIVED"
    
    // PAYMENT_FAILED fires when payment processing fails
    // Fields: payment_id, user_id, amount, currency, error_code, reason
    PAYMENT_FAILED = "PAYMENT_FAILED"
)
```

### Monitoring Documentation

Document your monitoring strategy:

```markdown
## Logging Architecture

### Signal Routing

- **Standard Logging**: All ERROR/FATAL � stderr (for ops)
- **Business Metrics**: USER_*, PAYMENT_*, ORDER_* � Prometheus (for business)
- **Security Events**: SECURITY_* � SIEM (for security team)
- **Audit Trail**: All events � PostgreSQL (for compliance)

### Alert Thresholds

- Payment failure rate > 5% � Slack #payments
- Login failure rate > 10% � Slack #security  
- Any FATAL event � PagerDuty

### Retention Policies

- Operational logs: 30 days
- Security logs: 1 year
- Audit logs: 7 years
- Metrics: 13 months
```

Following these best practices will help you build a robust, maintainable, and secure logging architecture with zlog.