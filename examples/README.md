# zlog Examples

This directory contains practical examples demonstrating zlog's capabilities, from simple logging to complex signal routing patterns.

## Examples Overview

### Basic Usage

#### [01-basic-logging](./01-basic-logging)
Simple application logging with structured fields. Shows the minimal setup needed to get started with zlog.

```go
zlog.EnableStandardLogging(os.Stderr)
zlog.Info("Server starting", zlog.Int("port", 8080))
```

#### [02-debug-separation](./02-debug-separation)
Demonstrates separating debug logs from standard logs, sending them to different destinations.

```go
zlog.EnableDebugLogging(debugFile)
zlog.EnableStandardLogging(os.Stderr)
```

### Specialized Logging

#### [03-audit-trail](./03-audit-trail)
Implements a compliance-ready audit trail that tracks sensitive operations separately from application logs.

```go
zlog.EnableAuditLogging(auditFile)
zlog.Emit(zlog.AUDIT, "User permission changed", ...)
```

#### [04-metrics-pipeline](./04-metrics-pipeline)
Shows how to use zlog as a metrics collection system, routing metric signals to Prometheus.

```go
zlog.RouteSignal(METRIC, prometheusSink)
zlog.Emit(METRIC, "api_latency", zlog.Float64("value", 0.125))
```

### Advanced Patterns

#### [05-error-alerting](./05-error-alerting)
Multi-destination error routing - errors go to logs, Slack, and PagerDuty based on severity.

```go
zlog.RouteSignal(zlog.ERROR, slackSink)
zlog.RouteSignal(zlog.FATAL, pagerDutySink)
```

#### [06-request-tracing](./06-request-tracing)
HTTP middleware that tracks request lifecycle events for analysis and debugging.

```go
zlog.Emit(REQUEST_START, "Request received", ...)
zlog.Emit(REQUEST_END, "Request completed", ...)
```

#### [07-security-monitoring](./07-security-monitoring)
Security event monitoring with SIEM integration for login attempts, failures, and permission issues.

```go
zlog.RouteSignal(LOGIN_FAILED, siemSink)
```

#### [08-business-events](./08-business-events)
Tracks business events (orders, payments, shipments) and routes them to a data warehouse.

```go
zlog.Emit(ORDER_PLACED, "New order", zlog.Float64("total", order.Total))
```

### Environment-Specific

#### [09-dev-vs-prod](./09-dev-vs-prod)
Different logging configurations for development and production environments, including sampling.

```go
// Dev: everything to stdout
// Prod: structured logs with sampling
```

#### [10-distributed-tracing](./10-distributed-tracing)
Integration with OpenTelemetry to bridge logs with distributed traces.

```go
// All logs become trace events
zlog.RouteSignal(zlog.INFO, otelSink)
```

## Running the Examples

Each example includes:
- `main.go` - The example implementation
- `main_test.go` - Tests demonstrating the behavior
- `README.md` - Detailed explanation

To run an example:
```bash
cd examples/01-basic-logging
go run main.go
```

To test an example:
```bash
cd examples/01-basic-logging
go test -v
```

## Common Patterns

### Creating Custom Sinks

Most examples show how to create custom sinks for specific purposes:

```go
type CustomSink struct {
    // sink-specific fields
}

func (s *CustomSink) Write(event zlog.Event) error {
    // Process the event
    return nil
}

func (s *CustomSink) Name() string {
    return "custom"
}
```

### Defining Custom Signals

Many examples define domain-specific signals:

```go
const (
    PAYMENT   zlog.Signal = "PAYMENT"
    SHIPMENT  zlog.Signal = "SHIPMENT"
)
```

### Self-Registering Sinks

Examples follow the pattern of sinks that register themselves:

```go
func NewCustomSink() zlog.Sink {
    sink := &CustomSink{}
    zlog.RouteSignal(CUSTOM_SIGNAL, sink)
    return sink
}
```

## Learn More

- Start with [01-basic-logging](./01-basic-logging) for the simplest use case
- See [04-metrics-pipeline](./04-metrics-pipeline) for non-logging use cases
- Check [05-error-alerting](./05-error-alerting) for multi-sink patterns
- Explore [09-dev-vs-prod](./09-dev-vs-prod) for environment-specific setups