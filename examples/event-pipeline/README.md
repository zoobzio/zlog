# Event Pipeline Example

This example demonstrates using zlog as a complete event processing pipeline and application event bus.

## What This Shows

- Event correlation across user sessions
- Metrics extraction and aggregation
- Business analytics processing
- Real-time alerting for critical events
- Audit trail for compliance
- Multiple sinks working together

## Running the Example

```bash
go run main.go
```

## Architecture

```
Events → zlog → ┌─→ Correlation Engine
                ├─→ Metrics Aggregator
                ├─→ Business Analytics
                ├─→ Alerting System
                ├─→ Audit Logger
                └─→ File Storage
```

## Event Categories

### User Events
Track user lifecycle and actions:
- `USER_LOGIN` - Authentication events
- `USER_SIGNUP` - New registrations
- `USER_PROFILE_UPDATE` - Profile changes

### Commerce Events
Business-critical transactions:
- `ORDER_PLACED` - New orders
- `PAYMENT_PROCESSED` - Successful payments
- `CART_UPDATED` - Shopping behavior

### System Events
Technical and operational:
- `CACHE_HIT/MISS` - Performance tracking
- `API_CALLED` - Request monitoring
- `RATE_LIMITED` - Security events

## Key Features

### Event Correlation
Links related events by session ID to track complete user journeys.

### Metrics Aggregation
Automatically extracts and aggregates:
- Event counts by type
- Response times
- Business metrics (cart value, order totals)

### Multi-Sink Processing
Each event can trigger multiple actions:
- Write to file
- Update metrics
- Send alerts
- Log audit trail

## Benefits

1. **Single Source of Truth**: All events flow through zlog
2. **Flexible Routing**: Easy to add new handlers
3. **Performance**: Async processing doesn't block
4. **Observability**: Complete visibility into system behavior
5. **Compliance**: Audit trail built-in