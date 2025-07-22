# Example 05: Error Alerting

This example shows how to create a dedicated error alerting pipeline that captures errors with full context for incident response.

## What It Shows

- Creating a custom ALERT signal for critical errors
- Enriching error events with stack traces and context
- Separating alerts from regular error logs
- How to integrate with alerting systems (PagerDuty, Slack, etc.)

## Key Concepts

1. **Not all errors are alerts** - Only critical issues should page someone
2. **Rich context** - Include stack traces, user info, request IDs
3. **Deduplication ready** - Include error fingerprints for grouping
4. **Action required** - Alerts should be actionable, not just informative

## Running the Example

```bash
go run main.go
```

You'll see:
- Standard logs (including regular errors) go to stderr
- Critical alerts go to `alerts.json` file
- Stack traces are captured for alerts

Expected console output:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Payment service started","caller":"main.go:15"}
{"time":"2024-01-20T10:30:45Z","signal":"ERROR","message":"Failed to process payment","caller":"main.go:25","error":"insufficient funds","user_id":"user123"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Processed 5 payments, 2 failures","caller":"main.go:35"}
```

Expected alerts.json content:
```json
{"time":"2024-01-20T10:30:45Z","signal":"ALERT","message":"Database connection lost","caller":"main.go:30","error":"connection refused","stack_trace":"main.processPayment() main.go:30\nmain.Run() main.go:20","severity":"critical","service":"payment-api","fingerprint":"db_connection_refused"}
{"time":"2024-01-20T10:30:45Z","signal":"ALERT","message":"Payment gateway timeout","caller":"main.go:40","error":"context deadline exceeded","duration_ms":30000,"severity":"high","service":"payment-api","fingerprint":"gateway_timeout"}
```

## Use Cases

- **Database outages**: Connection failures, query timeouts
- **External service failures**: API timeouts, authentication failures
- **Data corruption**: Validation failures, inconsistent state
- **Resource exhaustion**: OOM, disk full, rate limits

## Alert Routing

In production, alerts would be routed to:
- PagerDuty (via API sink)
- Slack (via webhook sink)
- Email (via SMTP sink)
- Incident management systems

## Next Steps

- See [07-security-monitoring](../07-security-monitoring) for security alerts
- See [09-dev-vs-prod](../09-dev-vs-prod) for environment-specific alerting