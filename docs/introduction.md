# Introduction to zlog

## The Problem with Traditional Logging

Most logging libraries force you to categorize every event by "severity": debug, info, warn, error, fatal. But this creates real problems:

**Severity is subjective and context-dependent:**
- Is a failed payment an ERROR or a WARN? Depends if it's due to insufficient funds (normal) or system failure (critical).
- Is a successful login INFO or DEBUG? Depends if you're debugging auth issues or running in production.
- Is a cache miss WARN or INFO? Depends on your cache hit rate expectations.

**Severity doesn't determine destination:**
- Payment events need audit trails regardless of success/failure
- Security events need alerts regardless of their "severity"  
- Metrics need time-series storage, not log files
- Debug info needs filtering, not severity promotion

**This approach leads to complexity:**
```go
// This is what we're forced to do today
if payment.Failed() {
    log.Error("Payment failed", fields...)  // → All errors go to same place
    audit.Log(payment)                      // → Separate system
    metrics.Inc("payment.failures")        // → Another separate system
    if critical(payment) {
        alert.Send("Critical payment failure") // → Yet another system
    }
}
```

You end up with logging scattered across your codebase, duplicated context, and no single place to understand what events your system produces.

## The zlog Solution: Signal-Based Routing

zlog replaces severity levels with **signals** - semantic event types that describe what happened, not how important it is.

```go
// Define signals that match your domain
const (
    PAYMENT_RECEIVED = zlog.Signal("PAYMENT_RECEIVED")  
    PAYMENT_FAILED   = zlog.Signal("PAYMENT_FAILED")
    USER_LOGIN       = zlog.Signal("USER_LOGIN")
    CACHE_MISS       = zlog.Signal("CACHE_MISS")
)

// Route signals to appropriate destinations
zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)
zlog.RouteSignal(PAYMENT_RECEIVED, metricsSink)
zlog.RouteSignal(PAYMENT_FAILED, auditSink)
zlog.RouteSignal(PAYMENT_FAILED, alertSink)
zlog.RouteSignal(PAYMENT_FAILED, metricsSink)

// Emit events with clear meaning
zlog.Emit(PAYMENT_RECEIVED, "Payment processed",
    zlog.String("user_id", userID),
    zlog.Float64("amount", amount),
    zlog.String("currency", "USD"))
```

Now your payment logic just emits the right signal. The routing system handles getting that event to audit logs, metrics systems, and alert channels automatically.

## Key Benefits

### 1. **Events Have Meaning**
Instead of arguing whether something is "warn" or "error", you describe what actually happened. `PAYMENT_DECLINED` is clearer than `log.Warn("payment declined")`.

### 2. **Routing is Explicit**
You can see exactly where each type of event goes. No more hunting through code to understand your logging architecture.

### 3. **Multiple Destinations**
Events naturally go to multiple places. Errors can trigger alerts, update metrics, and create audit trails - all from one emit.

### 4. **Domain-Driven**
Your logging vocabulary matches your business domain. `USER_REGISTERED`, `ORDER_SHIPPED`, `FRAUD_DETECTED` - these mean something to your team.

### 5. **Gradual Adoption**
You can start with traditional logging using `zlog.EnableStandardLogging(zlog.INFO)` and gradually adopt signal-based routing as you identify patterns.

### 6. **Extensible Through pipz**
Built on [pipz](https://github.com/zoobzio/pipz), zlog can tap into pipeline patterns like retries, fallbacks, and transformations when you need them.

## Why Choose zlog

zlog is designed for applications where:
- Events need different handling based on their type
- You want explicit control over where events go
- Business events are as important as errors
- Domain-specific vocabulary makes more sense than severity levels
- You need structured logging with type safety
- Event routing flexibility matters more than filtering by level

## Core Philosophy

1. **Events have types, not severities** - Focus on what happened, not how important it is
2. **Structured data is primary** - Messages are for humans, fields are for machines  
3. **Multiple handlers are normal** - Real events often need multiple actions
4. **Simple things stay simple** - You can use zlog like any other logger when needed
5. **Power when you need it** - Through pipz integration, any event processing pattern is possible

## Next Steps

- [Quick Start](./quick-start.md) - Get up and running in 5 minutes
- [Signals Concept](./concepts/signals.md) - Deep dive into signal-based thinking