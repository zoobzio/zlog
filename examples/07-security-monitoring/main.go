package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/zoobzio/zlog"
)

// ThreatLevel represents the severity of a security event.
type ThreatLevel string

const (
	ThreatLow      ThreatLevel = "low"
	ThreatMedium   ThreatLevel = "medium"
	ThreatHigh     ThreatLevel = "high"
	ThreatCritical ThreatLevel = "critical"
)

func main() {
	// Standard logs to stderr
	zlog.EnableStandardLogging(os.Stderr)

	// Security events to a separate file
	securityFile, err := os.Create("security.log")
	if err != nil {
		zlog.Fatal("Failed to create security file", zlog.Err(err))
	}
	defer securityFile.Close()

	// Route SECURITY signals to the security file
	securitySink := zlog.NewWriterSink(securityFile)
	zlog.RouteSignal(zlog.SECURITY, securitySink)

	// Start security monitoring
	zlog.Info("Security monitor started")

	monitor := NewSecurityMonitor()

	// Simulate various security scenarios
	monitor.SimulateActivity()

	zlog.Info("Security monitoring session complete",
		zlog.Int("threats_detected", monitor.threatCount),
		zlog.Int("users_blocked", len(monitor.blockedUsers)),
		zlog.Int("ips_blocked", len(monitor.blockedIPs)),
	)
}

// SecurityMonitor tracks and responds to security events.
type SecurityMonitor struct {
	loginAttempts map[string][]time.Time
	requestCounts map[string]int
	blockedIPs    map[string]time.Time
	blockedUsers  map[string]time.Time
	threatCount   int
	mu            sync.RWMutex
}

func NewSecurityMonitor() *SecurityMonitor {
	return &SecurityMonitor{
		loginAttempts: make(map[string][]time.Time),
		requestCounts: make(map[string]int),
		blockedIPs:    make(map[string]time.Time),
		blockedUsers:  make(map[string]time.Time),
	}
}

func (m *SecurityMonitor) SimulateActivity() {
	// Scenario 1: Brute force login attempts
	m.SimulateBruteForce("alice", "192.168.1.100")

	// Scenario 2: SQL injection attempts
	m.SimulateSQLInjection("10.0.0.50")

	// Scenario 3: Path traversal attempts
	m.SimulatePathTraversal("172.16.0.10")

	// Scenario 4: Rate limit violations
	m.SimulateRateLimitViolation("172.16.0.1")

	// Scenario 5: Privilege escalation attempt
	m.SimulatePrivilegeEscalation("bob", "192.168.1.50")

	// Scenario 6: Data exfiltration attempt
	m.SimulateDataExfiltration("charlie", "10.10.10.10")
}

func (m *SecurityMonitor) SimulateBruteForce(user, ip string) {
	zlog.Info("Processing login attempts")

	// Simulate multiple failed login attempts
	for i := 0; i < 5; i++ {
		m.RecordFailedLogin(user, ip)
		time.Sleep(100 * time.Millisecond)
	}

	// Check for brute force
	if m.IsBruteForce(user, ip) {
		m.threatCount++
		zlog.Emit(zlog.SECURITY, "Multiple failed login attempts",
			zlog.String("user", user),
			zlog.String("ip", ip),
			zlog.Int("attempts", 5),
			zlog.String("period", "5m"),
			zlog.String("threat_level", string(ThreatMedium)),
			zlog.String("action", "account_locked"),
		)
		m.BlockUser(user, 30*time.Minute)
	}
}

func (m *SecurityMonitor) SimulateSQLInjection(ip string) {
	// Common SQL injection patterns
	payloads := []string{
		"'; DROP TABLE users; --",
		"1' OR '1'='1",
		"admin'--",
		"1; DELETE FROM products",
	}

	for _, payload := range payloads {
		if m.IsSQLInjection(payload) {
			m.threatCount++
			zlog.Emit(zlog.SECURITY, "SQL injection attempt detected",
				zlog.String("ip", ip),
				zlog.String("path", "/api/users"),
				zlog.String("payload", payload),
				zlog.String("threat_level", string(ThreatHigh)),
				zlog.String("action", "request_blocked"),
			)
			m.BlockIP(ip, 1*time.Hour)
			break
		}
	}
}

func (m *SecurityMonitor) SimulatePathTraversal(ip string) {
	paths := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"/api/files?path=../../../../etc/shadow",
	}

	for _, path := range paths {
		if m.IsPathTraversal(path) {
			m.threatCount++
			zlog.Emit(zlog.SECURITY, "Path traversal attempt",
				zlog.String("ip", ip),
				zlog.String("path", path),
				zlog.String("threat_level", string(ThreatHigh)),
				zlog.String("action", "request_blocked"),
			)
			m.BlockIP(ip, 2*time.Hour)
			break
		}
	}
}

