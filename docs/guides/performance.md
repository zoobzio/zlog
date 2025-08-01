# Performance Guide

This guide covers performance characteristics of zlog and strategies for optimization.

## Performance Characteristics

### Event Creation

Event creation and field constructors are designed to minimize allocations:

```go
// Field constructors show 0 allocations in benchmarks
zlog.Info("User action", zlog.String("user_id", userID))
```

### Concurrent Processing

When multiple sinks handle the same signal, zlog automatically uses concurrent processing:

```go
// Single sink: Direct call (fastest)
zlog.RouteSignal(PAYMENT, auditSink)

// Multiple sinks: Automatic concurrency
zlog.RouteSignal(PAYMENT, auditSink)
zlog.RouteSignal(PAYMENT, metricsSink)  // Triggers concurrent mode
zlog.RouteSignal(PAYMENT, alertSink)    // All three run concurrently
```

When multiple sinks are registered for a signal, events are cloned and processed concurrently for isolation.

### Memory Characteristics

```go
// Field creation benchmarks show 0 allocations
zlog.String("key", value)    // 0 allocs
zlog.Int("count", 42)        // 0 allocs

// Events are cloned when sent to multiple sinks
event.Clone()  // Creates new event with copied fields
```

## Benchmarking Your Setup

### Basic Benchmarks

Test your logging performance:

```go
func BenchmarkLogging(b *testing.B) {
    // Setup
    zlog.EnableStandardLogging(zlog.INFO)
    
    userID := "user_123"
    requestID := "req_456"
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        zlog.Info("API request",
            zlog.String("user_id", userID),
            zlog.String("request_id", requestID),
            zlog.Int("status", 200),
            zlog.Duration("latency", 45*time.Millisecond))
    }
}
```

### Sink Performance Testing

Benchmark specific sinks:

```go
func BenchmarkCustomSink(b *testing.B) {
    sink := zlog.NewSink("test", func(ctx context.Context, event zlog.Event) error {
        // Your sink logic here
        return processEvent(event)
    })
    
    zlog.RouteSignal("TEST", sink)
    
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        zlog.Emit("TEST", "test message", zlog.Int("value", i))
    }
}
```

### Memory Allocation Tracking

Track allocations per operation:

```go
func BenchmarkMemoryUsage(b *testing.B) {
    zlog.EnableStandardLogging(zlog.INFO)
    
    b.ReportAllocs()  // Shows allocations per op
    b.ResetTimer()
    
    for i := 0; i < b.N; i++ {
        zlog.Info("Memory test", 
            zlog.String("key1", "value1"),
            zlog.String("key2", "value2"),
            zlog.Int("number", i))
    }
}
```

## High-Performance Patterns

### Efficient Field Usage

```go
// Good - reuse field variables when possible
var (
    userIDField = zlog.String("user_id", "")  // Template field
    statusField = zlog.Int("status", 0)       // Template field
)

func logRequest(userID string, status int) {
    zlog.Info("Request completed",
        zlog.String("user_id", userID),  // Efficient creation
        zlog.Int("status", status))      // Type-safe, no boxing
}

// Avoid - unnecessary field creation in hot paths
func slowLogging() {
    // Don't recreate complex objects repeatedly
    complexData := buildComplexObject()  // Expensive
    zlog.Info("Event", zlog.Any("data", complexData))
}
```

### Conditional Logging

Use conditional logic to avoid expensive operations:

```go
// Expensive operation only when needed
func conditionalLogging(level zlog.Signal) {
    if level >= zlog.DEBUG {  // Check level first
        expensiveData := generateExpensiveDebugInfo()  // Only if needed
        zlog.Debug("Debug info", zlog.Any("data", expensiveData))
    }
}

// Or use context to skip entirely
func smartLogging(ctx context.Context) {
    if !shouldLog(ctx) {
        return  // Skip entirely
    }
    
    zlog.Info("Operation completed", extractContextFields(ctx)...)
}
```

### Batched Operations

For very high throughput, consider batching:

