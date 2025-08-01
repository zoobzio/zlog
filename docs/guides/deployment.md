# Deployment Guide

This guide covers considerations for deploying applications that use zlog, including configuration patterns, performance considerations, and integration strategies.

## Configuration Patterns

### Environment-Based Setup

You can configure zlog differently based on your environment:

```go
func setupLogging() error {
    env := os.Getenv("ENV")
    
    switch env {
    case "production":
        return setupProductionLogging()
    case "staging":
        return setupStagingLogging()
    case "development":
        return setupDevelopmentLogging()
    default:
        return setupDevelopmentLogging()
    }
}

func setupProductionLogging() error {
    // Conservative log level for production
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Enable structured monitoring
    if err := enableMetrics(); err != nil {
        return err
    }
    
    // Enable audit trails for compliance
    if err := enableAuditLogging(); err != nil {
        return err
    }
    
    // Enable alerting for critical issues
    if err := enableAlerting(); err != nil {
        return err
    }
    
    return nil
}
```

### Configuration from Environment

Use environment variables for production configuration:

```go
type ProductionConfig struct {
    LogLevel     string
    MetricsPort  int
    AuditDB      string
    SlackWebhook string
    SentryDSN    string
}

func loadConfig() ProductionConfig {
    return ProductionConfig{
        LogLevel:     getEnv("LOG_LEVEL", "INFO"),
        MetricsPort:  getEnvInt("METRICS_PORT", 9090),
        AuditDB:      getEnv("AUDIT_DB_URL", ""),
        SlackWebhook: getEnv("SLACK_WEBHOOK", ""),
        SentryDSN:    getEnv("SENTRY_DSN", ""),
    }
}

func getEnv(key, defaultVal string) string {
    if val := os.Getenv(key); val != "" {
        return val
    }
    return defaultVal
}
```

## Performance Considerations

### Sink Performance

Design sinks for high throughput:

```go
// Efficient HTTP sink with connection pooling
var httpClient = &http.Client{
    Timeout: 5 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
        IdleConnTimeout:     90 * time.Second,
    },
}

func createHTTPSink(url string) zlog.Sink {
    return zlog.NewSink("http-logs", func(ctx context.Context, event zlog.Event) error {
        data, err := json.Marshal(event)
        if err != nil {
            return err
        }
        
        req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
        if err != nil {
            return err
        }
        req.Header.Set("Content-Type", "application/json")
        
        resp, err := httpClient.Do(req)
        if err != nil {
            return err
        }
        defer resp.Body.Close()
        
        if resp.StatusCode >= 400 {
            return fmt.Errorf("HTTP error: %d", resp.StatusCode)
        }
        
        return nil
    })
}
```

### Buffering for Performance

Use buffering for high-volume sinks:

```go
type BufferedSink struct {
    buffer   []zlog.Event
    maxSize  int
    timeout  time.Duration
    sender   func([]zlog.Event) error
    mutex    sync.Mutex
    timer    *time.Timer
}

func NewBufferedSink(maxSize int, timeout time.Duration, sender func([]zlog.Event) error) *BufferedSink {
    bs := &BufferedSink{
        buffer:  make([]zlog.Event, 0, maxSize),
        maxSize: maxSize,
        timeout: timeout,
        sender:  sender,
    }
    
    return bs
}

func (bs *BufferedSink) Process(ctx context.Context, event zlog.Event) (zlog.Event, error) {
    bs.mutex.Lock()
    defer bs.mutex.Unlock()
    
    bs.buffer = append(bs.buffer, event)
    
    // Start timer on first event
    if len(bs.buffer) == 1 {
        bs.timer = time.AfterFunc(bs.timeout, bs.flush)
    }
    
    // Flush if buffer is full
    if len(bs.buffer) >= bs.maxSize {
        bs.flushNow()
    }
    
    return event, nil
}

func (bs *BufferedSink) flush() {
    bs.mutex.Lock()
    defer bs.mutex.Unlock()
    bs.flushNow()
}

func (bs *BufferedSink) flushNow() {
    if len(bs.buffer) == 0 {
        return
    }
    
    // Stop timer
    if bs.timer != nil {
        bs.timer.Stop()
        bs.timer = nil
    }
    
    // Send buffer
    events := make([]zlog.Event, len(bs.buffer))
    copy(events, bs.buffer)
    bs.buffer = bs.buffer[:0]
    
    // Send asynchronously to avoid blocking
    go func() {
        if err := bs.sender(events); err != nil {
            // Log error (but avoid infinite loops)
            fmt.Printf("Failed to send buffered events: %v\n", err)
        }
    }()
}
```

