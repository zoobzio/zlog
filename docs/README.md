# zlog Documentation

Welcome to the zlog documentation! This guide covers everything you need to know about signal-based structured logging with zlog.

## Documentation Structure

### Getting Started
- [Introduction](./introduction.md) - Why signal-based logging beats traditional levels
- [Quick Start](./quick-start.md) - Get logging in 5 minutes
- [Installation](./installation.md) - Installation and setup instructions

### Core Concepts
- [Signals](./concepts/signals.md) - Understanding signals vs traditional log levels
- [Events](./concepts/events.md) - Event structure and structured fields
- [Routing](./concepts/routing.md) - How signal-based routing works
- [Sinks](./concepts/sinks.md) - Creating and using sinks
- [Modules](./concepts/modules.md) - The module pattern for reusable configurations

### Guides
- [Deployment](./guides/deployment.md) - Deployment considerations and configuration patterns
- [Performance](./guides/performance.md) - Performance characteristics and optimization
- [Testing](./guides/testing.md) - Testing applications that use zlog
- [Best Practices](./guides/best-practices.md) - Recommended patterns and conventions
- [Advanced Pipelines](./guides/advanced-pipelines.md) - Sophisticated event processing with pipz

### Real-World Examples
- [Web Service Logging](./examples/web-service.md) - HTTP APIs and request tracking
- [Background Jobs](./examples/background-jobs.md) - Worker queues and batch processing
- [Microservices](./examples/microservices.md) - Distributed system logging patterns
- [Audit & Compliance](./examples/audit-compliance.md) - Regulatory and security logging
- [Custom Modules](./examples/custom-modules.md) - Building your own logging modules

### API Reference
- [Core Functions](./api/core.md) - Emit, Debug, Info, Error, Fatal
- [Field Constructors](./api/fields.md) - String, Int, Err, Data, etc.
- [Routing Functions](./api/routing.md) - RouteSignal and signal management
- [Standard Module](./api/standard-module.md) - EnableStandardLogging and built-in signals

### Patterns & Design
- [Signal Design](./patterns/signal-design.md) - How to design effective signals
- [Multi-Destination Routing](./patterns/multi-destination.md) - Routing events to multiple sinks
- [Error Handling](./patterns/error-handling.md) - Handling errors in sinks gracefully

## Quick Links

- [GitHub Repository](https://github.com/zoobzio/zlog)
- [Go Package Documentation](https://pkg.go.dev/github.com/zoobzio/zlog)
- [Contributing Guide](../CONTRIBUTING.md)

## Philosophy

zlog is built on the principle that **events have types, not severities**. Instead of forcing you to choose between debug/info/warn/error, zlog lets you route events based on what they actually represent: payments, user actions, system metrics, security events, etc.

This approach enables sophisticated logging architectures where different types of events flow to different destinations automatically, making your logs more useful and your systems more observable.