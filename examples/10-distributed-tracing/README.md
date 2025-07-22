# Example 10: Distributed Tracing

This example shows how to use zlog in a microservices architecture with distributed tracing, demonstrating how logs can be correlated across service boundaries.

## What It Shows

- Trace ID propagation across services
- Service-to-service correlation
- Combining logs with distributed tracing
- Integration with OpenTelemetry concepts

## Key Concepts

1. **Trace context** - Unique trace ID follows requests across services
2. **Span relationships** - Parent/child spans show request flow
3. **Service correlation** - All services log with same trace ID
4. **Distributed debugging** - Find all logs for a request across all services

## Running the Example

```bash
go run main.go
```

Simulates multiple services:
- API Gateway (receives requests)
- User Service (handles user data)
- Order Service (processes orders)
- Notification Service (sends notifications)

## Expected Output

You'll see logs from all services with correlated trace IDs:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Request received","service":"api-gateway","trace_id":"trace_123","span_id":"span_001"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Fetching user","service":"user-service","trace_id":"trace_123","span_id":"span_002","parent_span":"span_001"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Creating order","service":"order-service","trace_id":"trace_123","span_id":"span_003","parent_span":"span_001"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Sending notification","service":"notification-service","trace_id":"trace_123","span_id":"span_004","parent_span":"span_003"}
```

## Tracing Across Services

To find all logs for a distributed request:
```bash
# Find all logs for a specific trace
grep '"trace_id":"trace_123"' service-*.log | jq .

# See the request flow through services
grep '"trace_id":"trace_123"' *.log | jq -s 'sort_by(.time)'

# Find slow spans
grep '"trace_id":"trace_123"' *.log | jq 'select(.duration_ms > 100)'
```

## Integration with Tracing Systems

While zlog handles structured logging, it complements:
- **OpenTelemetry**: For detailed performance tracing
- **Jaeger/Zipkin**: For trace visualization
- **APM tools**: For holistic monitoring

## Use Cases

- **Debugging distributed requests**: See the complete flow
- **Performance analysis**: Find bottlenecks across services
- **Error investigation**: Trace errors back to root cause
- **Service dependencies**: Understand service interactions

## Best Practices

1. **Always propagate trace context** - Pass trace/span IDs between services
2. **Log at service boundaries** - Entry/exit of each service
3. **Include service metadata** - Service name, version, instance
4. **Correlate with metrics** - Same trace ID in metrics and logs

## Next Steps

- Implement trace context propagation in your services
- Add span timing for performance analysis
- Integrate with OpenTelemetry for full observability