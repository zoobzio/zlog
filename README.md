# zlog

Signal-based structured logging for Go that acknowledges different events need different handling.

A different approach to logging that uses semantic signals instead of severity levels. Route payment events to your audit system, errors to your alerts, and metrics to your time-series database - all through one simple API.

```go
// Traditional logging forces everything into severity levels
log.Info("payment processed")  // Is this info or audit?
log.Error("rate limit hit")    // Is this error or metric?

// zlog uses signals to route events where they belong
zlog.Emit(PAYMENT_RECEIVED, "Payment processed",
    zlog.String("user_id", "123"),
    zlog.Float64("amount", 99.99))

zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)   // → Audit trail
zlog.RouteSignal(PAYMENT_RECEIVED, metricsSink) // → Revenue metrics
```

## Why zlog?

- **Signal-based routing**: Events go where they belong, not into severity buckets
- **True structured logging**: Type-safe fields with compile-time safety
- **Simple and fast**: Sequential sink processing for predictable performance
- **Extensible**: Easy to add custom sinks for any destination
- **Zero-allocation fields**: Field constructors create no heap allocations
- **Simple**: Clean API that's easy to understand and use
- **Built on pipz**: Access to pipeline patterns like retries and fallbacks when needed

## Installation

```bash
go get github.com/zoobzio/zlog
```

Requirements: Go 1.21+ (for generics)

## Quick Start

### Traditional Logging

For a familiar logging experience with structured fields:

```go
import "github.com/zoobzio/zlog"

func main() {
    // Enable JSON logging to stderr
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Use familiar log levels
    zlog.Info("Server starting", zlog.Int("port", 8080))
    zlog.Debug("This won't show with INFO level")
    
    // Structured fields with type safety
    zlog.Error("Connection failed",
        zlog.Err(err),
        zlog.String("host", "db.example.com"),
        zlog.Duration("timeout", 30*time.Second))
}
```

### Signal-Based Routing

Define signals that match your application's events:

```go
// Define domain-specific signals
const (
    PAYMENT_RECEIVED = zlog.Signal("PAYMENT_RECEIVED")
    PAYMENT_FAILED   = zlog.Signal("PAYMENT_FAILED")
    USER_LOGIN       = zlog.Signal("USER_LOGIN")
    CACHE_MISS       = zlog.Signal("CACHE_MISS")
)

// Create specialized sinks
auditSink := zlog.NewSink("audit", func(ctx context.Context, e zlog.Event) error {
    // Write to audit log with regulatory compliance formatting
    return auditWriter.WriteEvent(e)
})

alertSink := zlog.NewSink("alerts", func(ctx context.Context, e zlog.Event) error {
    if e.Signal == PAYMENT_FAILED {
        return slack.PostAlert(e.Message, e.Fields)
    }
    return nil
})

// Route signals to appropriate handlers (multiple sinks per signal)
zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)
zlog.RouteSignal(PAYMENT_FAILED, auditSink, alertSink)  // Goes to both!
zlog.RouteSignal(USER_LOGIN, auditSink)
zlog.RouteSignal(CACHE_MISS, metricsSink)

// Emit events with meaning
zlog.Emit(PAYMENT_RECEIVED, "Payment processed successfully",
    zlog.String("user_id", userID),
    zlog.String("payment_id", paymentID),
    zlog.Float64("amount", amount),
    zlog.String("currency", "USD"))
```

## Core Concepts

### Signals vs Levels

Traditional logging makes you choose a "severity" for every event. But is a failed payment an ERROR or a WARN? Is a successful login INFO or DEBUG? These aren't severity decisions - they're routing decisions.

Signals let you say what happened, not how important it is:

```go
// Instead of arguing about severity...
log.Warn("Payment declined")  // or is it Error? Info?

// Say what actually happened
zlog.Emit(PAYMENT_DECLINED, "Card declined", 
    zlog.String("reason", "insufficient_funds"))
```

### Structured Fields

Type-safe field constructors prevent errors at compile time:

