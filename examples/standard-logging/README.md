# Standard Logging Example

This example demonstrates using zlog as a traditional logger with familiar severity levels (DEBUG, INFO, WARN, ERROR, FATAL).

## What This Shows

- Using `EnableStandardLogging()` to set up JSON output to stderr
- Structured logging with typed fields
- Common web server logging patterns
- Application startup sequence logging
- Different log levels and when to use them

## Running the Example

```bash
go run main.go
```

The output will be JSON-formatted logs to stderr, perfect for:
- Local development (pipe to `jq` for pretty printing)
- Container environments (stdout/stderr collection)
- Log aggregation systems (ELK, Datadog, etc.)

## Key Concepts

1. **Log Levels**: Traditional severity-based logging
2. **Structured Fields**: Type-safe field constructors
3. **JSON Output**: Machine-readable format
4. **Caller Info**: Automatic file:line information

## Try It With Different Levels

```bash
# Modify the code to use DEBUG level:
# zlog.EnableStandardLogging(zlog.DEBUG)

# Then run again to see debug messages
go run main.go
```

## Pretty Print Output

```bash
go run main.go 2>&1 | jq '.'
```