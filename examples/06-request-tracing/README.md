# Example 06: Request Tracing

This example shows how to use structured fields to trace requests through your system, making it easy to correlate all logs for a single request.

## What It Shows

- Adding request IDs to all logs within a request context
- Using structured fields for correlation
- Tracing requests across service boundaries
- How to search/filter logs by request ID

## Key Concepts

1. **Request correlation** - All logs for a request share the same ID
2. **Contextual logging** - Pass request context through your app
3. **Service boundaries** - Trace IDs follow requests between services
4. **Debugging power** - Find all logs for a problematic request instantly

## Running the Example

```bash
go run main.go
```

You'll see logs with consistent request IDs:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Request started","caller":"main.go:25","request_id":"req_abc123","method":"GET","path":"/api/users"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Database query","caller":"main.go:30","request_id":"req_abc123","query":"SELECT * FROM users"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Cache hit","caller":"main.go:35","request_id":"req_abc123","key":"users:all"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Request completed","caller":"main.go:40","request_id":"req_abc123","status":200,"duration_ms":45}
```

## Searching by Request ID

In production, you'd search logs like:
```bash
# Find all logs for a specific request
grep '"request_id":"req_abc123"' app.log | jq .

# Find slow requests
jq 'select(.duration_ms > 1000)' app.log

# Trace a request across services
grep '"trace_id":"trace_xyz789"' service-*.log
```

## Use Cases

- **Debugging**: Follow a request through all services
- **Performance analysis**: Find bottlenecks in request flow
- **Error investigation**: See full context when requests fail
- **Customer support**: Look up exactly what happened for a user

## Integration with Tracing

Request IDs complement distributed tracing:
- Request ID: Application-level correlation
- Trace ID: Infrastructure-level correlation (OpenTelemetry)
- Both together: Complete observability

## Next Steps

- See [10-distributed-tracing](../10-distributed-tracing) for cross-service tracing
- See [04-metrics-pipeline](../04-metrics-pipeline) for performance metrics