```go
type BatchingSink struct {
    events []zlog.Event
    mutex  sync.Mutex
    timer  *time.Timer
}

func NewBatchingSink(batchSize int, timeout time.Duration) *BatchingSink {
    bs := &BatchingSink{
        events: make([]zlog.Event, 0, batchSize),
    }
    
    return bs
}

func (bs *BatchingSink) Process(ctx context.Context, event zlog.Event) error {
    bs.mutex.Lock()
    defer bs.mutex.Unlock()
    
    bs.events = append(bs.events, event)
    
    // Trigger batch processing
    if len(bs.events) >= cap(bs.events) {
        return bs.flushBatch()
    }
    
    // Set timeout for partial batches
    if bs.timer == nil {
        bs.timer = time.AfterFunc(timeout, func() {
            bs.mutex.Lock()
            bs.flushBatch()
            bs.mutex.Unlock()
        })
    }
    
    return nil
}
```

## Sink Optimization

### Non-Blocking Sinks

Design sinks that don't block the caller:

```go
func createAsyncSink(bufferSize int) zlog.Sink {
    eventChan := make(chan zlog.Event, bufferSize)
    
    // Background processor
    go func() {
        for event := range eventChan {
            processEventSlowly(event)  // This can take time
        }
    }()
    
    return zlog.NewSink("async", func(ctx context.Context, event zlog.Event) error {
        select {
        case eventChan <- event:
            return nil  // Non-blocking send
        default:
            return fmt.Errorf("sink buffer full")  // Handle overflow
        }
    })
}
```

### Connection Pooling

Reuse expensive resources:

```go
type HTTPSink struct {
    client *http.Client
    url    string
    pool   sync.Pool  // For request buffers
}

func NewHTTPSink(url string) *HTTPSink {
    return &HTTPSink{
        url: url,
        client: &http.Client{
            Timeout: 5 * time.Second,
            Transport: &http.Transport{
                MaxIdleConns:        100,
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     90 * time.Second,
            },
        },
        pool: sync.Pool{
            New: func() interface{} {
                return bytes.NewBuffer(make([]byte, 0, 1024))
            },
        },
    }
}

func (h *HTTPSink) Process(ctx context.Context, event zlog.Event) error {
    // Reuse buffer from pool
    buf := h.pool.Get().(*bytes.Buffer)
    defer h.pool.Put(buf)
    buf.Reset()
    
    // Serialize to reused buffer
    if err := json.NewEncoder(buf).Encode(event); err != nil {
        return err
    }
    
    // Send with context
    req, err := http.NewRequestWithContext(ctx, "POST", h.url, buf)
    if err != nil {
        return err
    }
    
    resp, err := h.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    return nil
}
```

### Local Buffering

Buffer writes for better I/O performance:

```go
type BufferedFileSink struct {
    file   *os.File
    writer *bufio.Writer
    mutex  sync.Mutex
}

func NewBufferedFileSink(filename string, bufferSize int) (*BufferedFileSink, error) {
    file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return nil, err
    }
    
    return &BufferedFileSink{
        file:   file,
        writer: bufio.NewWriterSize(file, bufferSize),
    }, nil
}

func (b *BufferedFileSink) Process(ctx context.Context, event zlog.Event) error {
    b.mutex.Lock()
    defer b.mutex.Unlock()
    
    // Write to buffer
    data, _ := json.Marshal(event)
    _, err := b.writer.Write(append(data, '\n'))
    
    // Flush periodically or on errors
    if err != nil || b.writer.Buffered() > 8192 {
        b.writer.Flush()
    }
    
    return err
}
```

## Performance Monitoring

### Built-in Metrics

Track zlog performance:

