# Incident Investigation Checklist

Printable checklist for systematic incident investigation using IBM Cloud
Logs and the `smart_investigate` tool.

---

## Pre-Investigation

- [ ] Confirm the incident report: what symptoms are observed?
- [ ] Identify the time window when the issue started
- [ ] Determine if a specific service, trace ID, or correlation ID is known
- [ ] Select the appropriate investigation mode:
  - **Global** -- no specific service or trace; system-wide scan
  - **Component** -- specific service name is known
  - **Flow** -- specific trace_id or correlation_id is available

---

## Global Mode Checklist

Use when: no specific service is identified, or performing a broad health scan.

- [ ] Run `smart_investigate` with no application/trace parameters
- [ ] Review error rate per application (top 20 by volume)
- [ ] Check error timeline for spikes (> 3x average)
- [ ] Review CRITICAL severity events for recurring patterns
- [ ] Identify the top affected services
- [ ] For each high-error service, drill down with component mode
- [ ] Check if spikes correlate with deployment times
- [ ] Review cross-service error patterns for cascading failures

---

## Component Mode Checklist

Use when: a specific service is identified as the target.

- [ ] Run `smart_investigate` with `application: <service_name>`
- [ ] Review all errors from the service (up to 200)
- [ ] Identify top 5 recurring error patterns by message
- [ ] Check error distribution across subsystems
- [ ] Scan for dependency failure patterns:
  - [ ] Connection refused
  - [ ] Timeout / timed out
  - [ ] Connection reset (econnreset)
  - [ ] Connection timed out (etimedout)
  - [ ] Pool exhausted
  - [ ] Deadlock
  - [ ] Too many connections
- [ ] Check downstream service health if dependency issues found
- [ ] Correlate errors with recent deployments
- [ ] Review subsystem-level error counts (flag if > 20)

---

## Flow Mode Checklist

Use when: a specific request needs to be traced across services.

- [ ] Obtain trace_id or correlation_id from the failed request
- [ ] Run `smart_investigate` with `trace_id` or `correlation_id`
- [ ] Review the service traversal path
- [ ] Identify the first error event in the request flow
- [ ] Note which service the request failed at
- [ ] Drill down into the failing service with component mode
- [ ] Check if the failure pattern is isolated or recurring

---

## Heuristic Pattern Review

After any investigation mode, check if these patterns were detected:

- [ ] **Timeout** -- downstream slow, network issues, connection pool problems
- [ ] **Memory** -- OOM, heap exhaustion, GC overhead, memory leaks
- [ ] **Database** -- connection pool, deadlocks, slow queries, max connections
- [ ] **Auth** -- 401/403 errors, expired tokens, invalid credentials
- [ ] **Rate Limit** -- 429 errors, throttling, quota exceeded
- [ ] **Network** -- connection refused/reset, DNS, TLS/SSL, 502/503

---

## Remediation Actions

- [ ] Review suggested next actions from the investigation report
- [ ] Execute follow-up queries for identified heuristic patterns
- [ ] If `generate_assets: true` was used, review generated configurations:
  - [ ] Alert configuration (Terraform HCL or JSON)
  - [ ] Dashboard configuration (widget definitions)
- [ ] Apply relevant SOP based on matched pattern:

### Timeout SOP
- [ ] Check downstream service health status
- [ ] Review network latency metrics
- [ ] Verify connection pool settings
- [ ] Check for resource contention (CPU/Memory)
- [ ] Review recent deployments or configuration changes
- [ ] Escalate to Platform team if unresolved in 15 minutes

### Memory SOP
- [ ] Check container memory limits (`kubectl top pods`)
- [ ] Review JVM heap settings (`-Xmx`, `-Xms`)
- [ ] Analyze heap dumps if available
- [ ] Check for memory leaks in recent deployments
- [ ] Consider horizontal scaling
- [ ] Review object caching configurations
- [ ] Escalate to Development team if OOMKilled

### Database SOP
- [ ] Check database connection pool settings
- [ ] Review slow query logs
- [ ] Check database CPU and memory utilization
- [ ] Verify `max_connections` settings
- [ ] Look for long-running transactions
- [ ] Check for table locks or deadlocks
- [ ] Escalate to DBA team

### Auth SOP
- [ ] Verify service credentials and API keys
- [ ] Check IAM policy changes
- [ ] Review token expiration settings
- [ ] Check for certificate issues
- [ ] Verify OAuth/OIDC provider status
- [ ] Review recent permission changes
- [ ] Escalate to Security team if security incident suspected

### Rate Limit SOP
- [ ] Identify the source of excessive requests
- [ ] Review rate limit configurations
- [ ] Check for retry storms
- [ ] Implement exponential backoff if not present
- [ ] Consider request caching or batching
- [ ] Contact API provider if external limit
- [ ] Escalate to Engineering lead if business-critical

### Network SOP
- [ ] Verify DNS resolution
- [ ] Check network policies and security groups
- [ ] Verify service endpoints are accessible
- [ ] Check load balancer health
- [ ] Review SSL/TLS certificate validity
- [ ] Check for network partitions
- [ ] Escalate to Platform/Network team if infrastructure-wide

---

## Post-Investigation

- [ ] Document root cause and confidence level
- [ ] Record affected services and impact summary
- [ ] Deploy generated alerts if applicable
- [ ] Deploy generated dashboards if applicable
- [ ] Schedule post-incident review
- [ ] Update runbooks with new findings
