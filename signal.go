package zlog

// Signal represents a signal type for the logging system.
// Standard logging signals are provided as constants, but any string can be used as a custom signal.
type Signal string

// Standard logging signals for compatibility.
const (
	DEBUG Signal = "DEBUG"
	INFO  Signal = "INFO"
	WARN  Signal = "WARN"
	ERROR Signal = "ERROR"
	FATAL Signal = "FATAL"
)

// Specialized signals with clear use cases.
const (
	// Audit trail for compliance and security tracking.
	AUDIT    Signal = "AUDIT"
	SECURITY Signal = "SECURITY"

	// Metrics for monitoring and observability.
	METRIC Signal = "METRIC"
)
