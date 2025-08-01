# Custom Signals Example

This example shows how to use domain-specific signals instead of traditional severity levels. Business events get meaningful names and appropriate routing.

## What This Shows

- Defining custom signals for business events
- Routing different signals to different sinks
- Multiple sinks processing the same signal
- Specialized sinks for audit, metrics, and alerts
- Combining custom signals with standard logging

## Running the Example

```bash
go run main.go
```

## Signal Categories

### User Events
- `USER_REGISTERED` - New user account
- `USER_LOGIN` - Authentication success
- `PASSWORD_CHANGED` - Security event

### Commerce Events  
- `ORDER_PLACED` - Purchase initiated
- `PAYMENT_PROCESSED` - Payment success
- `PAYMENT_FAILED` - Payment issue (triggers alert)

### System Events
- `CACHE_HIT/MISS` - Performance metrics
- `FRAUD_DETECTED` - Security alert

## Routing Strategy

```
USER_* events      → Audit Sink
ORDER_PLACED       → Audit + Metrics + Analytics
PAYMENT_*          → Audit + Metrics (+ Alerts if failed)
CACHE_HIT          → Metrics (10% sampling)
CACHE_MISS         → Metrics (no sampling)
FRAUD_DETECTED     → Audit + Alerts
PRODUCT_VIEWED     → Analytics (25% sampling)
```

## New Features Demonstrated

### Variadic RouteSignal
Route to multiple sinks in one call:
```go
zlog.RouteSignal(ORDER_PLACED, auditSink, metricsSink, analyticsSink)
```

### Sampling for High-Volume Events
Reduce load while maintaining visibility:
```go
sampledSink := metricsSink.WithSampling(0.1) // 10% of events
zlog.RouteSignal(CACHE_HIT, sampledSink)
```

## Key Benefits

1. **Meaningful Names**: Events have business meaning, not severity
2. **Flexible Routing**: Each event type goes where it needs to
3. **Multiple Handlers**: One event can trigger multiple actions
4. **Separation of Concerns**: Metrics, audit, and alerts are independent