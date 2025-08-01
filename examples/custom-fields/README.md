# Custom Fields Example

This example demonstrates creating custom field constructors that transform data before it enters the logging pipeline. Essential for security, compliance, and privacy requirements.

## What This Shows

- Redacting sensitive data (credit cards, SSNs)
- Masking emails while maintaining traceability
- One-way hashing for correlation without exposure
- Truncating API keys and tokens
- IP address anonymization
- Compliance tagging for audit trails

## Running the Example

```bash
go run main.go
```

## Field Transformers

### RedactedString
Automatically detects and masks:
- Credit card numbers: `4532********9012`
- Social Security Numbers: `XXX-XX-6789`
- Generic long strings: `abc*****xyz`

### MaskedEmail
Preserves domain while hiding username:
- `john.doe@example.com` → `j***@example.com`

### HashedField
One-way hash for correlation:
- Passwords, user IDs
- Allows matching without exposing values

### TruncatedToken
Shows partial API keys:
- `sk-1234567890abcdef` → `sk-1234...cdef`

### IPAnonymized
GDPR-compliant IP logging:
- `192.168.1.100` → `192.168.1.xxx`

### SensitiveInt
Converts exact values to ranges:
- Age 28 → `25-34`
- Useful for demographics without PII

## Compliance Features

```go
compliance := NewComplianceFields("GDPR", "PCI-DSS").
    Add(HashedField("user_id", userId)).
    Add(RedactedString("card", cardNumber))
```

Automatically adds:
- Compliance tags
- PII presence indicators

## Best Practices

1. **Never log**: passwords, CVVs, full SSNs
2. **Always transform**: emails, IPs, tokens
3. **Consider**: data retention policies
4. **Remember**: logs are often copied/backed up

## Creating Your Own

```go
func MyCustomField(key string, value SensitiveType) zlog.Field {
    transformed := transform(value)
    return zlog.String(key, transformed)
}
```