# IBM Cloud Logs DataPrime Query Templates Catalog

A comprehensive reference of ready-to-use DataPrime query templates organized by category. Each template includes a full query, parameters, use cases, and tips for effective usage.

---

## Category: discovery

Use these templates when you are starting an investigation without prior knowledge of where the problem is. They help you orient quickly by surfacing high-signal patterns across all services.

---

### 1. error_hotspots

**Find which services have the most errors - start investigations here**

```dataprime
source logs | filter $m.severity >= 5 | groupby $l.applicationname, $l.subsystemname | aggregate count() as error_count | sortby -error_count | limit 20
```

**Use cases:**
- Start incident investigation without prior knowledge
- Identify the source of problems
- Prioritize which service to investigate first

**Tips:**
- This is often the first query in an investigation
- High error counts indicate the problem area
- Follow up with error_details on top services

---

### 2. anomaly_detection

**Find services with unusually high error rates**

```dataprime
source logs | filter $m.severity >= 5 | groupby $l.applicationname | aggregate count() as error_count | filter error_count > 10 | sortby -error_count | limit 20
```

**Use cases:**
- Detect services behaving abnormally
- Find issues even in low-traffic services
- Identify degradation before it becomes critical

**Tips:**
- Error rate > 1% typically indicates problems
- Look at both rate AND absolute count
- Compare with historical baseline

---

### 3. top_error_messages

**Group similar errors to find the most common problems**

```dataprime
source logs | filter $m.severity >= 5 | groupby $d.message:string | aggregate count() as occurrences, min($m.timestamp) as first_seen, max($m.timestamp) as last_seen | filter occurrences >= 3 | sortby -occurrences | limit 30
```

**Use cases:**
- Find the most impactful errors
- Identify systemic issues vs one-offs
- Prioritize fixes by impact

**Tips:**
- High occurrence count = high impact
- Check first_seen vs last_seen for duration
- Similar messages indicate same root cause

---

### 4. noise_filtered_errors

**Error analysis excluding health checks and known noise**

```dataprime
source logs | filter $m.severity >= 5 && !$d.message:string.toLowerCase().contains('health') && !$d.message:string.toLowerCase().contains('ping') && !$d.message:string.toLowerCase().contains('heartbeat') && !$d.message:string.toLowerCase().contains('metrics') | groupby $l.applicationname | aggregate count() as errors | sortby -errors | limit 20
```

**Use cases:**
- Find real errors without health check noise
- Cleaner investigation starting point
- Focus on application errors only

**Tips:**
- Add more exclusions for your environment
- Good for production systems with monitoring
- Compare with unfiltered to verify exclusions

---

### 5. traffic_overview

**See overall traffic patterns by service**

```dataprime
source logs | groupby $l.applicationname | aggregate count() as volume, approx_count_distinct($l.subsystemname) as components | sortby -volume | limit 20
```

**Use cases:**
- Understand system topology
- Find unexpected traffic patterns
- Identify services to investigate

**Tips:**
- Unexpected high volume may indicate problems
- Compare with normal patterns
- Use before drilling into specific services

---

### 6. recent_changes

**Detect recent deployments or restarts that might explain issues**

```dataprime
source logs | filter $d.message:string.contains('start') || $d.message:string.contains('deploy') || $d.message:string.contains('version') || $d.message:string.contains('initializ') || $d.message:string.contains('shutdown') | groupby $l.applicationname | aggregate count() as events, min($m.timestamp) as earliest, max($m.timestamp) as latest | filter events >= 2 | sortby -latest | limit 20
```

**Use cases:**
- Correlate issues with recent changes
- Find which services were recently deployed
- Identify crash loops or restarts

**Tips:**
- Issues often follow deployments
- Multiple events may indicate crash loops
- Compare timing with error onset

---

## Category: error

Use these templates to investigate errors once you have identified a problem area. They provide increasing levels of detail from broad triage to deep per-service analysis.

