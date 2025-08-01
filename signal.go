package zlog

// Signal represents an event type in the logging system.
//
// Unlike traditional severity levels, signals categorize events by their meaning
// rather than their importance. This enables sophisticated routing where different
// types of events can be handled by different systems.
//
// While predefined signals are provided for compatibility with traditional logging,
// you are encouraged to define domain-specific signals:
//
//	const (
//	    PAYMENT_RECEIVED = Signal("PAYMENT_RECEIVED")
//	    USER_REGISTERED  = Signal("USER_REGISTERED")
//	    CACHE_MISS       = Signal("CACHE_MISS")
//	)
//
// Signals are just strings, making them easy to create and use. The routing
// system uses exact string matching to determine which sinks handle each signal.
type Signal string

// Standard logging signals provide compatibility with traditional level-based logging.
// These signals have implicit severity ordering when used with EnableStandardLogging.
const (
	// DEBUG indicates detailed information for diagnosing problems.
	// Typically disabled in production.
	DEBUG Signal = "DEBUG"

	// INFO indicates informational messages about normal operation.
	INFO Signal = "INFO"

	// WARN indicates potentially harmful situations that deserve attention.
	WARN Signal = "WARN"

	// ERROR indicates error events that might still allow the application to continue.
	ERROR Signal = "ERROR"

	// FATAL indicates severe errors that will cause the application to exit.
	FATAL Signal = "FATAL"
)

// Specialized signals for common use cases beyond traditional logging.
// These demonstrate how signals can represent domain concepts rather than severities.
const (
	// AUDIT events track user actions for compliance and forensics.
	// Route these to secure, tamper-proof storage.
	AUDIT Signal = "AUDIT"

	// SECURITY events indicate potential security issues.
	// Route these to security monitoring systems.
	SECURITY Signal = "SECURITY"

	// METRIC events carry measurement data for monitoring.
	// Route these to time-series databases or metrics aggregators.
	METRIC Signal = "METRIC"
)
