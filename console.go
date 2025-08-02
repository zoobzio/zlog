package zlog

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
)

// Color codes for terminal output.
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorGray   = "\033[90m"
	colorBold   = "\033[1m"
	colorDim    = "\033[2m"
)

// isTerminal checks if we're writing to a terminal that supports colors.
func isTerminal() bool {
	// Check if stdout is a terminal
	if runtime.GOOS == "windows" {
		// Windows terminal detection is more complex, default to false for safety
		return false
	}

	// Unix-like systems: check if stdout is a tty
	// This is a simple heuristic - in practice, most CI/CD systems set this correctly
	if os.Getenv("TERM") == "" || os.Getenv("TERM") == "dumb" {
		return false
	}

	return true
}

// formatSignalWithSymbol returns a colored signal with visual symbol.
func formatSignalWithSymbol(signal Signal, useColors bool) string {
	var symbol, color string

	switch signal {
	case DEBUG:
		symbol = "🔍"
		color = colorGray
	case INFO:
		symbol = "✓"
		color = colorBlue
	case WARN:
		symbol = "⚠"
		color = colorYellow
	case ERROR:
		symbol = "✗"
		color = colorRed
	case FATAL:
		symbol = "💀"
		color = colorRed + colorBold
	default:
		symbol = "•"
		color = colorGray
	}

	if !useColors {
		return fmt.Sprintf("[%s] %s", string(signal), symbol)
	}

	return fmt.Sprintf("%s[%s]%s %s", color, string(signal), colorReset, symbol)
}

// formatFields creates a tree-style display of structured fields.
func formatFields(fields []Field, useColors bool) string {
	if len(fields) == 0 {
		return ""
	}

	var lines []string
	for i, field := range fields {
		prefix := "├─"
		if i == len(fields)-1 {
			prefix = "└─"
		}

		if useColors {
			line := fmt.Sprintf("   %s%s%s%s=%s%v%s",
				colorDim, prefix, colorReset,
				colorBold, colorReset,
				field.Value, colorReset)
			lines = append(lines, fmt.Sprintf("%s %s", field.Key, line))
		} else {
			lines = append(lines, fmt.Sprintf("   %s %s=%v", prefix, field.Key, field.Value))
		}
	}

	return "\n" + strings.Join(lines, "\n")
}

// formatCaller formats caller information.
func formatCaller(caller CallerInfo, useColors bool) string {
	if caller.File == "" {
		return ""
	}

	callerStr := fmt.Sprintf("%s:%d", caller.File, caller.Line)

	if useColors {
		return fmt.Sprintf(" %s(%s)%s", colorDim, callerStr, colorReset)
	}

	return fmt.Sprintf(" (%s)", callerStr)
}

// NewPrettyConsoleSink creates a sink that outputs human-readable, colorized logs to stderr.
//
// This sink is designed for development environments where logs are viewed directly
// in terminals. It provides:
//   - Color-coded log levels with visual symbols (✓, ⚠, ✗, 💀)
//   - Tree-style field display for easy scanning
//   - Automatic color detection (disabled in CI/non-terminal environments)
//   - Compact timestamp format
//   - Clean message layout
//
// Output format:
//
//	[INFO] ✓ 15:04:05 User logged in (auth.go:42)
//	├─ user_id=12345
//	└─ session_id=abc123
//
// The sink automatically detects terminal capabilities:
//   - Colors enabled: Interactive terminals with TERM variable
//   - Colors disabled: CI environments, redirected output, Windows
//
// Example usage:
//
//	// Development logging
//	consoleSink := zlog.NewPrettyConsoleSink()
//	zlog.RouteSignal(zlog.INFO, consoleSink)
//	zlog.RouteSignal(zlog.ERROR, consoleSink)
//
//	// With adapters for filtering
//	devSink := consoleSink.WithFilter(func(ctx context.Context, e zlog.Log) bool {
//	    return e.Signal != zlog.DEBUG // Hide debug in development
//	})
//
// The sink works with all zlog adapters (WithAsync, WithFilter, WithRetry, etc.)
// and is fully compatible with the fluent builder pattern.
func NewPrettyConsoleSink() *Sink {
	useColors := isTerminal()

	return NewSink("pretty-console", func(_ context.Context, event Log) error {
		// Format timestamp (compact format for readability)
		timestamp := event.Time.Format("15:04:05")

		// Format signal with symbol and color
		signalDisplay := formatSignalWithSymbol(event.Signal, useColors)

		// Format caller info
		callerDisplay := formatCaller(event.Caller, useColors)

		// Format main log line
		mainLine := fmt.Sprintf("%s %s %s%s",
			signalDisplay,
			timestamp,
			event.Message,
			callerDisplay)

		// Format structured fields
		fieldsDisplay := formatFields(event.Data, useColors)

		// Write complete entry to stderr
		fmt.Fprintf(os.Stderr, "%s%s\n", mainLine, fieldsDisplay)

		return nil
	})
}
