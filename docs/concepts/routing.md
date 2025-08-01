# Understanding Routing

Routing is how zlog determines where events go based on their signals. It's the engine that enables signal-based logging and makes zlog more powerful than traditional severity-based loggers.

## How Routing Works

When you emit an event, zlog's routing system:

1. **Examines the signal** in the event
2. **Looks up registered sinks** for that signal
3. **Routes the event** to all matching sinks
4. **Processes concurrently** if multiple sinks exist

```go
// 1. Register sinks for signals
zlog.RouteSignal(PAYMENT_FAILED, auditSink)
zlog.RouteSignal(PAYMENT_FAILED, alertSink)
zlog.RouteSignal(PAYMENT_FAILED, metricsSink)

// 2. Emit event
zlog.Emit(PAYMENT_FAILED, "Payment declined", 
    zlog.String("reason", "insufficient_funds"))

// 3. Event automatically goes to all three sinks
```

## Basic Routing

### Single Destination

The simplest case - one signal, one sink:

```go
fileSink := zlog.NewSink("file-writer", func(ctx context.Context, event zlog.Event) error {
    return writeToFile("app.log", event)
})

zlog.RouteSignal(zlog.ERROR, fileSink)

// Now all ERROR events go to the file
zlog.Error("Something went wrong", zlog.Err(err))
```

### Multiple Destinations

Route one signal to multiple sinks for different purposes:

```go
// Create sinks for different purposes
fileSink := zlog.NewSink("file", writeToFile)
alertSink := zlog.NewSink("alerts", sendToSlack)
metricsSink := zlog.NewSink("metrics", updateMetrics)

// Route ERROR signal to all three
zlog.RouteSignal(zlog.ERROR, fileSink)   // Permanent record
zlog.RouteSignal(zlog.ERROR, alertSink)  // Team notification
zlog.RouteSignal(zlog.ERROR, metricsSink) // Error rate tracking

// One emit, three destinations
zlog.Error("Database connection lost", zlog.Err(err))
```

## Advanced Routing Patterns

### Signal Families

Group related signals and route them together:

```go
// All payment events go to audit
zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)
zlog.RouteSignal(PAYMENT_FAILED, auditSink)
zlog.RouteSignal(PAYMENT_REFUNDED, auditSink)
zlog.RouteSignal(PAYMENT_DISPUTED, auditSink)

// Failure events also trigger alerts
zlog.RouteSignal(PAYMENT_FAILED, alertSink)
zlog.RouteSignal(PAYMENT_DISPUTED, alertSink)

// All events generate metrics
zlog.RouteSignal(PAYMENT_RECEIVED, metricsSink)
zlog.RouteSignal(PAYMENT_FAILED, metricsSink)
zlog.RouteSignal(PAYMENT_REFUNDED, metricsSink)
```

### Traditional Level Routing

You can route traditional log levels alongside custom signals:

```go
// Route traditional levels
zlog.RouteSignal(zlog.DEBUG, devSink)     // Development only
zlog.RouteSignal(zlog.INFO, stdoutSink)   // Operational info
zlog.RouteSignal(zlog.WARN, logSink)      // Warnings to file
zlog.RouteSignal(zlog.ERROR, alertSink)   // Errors trigger alerts
zlog.RouteSignal(zlog.FATAL, alertSink)   // Fatal also alerts

// Mix with custom signals
zlog.RouteSignal(SECURITY_VIOLATION, siemSink)
zlog.RouteSignal(BUSINESS_MILESTONE, analyticsSink)
```

### Conditional Routing

Create sinks that route based on event content:

```go
conditionalSink := zlog.NewSink("conditional", func(ctx context.Context, event zlog.Event) error {
    // Route high-value payments differently
    if event.Signal == PAYMENT_RECEIVED {
        for _, field := range event.Fields {
            if field.Key == "amount" {
                if amount, ok := field.Value.(float64); ok && amount > 10000 {
                    return sendToHighValueAudit(event)
                }
            }
        }
        return sendToStandardAudit(event)
    }
    
    // Route errors during business hours differently
    if event.Signal == zlog.ERROR {
        hour := event.Time.Hour()
        if hour >= 9 && hour <= 17 {
            return sendToBusinessHoursAlert(event)
        }
        return sendToOffHoursAlert(event)
    }
    
    return nil
})

zlog.RouteSignal(PAYMENT_RECEIVED, conditionalSink)
zlog.RouteSignal(zlog.ERROR, conditionalSink)
```

## Routing Architecture

### Performance Optimization

zlog optimizes routing based on the number of sinks:

```go
// 1 sink: Direct routing (fastest)
zlog.RouteSignal(SIGNAL_A, sink1)

// 2+ sinks: Automatic concurrent processing
zlog.RouteSignal(SIGNAL_B, sink1)
zlog.RouteSignal(SIGNAL_B, sink2)  // Triggers concurrent mode
zlog.RouteSignal(SIGNAL_B, sink3)  // Added to concurrent processor
```

### Concurrent Processing

When multiple sinks handle the same signal, zlog processes them concurrently:

