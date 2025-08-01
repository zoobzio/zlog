# zlog Examples

These examples demonstrate key patterns and use cases for zlog's signal-based logging system.

## Examples Overview

### 1. [Standard Logging](./standard-logging/)
Shows traditional logging patterns using zlog's convenience functions.
- Migrating from other loggers
- Using severity levels (DEBUG, INFO, WARN, ERROR)
- Structured logging with typed fields

### 2. [Custom Signals](./custom-signals/)
Demonstrates domain-specific signals for business events.
- Defining meaningful signals (PAYMENT_PROCESSED, USER_LOGIN)
- Routing different signals to different sinks
- Separating audit, metrics, and alerts

### 3. [Custom Sink](./custom-sink/)
Building sinks to integrate with external systems.
- Metrics collection (Prometheus/StatsD style)
- Message queue publishing
- Database audit logging
- Conditional and batching patterns

### 4. [Custom Fields](./custom-fields/)
Creating field transformers for security and compliance.
- Redacting sensitive data (credit cards, SSNs)
- Masking emails and IPs
- Hashing for correlation without exposure
- Compliance tagging

### 5. [Event Pipeline](./event-pipeline/)
Using zlog as a complete event processing pipeline.
- Event correlation across sessions
- Real-time metrics aggregation
- Multi-sink processing
- Building an application event bus

## Running Examples

Each example is self-contained:

```bash
cd examples/standard-logging
go run main.go
```

## Key Concepts

1. **Signals Over Severity**: Events have business meaning, not just log levels
2. **Flexible Routing**: Different events go to different destinations
3. **Composable Sinks**: Build complex behavior from simple pieces
4. **Type-Safe Fields**: Structured data with compile-time safety
5. **Performance**: Async processing and minimal allocations

## Learning Path

1. Start with **Standard Logging** if coming from traditional loggers
2. Move to **Custom Signals** to see the real power of zlog
3. Explore **Custom Sinks** to integrate with your infrastructure
4. Use **Custom Fields** for security and compliance needs
5. Study **Event Pipeline** for a complete architectural example