```go
zlog.Info("Request completed",
    zlog.String("method", "POST"),
    zlog.String("path", "/api/users"),
    zlog.Int("status", 201),
    zlog.Duration("latency", time.Since(start)),
    zlog.Time("timestamp", time.Now()),
    zlog.Err(err),  // nil-safe
    zlog.Data("user", user))  // arbitrary types
```

### Multiple Sinks

Events can go to multiple destinations in a single call:

```go
// Route errors to multiple handlers at once
zlog.RouteSignal(zlog.ERROR, fileSink, consoleSink, alertSink, metricsSink)

// Or add them separately - same effect
zlog.RouteSignal(zlog.ERROR, fileSink)     // Permanent record
zlog.RouteSignal(zlog.ERROR, consoleSink)  // Developer visibility
```

### Sampling High-Volume Events

Reduce load while maintaining visibility with sampling:

```go
// Sample 10% of cache hits (high volume)
cacheSink := metricsSink.WithSampling(0.1)
zlog.RouteSignal(CACHE_HIT, cacheSink)

// Sample 1% of API requests
apiSink := fileSink.WithSampling(0.01).WithAsync()
zlog.RouteSignal(API_REQUEST, apiSink)

// For statistical sampling use probabilistic mode
randomSink := debugSink.WithProbabilisticSampling(0.25) // 25% random sample
```

### Creating Modules

Modules are just functions that set up routing. See `log.go` for the standard logging module:

```go
// myapp/logging/siem.go
package logging

import (
    "github.com/zoobzio/zlog"
    "github.com/splunk/splunk-sdk-go"
)

var siemSink = zlog.NewSink("siem-forwarder", func(ctx context.Context, e zlog.Event) error {
    return splunk.Send(convertToSplunkEvent(e))
})

// EnableSIEMForwarding routes security events to your SIEM
func EnableSIEMForwarding(config SIEMConfig) error {
    if err := splunk.Connect(config); err != nil {
        return err
    }
    
    zlog.RouteSignal(zlog.SECURITY, siemSink)
    zlog.RouteSignal(zlog.AUDIT, siemSink)
    zlog.RouteSignal("INTRUSION_DETECTED", siemSink)
    zlog.RouteSignal("PRIVILEGE_ESCALATION", siemSink)
    
    return nil
}
```

## Examples

### Web Service

```go
const (
    REQUEST_START    = zlog.Signal("REQUEST_START")
    REQUEST_COMPLETE = zlog.Signal("REQUEST_COMPLETE")
    AUTH_FAILED      = zlog.Signal("AUTH_FAILED")
)

func handler(w http.ResponseWriter, r *http.Request) {
    start := time.Now()
    
    zlog.Emit(REQUEST_START, "Handling request",
        zlog.String("method", r.Method),
        zlog.String("path", r.URL.Path),
        zlog.String("remote_addr", r.RemoteAddr))
    
    if !authenticate(r) {
        zlog.Emit(AUTH_FAILED, "Authentication failed",
            zlog.String("path", r.URL.Path),
            zlog.String("auth_header", r.Header.Get("Authorization")))
        http.Error(w, "Unauthorized", 401)
        return
    }
    
    // ... handle request ...
    
    zlog.Emit(REQUEST_COMPLETE, "Request completed",
        zlog.String("method", r.Method),
        zlog.String("path", r.URL.Path),
        zlog.Int("status", 200),
        zlog.Duration("latency", time.Since(start)))
}
```

### Background Jobs

```go
const (
    JOB_STARTED   = zlog.Signal("JOB_STARTED")
    JOB_COMPLETED = zlog.Signal("JOB_COMPLETED")
    JOB_FAILED    = zlog.Signal("JOB_FAILED")
    JOB_RETRY     = zlog.Signal("JOB_RETRY")
)

func processJob(job Job) {
    zlog.Emit(JOB_STARTED, "Processing job",
        zlog.String("job_id", job.ID),
        zlog.String("type", job.Type))
    
    for attempt := 0; attempt < maxRetries; attempt++ {
        if err := job.Execute(); err != nil {
            zlog.Emit(JOB_RETRY, "Job failed, retrying",
                zlog.String("job_id", job.ID),
                zlog.Int("attempt", attempt+1),
                zlog.Err(err))
            time.Sleep(backoff(attempt))
            continue
        }
        
        zlog.Emit(JOB_COMPLETED, "Job completed successfully",
            zlog.String("job_id", job.ID),
            zlog.Duration("duration", time.Since(start)))
        return
    }
    
    zlog.Emit(JOB_FAILED, "Job failed after all retries",
        zlog.String("job_id", job.ID),
        zlog.Int("attempts", maxRetries))
}
```

