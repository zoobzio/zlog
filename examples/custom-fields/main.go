// Package main demonstrates creating custom field constructors for data transformation.
//
// This example shows how to extend zlog's field system to handle sensitive data,
// implement security requirements, and transform data before it enters the logging pipeline.
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/zoobzio/zlog"
)

// Custom field constructors that transform data for security and compliance.

// RedactedString creates a field with sensitive data masked.
// Useful for: SSNs, credit cards, API keys, passwords
func RedactedString(key, value string) zlog.Field {
	// Detect and redact common patterns
	redacted := value
	
	// Credit card pattern (simplified)
	if regexp.MustCompile(`\d{4}[\s-]?\d{4}[\s-]?\d{4}[\s-]?\d{4}`).MatchString(value) {
		// Show first and last 4 digits
		digits := regexp.MustCompile(`\d`).FindAllString(value, -1)
		if len(digits) >= 8 {
			redacted = digits[0] + digits[1] + digits[2] + digits[3] + 
				"********" + 
				digits[len(digits)-4] + digits[len(digits)-3] + 
				digits[len(digits)-2] + digits[len(digits)-1]
		}
	} else if regexp.MustCompile(`\d{3}-\d{2}-\d{4}`).MatchString(value) {
		// SSN pattern
		redacted = "XXX-XX-" + value[len(value)-4:]
	} else if len(value) > 8 {
		// Generic redaction for long strings
		redacted = value[:3] + strings.Repeat("*", len(value)-6) + value[len(value)-3:]
	}
	
	return zlog.String(key, redacted)
}

// MaskedEmail creates a field with email partially hidden.
// Shows: j***@example.com
func MaskedEmail(key, email string) zlog.Field {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return zlog.String(key, "invalid-email")
	}
	
	username := parts[0]
	domain := parts[1]
	
	if len(username) <= 1 {
		return zlog.String(key, "*@"+domain)
	}
	
	masked := string(username[0]) + strings.Repeat("*", len(username)-1) + "@" + domain
	return zlog.String(key, masked)
}

// HashedField creates a field with one-way hashed value.
// Useful for: correlation without exposing actual values
func HashedField(key, value string) zlog.Field {
	hash := sha256.Sum256([]byte(value))
	hashStr := hex.EncodeToString(hash[:])[:12] // First 12 chars of hash
	return zlog.String(key, "hash:"+hashStr)
}

// TruncatedToken shows only part of an API key or token.
// Shows: sk-abc...xyz (first 6 and last 4 chars)
func TruncatedToken(key, token string) zlog.Field {
	if len(token) <= 10 {
		return zlog.String(key, strings.Repeat("*", len(token)))
	}
	
	truncated := token[:6] + "..." + token[len(token)-4:]
	return zlog.String(key, truncated)
}

// EncryptedField simulates field encryption (in practice, use real encryption).
func EncryptedField(key, value string) zlog.Field {
	// In a real implementation, this would use AES or similar
	// For demo, we'll just base64 encode with a marker
	encoded := []byte(value)
	for i := range encoded {
		encoded[i] = encoded[i] + 1 // Simple Caesar cipher for demo
	}
	return zlog.String(key, fmt.Sprintf("enc:%x", encoded))
}

// IPAnonymized removes the last octet from an IP address.
// 192.168.1.100 -> 192.168.1.xxx
func IPAnonymized(key, ip string) zlog.Field {
	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		parts[3] = "xxx"
		ip = strings.Join(parts, ".")
	}
	return zlog.String(key, ip)
}

// SensitiveInt creates a field that shows a range instead of exact value.
// Useful for: ages, salaries, counts
func SensitiveInt(key string, value int) zlog.Field {
	var rangeStr string
	
	switch {
	case value < 18:
		rangeStr = "<18"
	case value < 25:
		rangeStr = "18-24"
	case value < 35:
		rangeStr = "25-34"  
	case value < 50:
		rangeStr = "35-49"
	case value < 65:
		rangeStr = "50-64"
	default:
		rangeStr = "65+"
	}
	
	return zlog.String(key, rangeStr)
}

// ComplianceFields groups multiple fields with compliance tags.
type ComplianceFields struct {
	fields []zlog.Field
	tags   []string
}

func NewComplianceFields(tags ...string) *ComplianceFields {
	return &ComplianceFields{
		fields: make([]zlog.Field, 0),
		tags:   tags,
	}
}

func (cf *ComplianceFields) Add(field zlog.Field) *ComplianceFields {
	cf.fields = append(cf.fields, field)
	return cf
}