---

### 7. error_spike

**Find error spikes in the last hour grouped by application**

```dataprime
source logs | filter $m.severity >= 5 | groupby $l.applicationname | aggregate count() as error_count | sortby -error_count | limit 20
```

**Use cases:**
- Identify which applications are producing the most errors
- Quick triage during incidents
- Prioritize debugging efforts

**Tips:**
- Severity 5 = Error, 6 = Critical
- Expand time range if no results
- Follow up with error_details template for specific app

---

### 8. error_details

**Get detailed error logs for a specific application**

```dataprime
source logs | filter $l.applicationname == '{APPLICATION}' && $m.severity >= 5 | select $m.timestamp, $m.severity, $l.subsystemname, $d.message, $d.error, $d.stack_trace | sortby -$m.timestamp | limit 100
```

**Parameters:**
- `APPLICATION` (required) - Application name to investigate (e.g., `payment-service`)

**Use cases:**
- Deep dive into errors for a specific service
- Find error messages and stack traces
- Identify error patterns

**Tips:**
- Replace `{APPLICATION}` with your app name
- Add time filters for specific incidents
- Look for common patterns in error messages

---

### 9. error_timeline

**Error rate over time for trend analysis**

```dataprime
source logs | filter $m.severity >= 5 | groupby roundTime($m.timestamp, 1m) as time_bucket | aggregate count() as errors | sortby time_bucket
```

**Use cases:**
- Visualize error trends over time
- Identify when errors started
- Correlate with deployments or changes

**Tips:**
- Adjust roundTime interval for different granularity (1m, 5m, 1h)
- Compare with normal periods
- Use for dashboard creation

---

## Category: performance

Use these templates to investigate latency and throughput issues. They are most effective when your application logs include response time fields.

---

### 10. slow_requests

**Find slow requests with response time above threshold**

```dataprime
source logs | filter $d.response_time_ms > 1000 | select $m.timestamp, $l.applicationname, $d.endpoint, $d.response_time_ms, $d.status_code | sortby -$d.response_time_ms | limit 50
```

**Parameters:**
- `threshold_ms` (optional) - Response time threshold in milliseconds (default: `1000`, example: `500`)

**Use cases:**
- Identify performance bottlenecks
- Find slowest endpoints
- SLA compliance checking

**Tips:**
- Adjust threshold based on your SLOs
- Check if slow requests correlate with errors
- Group by endpoint to find problematic routes

---

### 11. latency_percentiles

**Calculate latency percentiles by endpoint**

```dataprime
source logs | filter $d.response_time_ms > 0 | groupby $d.endpoint | aggregate percentile($d.response_time_ms, 50) as p50, percentile($d.response_time_ms, 95) as p95, percentile($d.response_time_ms, 99) as p99, count() as requests | sortby -requests | limit 20
```

**Use cases:**
- Understand latency distribution
- Set realistic SLO targets
- Identify outlier endpoints

**Tips:**
- P50 = typical user experience
- P95/P99 = worst case scenarios
- Compare across time periods

---

### 12. throughput_analysis

**Request throughput over time by application**

```dataprime
source logs | filter $d.request_id != '' | groupby roundTime($m.timestamp, 1m) as time_bucket, $l.applicationname | aggregate count() as requests | sortby time_bucket
```

**Use cases:**
- Monitor traffic patterns
- Capacity planning
- Detect traffic anomalies

**Tips:**
- Adjust roundTime interval for different granularity
- Compare with baseline periods
- Useful for dashboard widgets

---

## Category: security

Use these templates to detect and investigate security events. Adapt the filter patterns to match the event schema of your application's audit logs.

---

### 13. auth_failures

**Find authentication failures grouped by source**

```dataprime
source logs | filter $d.event_type == 'auth_failure' || $d.message.contains('authentication failed') || $d.message.contains('invalid credentials') | groupby $d.source_ip, $d.username | aggregate count() as failures | filter failures > 3 | sortby -failures
```

