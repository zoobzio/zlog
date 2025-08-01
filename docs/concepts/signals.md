# Understanding Signals

Signals are the foundation of zlog's approach to logging. They replace traditional severity levels with semantic event types that describe what happened in your system.

## What Are Signals?

A signal is simply a string that categorizes an event by its meaning rather than its importance:

```go
const (
    USER_REGISTERED    = zlog.Signal("USER_REGISTERED")
    PAYMENT_PROCESSED  = zlog.Signal("PAYMENT_PROCESSED")
    CACHE_INVALIDATED  = zlog.Signal("CACHE_INVALIDATED")
    API_RATE_LIMITED   = zlog.Signal("API_RATE_LIMITED")
)
```

Unlike traditional log levels (debug, info, warn, error), signals tell you exactly what type of event occurred in your business domain.

## Signals vs Traditional Levels

### Traditional Approach Problems

Traditional logging forces every event into severity buckets:

```go
// Traditional logging - semantic meaning is lost
log.Info("User registered")           // Business event
log.Info("Cache hit")                 // Performance event  
log.Info("Request completed")         // Operational event

log.Error("Payment failed")           // Could be normal (insufficient funds)
log.Error("Database connection lost") // Could be critical system failure
log.Error("Validation failed")        // Could be expected user error
```

**Problems:**
- **Loss of semantic meaning**: "Info" tells you nothing about what actually happened
- **Routing ambiguity**: All "errors" go to the same place regardless of their nature
- **Subjective categorization**: Is a failed payment "error" or "warn"?
- **No business context**: Your logs don't speak your domain language

### Signal-Based Approach

Signals preserve the semantic meaning of events:

```go
// Signal-based - meaning is preserved
zlog.Emit(USER_REGISTERED, "New user registered")
zlog.Emit(CACHE_HIT, "Cache hit for key")
zlog.Emit(REQUEST_COMPLETED, "API request completed")

zlog.Emit(PAYMENT_DECLINED, "Payment declined", 
    zlog.String("reason", "insufficient_funds"))
zlog.Emit(DATABASE_CONNECTION_LOST, "Database connection lost")
zlog.Emit(VALIDATION_ERROR, "Invalid input received")
```

**Benefits:**
- **Semantic clarity**: Each signal describes exactly what happened
- **Intelligent routing**: Events go where they need to based on their type
- **Domain language**: Your logs speak the language of your business
- **Context preservation**: The signal carries meaning beyond just severity

## Designing Good Signals

### Use Domain Language

Your signals should match how your team talks about your system:

```go
// Good - matches business domain
const (
    ORDER_PLACED     = zlog.Signal("ORDER_PLACED")
    ORDER_SHIPPED    = zlog.Signal("ORDER_SHIPPED")
    ORDER_CANCELLED  = zlog.Signal("ORDER_CANCELLED")
    INVENTORY_LOW    = zlog.Signal("INVENTORY_LOW")
    FRAUD_DETECTED   = zlog.Signal("FRAUD_DETECTED")
)

// Avoid - generic technical terms
const (
    BUSINESS_EVENT_1 = zlog.Signal("BUSINESS_EVENT_1")
    SUCCESS_TYPE_2   = zlog.Signal("SUCCESS_TYPE_2")
    ERROR_CLASS_A    = zlog.Signal("ERROR_CLASS_A")
)
```

### Be Specific But Not Too Granular

Find the right level of specificity:

```go
// Good specificity
const (
    PAYMENT_AUTHORIZED = zlog.Signal("PAYMENT_AUTHORIZED")
    PAYMENT_CAPTURED   = zlog.Signal("PAYMENT_CAPTURED")
    PAYMENT_REFUNDED   = zlog.Signal("PAYMENT_REFUNDED")
)

// Too granular - hard to manage
const (
    PAYMENT_AUTHORIZED_VISA        = zlog.Signal("PAYMENT_AUTHORIZED_VISA")
    PAYMENT_AUTHORIZED_MASTERCARD  = zlog.Signal("PAYMENT_AUTHORIZED_MASTERCARD")
    PAYMENT_AUTHORIZED_AMEX        = zlog.Signal("PAYMENT_AUTHORIZED_AMEX")
)

// Too generic - loses meaning
const (
    PAYMENT_EVENT = zlog.Signal("PAYMENT_EVENT")
)
```

Use fields to capture variations within a signal:

```go
zlog.Emit(PAYMENT_AUTHORIZED, "Payment authorized",
    zlog.String("card_type", "visa"),
    zlog.String("processor", "stripe"))
```

### Use Action-Based Names

Signals should describe actions or state changes:

```go
// Good - action-based
const (
    USER_LOGGED_IN      = zlog.Signal("USER_LOGGED_IN")
    SESSION_EXPIRED     = zlog.Signal("SESSION_EXPIRED")
    EMAIL_SENT          = zlog.Signal("EMAIL_SENT")
    BACKUP_COMPLETED    = zlog.Signal("BACKUP_COMPLETED")
)

// Avoid - noun-based
const (
    USER        = zlog.Signal("USER")
    SESSION     = zlog.Signal("SESSION")
    EMAIL       = zlog.Signal("EMAIL")
    BACKUP      = zlog.Signal("BACKUP")
)
```

