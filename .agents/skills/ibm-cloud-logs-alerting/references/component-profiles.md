# Component Profiles

Detailed profiles for all 15 component types, including detection keywords from
`DetectComponentType()`, labels, tier assignments, and methodology selection.

## Component Detection

The `DetectComponentType()` function inspects the combined query and use-case text
(lowercased) for keywords. Detection is ordered by specificity -- more specific patterns
are checked first to avoid false matches.

---

## cache
- **Methodology**: USE
- **Tier**: caching
- **Labels**: `methodology=USE`, `tier=caching`
- **Detection Keywords**: `cache`, `memcached`, `hit rate`, `miss rate`, `eviction`, `cache hit`, `cache miss`
- **Detection Priority**: 1 (checked first -- before database to avoid redis confusion)
- **Description**: In-memory caching systems (Redis, Memcached). Monitors memory utilization, hit/miss rates, and eviction pressure.

## kubernetes
- **Methodology**: USE
- **Tier**: platform
- **Labels**: `methodology=USE`, `tier=platform`
- **Detection Keywords**: `kubernetes`, `k8s`, `pod`, `kubectl`, `deployment`, `container restart`
- **Detection Priority**: 2
- **Description**: Kubernetes cluster and workload monitoring. Tracks pod lifecycle, resource utilization, and scheduling health.

## message_queue
- **Methodology**: USE
- **Tier**: messaging
- **Labels**: `methodology=USE`, `tier=messaging`
- **Detection Keywords**: `kafka`, `rabbitmq`, `sqs`, `queue depth`, `consumer lag`, `consumer`, `producer`, `message queue`, `dead letter`
- **Detection Priority**: 3
- **Description**: Message brokers and queuing systems. Monitors queue depth, consumer lag, and message processing failures.

## api_gateway
- **Methodology**: RED
- **Tier**: infrastructure
- **Labels**: `methodology=RED`, `tier=infrastructure`
- **Detection Keywords**: `gateway`, `proxy`, `routing`, `upstream`, `downstream`, `rate limit`
- **Detection Priority**: 4
- **Description**: API gateways and reverse proxies. Tracks upstream errors, gateway latency, and rate limiting.

## worker
- **Methodology**: RED
- **Tier**: background
- **Labels**: `methodology=RED`, `tier=background`
- **Detection Keywords**: `worker`, `job`, `background job`, `async`, `cron`, `scheduler`, `task queue`
- **Detection Priority**: 5
- **Description**: Background job processors and async workers. Monitors job success rate, duration, and retry behavior.

## serverless
- **Methodology**: RED (default for unknown -- not in strategy matrix)
- **Tier**: compute
- **Labels**: `methodology=RED`, `tier=compute`
- **Detection Keywords**: `lambda`, `serverless`, `faas`, `function invocation`
- **Detection Priority**: 6
- **Description**: Serverless functions (AWS Lambda, IBM Cloud Functions). Tracks invocation rate, errors, and cold start latency.

## database
- **Methodology**: USE
- **Tier**: data
- **Labels**: `methodology=USE`, `tier=data`
- **Detection Keywords**: `database`, `db`, `sql`, `postgres`, `mysql`, `mongodb`, `query duration`, `slow query`, `connection pool`
- **Detection Priority**: 7 (checked after cache to avoid redis confusion)
- **Description**: Relational and NoSQL databases. Monitors connection utilization, query performance, replication lag, and saturation.

## web_service
- **Methodology**: RED
- **Tier**: service
- **Labels**: `methodology=RED`, `tier=service`
- **Detection Keywords**: `http`, `request`, `response`, `api`, `endpoint`, `rest`, `graphql`, `status_code`
- **Detection Priority**: 8 (general, checked later to avoid false matches)
- **Description**: HTTP-based web services and APIs. Primary service type for request-driven monitoring.

## storage
- **Methodology**: USE (default for unknown)
- **Tier**: infrastructure
- **Labels**: `methodology=USE`, `tier=infrastructure`
- **Detection Keywords**: `storage`, `disk`, `volume`, `s3`, `blob`, `file system`
- **Detection Priority**: 9
- **Description**: Storage systems (block, object, file). Monitors capacity utilization, I/O throughput, and error rates.

## network
- **Methodology**: USE (default for unknown)
- **Tier**: infrastructure
- **Labels**: `methodology=USE`, `tier=infrastructure`
- **Detection Keywords**: `network`, `dns`, `tcp`, `socket`, `connection timeout`
- **Detection Priority**: 10
- **Description**: Network infrastructure monitoring. Tracks connectivity, DNS resolution, and connection patterns.

## load_balancer
- **Methodology**: RED (default for unknown -- not in strategy matrix)
- **Tier**: infrastructure
- **Labels**: `methodology=RED`, `tier=infrastructure`
- **Detection Keywords**: None (must be specified explicitly)
- **Detection Priority**: N/A
- **Description**: Load balancers and traffic distribution. Monitors request distribution, backend health, and connection draining.

## cron_job
- **Methodology**: RED (default for unknown -- not in strategy matrix)
- **Tier**: background
- **Labels**: `methodology=RED`, `tier=background`
- **Detection Keywords**: None (detected via `worker` keywords: `cron`, `scheduler`)
- **Detection Priority**: N/A (falls under worker detection)
- **Description**: Scheduled jobs and cron tasks. Monitors execution success, duration, and schedule adherence.

## microservice
- **Methodology**: RED (default for unknown -- not in strategy matrix)
- **Tier**: service
- **Labels**: `methodology=RED`, `tier=service`
- **Detection Keywords**: None (must be specified explicitly)
- **Detection Priority**: N/A
- **Description**: Individual microservices in a distributed architecture. Uses web_service strategy by default.

## monolith
- **Methodology**: RED (default for unknown -- not in strategy matrix)
- **Tier**: service
- **Labels**: `methodology=RED`, `tier=service`
- **Detection Keywords**: None (must be specified explicitly)
- **Detection Priority**: N/A
- **Description**: Monolithic applications. Uses web_service strategy by default with additional resource monitoring.

## custom
- **Methodology**: RED (default fallback)
- **Tier**: custom
- **Labels**: `methodology=RED`, `tier=custom`
- **Detection Keywords**: None (fallback when no pattern matches)
- **Detection Priority**: Last (returned when no keywords match)
- **Description**: Custom or unrecognized component types. Falls back to RED methodology with web_service strategy.

---

## Detection Order Summary

The detection function processes keywords in this order. The first match wins:

1. cache
2. kubernetes
3. message_queue
4. api_gateway
5. worker (also catches serverless-adjacent patterns)
6. serverless
7. database
8. web_service
9. storage
10. network

Components not in this list (`load_balancer`, `cron_job`, `microservice`, `monolith`) must be
specified explicitly via the `service_type` parameter.

## Tier Assignments

| Tier | Components | Purpose |
|------|-----------|---------|
| service | web_service, microservice, monolith | User-facing request handlers |
| infrastructure | api_gateway, load_balancer, storage, network | Shared infrastructure |
| data | database | Data persistence |
| caching | cache | In-memory data stores |
| messaging | message_queue | Async communication |
| platform | kubernetes | Container orchestration |
| background | worker, cron_job | Background processing |
| compute | serverless | On-demand compute |
| custom | custom | Unclassified |
