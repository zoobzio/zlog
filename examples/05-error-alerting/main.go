package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime/debug"
	"strings"
	"time"

	"github.com/zoobzio/zlog"
)

// Define a custom ALERT signal for critical errors.
const ALERT = zlog.Signal("ALERT")

func main() {
	// Standard logs to stderr
	zlog.EnableStandardLogging(os.Stderr)

	// Critical alerts to a separate file
	alertFile, err := os.Create("alerts.json")
	if err != nil {
		zlog.Fatal("Failed to create alerts file", zlog.Err(err))
	}
	defer alertFile.Close()

	// Route ALERT signals to the alerts file
	alertSink := zlog.NewWriterSink(alertFile)
	zlog.RouteSignal(ALERT, alertSink)

	// Start the payment service
	zlog.Info("Payment service started")

	service := &PaymentService{
		dbConnected: true,
		gatewayURL:  "https://payment-gateway.example.com",
	}

	// Process some payments
	service.ProcessBatch()

	zlog.Info("Payment service shutdown")
}

type PaymentService struct {
	gatewayURL     string
	processedCount int
	failureCount   int
	dbConnected    bool
}

func (s *PaymentService) ProcessBatch() {
	payments := []Payment{
		{ID: "pay_001", UserID: "user123", Amount: 100.00},
		{ID: "pay_002", UserID: "user456", Amount: 250.00},
		{ID: "pay_003", UserID: "user789", Amount: 50.00},
		{ID: "pay_004", UserID: "user123", Amount: 1000000.00}, // Will fail
		{ID: "pay_005", UserID: "user456", Amount: 75.00},
	}

	// Simulate database connection failure
	s.dbConnected = false

	for _, payment := range payments {
		if err := s.ProcessPayment(payment); err != nil {
			s.failureCount++

			// Determine if this error should trigger an alert
			if shouldAlert(err) {
				s.sendAlert(err, payment)
			} else {
				// Regular error logging
				zlog.Error("Failed to process payment",
					zlog.Err(err),
					zlog.String("payment_id", payment.ID),
					zlog.String("user_id", payment.UserID),
				)
			}
		} else {
			s.processedCount++
		}
	}

	zlog.Info("Batch processing complete",
		zlog.Int("processed", s.processedCount),
		zlog.Int("failures", s.failureCount),
	)
}

func (s *PaymentService) ProcessPayment(payment Payment) error {
	// Check database connection
	if !s.dbConnected {
		return &CriticalError{
			Err:         errors.New("database connection lost"),
			Fingerprint: "db_connection_lost",
			Severity:    "critical",
		}
	}

	// Validate payment
	if payment.Amount > 10000 {
		return fmt.Errorf("payment amount exceeds limit: %.2f", payment.Amount)
	}

	// Simulate gateway timeout
	if payment.ID == "pay_003" {
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		// Simulate slow gateway
		time.Sleep(200 * time.Millisecond)

		select {
		case <-ctx.Done():
			return &CriticalError{
				Err:         ctx.Err(),
				Fingerprint: "gateway_timeout",
				Severity:    "high",
				Context: map[string]interface{}{
					"gateway_url": s.gatewayURL,
					"timeout_ms":  100,
				},
			}
		default:
		}
	}

	return nil
}

func (s *PaymentService) sendAlert(err error, payment Payment) {
	fields := []zlog.Field{
		zlog.Err(err),
		zlog.String("service", "payment-api"),
		zlog.String("payment_id", payment.ID),
		zlog.String("user_id", payment.UserID),
		zlog.Float64("amount", payment.Amount),
	}

	// Add error-specific context
	if critErr, ok := err.(*CriticalError); ok {
		fields = append(fields,
			zlog.String("severity", critErr.Severity),
			zlog.String("fingerprint", critErr.Fingerprint),
		)

		// Add any additional context
		for k, v := range critErr.Context {
			fields = append(fields, zlog.Data(k, v))
		}

		// Capture stack trace for critical errors
		if critErr.Severity == "critical" {
			fields = append(fields, zlog.String("stack_trace", captureStackTrace()))
		}
	}

	// Emit the alert
	zlog.Emit(ALERT, "Critical error in payment processing", fields...)
}

// shouldAlert determines if an error warrants an alert.
func shouldAlert(err error) bool {
	// Check if it's a critical error type
	var critErr *CriticalError
	if errors.As(err, &critErr) {
		return true
	}

	// Check for specific error conditions
	errMsg := err.Error()
	alertKeywords := []string{
		"connection refused",
		"connection lost",
		"timeout",
		"deadline exceeded",
		"internal server error",
		"panic",
	}

	for _, keyword := range alertKeywords {
		if strings.Contains(strings.ToLower(errMsg), keyword) {
			return true
		}
	}

	return false
}

// captureStackTrace returns a simplified stack trace.
func captureStackTrace() string {
	stack := string(debug.Stack())
	lines := strings.Split(stack, "\n")

	// Skip runtime frames and focus on app frames
	var appFrames []string
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		// Look for function names in our package
		if strings.Contains(line, "main.") && !strings.Contains(line, "runtime") {
			// Next line usually has the file:line info
			if i+1 < len(lines) {
				funcName := strings.TrimSpace(line)
				fileLine := strings.TrimSpace(lines[i+1])
				// Combine function and location
				appFrames = append(appFrames, funcName+" at "+fileLine)
				i++ // Skip the file line we just processed
			}
		}
	}

	if len(appFrames) == 0 {
		return "stack trace unavailable"
	}

	return strings.Join(appFrames, " <- ")
}

// CriticalError represents an error that should trigger an alert.
type CriticalError struct {
	Err         error
	Context     map[string]interface{}
	Severity    string
	Fingerprint string
}

func (e *CriticalError) Error() string {
	return e.Err.Error()
}

func (e *CriticalError) Unwrap() error {
	return e.Err
}

// Payment represents a payment to process.
type Payment struct {
	ID     string
	UserID string
	Amount float64
}

func init() {
	// Clean up from previous runs
	os.Remove("alerts.json")
}
