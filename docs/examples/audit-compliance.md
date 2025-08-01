# Audit and Compliance Example

This example demonstrates implementing comprehensive audit logging and compliance monitoring using zlog's signal-based approach.

## Compliance Logging System

```go
package main

import (
    "context"
    "crypto/sha256"
    "database/sql"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "os"
    "time"

    "github.com/zoobzio/zlog"
    _ "github.com/lib/pq"
)

// Compliance and audit signals
const (
    // User lifecycle events (SOX, GDPR)
    USER_CREATED        = "USER_CREATED"
    USER_AUTHENTICATED  = "USER_AUTHENTICATED"
    USER_UPDATED        = "USER_UPDATED"
    USER_DATA_ACCESSED  = "USER_DATA_ACCESSED"
    USER_DATA_EXPORTED  = "USER_DATA_EXPORTED"
    USER_DELETED        = "USER_DELETED"
    
    // Financial events (SOX, PCI-DSS)
    PAYMENT_INITIATED   = "PAYMENT_INITIATED"
    PAYMENT_AUTHORIZED  = "PAYMENT_AUTHORIZED"
    PAYMENT_CAPTURED    = "PAYMENT_CAPTURED"
    PAYMENT_REFUNDED    = "PAYMENT_REFUNDED"
    FINANCIAL_REPORT_GENERATED = "FINANCIAL_REPORT_GENERATED"
    
    // Security events (SOC2, ISO27001)
    LOGIN_ATTEMPT       = "LOGIN_ATTEMPT"
    LOGIN_FAILED        = "LOGIN_FAILED"
    PASSWORD_CHANGED    = "PASSWORD_CHANGED"
    PERMISSION_GRANTED  = "PERMISSION_GRANTED"
    PERMISSION_REVOKED  = "PERMISSION_REVOKED"
    ADMIN_ACTION        = "ADMIN_ACTION"
    PRIVILEGED_ACCESS   = "PRIVILEGED_ACCESS"
    
    // Data events (GDPR, HIPAA)
    DATA_CREATED        = "DATA_CREATED"
    DATA_MODIFIED       = "DATA_MODIFIED"
    DATA_DELETED        = "DATA_DELETED"
    DATA_BACKUP_CREATED = "DATA_BACKUP_CREATED"
    DATA_RETENTION_APPLIED = "DATA_RETENTION_APPLIED"
    
    // System events (SOC2)
    SYSTEM_STARTED      = "SYSTEM_STARTED"
    SYSTEM_SHUTDOWN     = "SYSTEM_SHUTDOWN"
    CONFIG_CHANGED      = "CONFIG_CHANGED"
    MAINTENANCE_MODE    = "MAINTENANCE_MODE"
    
    // Compliance violations
    COMPLIANCE_VIOLATION = "COMPLIANCE_VIOLATION"
    AUDIT_FAILURE       = "AUDIT_FAILURE"
    RETENTION_VIOLATION = "RETENTION_VIOLATION"
)

// Compliance metadata
type ComplianceContext struct {
    UserID       string `json:"user_id"`
    SessionID    string `json:"session_id"`
    IPAddress    string `json:"ip_address"`
    UserAgent    string `json:"user_agent"`
    ComplianceID string `json:"compliance_id"` // Unique ID for audit trail
}

// Audit record for tamper-proof storage
type AuditRecord struct {
    ID           string                 `json:"id"`
    Timestamp    time.Time              `json:"timestamp"`
    Signal       string                 `json:"signal"`
    Message      string                 `json:"message"`
    ComplianceContext                   `json:"compliance_context"`
    Fields       map[string]interface{} `json:"fields"`
    Checksum     string                 `json:"checksum"`
    Signature    string                 `json:"signature,omitempty"`
}

func main() {
    setupComplianceLogging()
    
    // Simulate compliance-relevant events
    simulateUserLifecycle()
    simulateFinancialOperations()
    simulateSecurityEvents()
    simulateDataOperations()
    
    time.Sleep(5 * time.Second)
    
    // Generate compliance reports
    generateComplianceReports()
}

func setupComplianceLogging() {
    zlog.EnableStandardLogging(zlog.INFO)
    
    // Setup tamper-proof audit logging
    setupSecureAuditLogging()
    
    // Setup real-time compliance monitoring
    setupComplianceMonitoring()
    
    // Setup retention policy enforcement
    setupRetentionPolicyEnforcement()
    
    // Setup compliance reporting
    setupComplianceReporting()
}

func setupSecureAuditLogging() {
    dbURL := os.Getenv("AUDIT_DB_URL")
    if dbURL == "" {
        dbURL = "postgres://audit:password@localhost/audit_db?sslmode=require"
    }
    
    db, err := sql.Open("postgres", dbURL)
    if err != nil {
        zlog.Fatal("Failed to connect to audit database",
            zlog.Err(err),
            zlog.String("component", "audit_logging"))
    }
    
    // Create tamper-proof audit sink
    auditSink := zlog.NewSink("secure-audit", func(ctx context.Context, event zlog.Event) error {
        return writeSecureAuditRecord(db, event)
    })
    
    // Route all audit-relevant events
    auditSignals := []string{
        USER_CREATED, USER_AUTHENTICATED, USER_UPDATED, USER_DATA_ACCESSED,
        USER_DATA_EXPORTED, USER_DELETED,
        PAYMENT_INITIATED, PAYMENT_AUTHORIZED, PAYMENT_CAPTURED, PAYMENT_REFUNDED,
        LOGIN_ATTEMPT, LOGIN_FAILED, PASSWORD_CHANGED,
        PERMISSION_GRANTED, PERMISSION_REVOKED, ADMIN_ACTION, PRIVILEGED_ACCESS,
        DATA_CREATED, DATA_MODIFIED, DATA_DELETED,
        SYSTEM_STARTED, SYSTEM_SHUTDOWN, CONFIG_CHANGED,
        COMPLIANCE_VIOLATION, AUDIT_FAILURE,
    }
    
    for _, signal := range auditSignals {
        zlog.RouteSignal(signal, auditSink)
    }
}

func writeSecureAuditRecord(db *sql.DB, event zlog.Event) error {
    // Extract compliance context from event fields
    complianceCtx := extractComplianceContext(event.Fields)
    
    // Create audit record
    record := AuditRecord{
        ID:                generateAuditID(),
        Timestamp:         event.Time,
        Signal:            string(event.Signal),
        Message:           event.Message,
        ComplianceContext: complianceCtx,
        Fields:            fieldsToMap(event.Fields),
    }
    
    // Calculate integrity checksum
    recordJSON, err := json.Marshal(record)
    if err != nil {
        return fmt.Errorf("failed to serialize audit record: %w", err)
    }
    
    hash := sha256.Sum256(recordJSON)
    record.Checksum = hex.EncodeToString(hash[:])
    
    // Sign record if signing key is available
    if signingKey := os.Getenv("AUDIT_SIGNING_KEY"); signingKey != "" {
        record.Signature = signRecord(recordJSON, signingKey)
    }
    
    // Insert into tamper-proof table
    query := `
        INSERT INTO audit_log (
            id, timestamp, signal, message, 
            user_id, session_id, ip_address, user_agent, compliance_id,
            fields, checksum, signature
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `
    
    fieldsJSON, _ := json.Marshal(record.Fields)
    
    _, err = db.ExecContext(context.Background(), query,
        record.ID, record.Timestamp, record.Signal, record.Message,
        record.UserID, record.SessionID, record.IPAddress, record.UserAgent, record.ComplianceID,
        fieldsJSON, record.Checksum, record.Signature)
    
    if err != nil {
        return fmt.Errorf("failed to insert audit record: %w", err)
    }
    
    return nil
}

func setupComplianceMonitoring() {
    complianceSink := zlog.NewSink("compliance-monitor", func(ctx context.Context, event zlog.Event) error {
        // Real-time compliance violation detection
        if err := checkComplianceViolations(event); err != nil {
            zlog.Emit(COMPLIANCE_VIOLATION, "Compliance violation detected",
                zlog.String("original_signal", string(event.Signal)),
                zlog.String("violation_type", err.Error()),
                zlog.String("event_id", generateEventID()))
        }
        
        // Update compliance metrics
        updateComplianceMetrics(event)
        
        return nil
    })
    
    // Monitor all events for compliance
    zlog.RouteSignal(zlog.DEBUG, complianceSink)
    zlog.RouteSignal(zlog.INFO, complianceSink)
    zlog.RouteSignal(zlog.WARN, complianceSink)
    zlog.RouteSignal(zlog.ERROR, complianceSink)
    zlog.RouteSignal(zlog.FATAL, complianceSink)
}

func setupRetentionPolicyEnforcement() {
    retentionSink := zlog.NewSink("retention-policy", func(ctx context.Context, event zlog.Event) error {
        // Check if data should be retained or purged based on policy
        if shouldApplyRetention(event) {
            err := applyRetentionPolicy(event)
            if err != nil {
                zlog.Emit(RETENTION_VIOLATION, "Failed to apply retention policy",
                    zlog.String("signal", string(event.Signal)),
                    zlog.Err(err))
            } else {
                zlog.Emit(DATA_RETENTION_APPLIED, "Retention policy applied",
                    zlog.String("signal", string(event.Signal)),
                    zlog.String("policy", "7_year_financial"))
            }
        }
        return nil
    })
    
    // Apply to data lifecycle events
    zlog.RouteSignal(DATA_CREATED, retentionSink)
    zlog.RouteSignal(DATA_MODIFIED, retentionSink)
    zlog.RouteSignal(USER_DELETED, retentionSink)
}

func setupComplianceReporting() {
    reportingSink := zlog.NewSink("compliance-reporting", func(ctx context.Context, event zlog.Event) error {
        // Aggregate data for compliance reports
        return aggregateComplianceData(event)
    })
    
    // Route financial and security events for reporting
    financialSignals := []string{
        PAYMENT_INITIATED, PAYMENT_AUTHORIZED, PAYMENT_CAPTURED, PAYMENT_REFUNDED,
        FINANCIAL_REPORT_GENERATED,
    }
    
    securitySignals := []string{
        LOGIN_ATTEMPT, LOGIN_FAILED, PASSWORD_CHANGED,
        PERMISSION_GRANTED, PERMISSION_REVOKED, ADMIN_ACTION,
    }
    
    for _, signal := range append(financialSignals, securitySignals...) {
        zlog.RouteSignal(signal, reportingSink)
    }
}

// Simulate compliance events

func simulateUserLifecycle() {
    ctx := createComplianceContext("user_123", "192.168.1.100")
    
    // User registration (GDPR Article 30)
    zlog.Emit(USER_CREATED, "User account created",
        append(complianceFields(ctx),
            zlog.String("email", "alice@example.com"),
            zlog.Bool("gdpr_consent", true),
            zlog.Time("consent_timestamp", time.Now()))...)
    
    // Authentication event (SOC2 CC6.1)
    zlog.Emit(USER_AUTHENTICATED, "User successfully authenticated",
        append(complianceFields(ctx),
            zlog.String("auth_method", "password"),
            zlog.String("mfa_status", "enabled"))...)
    
    // Data access (GDPR Article 30, HIPAA)
    zlog.Emit(USER_DATA_ACCESSED, "User personal data accessed",
        append(complianceFields(ctx),
            zlog.String("data_type", "personal_information"),
            zlog.String("access_purpose", "profile_update"),
            zlog.String("legal_basis", "legitimate_interest"))...)
    
    // Data export (GDPR Article 20)
    zlog.Emit(USER_DATA_EXPORTED, "User data exported",
        append(complianceFields(ctx),
            zlog.String("export_format", "json"),
            zlog.String("export_reason", "data_portability_request"),
            zlog.Int("records_exported", 15))...)
}

func simulateFinancialOperations() {
    ctx := createComplianceContext("user_456", "10.0.1.50")
    
    // Payment processing (SOX, PCI-DSS)
    paymentID := "pay_" + generateEventID()
    
    zlog.Emit(PAYMENT_INITIATED, "Payment processing initiated",
        append(complianceFields(ctx),
            zlog.String("payment_id", paymentID),
            zlog.Float64("amount", 299.99),
            zlog.String("currency", "USD"),
            zlog.String("payment_method", "credit_card"),
            zlog.String("merchant_id", "merchant_123"))...)
    
    zlog.Emit(PAYMENT_AUTHORIZED, "Payment authorized",
        append(complianceFields(ctx),
            zlog.String("payment_id", paymentID),
            zlog.String("authorization_code", "AUTH123456"),
            zlog.String("processor", "stripe"))...)
    
    zlog.Emit(PAYMENT_CAPTURED, "Payment captured",
        append(complianceFields(ctx),
            zlog.String("payment_id", paymentID),
            zlog.String("transaction_id", "txn_789"),
            zlog.Float64("captured_amount", 299.99))...)
    
    // Financial report generation (SOX Section 404)
    zlog.Emit(FINANCIAL_REPORT_GENERATED, "Financial report generated",
        append(complianceFields(ctx),
            zlog.String("report_type", "monthly_revenue"),
            zlog.String("report_period", "2023-10"),
            zlog.String("generated_by", "automated_system"),
            zlog.String("approval_status", "pending"))...)
}

func simulateSecurityEvents() {
    // Failed login attempt (SOC2 CC6.1)
    suspiciousCtx := createComplianceContext("unknown", "suspicious.example.com")
    
    zlog.Emit(LOGIN_FAILED, "Authentication failed",
        append(complianceFields(suspiciousCtx),
            zlog.String("attempted_username", "admin"),
            zlog.String("failure_reason", "invalid_credentials"),
            zlog.Int("attempt_count", 5))...)
    
    // Admin action (SOC2 CC6.2)
    adminCtx := createComplianceContext("admin_789", "192.168.1.200")
    
    zlog.Emit(ADMIN_ACTION, "Administrative action performed",
        append(complianceFields(adminCtx),
            zlog.String("action", "user_role_change"),
            zlog.String("target_user", "user_123"),
            zlog.String("old_role", "user"),
            zlog.String("new_role", "moderator"),
            zlog.String("justification", "promotion_approved"))...)
    
    // Permission changes (ISO27001 A.9.2.6)
    zlog.Emit(PERMISSION_GRANTED, "Permission granted",
        append(complianceFields(adminCtx),
            zlog.String("permission", "financial_reports_read"),
            zlog.String("granted_to", "user_456"),
            zlog.String("granted_by", "admin_789"),
            zlog.String("approval_ticket", "TICKET-12345"))...)
}

func simulateDataOperations() {
    ctx := createComplianceContext("user_123", "192.168.1.100")
    
    // Data modification (GDPR Article 30)
    zlog.Emit(DATA_MODIFIED, "User data modified",
        append(complianceFields(ctx),
            zlog.String("data_type", "personal_information"),
            zlog.String("field_modified", "email_address"),
            zlog.String("old_value_hash", "sha256:abc123"),
            zlog.String("new_value_hash", "sha256:def456"))...)
    
    // Data backup (SOC2 CC6.1)
    zlog.Emit(DATA_BACKUP_CREATED, "Data backup created",
        zlog.String("backup_id", "backup_"+generateEventID()),
        zlog.String("backup_type", "incremental"),
        zlog.String("data_classification", "sensitive"),
        zlog.Int64("backup_size_bytes", 1073741824),
        zlog.String("encryption_status", "aes256"))
}

// Utility functions

func createComplianceContext(userID, ipAddress string) ComplianceContext {
    return ComplianceContext{
        UserID:       userID,
        SessionID:    "sess_" + generateEventID(),
        IPAddress:    ipAddress,
        UserAgent:    "ComplianceApp/1.0",
        ComplianceID: generateComplianceID(),
    }
}

func complianceFields(ctx ComplianceContext) []zlog.Field {
    return []zlog.Field{
        zlog.String("user_id", ctx.UserID),
        zlog.String("session_id", ctx.SessionID),
        zlog.String("ip_address", ctx.IPAddress),
        zlog.String("user_agent", ctx.UserAgent),
        zlog.String("compliance_id", ctx.ComplianceID),
    }
}

func extractComplianceContext(fields []zlog.Field) ComplianceContext {
    ctx := ComplianceContext{}
    
    for _, field := range fields {
        switch field.Key {
        case "user_id":
            if val, ok := field.Value.(string); ok {
                ctx.UserID = val
            }
        case "session_id":
            if val, ok := field.Value.(string); ok {
                ctx.SessionID = val
            }
        case "ip_address":
            if val, ok := field.Value.(string); ok {
                ctx.IPAddress = val
            }
        case "user_agent":
            if val, ok := field.Value.(string); ok {
                ctx.UserAgent = val
            }
        case "compliance_id":
            if val, ok := field.Value.(string); ok {
                ctx.ComplianceID = val
            }
        }
    }
    
    return ctx
}

func fieldsToMap(fields []zlog.Field) map[string]interface{} {
    result := make(map[string]interface{})
    for _, field := range fields {
        result[field.Key] = field.Value
    }
    return result
}

func generateEventID() string {
    return fmt.Sprintf("%d", time.Now().UnixNano())
}

func generateAuditID() string {
    return "audit_" + generateEventID()
}

func generateComplianceID() string {
    return "compliance_" + generateEventID()
}

func signRecord(data []byte, key string) string {
    // In real implementation, use proper digital signatures
    hash := sha256.Sum256(append(data, []byte(key)...))
    return hex.EncodeToString(hash[:])
}

// Compliance monitoring functions

func checkComplianceViolations(event zlog.Event) error {
    // Example compliance checks
    
    // Check for PCI-DSS violations (credit card data in logs)
    for _, field := range event.Fields {
        if containsCreditCardData(fmt.Sprintf("%v", field.Value)) {
            return fmt.Errorf("PCI-DSS violation: credit card data in logs")
        }
    }
    
    // Check for GDPR violations (personal data without consent)
    if event.Signal == USER_DATA_ACCESSED {
        if !hasGDPRConsent(event.Fields) {
            return fmt.Errorf("GDPR violation: personal data access without consent")
        }
    }
    
    return nil
}

func containsCreditCardData(value string) bool {
    // Simplified credit card detection
    // In real implementation, use proper PAN detection
    return len(value) == 16 && isNumeric(value)
}

func hasGDPRConsent(fields []zlog.Field) bool {
    for _, field := range fields {
        if field.Key == "gdpr_consent" {
            if consent, ok := field.Value.(bool); ok {
                return consent
            }
        }
    }
    return false
}

func isNumeric(s string) bool {
    for _, c := range s {
        if c < '0' || c > '9' {
            return false
        }
    }
    return true
}

func updateComplianceMetrics(event zlog.Event) {
    // Update Prometheus metrics for compliance monitoring
    // complianceEventsTotal.WithLabelValues(string(event.Signal)).Inc()
}

func shouldApplyRetention(event zlog.Event) bool {
    // Determine if retention policy should be applied
    retentionSignals := map[string]bool{
        DATA_CREATED: true,
        USER_DELETED: true,
    }
    
    return retentionSignals[string(event.Signal)]
}

func applyRetentionPolicy(event zlog.Event) error {
    // Apply data retention policies
    // This would integrate with your data retention system
    return nil
}

func aggregateComplianceData(event zlog.Event) error {
    // Aggregate data for compliance reports
    // This would write to a compliance data warehouse
    return nil
}

func generateComplianceReports() {
    zlog.Info("Generating compliance reports",
        zlog.String("report_type", "sox_monthly"),
        zlog.String("period", "2023-10"),
        zlog.Time("generated_at", time.Now()))
    
    zlog.Info("Generating compliance reports",
        zlog.String("report_type", "gdpr_data_processing"),
        zlog.String("period", "2023-10"),
        zlog.Time("generated_at", time.Now()))
}
```

## Example Output

Secure audit log entries:

```json
{"id":"audit_1640123420000000001","timestamp":"2023-10-20T14:30:00Z","signal":"USER_CREATED","message":"User account created","compliance_context":{"user_id":"user_123","session_id":"sess_1640123420000000002","ip_address":"192.168.1.100","user_agent":"ComplianceApp/1.0","compliance_id":"compliance_1640123420000000003"},"fields":{"email":"alice@example.com","gdpr_consent":true,"consent_timestamp":"2023-10-20T14:30:00Z"},"checksum":"sha256:abc123def456","signature":"digital_signature_here"}

{"id":"audit_1640123420000000004","timestamp":"2023-10-20T14:30:05Z","signal":"PAYMENT_CAPTURED","message":"Payment captured","compliance_context":{"user_id":"user_456","session_id":"sess_1640123420000000005","ip_address":"10.0.1.50","user_agent":"ComplianceApp/1.0","compliance_id":"compliance_1640123420000000006"},"fields":{"payment_id":"pay_1640123420000000007","transaction_id":"txn_789","captured_amount":299.99},"checksum":"sha256:def456ghi789","signature":"digital_signature_here"}

{"id":"audit_1640123420000000008","timestamp":"2023-10-20T14:30:10Z","signal":"ADMIN_ACTION","message":"Administrative action performed","compliance_context":{"user_id":"admin_789","session_id":"sess_1640123420000000009","ip_address":"192.168.1.200","user_agent":"ComplianceApp/1.0","compliance_id":"compliance_1640123420000000010"},"fields":{"action":"user_role_change","target_user":"user_123","old_role":"user","new_role":"moderator","justification":"promotion_approved"},"checksum":"sha256:ghi789jkl012","signature":"digital_signature_here"}
```

This example demonstrates:

- **Tamper-proof audit logging**: Digital signatures and checksums for data integrity
- **Comprehensive compliance coverage**: SOX, GDPR, PCI-DSS, SOC2, HIPAA, ISO27001
- **Real-time violation detection**: Automatic scanning for compliance violations
- **Retention policy enforcement**: Automated data lifecycle management
- **Compliance reporting**: Automated generation of regulatory reports
- **Secure storage**: Encrypted database with proper access controls
- **Audit trail**: Complete chain of custody for all compliance events

The signal-based approach makes it easy to ensure all compliance-relevant events are captured, monitored, and reported according to regulatory requirements.