# Installation

## Requirements

- Go 1.21 or later (for generics support)
- No external dependencies beyond the standard library

## Install with go get

```bash
go get github.com/zoobzio/zlog
```

## Verify Installation

Create a simple test file to verify zlog is working:

```go
// test.go
package main

import "github.com/zoobzio/zlog"

func main() {
    zlog.EnableStandardLogging(zlog.INFO)
    zlog.Info("zlog is working!", zlog.String("version", "latest"))
}
```

Run it:
```bash
go run test.go
```

You should see JSON output like:
```json
{"time":"2023-10-20T15:04:05Z","signal":"INFO","message":"zlog is working!","caller":"test.go:7","version":"latest"}
```

## Version Pinning

Pin to a specific version in production:

```bash
go get github.com/zoobzio/zlog@v1.0.0
```

Or in your `go.mod`:
```go
require github.com/zoobzio/zlog v1.0.0
```

## Development Setup

For contributing or local development:

```bash
git clone https://github.com/zoobzio/zlog.git
cd zlog
go test ./...
```