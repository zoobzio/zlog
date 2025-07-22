# Example 04: Metrics Pipeline

This example shows how to use the METRIC signal to create a metrics collection pipeline separate from logs.

## What It Shows

- Using the METRIC signal for performance data
- Separating metrics from logs for specialized processing
- Structured metric fields (counters, gauges, histograms)
- How metrics can feed into monitoring systems

## Key Concepts

1. **Metrics are not logs** - They're structured data points for monitoring
2. **Standard fields** - name, value, unit, tags for compatibility
3. **Pipeline ready** - Metrics can be routed to Prometheus, StatsD, etc.
4. **High volume** - Metrics often outnumber logs 100:1

## Running the Example

```bash
go run main.go
```

You'll see:
- Standard logs go to stderr
- Metrics go to `metrics.json` file (could be a metrics aggregator)

Expected console output:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Server started","caller":"main.go:15","port":8080}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Metrics collection enabled","caller":"main.go:20"}
```

Expected metrics.json content:
```json
{"time":"2024-01-20T10:30:45Z","signal":"METRIC","message":"http.request.duration","caller":"main.go:25","value":125.5,"unit":"ms","method":"GET","status":200}
{"time":"2024-01-20T10:30:45Z","signal":"METRIC","message":"memory.usage","caller":"main.go:30","value":45.2,"unit":"MB","type":"heap"}
{"time":"2024-01-20T10:30:45Z","signal":"METRIC","message":"active.connections","caller":"main.go:35","value":42,"unit":"count"}
```

## Use Cases

- **Performance monitoring**: Track request latencies, throughput
- **Resource monitoring**: Memory, CPU, disk usage
- **Business metrics**: User signups, revenue, conversion rates
- **SLI/SLO tracking**: Availability, error rates, latency percentiles

## Integration Points

In production, metrics would typically go to:
- Prometheus (via an exporter sink)
- StatsD (via UDP sink)
- CloudWatch (via AWS SDK sink)
- DataDog (via agent sink)

## Next Steps

- See [05-error-alerting](../05-error-alerting) for error tracking
- See [08-business-events](../08-business-events) for business metrics