### Sampling for High-Volume Events

Sample high-frequency events to reduce load:

```go
func createSamplingSink(rate float64, underlying zlog.Sink) zlog.Sink {
    return zlog.NewSink("sampling", func(ctx context.Context, event zlog.Event) error {
        // Always process critical events
        if event.Signal == zlog.ERROR || event.Signal == zlog.FATAL {
            return underlying.Process(ctx, event)
        }
        
        // Sample other events
        if rand.Float64() < rate {
            return underlying.Process(ctx, event)
        }
        
        return nil
    })
}

// Usage: sample 10% of INFO events, but all ERROR/FATAL
samplingSink := createSamplingSink(0.1, expensiveSink)
zlog.RouteSignal(zlog.INFO, samplingSink)
zlog.RouteSignal(zlog.ERROR, expensiveSink)  // No sampling for errors
```

## Reliability

### Error Handling and Fallbacks

Implement robust error handling:

```go
func createResilientSink(primary, fallback zlog.Sink) zlog.Sink {
    return zlog.NewSink("resilient", func(ctx context.Context, event zlog.Event) error {
        // Try primary sink
        if err := primary.Process(ctx, event); err != nil {
            // Log the failure (but avoid loops)
            fmt.Printf("Primary sink failed: %v\n", err)
            
            // Try fallback
            if fallbackErr := fallback.Process(ctx, event); fallbackErr != nil {
                // Both failed - this is the error we return
                return fmt.Errorf("both sinks failed - primary: %v, fallback: %v", err, fallbackErr)
            }
        }
        return nil
    })
}

// Usage
primarySink := createHTTPSink("https://logs.example.com")
fallbackSink := createFileSink("/var/log/app-fallback.log")
resilientSink := createResilientSink(primarySink, fallbackSink)

zlog.RouteSignal(zlog.ERROR, resilientSink)
```

### Circuit Breaker Pattern

Protect against failing downstream services:

```go
type CircuitBreaker struct {
    maxFailures   int
    resetTimeout  time.Duration
    failures      int
    lastFailTime  time.Time
    state         string // "closed", "open", "half-open"
    mutex         sync.Mutex
}

func (cb *CircuitBreaker) Call(fn func() error) error {
    cb.mutex.Lock()
    defer cb.mutex.Unlock()
    
    switch cb.state {
    case "open":
        if time.Since(cb.lastFailTime) > cb.resetTimeout {
            cb.state = "half-open"
            cb.failures = 0
        } else {
            return fmt.Errorf("circuit breaker is open")
        }
    }
    
    err := fn()
    if err != nil {
        cb.failures++
        cb.lastFailTime = time.Now()
        
        if cb.failures >= cb.maxFailures {
            cb.state = "open"
        }
        return err
    }
    
    // Success
    cb.failures = 0
    cb.state = "closed"
    return nil
}

func createCircuitBreakerSink(underlying zlog.Sink) zlog.Sink {
    cb := &CircuitBreaker{
        maxFailures:  5,
        resetTimeout: 30 * time.Second,
        state:        "closed",
    }
    
    return zlog.NewSink("circuit-breaker", func(ctx context.Context, event zlog.Event) error {
        return cb.Call(func() error {
            return underlying.Process(ctx, event)
        })
    })
}
```

## Security

### Sensitive Data Handling

Never log sensitive information:

```go
func sanitizeEvent(event zlog.Event) zlog.Event {
    sanitized := event.Clone()
    
    for i, field := range sanitized.Fields {
        switch field.Key {
        case "password", "secret", "token", "api_key":
            sanitized.Fields[i].Value = "[REDACTED]"
        case "credit_card", "ssn":
            sanitized.Fields[i].Value = "[REDACTED]"
        case "email":
            // Partially redact emails
            if email, ok := field.Value.(string); ok {
                sanitized.Fields[i].Value = redactEmail(email)
            }
        }
    }
    
    return sanitized
}

func createSanitizingSink(underlying zlog.Sink) zlog.Sink {
    return zlog.NewSink("sanitizing", func(ctx context.Context, event zlog.Event) error {
        sanitized := sanitizeEvent(event)
        return underlying.Process(ctx, sanitized)
    })
}
```

### Audit Trail Security

Ensure audit logs are tamper-proof:

