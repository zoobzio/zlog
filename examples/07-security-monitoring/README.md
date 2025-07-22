# Example 07: Security Monitoring

This example shows how to use zlog for security monitoring, detecting and tracking suspicious activities in real-time.

## What It Shows

- Using SECURITY signal for security events
- Detecting patterns of malicious behavior
- Rate limiting and anomaly detection
- Creating security dashboards and alerts

## Key Concepts

1. **Security signals** - Dedicated signal for security events
2. **Pattern detection** - Failed logins, suspicious requests, rate violations
3. **User tracking** - Monitor activity by user/IP for anomalies
4. **Real-time response** - Trigger automatic responses to threats

## Running the Example

```bash
go run main.go
```

You'll see:
- Normal activity logs go to stderr
- Security events go to `security.log` file

Expected console output:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Security monitor started","caller":"main.go:15"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Processing login attempts","caller":"main.go:25"}
```

Expected security.log content:
```json
{"time":"2024-01-20T10:30:45Z","signal":"SECURITY","message":"Multiple failed login attempts","caller":"main.go:30","user":"alice","ip":"192.168.1.100","attempts":5,"period":"5m","threat_level":"medium"}
{"time":"2024-01-20T10:30:45Z","signal":"SECURITY","message":"SQL injection attempt detected","caller":"main.go:35","ip":"10.0.0.50","path":"/api/users","payload":"'; DROP TABLE users; --","threat_level":"high"}
{"time":"2024-01-20T10:30:45Z","signal":"SECURITY","message":"Rate limit exceeded","caller":"main.go:40","ip":"172.16.0.1","requests":1000,"window":"1m","action":"blocked","threat_level":"low"}
```

## Security Patterns Detected

- **Brute force**: Multiple failed login attempts
- **SQL injection**: Malicious SQL in parameters
- **Path traversal**: Attempts to access unauthorized files
- **Rate violations**: Excessive requests from single source
- **Privilege escalation**: Attempts to access admin functions
- **Data exfiltration**: Large data transfers or bulk exports

## Response Actions

Based on threat level:
- **Low**: Log and monitor
- **Medium**: Rate limit or temporary block
- **High**: Immediate block and alert
- **Critical**: Full lockdown and incident response

## Integration Points

Security logs can trigger:
- WAF rules updates
- IP blacklisting
- User account locks
- Incident response workflows
- SIEM integration

## Next Steps

- See [05-error-alerting](../05-error-alerting) for security alerts
- See [03-audit-trail](../03-audit-trail) for compliance logging