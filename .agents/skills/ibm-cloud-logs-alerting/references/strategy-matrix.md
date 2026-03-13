# Alerting Strategy Matrix

Complete reference of the `AlertingStrategyMatrix` mapping component types to alerting strategies,
recommended metrics, queries, thresholds, and best practices.

For DataPrime syntax, see [IBM Cloud Logs Query Skill](../../ibm-cloud-logs-query/SKILL.md).

---

## web_service

- **Methodology**: RED
- **Labels**: `methodology=RED`, `tier=service`

### Metrics

#### request_rate
- **Type**: counter
- **Signal**: rate
- **Default Threshold**: dynamic baseline (0 = use baselines)
- **Unit**: requests/second
- **Description**: Request rate indicates service load and can detect traffic anomalies
- **Query**:
  ```
  source logs | filter $d.type == 'http_request' | stats count() as requests by bin(1m)
  ```
- **Best Practices**:
  - Set baseline from historical p50 traffic
  - Alert on both high AND low traffic (traffic drop may indicate upstream issues)
  - Use dynamic baselines for services with variable traffic patterns

#### error_rate
- **Type**: counter
- **Signal**: errors
- **Default Threshold**: 0.01 (1%)
- **Unit**: percentage
- **Description**: Error rate as percentage of total requests -- primary SLI for availability SLO
- **Query**:
  ```
  source logs | filter $m.severity >= 5 OR $d.status_code >= 500 | stats count() as errors by bin(1m)
  ```
- **Best Practices**:
  - Use burn rate alerting instead of static thresholds
  - Alert on error budget consumption rate, not absolute errors
  - Differentiate between client errors (4xx) and server errors (5xx)

#### latency_p99
- **Type**: histogram
- **Signal**: duration
- **Default Threshold**: 500ms
- **Unit**: milliseconds
- **Description**: P99 latency represents tail latency experienced by 1% of users
- **Query**:
  ```
  source logs | filter $d.response_time_ms exists | stats percentile($d.response_time_ms, 99) as p99_latency by bin(5m)
  ```
- **Best Practices**:
  - Alert on P99, not average latency
  - Set thresholds based on SLO (e.g., "P99 < 200ms for 99.9% of requests")
  - Consider separate alerts for different endpoints with different latency budgets

---

## api_gateway

- **Methodology**: RED
- **Labels**: `methodology=RED`, `tier=infrastructure`

### Metrics

#### upstream_error_rate
- **Type**: counter
- **Signal**: errors
- **Default Threshold**: 0.005 (0.5%)
- **Unit**: percentage
- **Description**: Upstream service errors proxied through the gateway
- **Query**:
  ```
  source logs | filter $d.component == 'api_gateway' AND $d.upstream_status >= 500 | stats count() as errors by bin(1m)
  ```
- **Best Practices**:
  - Separate upstream errors from gateway errors
  - Track error rates per upstream service
  - Alert on circuit breaker activations

#### gateway_latency
- **Type**: histogram
- **Signal**: duration
- **Default Threshold**: 100ms
- **Unit**: milliseconds
- **Description**: Gateway processing latency (excluding upstream time)
- **Query**:
  ```
  source logs | filter $d.component == 'api_gateway' | stats percentile($d.latency_ms, 99) as p99 by bin(1m)
  ```
- **Best Practices**:
  - Gateway latency should be minimal (< 50ms typically)
  - High gateway latency indicates gateway resource constraints

#### rate_limit_triggers
- **Type**: counter
- **Signal**: saturation
- **Default Threshold**: 100
- **Unit**: count
- **Description**: Rate limiting activations indicate potential abuse or capacity issues
- **Query**:
  ```
  source logs | filter $d.rate_limited == true | stats count() by bin(5m)
  ```
- **Best Practices**:
  - Track rate limits per API key/client
  - Alert on sudden spikes in rate limiting

---

## database

- **Methodology**: USE
- **Labels**: `methodology=USE`, `tier=data`

### Metrics

#### connection_utilization
- **Type**: gauge
- **Signal**: utilization
- **Default Threshold**: 80%
- **Unit**: percentage
- **Description**: Connection pool utilization -- high values indicate capacity constraints
- **Query**:
  ```
  source logs | filter $d.component == 'database' | stats avg($d.connections_active / $d.connections_max * 100) as utilization by bin(1m)
  ```
- **Best Practices**:
  - Alert at 80% utilization (warning) and 95% (critical)
  - Track connection pool exhaustion events separately
  - Consider connection pooling or read replicas if consistently high

