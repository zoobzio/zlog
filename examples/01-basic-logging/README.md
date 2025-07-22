# Example 01: Basic Logging

This example demonstrates the simplest use case for zlog - basic application logging with structured fields.

## What It Shows

- How to enable standard logging
- Using different log levels (Info, Warn, Error)
- Adding structured fields to log entries
- JSON output format

## Key Concepts

1. **Enable logging first** - zlog has no default output
2. **Structured fields** - Use typed field constructors
3. **Caller information** - Automatically captured

## Running the Example

```bash
go run main.go
```

Expected output (to stderr):
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Application starting","caller":"main.go:12","version":"1.0.0","pid":12345}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Database connected","caller":"main.go:18","host":"localhost","port":5432}
{"time":"2024-01-20T10:30:45Z","signal":"WARN","message":"Cache miss","caller":"main.go:24","key":"user:123"}
{"time":"2024-01-20T10:30:45Z","signal":"ERROR","message":"Failed to send email","caller":"main.go:30","error":"connection timeout","recipient":"user@example.com"}
```

## Next Steps

- See [02-debug-separation](../02-debug-separation) to learn about separating debug logs
- See [03-audit-trail](../03-audit-trail) for compliance logging patterns