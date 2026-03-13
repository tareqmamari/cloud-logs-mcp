# Runbook Templates

Per-component runbook templates extracted from the `AlertingStrategyMatrix`.
Customize these for your specific services and infrastructure.

---

## Web Service Alert Runbook

### Initial Triage
1. Check service health endpoint: `GET /health`
2. Review recent deployments in last 24 hours
3. Check dependent service health

### Investigation Steps
1. Query error logs:
   ```
   source logs | filter $m.severity >= 5 | top 10 $d.error_type
   ```
2. Check latency distribution:
   ```
   source logs | stats percentile($d.response_time_ms, 50, 90, 99)
   ```
3. Identify affected endpoints:
   ```
   source logs | filter $d.status_code >= 500 | top 10 $d.path
   ```

### Escalation
- P1: Page on-call engineer immediately
- P2: Create ticket, notify team channel

---

## API Gateway Alert Runbook

### Initial Triage
1. Check gateway health metrics
2. Verify upstream service connectivity
3. Review rate limiting configuration

### Investigation Steps
1. Identify affected routes:
   ```
   source logs | filter $d.component == 'api_gateway' | top 10 $d.route
   ```
2. Check client distribution:
   ```
   source logs | filter $d.component == 'api_gateway' | top 10 $d.client_id
   ```

### Escalation
- P1: Page platform team
- P2: Create ticket for API team

---

## Database Alert Runbook

### Initial Triage
1. Check database server resource utilization (CPU, Memory, Disk I/O)
2. Review active connections and running queries
3. Check replication status if applicable

### Investigation Steps
1. Identify slow queries:
   ```
   source logs | filter $d.query_duration_ms > 1000 | top 10 $d.query_fingerprint
   ```
2. Check connection sources:
   ```
   source logs | filter $d.component == 'database' | top 10 $d.client_host
   ```
3. Review lock contention:
   ```
   source logs | filter $d.lock_wait_ms > 0 | stats avg($d.lock_wait_ms)
   ```

### Escalation
- P1: Page DBA on-call
- P2: Create ticket for database team

---

## Cache Alert Runbook

### Initial Triage
1. Check cache server memory utilization
2. Review hit/miss ratio trends
3. Check eviction statistics

### Investigation Steps
1. Identify hot keys:
   ```
   source logs | filter $d.component == 'cache' | top 10 $d.key_prefix
   ```
2. Check client connections:
   ```
   source logs | filter $d.component == 'cache' | stats count() by $d.client
   ```

### Escalation
- P1: Page infrastructure on-call
- P2: Create ticket for platform team

---

## Message Queue Alert Runbook

### Initial Triage
1. Check broker health and connectivity
2. Review consumer group status
3. Check for dead letter queue messages

### Investigation Steps
1. Identify affected queues:
   ```
   source logs | filter $d.component == 'queue' | top 10 $d.queue_name by $d.queue_depth
   ```
2. Check consumer status:
   ```
   source logs | filter $d.consumer_group exists | stats count() by $d.consumer_group, $d.status
   ```
3. Review error patterns:
   ```
   source logs | filter $d.component == 'queue' AND $m.severity >= 5 | top 10 $d.error_type
   ```

### Escalation
- P1: Page platform on-call (message loss risk)
- P2: Create ticket for application team (processing delay)

---

## Worker Alert Runbook

### Initial Triage
1. Check worker process health
2. Review job queue depth
3. Check dependent service connectivity

### Investigation Steps
1. Identify failing jobs:
   ```
   source logs | filter $d.component == 'worker' AND $d.job_status == 'failed' | top 10 $d.job_type
   ```
2. Check error reasons:
   ```
   source logs | filter $d.component == 'worker' AND $m.severity >= 5 | top 10 $d.error_message
   ```

### Escalation
- P1: Page on-call (critical job failures)
- P2: Create ticket (degraded performance)

---

## Kubernetes Alert Runbook

### Initial Triage
1. Check cluster node health: `kubectl get nodes`
2. Review pod status: `kubectl get pods -A | grep -v Running`
3. Check resource quotas: `kubectl describe resourcequota`

### Investigation Steps
1. Check pod events: `kubectl describe pod <pod-name>`
2. Review container logs: `kubectl logs <pod-name> --previous`
3. Check resource usage: `kubectl top pods`

### Escalation
- P1: Page platform on-call (cluster-wide issues)
- P2: Create ticket for application team (single service)

---

## General Runbook Template

Use this template for component types not listed above.

### Initial Triage
1. Acknowledge alert and check current status
2. Review recent changes (deployments, config changes)
3. Check dependent services health

### Signal-Specific Investigation

#### For Errors
4. Query recent error logs to identify error patterns
5. Check error rate trend to determine if improving or worsening
6. Identify affected endpoints/users

#### For Duration/Latency
4. Check P50/P90/P99 latency distribution
5. Identify slow endpoints or queries
6. Check resource utilization (CPU, memory, I/O)

#### For Saturation
4. Check resource capacity and utilization
5. Identify resource consumers
6. Consider scaling or capacity increase

#### For Utilization
4. Verify utilization trend direction
5. Identify top resource consumers
6. Plan capacity increase if trend continues

### Escalation
- P1: Page on-call engineer immediately
- P2: Create ticket for responsible team
- P3: Add to monitoring dashboard for trend tracking
