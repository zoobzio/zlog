package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/zoobzio/zlog"
)

func TestErrorAlerting(t *testing.T) {
	// Set up separate outputs
	logBuf := &bytes.Buffer{}
	alertBuf := &bytes.Buffer{}

	zlog.EnableStandardLogging(logBuf)
	zlog.RouteSignal(ALERT, zlog.NewWriterSink(alertBuf))

	t.Run("AlertSeparation", func(t *testing.T) {
		// Regular error
		zlog.Error("Regular error", zlog.Err(errors.New("not critical")))

		// Alert
		zlog.Emit(ALERT, "Critical error", zlog.Err(errors.New("database down")))

		// Check separation
		if strings.Contains(logBuf.String(), "ALERT") {
			t.Error("ALERT signals should not appear in standard logs")
		}
		if !strings.Contains(logBuf.String(), "Regular error") {
			t.Error("ERROR signals should appear in standard logs")
		}
		if strings.Contains(alertBuf.String(), "Regular error") {
			t.Error("ERROR signals should not appear in alerts")
		}
		if !strings.Contains(alertBuf.String(), "Critical error") {
			t.Error("ALERT signals should appear in alerts output")
		}
	})
}

func TestPaymentService(t *testing.T) {
	alertBuf := &bytes.Buffer{}
	zlog.RouteSignal(ALERT, zlog.NewWriterSink(alertBuf))

	service := &PaymentService{
		dbConnected: true,
		gatewayURL:  "https://test-gateway.com",
	}

	t.Run("DatabaseError", func(t *testing.T) {
		alertBuf.Reset()
		service.dbConnected = false

		payment := Payment{ID: "test_001", UserID: "user123", Amount: 100}
		err := service.ProcessPayment(payment)

		if err == nil {
			t.Fatal("Expected database error")
		}

		// Should be a critical error
		var critErr *CriticalError
		if !errors.As(err, &critErr) {
			t.Error("Expected CriticalError type")
		}

		// Check alert is sent
		service.sendAlert(err, payment)

		var alert map[string]interface{}
		if err := json.Unmarshal(alertBuf.Bytes(), &alert); err != nil {
			t.Fatalf("Failed to parse alert: %v", err)
		}

		// Verify alert fields
		if alert["signal"] != "ALERT" {
			t.Errorf("Expected ALERT signal, got %v", alert["signal"])
		}
		if alert["severity"] != "critical" {
			t.Errorf("Expected critical severity, got %v", alert["severity"])
		}
		if alert["fingerprint"] != "db_connection_lost" {
			t.Errorf("Expected db_connection_lost fingerprint, got %v", alert["fingerprint"])
		}
		if _, ok := alert["stack_trace"]; !ok {
			t.Error("Critical alerts should include stack trace")
		}
	})

	t.Run("RegularError", func(t *testing.T) {
		service.dbConnected = true

		payment := Payment{ID: "test_002", UserID: "user456", Amount: 50000} // Exceeds limit
		err := service.ProcessPayment(payment)

		if err == nil {
			t.Fatal("Expected validation error")
		}

		// Should NOT be a critical error
		var critErr *CriticalError
		if errors.As(err, &critErr) {
			t.Error("Validation errors should not be CriticalError")
		}

		// Should not trigger alert
		if shouldAlert(err) {
			t.Error("Validation errors should not trigger alerts")
		}
	})
}

func TestShouldAlert(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantAlert bool
	}{
		{
			name:      "CriticalError",
			err:       &CriticalError{Err: errors.New("test"), Severity: "high"},
			wantAlert: true,
		},
		{
			name:      "ConnectionRefused",
			err:       errors.New("dial tcp: connection refused"),
			wantAlert: true,
		},
		{
			name:      "Timeout",
			err:       errors.New("request timeout after 30s"),
			wantAlert: true,
		},
		{
			name:      "DeadlineExceeded",
			err:       errors.New("context deadline exceeded"),
			wantAlert: true,
		},
		{
			name:      "ValidationError",
			err:       errors.New("invalid email format"),
			wantAlert: false,
		},
		{
			name:      "NotFound",
			err:       errors.New("user not found"),
			wantAlert: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldAlert(tt.err); got != tt.wantAlert {
				t.Errorf("shouldAlert(%v) = %v, want %v", tt.err, got, tt.wantAlert)
			}
		})
	}
}

func TestAlertContext(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.RouteSignal(ALERT, zlog.NewWriterSink(buf))

	service := &PaymentService{}

	// Create error with context
	err := &CriticalError{
		Err:         errors.New("gateway timeout"),
		Severity:    "high",
		Fingerprint: "gateway_timeout",
		Context: map[string]interface{}{
			"gateway_url": "https://api.example.com",
			"timeout_ms":  5000,
			"retry_count": 3,
		},
	}

	payment := Payment{ID: "test_003", UserID: "user789", Amount: 200}
	service.sendAlert(err, payment)

	var alert map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &alert); err != nil {
		t.Fatalf("Failed to parse alert: %v", err)
	}

	// Check context fields are included
	if alert["gateway_url"] != "https://api.example.com" {
		t.Errorf("Missing gateway_url context")
	}
	if alert["timeout_ms"] != float64(5000) {
		t.Errorf("Missing timeout_ms context")
	}
	if alert["retry_count"] != float64(3) {
		t.Errorf("Missing retry_count context")
	}
}

func TestStackTrace(t *testing.T) {
	stack := captureStackTrace()

	// Should contain test function name
	if !strings.Contains(stack, "TestStackTrace") {
		t.Errorf("Stack trace should contain current function, got: %s", stack)
	}

	// Should not contain runtime frames
	if strings.Contains(stack, "runtime.") {
		t.Errorf("Stack trace should filter runtime frames, got: %s", stack)
	}
}