### Group Related Signals

Use consistent naming patterns for related events:

```go
// User lifecycle
const (
    USER_REGISTERED     = zlog.Signal("USER_REGISTERED")
    USER_ACTIVATED      = zlog.Signal("USER_ACTIVATED")
    USER_DEACTIVATED    = zlog.Signal("USER_DEACTIVATED")
    USER_DELETED        = zlog.Signal("USER_DELETED")
)

// Order processing
const (
    ORDER_RECEIVED      = zlog.Signal("ORDER_RECEIVED")
    ORDER_VALIDATED     = zlog.Signal("ORDER_VALIDATED")
    ORDER_PROCESSED     = zlog.Signal("ORDER_PROCESSED")
    ORDER_SHIPPED       = zlog.Signal("ORDER_SHIPPED")
    ORDER_DELIVERED     = zlog.Signal("ORDER_DELIVERED")
)
```

## Signal Hierarchies and Routing

Signals don't have built-in hierarchies like log levels, but you can create logical groupings through routing:

```go
// Route all payment events to audit system
zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)
zlog.RouteSignal(PAYMENT_FAILED, auditSink)
zlog.RouteSignal(PAYMENT_REFUNDED, auditSink)

// Route failure events to alerting
zlog.RouteSignal(PAYMENT_FAILED, alertSink)
zlog.RouteSignal(ORDER_FAILED, alertSink)
zlog.RouteSignal(USER_LOGIN_FAILED, alertSink)

// Route all events to metrics
zlog.RouteSignal(PAYMENT_RECEIVED, metricsSink)
zlog.RouteSignal(PAYMENT_FAILED, metricsSink)
zlog.RouteSignal(ORDER_RECEIVED, metricsSink)
```

## Mixing Signals and Levels

You can combine traditional log levels with signal-based routing:

```go
// Enable standard logging for operational events
zlog.EnableStandardLogging(zlog.INFO)

// Route traditional levels to appropriate destinations
zlog.RouteSignal(zlog.ERROR, alertSink)   // All errors trigger alerts
zlog.RouteSignal(zlog.FATAL, alertSink)   // Fatal events trigger alerts

// Use signals for business events
zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)
zlog.RouteSignal(USER_REGISTERED, analyticsSink)

// Now you can mix both approaches
zlog.Error("Database connection failed", zlog.Err(err))  // Traditional + alert
zlog.Emit(PAYMENT_RECEIVED, "Payment processed", fields...) // Signal-based
```

## Signal Best Practices

1. **Start with your domain**: What events does your business care about?
2. **Use past tense**: `USER_REGISTERED` not `REGISTER_USER`
3. **Be consistent**: Use the same naming patterns across your application
4. **Group logically**: Related events should have similar names
5. **Avoid abbreviations**: `USER_AUTHENTICATED` not `USER_AUTHED`
6. **Consider your audience**: Operations, business, compliance, developers
7. **Document your signals**: Keep a registry of what each signal means

## Common Signal Patterns

### Application Lifecycle
```go
const (
    APP_STARTING     = zlog.Signal("APP_STARTING")
    APP_READY        = zlog.Signal("APP_READY")
    APP_SHUTTING_DOWN = zlog.Signal("APP_SHUTTING_DOWN")
    APP_STOPPED      = zlog.Signal("APP_STOPPED")
)
```

### User Authentication
```go
const (
    USER_LOGIN_ATTEMPTED    = zlog.Signal("USER_LOGIN_ATTEMPTED")
    USER_LOGIN_SUCCEEDED    = zlog.Signal("USER_LOGIN_SUCCEEDED")
    USER_LOGIN_FAILED       = zlog.Signal("USER_LOGIN_FAILED")
    USER_LOGOUT             = zlog.Signal("USER_LOGOUT")
    USER_SESSION_EXPIRED    = zlog.Signal("USER_SESSION_EXPIRED")
)
```

### API Operations
```go
const (
    REQUEST_RECEIVED    = zlog.Signal("REQUEST_RECEIVED")
    REQUEST_COMPLETED   = zlog.Signal("REQUEST_COMPLETED")
    REQUEST_FAILED      = zlog.Signal("REQUEST_FAILED")
    REQUEST_TIMEOUT     = zlog.Signal("REQUEST_TIMEOUT")
    RATE_LIMIT_EXCEEDED = zlog.Signal("RATE_LIMIT_EXCEEDED")
)
```

### Security Events
```go
const (
    SUSPICIOUS_ACTIVITY    = zlog.Signal("SUSPICIOUS_ACTIVITY")
    UNAUTHORIZED_ACCESS    = zlog.Signal("UNAUTHORIZED_ACCESS")
    PRIVILEGE_ESCALATION   = zlog.Signal("PRIVILEGE_ESCALATION")
    SECURITY_SCAN_DETECTED = zlog.Signal("SECURITY_SCAN_DETECTED")
)
```

Signals are the vocabulary of your logging system. Choose them carefully, and they'll make your logs more meaningful, your routing more intelligent, and your system more observable.