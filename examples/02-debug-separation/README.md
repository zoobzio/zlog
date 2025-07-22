# Example 02: Debug Separation

This example shows how to separate debug logs from standard application logs, sending them to different destinations.

## What It Shows

- How to enable debug logging separately from standard logging
- Routing different log levels to different outputs
- Why debug separation is useful in production

## Key Concepts

1. **Debug logs are noisy** - Often contain implementation details
2. **Separate destinations** - Debug to file, standard logs to stderr
3. **Easy to disable** - Just don't call EnableDebugLogging() in production

## Running the Example

```bash
go run main.go
```

You'll see:
- Standard logs (INFO, WARN, ERROR) go to stderr (console output)
- Debug logs go to `debug.log` file

Expected console output:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Server starting","caller":"main.go:19","port":8080}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Request received","caller":"main.go:35","method":"GET","path":"/api/users"}
{"time":"2024-01-20T10:30:45Z","signal":"WARN","message":"Slow query","caller":"main.go:40","duration_ms":1500}
```

Expected debug.log content:
```json
{"time":"2024-01-20T10:30:45Z","signal":"DEBUG","message":"Configuration loaded","caller":"main.go:25","config_path":"/etc/app/config.json"}
{"time":"2024-01-20T10:30:45Z","signal":"DEBUG","message":"Database connection pool initialized","caller":"main.go:26","pool_size":10}
{"time":"2024-01-20T10:30:45Z","signal":"DEBUG","message":"Query execution","caller":"main.go:38","sql":"SELECT * FROM users WHERE active = true","rows":42}
```

## Use Cases

- **Development**: Enable debug logs to see everything
- **Production**: Disable debug logs or send to separate file
- **Troubleshooting**: Temporarily enable debug logs for specific issues

## Next Steps

- See [03-audit-trail](../03-audit-trail) for custom signal types
- See [09-dev-vs-prod](../09-dev-vs-prod) for environment-based configuration