**Use cases:**
- Detect brute force attempts
- Identify compromised accounts
- Security incident investigation

**Tips:**
- Threshold of 3 filters noise
- Group by IP to detect attacks
- Follow up with IP geolocation

---

### 14. privilege_escalation

**Detect privilege escalation attempts**

```dataprime
source logs | filter $d.event_type.contains('privilege') || $d.message.contains('sudo') || $d.message.contains('root access') || $d.message.contains('admin') | select $m.timestamp, $l.applicationname, $d.username, $d.action, $d.message | sortby -$m.timestamp | limit 100
```

**Use cases:**
- Detect unauthorized privilege changes
- Audit admin actions
- Compliance reporting

**Tips:**
- Customize patterns for your environment
- Correlate with known admin activities
- Set up alerts for critical matches

---

### 15. sensitive_data_access

**Track access to sensitive resources**

```dataprime
source logs | filter $d.resource.contains('pii') || $d.resource.contains('secrets') || $d.resource.contains('credentials') || $d.endpoint.contains('/admin') | select $m.timestamp, $d.username, $d.resource, $d.action, $d.source_ip | sortby -$m.timestamp | limit 100
```

**Use cases:**
- PII access auditing
- Secrets management monitoring
- Compliance evidence gathering

**Tips:**
- Adjust resource patterns for your data
- Export for compliance reports
- Set up alerts for unusual access

---

## Category: health

Use these templates for ongoing service health monitoring, SRE dashboards, and detecting silent failures where a service stops logging without producing visible errors.

---

### 16. service_health

**Overall health summary by service**

```dataprime
source logs | filter $m.severity >= 5 | groupby $l.applicationname | aggregate count() as error_count | sortby -error_count | limit 20
```

**Use cases:**
- Quick health overview
- Identify troubled services
- SRE dashboards

**Tips:**
- Error rate > 1% typically needs attention
- Compare with historical baselines
- Great for morning health checks

---

### 17. heartbeat_check

**Verify services are logging (heartbeat)**

```dataprime
source logs | groupby $l.applicationname | aggregate max($m.timestamp) as last_seen, count() as log_count | sortby last_seen
```

**Use cases:**
- Detect silent failures
- Verify logging is working
- Find stale services

**Tips:**
- Services missing from results may be down
- Check last_seen vs current time
- Set up alerts for missing heartbeats

---

### 18. restart_detection

**Detect service restarts and crashes**

```dataprime
source logs | filter $d.message.contains('starting') || $d.message.contains('started') || $d.message.contains('shutdown') || $d.message.contains('terminated') || $d.message.contains('OOMKilled') | select $m.timestamp, $l.applicationname, $l.subsystemname, $d.message | sortby -$m.timestamp | limit 50
```

**Use cases:**
- Detect crash loops
- Track deployment rollouts
- Investigate instability

**Tips:**
- Multiple restarts indicate problems
- OOMKilled suggests memory issues
- Correlate with error spikes

---

## Category: usage

Use these templates to understand how your system and APIs are being used. Useful for capacity planning, cost allocation, and detecting anomalous usage patterns.

---

### 19. top_endpoints

**Most frequently called endpoints**

```dataprime
source logs | filter $d.endpoint != '' | groupby $d.endpoint | aggregate count() as calls, avg($d.response_time_ms) as avg_latency | sortby -calls | limit 20
```

**Use cases:**
- Identify hot endpoints
- Capacity planning
- Cost allocation

**Tips:**
- Focus optimization on top endpoints
- High calls + high latency = priority fix
- Use for API documentation priorities

---

### 20. user_activity

**User activity summary**

```dataprime
source logs | filter $d.user_id != '' | groupby $d.user_id | aggregate count() as actions, approx_count_distinct($d.endpoint) as unique_endpoints | sortby -actions | limit 50
```