```go
// These three sinks will process events concurrently
zlog.RouteSignal(USER_REGISTERED, auditSink)      // ~5ms
zlog.RouteSignal(USER_REGISTERED, emailSink)      // ~100ms
zlog.RouteSignal(USER_REGISTERED, analyticsSink)  // ~20ms

// Total processing time: ~100ms (not 125ms)
zlog.Emit(USER_REGISTERED, "New user", fields...)
```

Each sink gets its own copy of the event (via `event.Clone()`), so they can't interfere with each other.

### Lock-Free Reads

The routing hot path is lock-free for performance:

```go
// This is very fast - no locks on the read path
zlog.Info("High frequency event")  // Just a map lookup + function call
```

Locks are only taken when adding new routes with `RouteSignal()`.

## Routing Strategies

### By Domain

Organize routing around business domains:

```go
// User domain
zlog.RouteSignal(USER_REGISTERED, userAuditSink)
zlog.RouteSignal(USER_ACTIVATED, userAuditSink)
zlog.RouteSignal(USER_DELETED, userAuditSink)

// Payment domain
zlog.RouteSignal(PAYMENT_RECEIVED, paymentAuditSink)
zlog.RouteSignal(PAYMENT_FAILED, paymentAuditSink)

// Order domain
zlog.RouteSignal(ORDER_CREATED, orderAuditSink)
zlog.RouteSignal(ORDER_SHIPPED, orderAuditSink)
```

### By Purpose

Route based on what you want to do with events:

```go
// Compliance/Audit
zlog.RouteSignal(USER_REGISTERED, complianceSink)
zlog.RouteSignal(PAYMENT_RECEIVED, complianceSink)
zlog.RouteSignal(DATA_EXPORTED, complianceSink)

// Real-time Alerts
zlog.RouteSignal(SECURITY_VIOLATION, alertSink)
zlog.RouteSignal(SYSTEM_DOWN, alertSink)
zlog.RouteSignal(PAYMENT_FRAUD, alertSink)

// Business Intelligence
zlog.RouteSignal(USER_REGISTERED, biSink)
zlog.RouteSignal(PAYMENT_RECEIVED, biSink)
zlog.RouteSignal(ORDER_COMPLETED, biSink)
```

### By Environment

Different routing for different environments:

```go
func setupRouting() {
    switch os.Getenv("ENV") {
    case "production":
        // Production: audit everything, alert on errors
        zlog.RouteSignal(zlog.ERROR, alertSink)
        zlog.RouteSignal(zlog.FATAL, alertSink)
        routeAllSignals(auditSink)
        
    case "staging":
        // Staging: log everything, no alerts
        routeAllSignals(stagingLogSink)
        
    case "development":
        // Development: pretty console output
        routeAllSignals(consoleSink)
    }
}
```

## Troubleshooting Routing

### Signal Not Being Routed

```go
// Check: Is the signal registered?
zlog.RouteSignal(MY_SIGNAL, mySink)  // Did you call this?

// Check: Are you using the right signal?
zlog.Emit(MY_SIGNAL, "message")  // Exact string match required

// Check: Is the sink working?
mySink := zlog.NewSink("test", func(ctx context.Context, event zlog.Event) error {
    fmt.Printf("Received: %s\n", event.Signal)  // Debug output
    return nil
})
```

### Performance Issues

```go
// Slow sink blocking others? They run concurrently automatically
// But check for:

// 1. Blocking operations in sinks
slowSink := zlog.NewSink("slow", func(ctx context.Context, event zlog.Event) error {
    time.Sleep(1 * time.Second)  // This won't block other sinks
    return nil
})

// 2. Too many allocations
efficientSink := zlog.NewSink("efficient", func(ctx context.Context, event zlog.Event) error {
    // Reuse buffers, pool connections, etc.
    return processEfficiently(event)
})
```

### Debug Routing

Create a debug sink to see what's happening:

```go
debugSink := zlog.NewSink("debug", func(ctx context.Context, event zlog.Event) error {
    fmt.Printf("[DEBUG] Signal: %s, Message: %s, Fields: %d\n", 
        event.Signal, event.Message, len(event.Fields))
    return nil
})

// Route everything to debug temporarily
zlog.RouteSignal(zlog.DEBUG, debugSink)
zlog.RouteSignal(zlog.INFO, debugSink)
zlog.RouteSignal(zlog.ERROR, debugSink)
zlog.RouteSignal(MY_CUSTOM_SIGNAL, debugSink)
```

## Routing Best Practices

1. **Set up routing early**: Configure routes in your main function or init
2. **Use consistent signals**: Don't mix `USER_LOGIN` and `user-login`
3. **Keep sinks fast**: Slow sinks don't block others, but they use resources
4. **Handle errors gracefully**: Sink errors don't affect other sinks or your app
5. **Consider fan-out**: Popular signals with many sinks use more resources
6. **Test your routing**: Verify events go where you expect
7. **Document your architecture**: Keep a map of signal → sink routing

Routing is what makes zlog powerful. By thoughtfully designing your signal vocabulary and routing strategy, you can create sophisticated logging architectures that automatically deliver the right information to the right systems.