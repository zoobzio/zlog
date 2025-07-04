package zlog

import (
	"fmt"
	"os"
	"time"
)

// Public API functions - Zero breaking changes

// Debug logs a debug message with structured fields
func Debug(msg string, fields ...ZlogField) {
	writeLog(DEBUG, msg, fields...)
}

// Info logs an info message with structured fields
func Info(msg string, fields ...ZlogField) {
	writeLog(INFO, msg, fields...)
}

// Warn logs a warning message with structured fields
func Warn(msg string, fields ...ZlogField) {
	writeLog(WARN, msg, fields...)
}

// Error logs an error message with structured fields
func Error(msg string, fields ...ZlogField) {
	writeLog(ERROR, msg, fields...)
}

// Fatal logs a fatal message with structured fields and exits
func Fatal(msg string, fields ...ZlogField) {
	writeLog(FATAL, msg, fields...)
	os.Exit(1)
}

// writeLog writes a log entry to the configured writer
func writeLog(level LogLevel, msg string, fields ...ZlogField) {
	// Simple JSON format for now
	entry := fmt.Sprintf(`{"time":"%s","level":"%s","msg":"%s"`, 
		time.Now().Format(time.RFC3339), 
		level.String(), 
		msg)
	
	for _, field := range fields {
		entry += fmt.Sprintf(`,"%s":%v`, field.Key, field.Value)
	}
	
	entry += "}\n"
	
	getWriter().Write([]byte(entry))
}
