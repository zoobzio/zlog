package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/zoobzio/zlog"
)

func TestSecurityMonitoring(t *testing.T) {
	// Set up separate outputs
	logBuf := &bytes.Buffer{}
	secBuf := &bytes.Buffer{}

	zlog.EnableStandardLogging(logBuf)
	zlog.RouteSignal(zlog.SECURITY, zlog.NewWriterSink(secBuf))

	t.Run("SignalSeparation", func(t *testing.T) {
		// Regular log
		zlog.Info("Normal operation")

		// Security event
		zlog.Emit(zlog.SECURITY, "Security event",
			zlog.String("type", "test"),
		)

		// Check separation
		if strings.Contains(logBuf.String(), "SECURITY") {
			t.Error("SECURITY signals should not appear in standard logs")
		}
		if !strings.Contains(logBuf.String(), "Normal operation") {
			t.Error("INFO signals should appear in standard logs")
		}
		if strings.Contains(secBuf.String(), "Normal operation") {
			t.Error("INFO signals should not appear in security logs")
		}
		if !strings.Contains(secBuf.String(), "Security event") {
			t.Error("SECURITY signals should appear in security logs")
		}
	})
}

func TestBruteForceDetection(t *testing.T) {
	monitor := NewSecurityMonitor()

	t.Run("DetectsBruteForce", func(t *testing.T) {
		user := "testuser"
		ip := "192.168.1.1"

		// Record 5 failed attempts
		for i := 0; i < 5; i++ {
			monitor.RecordFailedLogin(user, ip)
		}

		if !monitor.IsBruteForce(user, ip) {
			t.Error("Should detect brute force after 5 attempts")
		}
	})

	t.Run("NoFalsePositive", func(t *testing.T) {
		user := "gooduser"
		ip := "192.168.1.2"

		// Only 3 attempts
		for i := 0; i < 3; i++ {
			monitor.RecordFailedLogin(user, ip)
		}

		if monitor.IsBruteForce(user, ip) {
			t.Error("Should not detect brute force with only 3 attempts")
		}
	})
}