#### query_queue_depth
- **Type**: gauge
- **Signal**: saturation
- **Default Threshold**: 10 queries
- **Unit**: queries
- **Description**: Query queue depth indicates database saturation
- **Query**:
  ```
  source logs | filter $d.component == 'database' | stats max($d.query_queue_length) as queue_depth by bin(1m)
  ```
- **Best Practices**:
  - Queue depth > 0 indicates queries are waiting
  - Sustained queueing indicates need for optimization or scaling

#### replication_lag
- **Type**: gauge
- **Signal**: saturation
- **Default Threshold**: 5 seconds
- **Unit**: seconds
- **Description**: Replication lag affects read consistency and failover capability
- **Query**:
  ```
  source logs | filter $d.component == 'database' AND $d.role == 'replica' | stats max($d.replication_lag_seconds) as lag by bin(1m)
  ```
- **Best Practices**:
  - Alert if lag exceeds your consistency requirements
  - High lag during failover increases data loss risk

#### slow_queries
- **Type**: counter
- **Signal**: errors
- **Default Threshold**: 10
- **Unit**: count
- **Description**: Slow queries impact application performance and may indicate missing indexes
- **Query**:
  ```
  source logs | filter $d.component == 'database' AND $d.query_duration_ms > 1000 | stats count() as slow_queries by bin(5m)
  ```
- **Best Practices**:
  - Capture query fingerprints for slow query analysis
  - Track slow query rate as percentage of total queries

---

## cache

- **Methodology**: USE
- **Labels**: `methodology=USE`, `tier=caching`

### Metrics

#### memory_utilization
- **Type**: gauge
- **Signal**: utilization
- **Default Threshold**: 85%
- **Unit**: percentage
- **Description**: Memory utilization affects cache eviction behavior
- **Query**:
  ```
  source logs | filter $d.component == 'redis' OR $d.component == 'memcached' | stats avg($d.used_memory / $d.max_memory * 100) as utilization by bin(1m)
  ```
- **Best Practices**:
  - Alert before reaching maxmemory to prevent unexpected evictions
  - Track eviction rate alongside memory utilization

#### hit_rate
- **Type**: gauge
- **Signal**: utilization
- **Default Threshold**: 90% (alert below)
- **Unit**: percentage
- **Description**: Cache hit rate indicates cache effectiveness
- **Query**:
  ```
  source logs | filter $d.component == 'cache' | stats sum($d.hits) / (sum($d.hits) + sum($d.misses)) * 100 as hit_rate by bin(5m)
  ```
- **Best Practices**:
  - Low hit rate may indicate cache sizing issues or access pattern changes
  - Alert on sudden drops in hit rate
  - Track hit rate per key prefix if possible

#### eviction_rate
- **Type**: counter
- **Signal**: saturation
- **Default Threshold**: 100 keys/5min
- **Unit**: keys/5min
- **Description**: High eviction rate indicates memory pressure
- **Query**:
  ```
  source logs | filter $d.component == 'cache' | stats sum($d.evicted_keys) as evictions by bin(5m)
  ```
- **Best Practices**:
  - Evictions force cache rebuilding, increasing backend load
  - Sustained evictions indicate need for larger cache or TTL tuning

---

## message_queue

- **Methodology**: USE
- **Labels**: `methodology=USE`, `tier=messaging`

### Metrics

#### queue_depth
- **Type**: gauge
- **Signal**: saturation
- **Default Threshold**: 1000 messages
- **Unit**: messages
- **Description**: Queue depth indicates consumer lag -- primary saturation signal
- **Query**:
  ```
  source logs | filter $d.component == 'queue' | stats max($d.queue_depth) as depth by $d.queue_name, bin(1m)
  ```
- **Best Practices**:
  - Set threshold based on acceptable processing delay
  - Alert on rate of queue depth increase, not just absolute value
  - Track queue depth per queue/topic

#### consumer_lag
- **Type**: gauge
- **Signal**: saturation
- **Default Threshold**: 10000 messages
- **Unit**: messages
- **Description**: Consumer lag in Kafka indicates processing backlog
- **Query**:
  ```
  source logs | filter $d.component == 'kafka' | stats max($d.consumer_lag) as lag by $d.consumer_group, bin(1m)
  ```
- **Best Practices**:
  - Set lag threshold based on message rate and acceptable delay
  - Alert on lag increasing over time, not just threshold

#### dead_letter_queue
- **Type**: counter
- **Signal**: errors
- **Default Threshold**: 1 message (any DLQ activity)
- **Unit**: messages
- **Description**: Dead letter queue messages indicate processing failures
- **Query**:
  ```
  source logs | filter $d.queue_name contains 'dlq' OR $d.queue_name contains 'dead' | stats count() by bin(5m)
  ```
