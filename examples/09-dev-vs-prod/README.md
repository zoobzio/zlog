# Example 09: Dev vs Prod

This example shows how to configure zlog differently for development and production environments, demonstrating environment-specific logging strategies.

## What It Shows

- Environment-based sink configuration
- Different log verbosity for dev vs prod
- Local file logging in dev, cloud logging in prod
- Performance considerations for production

## Key Concepts

1. **Environment detection** - Use env vars to determine context
2. **Dev verbosity** - Everything including DEBUG in development
3. **Prod efficiency** - Only what's needed in production
4. **Different sinks** - Console for dev, structured logs for prod

## Running the Example

Development mode:
```bash
ENV=development go run main.go
```

Production mode:
```bash
ENV=production go run main.go
```

## Development Output

In development, you see everything:
```json
{"time":"2024-01-20T10:30:45Z","signal":"DEBUG","message":"Database connection pool stats","caller":"main.go:25","pool_size":10,"active":3}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Request handled","caller":"main.go:30","duration_ms":45}
{"time":"2024-01-20T10:30:45Z","signal":"METRIC","message":"response.time","caller":"main.go:35","value":45}
```

## Production Output

In production, only essential logs:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Request handled","caller":"main.go:30","duration_ms":45}
{"time":"2024-01-20T10:30:45Z","signal":"ERROR","message":"Database query failed","caller":"main.go:40","error":"connection timeout"}
```

## Configuration Strategies

### Development
- All signals to console (human-readable)
- DEBUG enabled for troubleshooting
- Local file storage for persistence
- No sampling or filtering

### Production
- Structured JSON to stdout for log aggregators
- DEBUG disabled to reduce volume
- Metrics to separate pipeline
- Sampling for high-volume events

## Performance Considerations

- **Dev**: Readability over performance
- **Prod**: Optimize for volume and parsing
- **Sampling**: Reduce noise in production
- **Buffering**: Batch writes in production

## Next Steps

- See [02-debug-separation](../02-debug-separation) for debug routing
- See [10-distributed-tracing](../10-distributed-tracing) for production tracing