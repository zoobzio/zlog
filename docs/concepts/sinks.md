# Understanding Sinks

Sinks are the destination endpoints for your log events. They determine what happens to events after they're emitted - whether they get written to files, sent to external services, stored in databases, or trigger alerts.

## What Are Sinks?

A sink is a function that processes events. It receives an event and can perform any action: write to a file, send to an API, update metrics, trigger notifications, etc.

```go
// Basic sink that writes to stdout
sink := zlog.NewSink("console", func(ctx context.Context, event zlog.Event) error {
    fmt.Printf("[%s] %s: %s\n", 
        event.Time.Format("15:04:05"),
        event.Signal,
        event.Message)
    return nil
})
```

## Creating Sinks

### Simple Function Sinks

The easiest way to create a sink is with `zlog.NewSink()`:

```go
// File writer sink
fileSink := zlog.NewSink("file-writer", func(ctx context.Context, event zlog.Event) error {
    logFile, err := os.OpenFile("app.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    defer logFile.Close()
    
    _, err = fmt.Fprintf(logFile, "%s [%s] %s\n", 
        event.Time.Format(time.RFC3339),
        event.Signal,
        event.Message)
    return err
})

// Slack notification sink
slackSink := zlog.NewSink("slack-alerts", func(ctx context.Context, event zlog.Event) error {
    if event.Signal == "CRITICAL_ERROR" {
        return slack.PostMessage(slack.Message{
            Channel: "#alerts",
            Text:    fmt.Sprintf("🚨 Critical Error: %s", event.Message),
        })
    }
    return nil
})
```

### Structured Sinks

For more complex formatting, extract structured data from events:

```go
// JSON sink with custom formatting
jsonSink := zlog.NewSink("json-formatter", func(ctx context.Context, event zlog.Event) error {
    entry := map[string]interface{}{
        "timestamp": event.Time.Format(time.RFC3339Nano),
        "level":     event.Signal,
        "message":   event.Message,
        "service":   "payment-api",
        "version":   "1.2.3",
    }
    
    // Add caller information
    if event.Caller.File != "" {
        entry["caller"] = fmt.Sprintf("%s:%d", event.Caller.File, event.Caller.Line)
    }
    
    // Add all fields
    for _, field := range event.Fields {
        entry[field.Key] = field.Value
    }
    
    data, err := json.Marshal(entry)
    if err != nil {
        return err
    }
    
    fmt.Println(string(data))
    return nil
})
```

### External Service Sinks

Sinks can send events to external services:

```go
// Elasticsearch sink
esSink := zlog.NewSink("elasticsearch", func(ctx context.Context, event zlog.Event) error {
    doc := map[string]interface{}{
        "@timestamp": event.Time,
        "signal":     string(event.Signal),
        "message":    event.Message,
        "service":    "my-service",
    }
    
    // Add fields
    for _, field := range event.Fields {
        doc[field.Key] = field.Value
    }
    
    return esClient.Index("logs", doc)
})

// Metrics sink
metricsSink := zlog.NewSink("prometheus", func(ctx context.Context, event zlog.Event) error {
    // Increment counter for this signal
    eventCounter.WithLabelValues(string(event.Signal)).Inc()
    
    // Extract duration metrics
    for _, field := range event.Fields {
        if field.Key == "duration" && field.Type == zlog.DurationType {
            if duration, ok := field.Value.(time.Duration); ok {
                durationHistogram.WithLabelValues(string(event.Signal)).Observe(duration.Seconds())
            }
        }
    }
    
    return nil
})
```

## Sink Patterns

### Filtering Sinks

Sinks can filter events based on any criteria:

```go
// Only process high-value payments
highValueSink := zlog.NewSink("high-value-audit", func(ctx context.Context, event zlog.Event) error {
    if event.Signal != "PAYMENT_RECEIVED" {
        return nil // Skip non-payment events
    }
    
    for _, field := range event.Fields {
        if field.Key == "amount" {
            if amount, ok := field.Value.(float64); ok && amount > 10000 {
                return auditHighValuePayment(event)
            }
        }
    }
    return nil // Skip low-value payments
})

// Only process errors during business hours
businessHoursSink := zlog.NewSink("business-alerts", func(ctx context.Context, event zlog.Event) error {
    hour := event.Time.Hour()
    if hour < 9 || hour > 17 { // Outside 9-5
        return nil
    }
    
    if event.Signal == zlog.ERROR || event.Signal == zlog.FATAL {
        return sendBusinessHoursAlert(event)
    }
    return nil
})
```