- **Best Practices**:
  - DLQ messages should always trigger investigation
  - Track DLQ rate and implement alerting on any DLQ activity

#### publish_errors
- **Type**: counter
- **Signal**: errors
- **Default Threshold**: 5 errors/min
- **Unit**: errors/min
- **Description**: Publish errors indicate producer issues or broker problems
- **Query**:
  ```
  source logs | filter $d.operation == 'publish' AND $m.severity >= 5 | stats count() by bin(1m)
  ```
- **Best Practices**:
  - Track publish success rate, not just errors
  - Alert on publish latency spikes as early warning

---

## worker

- **Methodology**: RED
- **Labels**: `methodology=RED`, `tier=background`

### Metrics

#### job_success_rate
- **Type**: counter
- **Signal**: errors
- **Default Threshold**: 99%
- **Unit**: percentage
- **Description**: Job success rate is the primary reliability metric for workers
- **Query**:
  ```
  source logs | filter $d.component == 'worker' | stats sum(case when $d.job_status == 'success' then 1 else 0 end) / count() * 100 as success_rate by bin(5m)
  ```
- **Best Practices**:
  - Track success rate per job type
  - Set different thresholds for critical vs non-critical jobs

#### job_duration
- **Type**: histogram
- **Signal**: duration
- **Default Threshold**: 60000ms (60 seconds)
- **Unit**: milliseconds
- **Description**: Job duration affects throughput and resource utilization
- **Query**:
  ```
  source logs | filter $d.component == 'worker' AND $d.job_duration_ms exists | stats percentile($d.job_duration_ms, 95) as p95_duration by $d.job_type, bin(5m)
  ```
- **Best Practices**:
  - Alert on jobs exceeding SLA duration
  - Track duration trends for capacity planning

#### retry_rate
- **Type**: counter
- **Signal**: errors
- **Default Threshold**: 10/5min
- **Unit**: count
- **Description**: Retry rate indicates transient failures affecting reliability
- **Query**:
  ```
  source logs | filter $d.component == 'worker' AND $d.retry_count > 0 | stats count() by bin(5m)
  ```
- **Best Practices**:
  - High retry rates indicate upstream instability
  - Track jobs exhausting retry budget

---

## kubernetes

- **Methodology**: USE
- **Labels**: `methodology=USE`, `tier=platform`

### Metrics

#### pod_restarts
- **Type**: counter
- **Signal**: errors
- **Default Threshold**: 3 restarts/5min
- **Unit**: restarts/5min
- **Description**: Pod restarts indicate application crashes or OOM kills
- **Query**:
  ```
  source logs | filter $d.kubernetes exists AND $d.event_type == 'container_restart' | stats count() by $d.kubernetes.pod_name, bin(5m)
  ```
- **Best Practices**:
  - Alert on restart rate, not just count
  - Track OOMKilled vs CrashLoopBackOff separately

#### cpu_throttling
- **Type**: gauge
- **Signal**: saturation
- **Default Threshold**: 25%
- **Unit**: percentage
- **Description**: CPU throttling indicates resource constraints
- **Query**:
  ```
  source logs | filter $d.kubernetes exists | stats avg($d.cpu_throttled_percentage) as throttled by $d.kubernetes.pod_name, bin(1m)
  ```
- **Best Practices**:
  - High throttling affects latency predictability
  - Consider increasing CPU limits or optimizing application

#### memory_utilization
- **Type**: gauge
- **Signal**: utilization
- **Default Threshold**: 90%
- **Unit**: percentage
- **Description**: Memory utilization near limits risks OOMKill
- **Query**:
  ```
  source logs | filter $d.kubernetes exists | stats avg($d.memory_usage_bytes / $d.memory_limit_bytes * 100) as utilization by $d.kubernetes.pod_name, bin(1m)
  ```
- **Best Practices**:
  - Alert before OOMKill occurs (e.g., 85% warning)
  - Track memory growth rate for leak detection

#### pending_pods
- **Type**: gauge
- **Signal**: saturation
- **Default Threshold**: 5 pods
- **Unit**: pods
- **Description**: Pending pods indicate scheduling constraints
- **Query**:
  ```
  source logs | filter $d.kubernetes.pod_phase == 'Pending' | stats distinctcount($d.kubernetes.pod_name) as pending by bin(1m)
  ```
- **Best Practices**:
  - Track pending duration, not just count
  - Alert on pods pending > 5 minutes
