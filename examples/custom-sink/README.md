# Custom Sink Example

This example demonstrates building custom sinks to integrate zlog with external systems.

## What This Shows

- Creating sinks for metrics collection
- Publishing events to message queues
- Database audit logging patterns
- Conditional sink processing
- Event batching for efficiency

## Running the Example

```bash
go run main.go
```

## Custom Sink Patterns

### 1. Metrics Sink
Extracts numeric data from events and updates counters/gauges:
- Counts events by signal type
- Extracts durations, amounts, counts
- Updates metrics collector (Prometheus/StatsD style)

### 2. Message Queue Sink
Publishes specific events to a queue:
- Filters events by signal type
- Serializes to JSON
- Publishes to topic (Kafka/RabbitMQ style)

### 3. Database Audit Sink
Writes audit trail to database:
- Extracts audit-relevant fields
- Simulates SQL insert
- Preserves user actions and results

### 4. Conditional Sink
Processes events based on conditions:
- High-value transaction detection
- Threshold-based alerting
- Custom business rules

### 5. Batching Sink
Accumulates events for batch processing:
- Reduces I/O operations
- Efficient bulk inserts
- Periodic processing

## Creating Your Own Sink

```go
sink := zlog.NewSink("my-sink", func(ctx context.Context, event zlog.Event) error {
    // Process the event
    // Return error to trigger retry/fallback
    return nil
})
```

## Using Sink Adapters

All custom sinks work with the built-in adapters:

```go
sink.WithRetry(3).
    WithTimeout(5 * time.Second).
    WithAsync()
```