## Advanced Capabilities with pipz

zlog is built on [pipz](https://github.com/zoobzio/pipz), giving you access to sophisticated event processing when you need it:

```go
// Start simple - basic routing
zlog.RouteSignal(PAYMENT_FAILED, alertSink)

// Add reliability when needed
reliableAudit := pipz.Retry("audit-write", 3, auditSink)
zlog.RouteSignal(PAYMENT_RECEIVED, reliableAudit)

// Or build complex processing pipelines
errorPipeline := pipz.NewSequence("error-handling",
    pipz.Apply("sanitize", removeSensitiveData),
    pipz.NewFallback("delivery",
        pipz.Retry("primary", 3, sendToElasticsearch),
        pipz.Apply("fallback", writeToLocalFile),
    ),
    pipz.Effect("metrics", updateErrorMetrics),
)
zlog.RouteSignal(ERROR, errorPipeline)
```

With pipz integration, you get:
- **Retry with backoff** - Automatic retries for transient failures
- **Fallback chains** - Primary/backup sink strategies  
- **Circuit breakers** - Protect against cascading failures
- **Concurrent processing** - Fan-out to multiple sinks in parallel
- **Event transformation** - Modify events before delivery
- **Conditional routing** - Route based on event content
- **And much more** - Full pipeline capabilities when you need them

The beauty is progressive complexity - use simple sinks for simple needs, tap into pipz power when you need sophisticated processing.

## Design Philosophy

1. **Events have types, not severities**: A payment failure isn't an "error level" - it's a payment failure that might need fraud detection, customer notification, and metric tracking.

2. **Structured data is primary**: Messages are for humans, fields are for machines. Every event should include rich context.

3. **Multiple handlers are normal**: Real events often need multiple actions. Sequential routing keeps it simple and fast.

4. **Simple things should be simple**: You can use zlog like a traditional logger with `EnableStandardLogging()` and gradually adopt signals.

5. **Progressive complexity**: Start with simple sequential processing. Add pipz pipelines when you need retries, concurrency, or transformations.

## Performance

zlog is designed with performance in mind:

- **Zero-allocation field constructors**: Field creation benchmarks show 0 allocations
- **Efficient routing**: Direct dispatch to sinks without cloning
- **Sequential processing**: Predictable performance without concurrency overhead
- **95.9% test coverage**: Comprehensive test suite
- **Benchmarked**: See BENCHMARKS.md for detailed performance metrics

## Questions & Answers

**How is this different from traditional loggers?**

Traditional loggers focus on severity (debug < info < warn < error). zlog focuses on event types. You don't filter by "level", you route by signal.

**Can I use both signals and levels?**

Yes! `EnableStandardLogging()` provides familiar level-based logging. You can mix approaches as needed.

**What about log sampling/filtering?**

Create a filtering sink using pipz capabilities:
```go
samplingSink := pipz.NewSampler(0.1, actualSink) // 10% sampling
zlog.RouteSignal(HIGH_VOLUME_SIGNAL, samplingSink)
```

**How do I rotate log files?**

File rotation belongs in the sink, not the logger. Use a proper file sink like lumberjack or let your platform handle it (systemd, Docker, K8s).

## Contributing

Contributions welcome! Please ensure:
- Tests pass: `go test ./...`
- Coverage maintained: `go test -cover` (currently 95.9%)
- Benchmarks pass: `go test -bench=.`
- Code is formatted: `go fmt ./...`
- Lint passes: `golangci-lint run`

## License

MIT License - see [LICENSE](LICENSE) file for details.