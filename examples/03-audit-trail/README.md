# Example 03: Audit Trail

This example shows how to use custom signals (AUDIT, SECURITY) to create a comprehensive audit trail separate from normal application logs.

## What It Shows

- Creating custom signals beyond standard log levels
- Using AUDIT and SECURITY signals for compliance
- Separating audit logs into a dedicated file
- Including detailed context in audit events

## Key Concepts

1. **Audit signals are special** - They capture who did what, when
2. **Immutable records** - Audit logs often have legal/compliance requirements
3. **Rich context** - User IDs, IPs, actions, results, and reasons
4. **Separate storage** - Audit logs typically go to secure, append-only storage

## Running the Example

```bash
go run main.go
```

You'll see:
- Standard logs (INFO) go to stderr
- Audit events go to `audit.log` file
- Security events also go to `audit.log` file

Expected console output:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Service started","caller":"main.go:15"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"User service ready","caller":"main.go:20"}
```

Expected audit.log content:
```json
{"time":"2024-01-20T10:30:45Z","signal":"AUDIT","message":"User login","caller":"main.go:25","user_id":"alice","ip":"192.168.1.100","success":true}
{"time":"2024-01-20T10:30:45Z","signal":"AUDIT","message":"Permission check","caller":"main.go:30","user_id":"alice","resource":"admin_panel","action":"view","granted":false}
{"time":"2024-01-20T10:30:45Z","signal":"SECURITY","message":"Failed login attempt","caller":"main.go:35","user_id":"bob","ip":"10.0.0.50","reason":"invalid_password"}
```

## Use Cases

- **Compliance**: SOC2, HIPAA, PCI-DSS require audit trails
- **Security**: Track authentication, authorization, and access
- **Forensics**: Investigate incidents with detailed event history
- **Analytics**: Understand user behavior and system usage

## Next Steps

- See [04-metrics-pipeline](../04-metrics-pipeline) for metrics collection
- See [07-security-monitoring](../07-security-monitoring) for security-focused logging