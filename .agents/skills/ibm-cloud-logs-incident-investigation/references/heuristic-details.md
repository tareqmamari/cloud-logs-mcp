# Heuristic Pattern Matching Details

The `HeuristicEngine` runs six matchers against investigation findings
and raw log events. Each matcher checks finding summaries (case-insensitive)
for specific text patterns and, when triggered, provides a suggested
follow-up action and a standard operating procedure (SOP).

Matchers are evaluated in order. Actions are deduplicated by description
and sorted by priority (lower number = higher priority).

## Timeout Heuristic

**Name:** `timeout_detector`

### Trigger Patterns

- `timeout`
- `timed out`
- `deadline exceeded`
- `context deadline`
- `read timeout`
- `write timeout`
- `connection timeout`
- `request timeout`
- `504`

### Suggested Action

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| Priority    | 1                                                        |
| Type        | correlate                                                |
| Description | Check downstream service health and network latency      |
| Rationale   | Timeout errors indicate slow downstream services or network issues |

**Follow-up query** (when service is known):

```dataprime
source logs
| filter $l.applicationname == '<service>'
| filter $d.duration_ms.exists()
| calculate
    avg($d.duration_ms) as avg_latency,
    percentile($d.duration_ms, 95) as p95_latency,
    percentile($d.duration_ms, 99) as p99_latency
| limit 1
```

### SOP

**Trigger:** Timeout errors detected

**Procedure:**
1. Check downstream service health status
2. Review network latency metrics
3. Verify connection pool settings
4. Check for resource contention (CPU/Memory)
5. Review recent deployments or configuration changes

**Escalation:** If unresolved in 15 minutes, escalate to Platform team

---

## Memory Heuristic

**Name:** `memory_detector`

### Trigger Patterns

- `out of memory`
- `oom`
- `heap space`
- `memory limit`
- `gc overhead`
- `allocation failure`
- `java.lang.outofmemory`
- `fatal error: runtime: out of memory`
- `oomkilled`
- `memory pressure`
- `memory leak`

### Suggested Action

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| Priority    | 1                                                        |
| Type        | query                                                    |
| Description | Check container resource limits and memory trends        |
| Rationale   | Memory errors indicate potential leaks or insufficient limits |

### SOP

**Trigger:** Memory pressure detected

**Procedure:**
1. Check container memory limits (`kubectl top pods`)
2. Review JVM heap settings (`-Xmx`, `-Xms`)
3. Analyze heap dumps if available
4. Check for memory leaks in recent deployments
5. Consider horizontal scaling
6. Review object caching configurations

**Escalation:** If OOMKilled, escalate to Development team immediately

---

## Database Heuristic

**Name:** `database_detector`

### Trigger Patterns

- `connection pool`
- `too many connections`
- `deadlock`
- `lock wait timeout`
- `cannot acquire`
- `database`
- `sql`
- `query failed`
- `transaction`
- `postgres`
- `mysql`
- `mongodb`
- `redis`
- `connection refused`
- `max_connections`
- `slow query`
- `query timeout`

### Suggested Action

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| Priority    | 1                                                        |
| Type        | query                                                    |
| Description | Analyze slow database queries                            |
| Rationale   | Database issues often cause cascading failures           |

**Follow-up query** (when service is known):

```dataprime
source logs
| filter $l.applicationname == '<service>' && $d.sql.exists()
| groupby $d.sql
| calculate
    avg($d.exec_ms) as avg_time,
    max($d.exec_ms) as max_time,
    count() as query_count
| sortby -avg_time
| limit 10
```

### SOP

**Trigger:** Database connection/query issues detected

**Procedure:**
1. Check database connection pool settings
2. Review slow query logs
3. Check database CPU and memory utilization
4. Verify `max_connections` settings
5. Look for long-running transactions
6. Check for table locks or deadlocks

**Escalation:** If database-related, escalate to DBA team

---

## Auth Heuristic

**Name:** `auth_detector`

### Trigger Patterns

- `unauthorized`
- `forbidden`
- `401`
- `403`
- `authentication failed`
- `invalid token`
- `expired token`
- `access denied`
- `permission denied`
- `invalid credentials`
- `jwt`
- `oauth`
- `saml`

### Suggested Action

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| Priority    | 2                                                        |
| Type        | query                                                    |
| Description | Investigate authentication failures                      |
| Rationale   | Auth failures may indicate credential issues or security incidents |

**Follow-up query** (when service is known):

```dataprime
source logs
| filter $l.applicationname == '<service>'
| filter $d.message.contains('401') || $d.message.contains('403') || $d.message.contains('auth')
| groupby $d.user_id, $d.endpoint
| calculate count() as failures
| sortby -failures
| limit 20
```

### SOP

**Trigger:** Authentication/Authorization failures detected

**Procedure:**
1. Verify service credentials and API keys
2. Check IAM policy changes
3. Review token expiration settings
4. Check for certificate issues
5. Verify OAuth/OIDC provider status
6. Review recent permission changes

**Escalation:** If security incident suspected, escalate to Security team immediately

---

## Rate Limit Heuristic

**Name:** `rate_limit_detector`

### Trigger Patterns

- `rate limit`
- `429`
- `too many requests`
- `throttled`
- `quota exceeded`
- `limit exceeded`
- `backoff`

### Suggested Action

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| Priority    | 2                                                        |
| Type        | correlate                                                |
| Description | Analyze request patterns and rate limits                 |
| Rationale   | Rate limiting indicates traffic spikes or misconfigured limits |

### SOP

**Trigger:** Rate limiting detected

**Procedure:**
1. Identify the source of excessive requests
2. Review rate limit configurations
3. Check for retry storms
4. Implement exponential backoff if not present
5. Consider request caching or batching
6. Contact API provider if external limit

**Escalation:** If business-critical, escalate to Engineering lead

---

## Network Heuristic

**Name:** `network_detector`

### Trigger Patterns

- `connection refused`
- `connection reset`
- `no route to host`
- `network unreachable`
- `dns`
- `econnrefused`
- `econnreset`
- `socket`
- `tcp`
- `ssl`
- `tls`
- `certificate`
- `502`
- `503`
- `bad gateway`
- `service unavailable`

### Suggested Action

| Field       | Value                                                    |
|-------------|----------------------------------------------------------|
| Priority    | 1                                                        |
| Type        | correlate                                                |
| Description | Check network connectivity and DNS resolution            |
| Rationale   | Network errors indicate infrastructure or connectivity issues |

### SOP

**Trigger:** Network connectivity issues detected

**Procedure:**
1. Verify DNS resolution
2. Check network policies and security groups
3. Verify service endpoints are accessible
4. Check load balancer health
5. Review SSL/TLS certificate validity
6. Check for network partitions

**Escalation:** If infrastructure-wide, escalate to Platform/Network team
