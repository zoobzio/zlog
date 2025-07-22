# Example 08: Business Events

This example shows how to use custom signals to track business events separately from technical logs, enabling product analytics and business intelligence.

## What It Shows

- Creating custom business event signals (REVENUE, ENGAGEMENT, CONVERSION)
- Tracking user behavior and business metrics
- Separating business insights from operational logs
- Building analytics pipelines with structured events

## Key Concepts

1. **Business signals** - Domain-specific signals for your business
2. **Product analytics** - User behavior, feature usage, conversion funnels
3. **Revenue tracking** - Purchases, subscriptions, churn events
4. **Clean separation** - Business events don't clutter operational logs

## Running the Example

```bash
go run main.go
```

You'll see:
- Operational logs go to stderr
- Business events go to `business-events.json` file

Expected console output:
```json
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"E-commerce platform started","caller":"main.go:15"}
{"time":"2024-01-20T10:30:45Z","signal":"INFO","message":"Processing user sessions","caller":"main.go:20"}
```

Expected business-events.json content:
```json
{"time":"2024-01-20T10:30:45Z","signal":"REVENUE","message":"Purchase completed","user_id":"user123","amount":99.99,"currency":"USD","product_id":"prod_456","product_name":"Premium Plan"}
{"time":"2024-01-20T10:30:45Z","signal":"ENGAGEMENT","message":"Feature used","user_id":"user456","feature":"advanced_search","duration_seconds":45,"success":true}
{"time":"2024-01-20T10:30:45Z","signal":"CONVERSION","message":"Trial to paid conversion","user_id":"user789","trial_days":14,"plan":"professional","mrr":49.99}
```

## Business Event Types

- **REVENUE**: Purchases, subscriptions, upgrades, refunds
- **ENGAGEMENT**: Feature usage, session duration, user actions
- **CONVERSION**: Signups, trial conversions, funnel progression
- **RETENTION**: User returns, churn, reactivation
- **EXPERIMENT**: A/B test events, feature flags

## Analytics Integration

Business events can feed into:
- Product analytics tools (Mixpanel, Amplitude)
- Data warehouses (BigQuery, Snowflake)
- BI tools (Looker, Tableau)
- Custom dashboards

## Use Cases

- **Product decisions**: Which features are actually used?
- **Revenue optimization**: What drives conversions?
- **User journey**: How do users progress through your app?
- **Experiments**: A/B test results and feature impact

## Next Steps

- See [04-metrics-pipeline](../04-metrics-pipeline) for technical metrics
- See [03-audit-trail](../03-audit-trail) for compliance events