**Use cases:**
- Identify power users
- Detect anomalous behavior
- User engagement metrics

**Tips:**
- Unusually high activity may indicate automation or abuse
- Low unique_endpoints may indicate bots
- Compare with user segments

---

### 21. data_volume

**Log volume by application over time**

```dataprime
source logs | groupby roundTime($m.timestamp, 1h) as time_bucket, $l.applicationname | aggregate count() as logs | sortby time_bucket
```

**Use cases:**
- Cost monitoring
- Capacity planning
- Detect logging anomalies

**Tips:**
- Sudden spikes may indicate problems
- Use for log retention decisions
- Identify chatty applications

---

## Category: audit

Use these templates for change management, compliance evidence gathering, and data governance. They are designed to produce audit trails suitable for export and review.

---

### 22. config_changes

**Track configuration changes**

```dataprime
source logs | filter $d.event_type.contains('config') || $d.message.contains('configuration changed') || $d.message.contains('settings updated') | select $m.timestamp, $d.username, $d.resource, $d.old_value, $d.new_value, $d.message | sortby -$m.timestamp | limit 100
```

**Use cases:**
- Change management audit
- Troubleshoot config-related issues
- Compliance evidence

**Tips:**
- Correlate changes with incidents
- Export for change management tickets
- Set up alerts for critical configs

---

### 23. data_exports

**Track data export activities**

```dataprime
source logs | filter $d.action.contains('export') || $d.action.contains('download') || $d.message.contains('exported') | select $m.timestamp, $d.username, $d.resource, $d.record_count, $d.destination | sortby -$m.timestamp | limit 100
```

**Use cases:**
- Data loss prevention
- Compliance auditing
- Detect data exfiltration

**Tips:**
- Large exports need attention
- Unusual destinations are suspicious
- Compare with business justifications

---

### 24. api_key_usage

**Track API key usage patterns**

```dataprime
source logs | filter $d.api_key_id != '' || $d.auth_type == 'api_key' | groupby $d.api_key_id, $l.applicationname | aggregate count() as calls, approx_count_distinct($d.source_ip) as unique_ips | sortby -calls | limit 50
```

**Use cases:**
- API key auditing
- Detect key sharing/abuse
- Key rotation planning

**Tips:**
- Multiple IPs per key may indicate sharing
- Inactive keys should be rotated
- Monitor for unusual patterns

---

## Quick Reference Index

| # | Template | Category | Primary Use |
|---|----------|----------|-------------|
| 1 | error_hotspots | discovery | Start incident investigation |
| 2 | anomaly_detection | discovery | Find abnormal error rates |
| 3 | top_error_messages | discovery | Identify most common errors |
| 4 | noise_filtered_errors | discovery | Clean error analysis |
| 5 | traffic_overview | discovery | Understand system topology |
| 6 | recent_changes | discovery | Detect deployments/restarts |
| 7 | error_spike | error | Triage errors by application |
| 8 | error_details | error | Deep dive per service |
| 9 | error_timeline | error | Error trend analysis |
| 10 | slow_requests | performance | Find slow requests |
| 11 | latency_percentiles | performance | P50/P95/P99 by endpoint |
| 12 | throughput_analysis | performance | Request throughput over time |
| 13 | auth_failures | security | Detect brute force |
| 14 | privilege_escalation | security | Detect privilege abuse |
| 15 | sensitive_data_access | security | PII and secrets access |
| 16 | service_health | health | Health summary dashboard |
| 17 | heartbeat_check | health | Detect silent failures |
| 18 | restart_detection | health | Detect crash loops |
| 19 | top_endpoints | usage | Identify hot endpoints |
| 20 | user_activity | usage | User behavior analysis |
| 21 | data_volume | usage | Log volume and cost |
| 22 | config_changes | audit | Change management trail |
| 23 | data_exports | audit | Data exfiltration detection |
| 24 | api_key_usage | audit | API key governance |