func (m *SecurityMonitor) SimulateRateLimitViolation(ip string) {
	// Simulate rapid requests
	for i := 0; i < 1000; i++ {
		m.RecordRequest(ip)
	}

	if count := m.GetRequestCount(ip); count > 100 {
		m.threatCount++
		zlog.Emit(zlog.SECURITY, "Rate limit exceeded",
			zlog.String("ip", ip),
			zlog.Int("requests", count),
			zlog.String("window", "1m"),
			zlog.String("threat_level", string(ThreatLow)),
			zlog.String("action", "rate_limited"),
		)
		// Implement rate limiting rather than blocking
	}
}

func (m *SecurityMonitor) SimulatePrivilegeEscalation(user, ip string) {
	// Detect attempts to access admin functions without privileges
	suspiciousActions := []string{
		"GET /admin/users",
		"POST /admin/settings",
		"DELETE /admin/data",
	}

	for _, action := range suspiciousActions {
		m.threatCount++
		zlog.Emit(zlog.SECURITY, "Privilege escalation attempt",
			zlog.String("user", user),
			zlog.String("ip", ip),
			zlog.String("action", action),
			zlog.String("user_role", "regular"),
			zlog.String("required_role", "admin"),
			zlog.String("threat_level", string(ThreatMedium)),
			zlog.String("response", "access_denied"),
		)
	}
}

func (m *SecurityMonitor) SimulateDataExfiltration(user, ip string) {
	// Detect large data transfers or bulk exports
	m.threatCount++
	zlog.Emit(zlog.SECURITY, "Potential data exfiltration",
		zlog.String("user", user),
		zlog.String("ip", ip),
		zlog.String("endpoint", "/api/export/all-customers"),
		zlog.Int("records_accessed", 50000),
		zlog.Int("data_size_mb", 245),
		zlog.String("threat_level", string(ThreatCritical)),
		zlog.String("action", "export_blocked_admin_notified"),
		zlog.Bool("anomaly", true),
		zlog.String("baseline_export_size", "5MB"),
	)

	// Critical threat - immediate response
	m.BlockUser(user, 24*time.Hour)
	m.BlockIP(ip, 24*time.Hour)
}

// Detection methods

func (m *SecurityMonitor) RecordFailedLogin(user, ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	key := fmt.Sprintf("%s:%s", user, ip)
	m.loginAttempts[key] = append(m.loginAttempts[key], time.Now())
}

func (m *SecurityMonitor) IsBruteForce(user, ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", user, ip)
	attempts := m.loginAttempts[key]

	// Check if more than 5 attempts in last 5 minutes
	cutoff := time.Now().Add(-5 * time.Minute)
	recentAttempts := 0
	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			recentAttempts++
		}
	}

	return recentAttempts >= 5
}

func (m *SecurityMonitor) IsSQLInjection(input string) bool {
	// Common SQL injection patterns
	patterns := []string{
		`(?i)(union|select|insert|update|delete|drop|create)\s+(select|from|into|table|database|where)`,
		`(?i)(union\s+select)`,
		`(?i)(;|--|'|")\s*(drop|delete|truncate|alter|create|insert|update)`,
		`(?i)(\sor\s|\sand\s).*=.*`,
		`(?i)'.*or.*'.*'.*=.*'`,
	}

	for _, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			return true
		}
	}

	return false
}

func (m *SecurityMonitor) IsPathTraversal(path string) bool {
	dangerous := []string{
		"../",
		"..\\",
		"%2e%2e",
		"%252e%252e",
		"etc/passwd",
		"etc/shadow",
		"windows\\system32",
	}

	lowercasePath := strings.ToLower(path)
	for _, pattern := range dangerous {
		if strings.Contains(lowercasePath, pattern) {
			return true
		}
	}

	return false
}

func (m *SecurityMonitor) RecordRequest(ip string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.requestCounts[ip]++
}

func (m *SecurityMonitor) GetRequestCount(ip string) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.requestCounts[ip]
}

// Response methods

func (m *SecurityMonitor) BlockIP(ip string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockedIPs[ip] = time.Now().Add(duration)

	zlog.Info("IP blocked",
		zlog.String("ip", ip),
		zlog.Duration("duration", duration),
	)
}

func (m *SecurityMonitor) BlockUser(user string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.blockedUsers[user] = time.Now().Add(duration)

	zlog.Info("User blocked",
		zlog.String("user", user),
		zlog.Duration("duration", duration),
	)
}

func (m *SecurityMonitor) IsIPBlocked(ip string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if until, exists := m.blockedIPs[ip]; exists {
		return time.Now().Before(until)
	}
	return false
}

func (m *SecurityMonitor) IsUserBlocked(user string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if until, exists := m.blockedUsers[user]; exists {
		return time.Now().Before(until)
	}
	return false
}

func init() {
	// Clean up from previous runs
	os.Remove("security.log")
}
