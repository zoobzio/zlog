# Understanding Modules

Modules are reusable logging configurations that package sinks and routing logic together. They make it easy to add specific logging capabilities to your application with a single function call.

## What Are Modules?

A module is simply a function that creates sinks and sets up routing. zlog's built-in `EnableStandardLogging()` is a module - it creates a JSON sink and routes traditional log levels to it.

```go
// EnableStandardLogging is a module
func EnableStandardLogging(level Signal) {
    sink := stderrJSONSink  // Create sink
    
    // Set up routing based on level
    switch level {
    case DEBUG:
        RouteSignal(DEBUG, sink)
        fallthrough
    case INFO:
        RouteSignal(INFO, sink)
        // ... more routing
    }
}
```

## Module Pattern

The basic module pattern follows this structure:

```go
// 1. Create sinks (usually as package variables for reuse)
var mySink = NewSink("module-name", func(ctx context.Context, event Event) error {
    // Process events
    return nil
})

// 2. Provide enable function that sets up routing
func EnableMyModule(config MyConfig) error {
    // Optional: configure sink based on config
    if err := configureSink(mySink, config); err != nil {
        return err
    }
    
    // Set up routing
    RouteSignal(MY_SIGNAL_1, mySink)
    RouteSignal(MY_SIGNAL_2, mySink)
    
    return nil
}
```

## Built-in Modules

### Standard Logging Module

The standard logging module (`log.go`) provides traditional level-based logging:

```go
// Enable traditional logging to stderr
zlog.EnableStandardLogging(zlog.INFO)

// Now you can use familiar log levels
zlog.Debug("This won't show with INFO level")
zlog.Info("Server starting")
zlog.Error("Connection failed", zlog.Err(err))
```

This module:
- Creates a JSON formatter sink that writes to stderr
- Routes DEBUG, INFO, WARN, ERROR, FATAL based on the level parameter
- Provides familiar migration path from other loggers

## Creating Custom Modules

### Simple Module Example

A basic metrics module that tracks event counts:

```go
// metrics_module.go
package myapp

import (
    "context"
    "github.com/zoobzio/zlog"
    "github.com/prometheus/client_golang/prometheus"
)

var (
    eventCounter = prometheus.NewCounterVec(
        prometheus.CounterOpts{
            Name: "app_events_total",
            Help: "Total number of application events",
        },
        []string{"signal"},
    )
    
    metricsSink = zlog.NewSink("prometheus-metrics", func(ctx context.Context, event zlog.Event) error {
        eventCounter.WithLabelValues(string(event.Signal)).Inc()
        return nil
    })
)

func EnableMetrics() {
    // Register Prometheus metrics
    prometheus.MustRegister(eventCounter)
    
    // Route all events to metrics
    zlog.RouteSignal(zlog.DEBUG, metricsSink)
    zlog.RouteSignal(zlog.INFO, metricsSink)
    zlog.RouteSignal(zlog.WARN, metricsSink)
    zlog.RouteSignal(zlog.ERROR, metricsSink)
    zlog.RouteSignal(zlog.FATAL, metricsSink)
    
    // Route custom signals too
    zlog.RouteSignal(USER_REGISTERED, metricsSink)
    zlog.RouteSignal(PAYMENT_RECEIVED, metricsSink)
}
```

### Configurable Module Example

A module with configuration options:

```go
// audit_module.go
package myapp

import (
    "context"
    "database/sql"
    "github.com/zoobzio/zlog"
)

type AuditConfig struct {
    DatabaseURL string
    TableName   string
    Enabled     bool
}

var auditSink zlog.Sink

func EnableAuditLogging(config AuditConfig) error {
    if !config.Enabled {
        return nil  // Module disabled
    }
    
    // Connect to database
    db, err := sql.Open("postgres", config.DatabaseURL)
    if err != nil {
        return err
    }
    
    // Create sink with database connection
    auditSink = zlog.NewSink("audit-db", func(ctx context.Context, event zlog.Event) error {
        // Insert event into audit table
        _, err := db.ExecContext(ctx, 
            "INSERT INTO "+config.TableName+" (timestamp, signal, message, fields) VALUES ($1, $2, $3, $4)",
            event.Time, event.Signal, event.Message, fieldsToJSON(event.Fields))
        return err
    })
    
    // Route audit-relevant signals
    zlog.RouteSignal(USER_REGISTERED, auditSink)
    zlog.RouteSignal(USER_DELETED, auditSink)
    zlog.RouteSignal(PAYMENT_RECEIVED, auditSink)
    zlog.RouteSignal(DATA_EXPORTED, auditSink)
    zlog.RouteSignal(ADMIN_ACTION, auditSink)
    
    return nil
}
```

### Multi-Sink Module Example

A SIEM module with multiple destinations:

```go
// siem_module.go
package myapp

import (
    "context"
    "github.com/zoobzio/zlog"
)

type SIEMConfig struct {
    SplunkURL    string
    SplunkToken  string
    SyslogHost   string
    LocalBackup  string
}

func EnableSIEMLogging(config SIEMConfig) error {
    // Primary: Splunk
    splunkSink := zlog.NewSink("splunk", func(ctx context.Context, event zlog.Event) error {
        return sendToSplunk(config.SplunkURL, config.SplunkToken, event)
    })
    
    // Secondary: Syslog
    syslogSink := zlog.NewSink("syslog", func(ctx context.Context, event zlog.Event) error {
        return sendToSyslog(config.SyslogHost, event)
    })
    
    // Backup: Local file
    backupSink := zlog.NewSink("local-backup", func(ctx context.Context, event zlog.Event) error {
        return appendToFile(config.LocalBackup, event)
    })
    
    // Route security events to all three
    securitySignals := []zlog.Signal{
        zlog.SECURITY,
        SECURITY_VIOLATION,
        UNAUTHORIZED_ACCESS,
        PRIVILEGE_ESCALATION,
        SUSPICIOUS_ACTIVITY,
    }
    
    for _, signal := range securitySignals {
        zlog.RouteSignal(signal, splunkSink)
        zlog.RouteSignal(signal, syslogSink)
        zlog.RouteSignal(signal, backupSink)
    }
    
    return nil
}
```