### Buffering Sinks

For performance, you might buffer events:

```go
type BufferedSink struct {
    buffer   []zlog.Event
    maxSize  int
    flushCh  chan struct{}
    mu       sync.Mutex
}

func NewBufferedSink(maxSize int, flushInterval time.Duration) *BufferedSink {
    bs := &BufferedSink{
        buffer:  make([]zlog.Event, 0, maxSize),
        maxSize: maxSize,
        flushCh: make(chan struct{}),
    }
    
    // Periodic flush
    go func() {
        ticker := time.NewTicker(flushInterval)
        defer ticker.Stop()
        
        for {
            select {
            case <-ticker.C:
                bs.flush()
            case <-bs.flushCh:
                bs.flush()
            }
        }
    }()
    
    return bs
}

func (bs *BufferedSink) Process(ctx context.Context, event zlog.Event) (zlog.Event, error) {
    bs.mu.Lock()
    defer bs.mu.Unlock()
    
    bs.buffer = append(bs.buffer, event)
    
    if len(bs.buffer) >= bs.maxSize {
        select {
        case bs.flushCh <- struct{}{}:
        default:
        }
    }
    
    return event, nil
}
```

### Conditional Sinks

Route events conditionally based on context:

```go
// Different behavior for different environments
envSink := zlog.NewSink("environment-aware", func(ctx context.Context, event zlog.Event) error {
    switch os.Getenv("ENV") {
    case "production":
        // In production, only log errors and above
        if event.Signal == zlog.ERROR || event.Signal == zlog.FATAL {
            return sendToDatadog(event)
        }
        
    case "staging":
        // In staging, log everything to file
        return writeToFile("staging.log", event)
        
    case "development":
        // In development, pretty print to console
        return prettyPrint(event)
    }
    
    return nil
})
```

## Error Handling in Sinks

Sinks should handle errors gracefully. Errors in one sink don't affect other sinks or the application:

```go
robustSink := zlog.NewSink("robust-sink", func(ctx context.Context, event zlog.Event) error {
    // Try primary destination
    if err := sendToPrimaryService(event); err != nil {
        // Log the error (but don't create infinite loops!)
        fmt.Printf("Primary service failed: %v\n", err)
        
        // Try fallback
        if err := sendToFallbackService(event); err != nil {
            fmt.Printf("Fallback service also failed: %v\n", err)
            
            // Last resort - write to local file
            return writeToLocalFile(event)
        }
    }
    
    return nil
})
```

## Sink Performance

Sinks run concurrently, but they should still be efficient:

```go
// Pool connections for better performance
var httpClient = &http.Client{
    Timeout: 5 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:        100,
        MaxIdleConnsPerHost: 10,
    },
}

efficientSink := zlog.NewSink("http-sink", func(ctx context.Context, event zlog.Event) error {
    // Reuse HTTP client
    resp, err := httpClient.Post("https://logs.example.com", "application/json", eventToJSON(event))
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode >= 400 {
        return fmt.Errorf("HTTP error: %d", resp.StatusCode)
    }
    
    return nil
})
```

## Testing Sinks

Sinks are easy to test since they're just functions:

```go
func TestSlackSink(t *testing.T) {
    var sentMessage string
    
    // Mock Slack client
    mockSlack := &MockSlackClient{
        PostFunc: func(msg slack.Message) error {
            sentMessage = msg.Text
            return nil
        },
    }
    
    sink := createSlackSink(mockSlack)
    
    event := zlog.NewEvent("CRITICAL_ERROR", "Database down", nil)
    err := sink.Process(context.Background(), event)
    
    assert.NoError(t, err)
    assert.Contains(t, sentMessage, "Database down")
}
```

## Built-in Sinks

zlog provides some common sink patterns:

```go
// Standard JSON sink (used by EnableStandardLogging)
jsonSink := zlog.JSONObserver(os.Stderr)

// You can create similar patterns
func FileObserver(filename string) zlog.Sink {
    return zlog.NewSink("file-observer", func(ctx context.Context, event zlog.Event) error {
        // File writing logic
    })
}

func MetricsObserver(client *prometheus.Client) zlog.Sink {
    return zlog.NewSink("metrics-observer", func(ctx context.Context, event zlog.Event) error {
        // Metrics logic
    })
}
```

Sinks are where the rubber meets the road in zlog. They're the bridge between your application events and your observability infrastructure. Design them thoughtfully, and they'll give you powerful insights into your system's behavior.