```go
var (
    eventsProcessed = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "zlog_events_total",
            Help: "Total events processed",
        },
        []string{"signal", "sink"},
    )
    
    processingDuration = prometheus.NewHistogramVec(
        prometheus.HistogramOpts{
            Name: "zlog_processing_seconds",
            Help: "Time spent processing events",
        },
        []string{"signal", "sink"},
    )
)

func createMetricsSink(name string, underlying zlog.Sink) zlog.Sink {
    return zlog.NewSink(name, func(ctx context.Context, event zlog.Event) error {
        start := time.Now()
        
        err := underlying.Process(ctx, event)
        
        signal := string(event.Signal)
        eventsProcessed.WithLabelValues(signal, name).Inc()
        processingDuration.WithLabelValues(signal, name).Observe(time.Since(start).Seconds())
        
        return err
    })
}
```

### Performance Alerting

Monitor for performance degradation:

```go
type PerformanceMonitor struct {
    thresholds map[string]time.Duration
    alerts     chan PerformanceAlert
}

type PerformanceAlert struct {
    Sink     string
    Duration time.Duration
    Event    zlog.Event
}

func (pm *PerformanceMonitor) wrapSink(name string, sink zlog.Sink) zlog.Sink {
    threshold := pm.thresholds[name]
    
    return zlog.NewSink(name, func(ctx context.Context, event zlog.Event) error {
        start := time.Now()
        err := sink.Process(ctx, event)
        duration := time.Since(start)
        
        if duration > threshold {
            select {
            case pm.alerts <- PerformanceAlert{
                Sink:     name,
                Duration: duration,
                Event:    event,
            }:
            default:  // Don't block on alert channel
            }
        }
        
        return err
    })
}
```

## Profiling

### CPU Profiling

Use Go's built-in profiler:

```go
import _ "net/http/pprof"

func main() {
    // Enable pprof endpoint
    go func() {
        log.Println(http.ListenAndServe("localhost:6060", nil))
    }()
    
    // Your application with zlog
    setupLogging()
    runApplication()
}
```

Then profile with:
```bash
go tool pprof http://localhost:6060/debug/pprof/profile?seconds=30
```

### Memory Profiling

Track memory usage:

```bash
# Heap profile
go tool pprof http://localhost:6060/debug/pprof/heap

# Allocation profile
go tool pprof http://localhost:6060/debug/pprof/allocs
```

### Trace Analysis

For detailed execution tracing:

```go
import "runtime/trace"

func main() {
    f, _ := os.Create("trace.out")
    defer f.Close()
    
    trace.Start(f)
    defer trace.Stop()
    
    // Your logging-heavy code
    performLoggingOperations()
}
```

View with: `go tool trace trace.out`

## Common Performance Issues

### Problem: High Memory Usage

**Cause**: Large events being cloned for multiple sinks
**Solution**: Reduce event size or use fewer concurrent sinks

```go
// Problem: Large data in events
zlog.Info("File processed", zlog.Any("content", largeFileData))

// Solution: Reference instead of content
zlog.Info("File processed", 
    zlog.String("file_id", fileID),
    zlog.String("path", filePath),
    zlog.Int64("size", fileSize))
```

### Problem: Slow Log Processing

**Cause**: Blocking operations in sinks
**Solution**: Use async sinks or optimize sink logic

```go
// Problem: Blocking network calls
func slowSink(ctx context.Context, event zlog.Event) error {
    return http.Post(url, "application/json", eventData)  // Blocks
}

// Solution: Async with buffering
func fastSink(ctx context.Context, event zlog.Event) error {
    select {
    case eventQueue <- event:
        return nil
    default:
        return ErrQueueFull
    }
}
```

### Problem: CPU Hotspots

**Cause**: Expensive serialization or field creation
**Solution**: Optimize hot paths and reuse objects

```go
// Problem: Repeated JSON marshaling
func inefficientSink(ctx context.Context, event zlog.Event) error {
    data, _ := json.Marshal(event)  // Expensive
    return writeData(data)
}

// Solution: Streaming encoder with buffer reuse
func efficientSink(ctx context.Context, event zlog.Event) error {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()
    
    encoder := json.NewEncoder(buf)
    encoder.Encode(event)
    return writeData(buf.Bytes())
}
```

These patterns can help optimize zlog performance for your specific use case.