## Module Composition

Modules can be composed together:

```go
func SetupLogging() error {
    // Enable standard console logging
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Add metrics tracking
    if err := EnableMetrics(); err != nil {
        return err
    }
    
    // Add audit logging in production
    if os.Getenv("ENV") == "production" {
        auditConfig := AuditConfig{
            DatabaseURL: os.Getenv("AUDIT_DB_URL"),
            TableName:   "audit_events",
            Enabled:     true,
        }
        if err := EnableAuditLogging(auditConfig); err != nil {
            return err
        }
    }
    
    // Add SIEM in production
    if os.Getenv("ENV") == "production" {
        siemConfig := SIEMConfig{
            SplunkURL:   os.Getenv("SPLUNK_URL"),
            SplunkToken: os.Getenv("SPLUNK_TOKEN"),
            SyslogHost:  os.Getenv("SYSLOG_HOST"),
            LocalBackup: "/var/log/security.log",
        }
        if err := EnableSIEMLogging(siemConfig); err != nil {
            return err
        }
    }
    
    return nil
}
```

## Module Best Practices

### 1. Single Responsibility

Each module should have a clear, single purpose:

```go
// Good - focused on one concern
func EnableSlackAlerts(webhookURL string) error { /* ... */ }
func EnableMetricsCollection() error { /* ... */ }
func EnableAuditTrail(dbURL string) error { /* ... */ }

// Avoid - doing too much
func EnableEverything(config MegaConfig) error { /* ... */ }
```

### 2. Configuration

Accept configuration parameters for flexibility:

```go
type EmailConfig struct {
    SMTPHost     string
    SMTPPort     int
    Username     string
    Password     string
    FromAddress  string
    AlertAddress string
    Enabled      bool
}

func EnableEmailAlerts(config EmailConfig) error {
    if !config.Enabled {
        return nil
    }
    // ... setup
}
```

### 3. Error Handling

Handle errors gracefully and provide useful messages:

```go
func EnableDatabaseLogging(dbURL string) error {
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        return fmt.Errorf("failed to connect to database for logging: %w", err)
    }
    
    if err := db.Ping(); err != nil {
        return fmt.Errorf("database connection test failed: %w", err)
    }
    
    // ... rest of setup
    return nil
}
```

### 4. Resource Management

Consider resource cleanup:

```go
func EnableFileLogging(filename string) error {
    file, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return err
    }
    
    fileSink := zlog.NewSink("file-logger", func(ctx context.Context, event zlog.Event) error {
        // Note: In real code, you'd want to handle file rotation and cleanup
        _, err := fmt.Fprintf(file, "%s\n", formatEvent(event))
        return err
    })
    
    zlog.RouteSignal(zlog.INFO, fileSink)
    zlog.RouteSignal(zlog.ERROR, fileSink)
    
    return nil
}
```

### 5. Testing

Make modules testable:

```go
func TestEnableMetrics(t *testing.T) {
    // Save original state
    originalRegistry := prometheus.DefaultRegisterer
    defer func() { prometheus.DefaultRegisterer = originalRegistry }()
    
    // Use test registry
    testRegistry := prometheus.NewRegistry()
    prometheus.DefaultRegisterer = testRegistry
    
    // Test module
    err := EnableMetrics()
    assert.NoError(t, err)
    
    // Verify metrics are registered
    families, err := testRegistry.Gather()
    assert.NoError(t, err)
    assert.Greater(t, len(families), 0)
}
```

## Module Organization

### File Structure

Organize modules in separate files:

```
myapp/
   logging/
      standard.go      // EnableStandardLogging
      metrics.go       // EnableMetrics
      audit.go         // EnableAuditLogging
      alerts.go        // EnableSlackAlerts, EnableEmailAlerts
      siem.go          // EnableSIEMLogging
   main.go
   config.go
```

### Package Structure

For larger applications, consider a dedicated logging package:

```go
// internal/logging/logging.go
package logging

import "github.com/zoobzio/zlog"

type Config struct {
    Level        zlog.Signal
    MetricsEnabled bool
    AuditConfig    AuditConfig
    SIEMConfig     SIEMConfig
}

func Setup(config Config) error {
    zlog.EnableStandardLogging(config.Level)
    
    if config.MetricsEnabled {
        if err := EnableMetrics(); err != nil {
            return err
        }
    }
    
    // ... more setup
    return nil
}
```

## Third-Party Modules

Modules can be distributed as separate packages:

```go
// go get github.com/company/zlog-datadog
import "github.com/company/zlog-datadog"

func main() {
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Enable third-party module
    datadog.EnableLogging(datadog.Config{
        APIKey: os.Getenv("DATADOG_API_KEY"),
        Tags:   []string{"service:myapp", "env:production"},
    })
}
```

Modules are the key to making zlog extensible and reusable. By following the module pattern, you can create focused, composable logging capabilities that can be easily shared and maintained.