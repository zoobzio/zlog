# Advanced Event Processing with pipz

zlog is built on [pipz](https://github.com/zoobzio/pipz), a powerful data pipeline library. This means your sinks aren't limited to simple functions - they can be sophisticated processing pipelines with retry logic, fallbacks, transformations, and more.

By default, zlog uses sequential processing for optimal performance. When you need concurrent processing, isolation, or other advanced patterns, pipz provides them.

## Why pipz Integration Matters

While simple logging needs are well-served by basic sinks, complex applications may require:
- **Reliability**: Retry failed operations, fallback to alternatives
- **Transformation**: Sanitize, enrich, or reformat events
- **Performance**: Add concurrency when needed, timeouts, buffering
- **Resilience**: Circuit breakers, rate limiting, backpressure handling

pipz provides implementations of these patterns.

## Progressive Complexity

Start simple and add sophistication as needed:

```go
// Level 1: Basic sink (no pipz needed)
basicSink := zlog.NewSink("console", func(ctx context.Context, e zlog.Event) error {
    fmt.Printf("%s: %s\n", e.Signal, e.Message)
    return nil
})

// Level 2: Add reliability with pipz
reliableSink := pipz.Retry("console-retry", 3, basicSink)

// Level 3: Add timeout protection
timedSink := pipz.Timeout("console-timeout", 5*time.Second, reliableSink)

// Level 4: Full pipeline with fallback
productionSink := pipz.NewFallback("production-logging",
    pipz.Timeout("primary", 5*time.Second, 
        pipz.Retry("remote", 3, remoteSink)),
    pipz.Apply("local-fallback", writeToLocalFile),
)
```

## Common Patterns

### Reliable Audit Logging

Ensure critical events are never lost:

```go
// Create a reliable audit pipeline
auditPipeline := pipz.NewSequence("audit-pipeline",
    // Step 1: Validate event has required fields
    pipz.Apply("validate", func(ctx context.Context, e zlog.Event) (zlog.Event, error) {
        if e.GetField("user_id") == "" {
            return e, errors.New("audit events must include user_id")
        }
        return e, nil
    }),
    
    // Step 2: Add audit metadata
    pipz.Transform("enrich", func(ctx context.Context, e zlog.Event) zlog.Event {
        e.Fields = append(e.Fields, 
            zlog.String("audit_id", generateAuditID()),
            zlog.Time("audit_timestamp", time.Now()),
        )
        return e
    }),
    
    // Step 3: Write with retries and fallback
    pipz.NewFallback("storage",
        pipz.Retry("database", 3, writeToAuditDB),
        pipz.Apply("file", writeToAuditFile),
    ),
)

// Route compliance events through the pipeline
zlog.RouteSignal(USER_DATA_ACCESSED, auditPipeline)
zlog.RouteSignal(PERMISSION_CHANGED, auditPipeline)
zlog.RouteSignal(ADMIN_ACTION, auditPipeline)
```

### Event Sanitization

Remove sensitive data before sending to external services:

```go
sanitizationPipeline := pipz.NewSequence("sanitize",
    // Remove sensitive fields
    pipz.Transform("remove-sensitive", func(ctx context.Context, e zlog.Event) zlog.Event {
        sanitized := e.Clone()
        for i, field := range sanitized.Fields {
            if isSensitive(field.Key) {
                sanitized.Fields[i].Value = "[REDACTED]"
            }
        }
        return sanitized
    }),
    
    // Send to external service
    pipz.Apply("send", sendToLoggingService),
)

// Route all ERROR events through sanitization
zlog.RouteSignal(zlog.ERROR, sanitizationPipeline)
```

### Multi-Destination Routing

Send events to multiple destinations with different processing:

```go
// Create specialized processors for each destination
metricsProcessor := pipz.Effect("metrics", func(ctx context.Context, e zlog.Event) error {
    metricName := fmt.Sprintf("event.%s", strings.ToLower(string(e.Signal)))
    metrics.Increment(metricName)
    return nil
})

alertProcessor := pipz.Apply("alerts", func(ctx context.Context, e zlog.Event) (zlog.Event, error) {
    if shouldAlert(e) {
        return e, sendAlert(e)
    }
    return e, nil
})

// When you need concurrent processing, use pipz.Concurrent
multiDestPipeline := pipz.NewConcurrent("multi-destination",
    pipz.Apply("log-file", writeToFile),
    metricsProcessor,
    alertProcessor,
    pipz.Retry("elasticsearch", 3, sendToElasticsearch),
)

// Critical events go everywhere
zlog.RouteSignal(PAYMENT_FAILED, multiDestPipeline)
zlog.RouteSignal(SECURITY_VIOLATION, multiDestPipeline)
```

### Circuit Breaker Pattern

Protect against cascading failures:

```go
// Wrap unreliable services with circuit breakers
type CircuitBreakerSink struct {
    breaker *pipz.CircuitBreaker
    sink    zlog.Sink
}

func NewCircuitBreakerSink(sink zlog.Sink) *CircuitBreakerSink {
    return &CircuitBreakerSink{
        breaker: pipz.NewCircuitBreaker(pipz.CircuitBreakerConfig{
            FailureThreshold: 5,
            ResetTimeout:     30 * time.Second,
        }),
        sink: sink,
    }
}

func (cb *CircuitBreakerSink) Process(ctx context.Context, e zlog.Event) error {
    return cb.breaker.Execute(func() error {
        return cb.sink.Process(ctx, e)
    })
}

// Use it
remoteSink := NewCircuitBreakerSink(elasticsearchSink)
zlog.RouteSignal(zlog.INFO, remoteSink)
```

### Event Enrichment

Add context to events before processing:

```go
enrichmentPipeline := pipz.NewSequence("enrichment",
    // Add user context
    pipz.Enrich("user-context", func(ctx context.Context, e zlog.Event) zlog.Event {
        if userID := e.GetField("user_id"); userID != "" {
            if user, err := fetchUser(userID); err == nil {
                e.Fields = append(e.Fields,
                    zlog.String("user_email", user.Email),
                    zlog.String("user_plan", user.Plan),
                )
            }
        }
        return e
    }),
    
    // Add geographic data
    pipz.Enrich("geo-data", func(ctx context.Context, e zlog.Event) zlog.Event {
        if ip := e.GetField("ip_address"); ip != "" {
            if geo, err := lookupGeoIP(ip); err == nil {
                e.Fields = append(e.Fields,
                    zlog.String("country", geo.Country),
                    zlog.String("city", geo.City),
                )
            }
        }
        return e
    }),
    
    // Send enriched event
    pipz.Apply("send", sendToAnalytics),
)
```

### Rate Limiting

Prevent log flooding:

```go
rateLimiter := pipz.NewRateLimiter("rate-limit", pipz.RateLimiterConfig{
    Rate:  100,           // 100 events
    Per:   time.Second,   // per second
    Burst: 200,           // allow bursts up to 200
})

throttledPipeline := pipz.NewSequence("throttled",
    rateLimiter,
    pipz.Apply("send", sendToExpensiveService),
)

// High-volume events get rate limited
zlog.RouteSignal(CACHE_MISS, throttledPipeline)
```

## Testing pipz-Enhanced Sinks

pipz components are independently testable:

```go
func TestAuditPipeline(t *testing.T) {
    // Test individual components
    validator := pipz.Apply("validate", validateAuditEvent)
    
    // Valid event should pass
    validEvent := createTestEvent(zlog.String("user_id", "123"))
    result, err := validator.Process(context.Background(), validEvent)
    assert.NoError(t, err)
    
    // Invalid event should fail
    invalidEvent := createTestEvent() // missing user_id
    _, err = validator.Process(context.Background(), invalidEvent)
    assert.Error(t, err)
    
    // Test the full pipeline with mocks
    mockDB := &MockAuditDB{}
    pipeline := createAuditPipeline(mockDB)
    
    err = pipeline.Process(context.Background(), validEvent)
    assert.NoError(t, err)
    assert.Equal(t, 1, mockDB.WriteCount)
}
```

## Performance Considerations

pipz is designed for performance, but complex pipelines have overhead:

- **Simple sinks**: ~100ns overhead
- **Retry wrapper**: ~150ns overhead  
- **Full pipeline**: ~500ns-1µs depending on complexity

For high-frequency events, keep pipelines simple. For critical events, the reliability is worth the overhead.

## Best Practices

1. **Start simple**: Don't add pipeline complexity until you need it
2. **Test components**: Each pipeline stage should be independently testable
3. **Monitor performance**: Track pipeline processing times
4. **Handle backpressure**: Use buffering or sampling for high-volume events
5. **Document pipelines**: Complex pipelines should have clear documentation

## Further Reading

- [pipz Documentation](https://github.com/zoobzio/pipz/docs) - Complete pipz reference
- [Pipeline Patterns](https://github.com/zoobzio/pipz/docs/patterns) - Common pipeline patterns
- [Performance Guide](./performance.md) - Optimizing zlog and pipz together

The combination of zlog's signal-based routing and pipz's pipeline capabilities gives you a complete event processing system, not just a logger. Start simple, add sophistication as needed.