```go
func createSecureAuditSink(dbURL string) (zlog.Sink, error) {
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        return nil, err
    }
    
    return zlog.NewSink("secure-audit", func(ctx context.Context, event zlog.Event) error {
        // Calculate checksum for integrity
        eventJSON, _ := json.Marshal(event)
        checksum := sha256.Sum256(eventJSON)
        
        // Insert with checksum
        _, err := db.ExecContext(ctx,
            "INSERT INTO audit_log (timestamp, event_data, checksum) VALUES ($1, $2, $3)",
            event.Time, eventJSON, hex.EncodeToString(checksum[:]))
        
        return err
    }), nil
}
```

## Monitoring and Observability

### Health Checks

Monitor zlog health:

```go
type LoggingHealth struct {
    sinkStats map[string]*SinkStats
    mutex     sync.RWMutex
}

type SinkStats struct {
    TotalEvents  int64
    FailedEvents int64
    LastError    error
    LastSeen     time.Time
}

func (lh *LoggingHealth) recordEvent(sinkName string, err error) {
    lh.mutex.Lock()
    defer lh.mutex.Unlock()
    
    stats := lh.sinkStats[sinkName]
    if stats == nil {
        stats = &SinkStats{}
        lh.sinkStats[sinkName] = stats
    }
    
    stats.TotalEvents++
    stats.LastSeen = time.Now()
    
    if err != nil {
        stats.FailedEvents++
        stats.LastError = err
    }
}

func createMonitoredSink(name string, underlying zlog.Sink, health *LoggingHealth) zlog.Sink {
    return zlog.NewSink(name, func(ctx context.Context, event zlog.Event) error {
        err := underlying.Process(ctx, event)
        health.recordEvent(name, err)
        return err
    })
}
```

### Metrics Integration

Expose logging metrics:

```go
var (
    eventsTotal = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "zlog_events_total",
            Help: "Total number of log events",
        },
        []string{"signal", "sink"},
    )
    
    eventsErrors = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "zlog_events_errors_total",
            Help: "Total number of log event errors",
        },
        []string{"signal", "sink"},
    )
    
    eventsDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "zlog_events_duration_seconds",
            Help:    "Time spent processing log events",
            Buckets: prometheus.DefBuckets,
        },
        []string{"signal", "sink"},
    )
)

func createMetricsSink(name string, underlying zlog.Sink) zlog.Sink {
    return zlog.NewSink(name, func(ctx context.Context, event zlog.Event) error {
        start := time.Now()
        
        err := underlying.Process(ctx, event)
        
        duration := time.Since(start)
        signal := string(event.Signal)
        
        eventsTotal.WithLabelValues(signal, name).Inc()
        eventsDuration.WithLabelValues(signal, name).Observe(duration.Seconds())
        
        if err != nil {
            eventsErrors.WithLabelValues(signal, name).Inc()
        }
        
        return err
    })
}
```

## Container Deployment

### Docker Configuration

Configure zlog for containerized environments:

```dockerfile
# Dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o myapp .

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/myapp .

# Set default log level
ENV LOG_LEVEL=INFO

CMD ["./myapp"]
```

### Kubernetes Deployment

```yaml
# k8s-deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
spec:
  replicas: 3
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
    spec:
      containers:
      - name: myapp
        image: myapp:latest
        env:
        - name: LOG_LEVEL
          value: "INFO"
        - name: METRICS_PORT
          value: "9090"
        - name: AUDIT_DB_URL
          valueFrom:
            secretKeyRef:
              name: myapp-secrets
              key: audit-db-url
        ports:
        - containerPort: 8080
        - containerPort: 9090  # Metrics
        resources:
          requests:
            memory: "128Mi"
            cpu: "100m"
          limits:
            memory: "256Mi"
            cpu: "200m"
```

## Debug Mode

You can add debug capabilities to help with troubleshooting:

```go
func enableDebugMode() {
    if os.Getenv("DEBUG_LOGGING") == "true" {
        debugSink := zlog.NewSink("debug", func(ctx context.Context, event zlog.Event) error {
            fmt.Printf("[DEBUG] %s: %s (fields: %d)\n", 
                event.Signal, event.Message, len(event.Fields))
            return nil
        })
        
        // Route all signals to debug
        zlog.RouteSignal(zlog.DEBUG, debugSink)
        zlog.RouteSignal(zlog.INFO, debugSink)
        zlog.RouteSignal(zlog.ERROR, debugSink)
        // Add custom signals as needed
    }
}
```

These patterns can help you deploy and manage zlog effectively in your applications.