func (cf *ComplianceFields) AddString(key, value string) *ComplianceFields {
	cf.fields = append(cf.fields, zlog.String(key, value))
	return cf
}

func (cf *ComplianceFields) Fields() []zlog.Field {
	// Add compliance metadata
	cf.fields = append(cf.fields, 
		zlog.Strings("compliance_tags", cf.tags),
		zlog.Bool("pii_present", true),
	)
	return cf.fields
}

// Custom signal for security events
const SECURITY_AUDIT = zlog.Signal("SECURITY_AUDIT")

func main() {
	fmt.Println("=== Custom Fields Example ===")
	fmt.Println("Demonstrating field transformations for security and compliance")
	fmt.Println()

	// Enable standard logging
	zlog.EnableStandardLogging(zlog.INFO)
	
	// Route security events
	zlog.RouteSignal(SECURITY_AUDIT, zlog.NewSink("security", func(_ context.Context, event zlog.Event) error {
		fmt.Printf("🔐 [SECURITY] %s\n", event.Message)
		for _, field := range event.Fields {
			fmt.Printf("   %s: %v\n", field.Key, field.Value)
		}
		return nil
	}))

	// Example 1: User registration with PII
	fmt.Println("--- User Registration ---")
	zlog.Info("New user registered",
		zlog.String("user_id", "usr_123456"),
		MaskedEmail("email", "john.doe@example.com"),
		RedactedString("ssn", "123-45-6789"),
		HashedField("password", "SuperSecret123!"),
		IPAnonymized("ip_address", "192.168.1.100"),
		SensitiveInt("age", 28),
	)

	// Example 2: Payment processing
	fmt.Println("\n--- Payment Processing ---")
	zlog.Info("Payment processed",
		RedactedString("credit_card", "4532-1234-5678-9012"),
		zlog.Float64("amount", 99.99),
		MaskedEmail("customer_email", "alice@company.com"),
		EncryptedField("cvv", "123"), // Never log CVV in production!
	)

	// Example 3: API key usage
	fmt.Println("\n--- API Authentication ---")
	zlog.Info("API request authenticated",
		TruncatedToken("api_key", "sk-1234567890abcdefghijklmnop"),
		zlog.String("endpoint", "/api/v1/users"),
		IPAnonymized("client_ip", "203.0.113.45"),
	)

	// Example 4: Compliance-tagged fields
	fmt.Println("\n--- GDPR Compliant Event ---")
	compliance := NewComplianceFields("GDPR", "PCI-DSS").
		AddString("event_type", "user_data_export").
		Add(HashedField("user_id", "user@example.com")).
		Add(RedactedString("account_number", "1234567890"))

	zlog.Emit(SECURITY_AUDIT, "User data export requested", compliance.Fields()...)

	// Example 5: Complex transformation
	fmt.Println("\n--- Session Tracking ---")
	sessionData := map[string]string{
		"session_id": "sess_abc123xyz789",
		"user_email": "admin@example.org",
		"auth_token": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
		"credit_card": "5555-4444-3333-2222",
	}

	// Transform all sensitive fields
	fields := []zlog.Field{
		HashedField("session_id", sessionData["session_id"]),
		MaskedEmail("user", sessionData["user_email"]),
		TruncatedToken("auth", sessionData["auth_token"]),
		RedactedString("payment_method", sessionData["credit_card"]),
		zlog.Time("login_time", time.Now()),
	}

	zlog.Info("User session started", fields...)

	// Example 6: Error with sensitive context
	fmt.Println("\n--- Error Handling ---")
	zlog.Error("Failed to process payment",
		zlog.Err(fmt.Errorf("insufficient funds")),
		RedactedString("account", "ACC-987654321"),
		zlog.Float64("attempted_amount", 500.00),
		zlog.Float64("available_balance", 50.00), // Be careful logging balances!
		MaskedEmail("notification_sent_to", "finance@example.com"),
	)

	fmt.Println("\n=== Example Complete ===")
	fmt.Println("Notice how sensitive data was transformed before logging!")
	fmt.Println("Key techniques demonstrated:")
	fmt.Println("- Redaction (credit cards, SSNs)")
	fmt.Println("- Masking (emails)")
	fmt.Println("- Hashing (passwords, IDs)")
	fmt.Println("- Truncation (API keys)")
	fmt.Println("- Anonymization (IP addresses)")
	fmt.Println("- Range buckets (ages)")
}