func TestSQLInjectionDetection(t *testing.T) {
	monitor := NewSecurityMonitor()

	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"DROP TABLE", "'; DROP TABLE users; --", true},
		{"OR 1=1", "1' OR '1'='1", true},
		{"Union Select", "1 UNION SELECT * FROM users", true},
		{"Delete From", "1; DELETE FROM products", true},
		{"Normal Input", "John O'Brien", false},
		{"Normal Email", "user@example.com", false},
		{"Normal Search", "product name with spaces", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := monitor.IsSQLInjection(tt.input); result != tt.expected {
				t.Errorf("IsSQLInjection(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestPathTraversalDetection(t *testing.T) {
	monitor := NewSecurityMonitor()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Unix Path Traversal", "../../../etc/passwd", true},
		{"Windows Path Traversal", "..\\..\\..\\windows\\system32", true},
		{"URL Encoded", "%2e%2e%2f%2e%2e%2f", true},
		{"Double Encoded", "%252e%252e%252f", true},
		{"Direct System File", "etc/shadow", true},
		{"Normal Path", "/api/users/123", false},
		{"Normal File", "report_2024.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if result := monitor.IsPathTraversal(tt.path); result != tt.expected {
				t.Errorf("IsPathTraversal(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestRateLimiting(t *testing.T) {
	monitor := NewSecurityMonitor()

	t.Run("TracksRequestCount", func(t *testing.T) {
		ip := "10.0.0.1"

		// Record 150 requests
		for i := 0; i < 150; i++ {
			monitor.RecordRequest(ip)
		}

		count := monitor.GetRequestCount(ip)
		if count != 150 {
			t.Errorf("Expected 150 requests, got %d", count)
		}
	})
}

func TestBlockingMechanisms(t *testing.T) {
	monitor := NewSecurityMonitor()

	t.Run("IPBlocking", func(t *testing.T) {
		ip := "192.168.1.100"

		// Should not be blocked initially
		if monitor.IsIPBlocked(ip) {
			t.Error("IP should not be blocked initially")
		}

		// Block for 1 hour
		monitor.BlockIP(ip, 1*time.Hour)

		// Should be blocked now
		if !monitor.IsIPBlocked(ip) {
			t.Error("IP should be blocked after BlockIP call")
		}
	})

	t.Run("UserBlocking", func(t *testing.T) {
		user := "malicious"

		// Should not be blocked initially
		if monitor.IsUserBlocked(user) {
			t.Error("User should not be blocked initially")
		}

		// Block for 30 minutes
		monitor.BlockUser(user, 30*time.Minute)

		// Should be blocked now
		if !monitor.IsUserBlocked(user) {
			t.Error("User should be blocked after BlockUser call")
		}
	})

	t.Run("BlockExpiry", func(t *testing.T) {
		ip := "10.0.0.50"

		// Block for 1 millisecond
		monitor.BlockIP(ip, 1*time.Millisecond)

		// Wait for expiry
		time.Sleep(2 * time.Millisecond)

		// Should not be blocked anymore
		if monitor.IsIPBlocked(ip) {
			t.Error("IP should not be blocked after expiry")
		}
	})
}

func TestSecurityEventLogging(t *testing.T) {
	buf := &bytes.Buffer{}
	zlog.RouteSignal(zlog.SECURITY, zlog.NewWriterSink(buf))

	monitor := NewSecurityMonitor()

	t.Run("BruteForceEvent", func(t *testing.T) {
		buf.Reset()
		monitor.SimulateBruteForce("alice", "192.168.1.1")

		// Parse security events
		events := parseSecurityEvents(t, buf.String())

		// Should have at least one brute force event
		found := false
		for _, event := range events {
			if strings.Contains(event["message"].(string), "Multiple failed login") {
				found = true

				// Check required fields
				if event["user"] != "alice" {
					t.Errorf("Expected user=alice, got %v", event["user"])
				}
				if event["threat_level"] != "medium" {
					t.Errorf("Expected threat_level=medium, got %v", event["threat_level"])
				}
				if event["action"] != "account_locked" {
					t.Errorf("Expected action=account_locked, got %v", event["action"])
				}
			}
		}

		if !found {
			t.Error("Brute force security event not found")
		}
	})

	t.Run("DataExfiltrationEvent", func(t *testing.T) {
		buf.Reset()
		monitor.SimulateDataExfiltration("charlie", "10.10.10.10")

		events := parseSecurityEvents(t, buf.String())

		// Should have data exfiltration event
		found := false
		for _, event := range events {
			if strings.Contains(event["message"].(string), "data exfiltration") {
				found = true

				// Check it's marked as critical
				if event["threat_level"] != "critical" {
					t.Errorf("Data exfiltration should be critical threat, got %v", event["threat_level"])
				}
				if event["anomaly"] != true {
					t.Error("Data exfiltration should be marked as anomaly")
				}
			}
		}

		if !found {
			t.Error("Data exfiltration event not found")
		}
	})
}

func TestThreatCounting(t *testing.T) {
	monitor := NewSecurityMonitor()

	// Simulate various threats
	monitor.SimulateSQLInjection("10.0.0.1")
	monitor.SimulatePathTraversal("10.0.0.2")
	monitor.SimulatePrivilegeEscalation("user1", "10.0.0.3")

	// Should count all threats
	if monitor.threatCount < 3 {
		t.Errorf("Expected at least 3 threats counted, got %d", monitor.threatCount)
	}
}

// Helper function to parse security events.
func parseSecurityEvents(t *testing.T, output string) []map[string]interface{} {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	events := make([]map[string]interface{}, 0)

	for _, line := range lines {
		if line == "" {
			continue
		}

		var event map[string]interface{}
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			t.Fatalf("Failed to parse security event: %v\nLine: %s", err, line)
		}

		// Only include SECURITY signal events
		if event["signal"] == "SECURITY" {
			events = append(events, event)
		}
	}

	return events
}
