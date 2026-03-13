# Burn Rate Math Reference

Detailed formulas and worked examples for multi-window, multi-burn-rate alerting.
Based on the Google SRE Workbook, Chapter 5: Alerting on SLOs.

## Core Formulas

### Error Budget

```
error_budget = 1 - slo_target
```

For SLO 99.9%: `error_budget = 1 - 0.999 = 0.001 (0.1%)`

### Error Budget in Hours

```
error_budget_hours = error_budget * window_days * 24
```

For 30-day window with SLO 99.9%: `0.001 * 30 * 24 = 0.72 hours = 43.2 minutes`

### Burn Rate

Burn rate is a unitless multiplier indicating how fast the error budget is being consumed
relative to a sustainable rate.

```
burn_rate = actual_error_rate / sustainable_error_rate
```

Where `sustainable_error_rate = error_budget / slo_window` (i.e., 1x burn rate exactly exhausts
the budget at the end of the window).

### GetBurnRateThreshold

Returns the error rate threshold for a given burn rate:

```
threshold = error_budget * burn_rate
          = (1 - slo_target) * burn_rate
```

**Example**: SLO 99.9%, burn rate 14.4x:
```
threshold = 0.001 * 14.4 = 0.0144 (1.44% error rate)
```

### CalculateErrorThreshold

Computes the error rate threshold that would consume a specified percentage of error budget
within a given time window:

```
threshold = (budget_consumption_% / 100) * error_budget * (slo_window_hours / alert_window_hours)
```

**Example**: 2% budget consumption in 1 hour, SLO 99.9%, 30-day window:
```
threshold = (2 / 100) * 0.001 * (720 / 1) = 0.0144 (1.44%)
```

### Budget Consumption Rate

How much of the error budget is consumed in a given alert window at a given burn rate:

```
budget_consumed_% = (alert_window_hours / slo_window_hours) * burn_rate * 100
```

**Example**: 14.4x burn rate over 1 hour, 30-day window:
```
budget_consumed = (1 / 720) * 14.4 * 100 = 2.0%
```

### Time to Budget Exhaustion

How long until the error budget is fully consumed at a given burn rate:

```
time_to_exhaustion = slo_window / burn_rate
```

**Example**: 14.4x burn rate, 30-day window:
```
time_to_exhaustion = 30 / 14.4 = 2.08 days
```

## Multi-Window Burn Rate Table

Standard windows from `CalculateBurnRate()`:

### Fast Burn Windows (Paging)

| Window Duration | Burn Rate | Budget Consumed | Severity | Alert Type |
|----------------|-----------|-----------------|----------|------------|
| 1 hour | 14.4x | 2% | P1 Critical | fast_burn |
| 6 hours | 6.0x | 5% | P1 Critical | fast_burn |

### Slow Burn Windows (Ticketing)

| Window Duration | Burn Rate | Budget Consumed | Severity | Alert Type |
|----------------|-----------|-----------------|----------|------------|
| 24 hours | 3.0x | 10% | P2 Warning | slow_burn |
| 72 hours | 1.0x | 10% | P3 Info | slow_burn |

## Worked Examples

### Example 1: SLO 99.9% (30-day window)

```
Error budget = 0.001 (0.1%)
Error budget in hours = 0.72 hours (43.2 minutes)
```

| Window | Burn Rate | Error Rate Threshold | Budget Consumed | Time to Exhaustion |
|--------|-----------|---------------------|-----------------|-------------------|
| 1h | 14.4x | 1.4400% | 2.0% | 2.1 days |
| 6h | 6.0x | 0.6000% | 5.0% | 5.0 days |
| 24h | 3.0x | 0.3000% | 10.0% | 10.0 days |
| 72h | 1.0x | 0.1000% | 10.0% | 30.0 days |

### Example 2: SLO 99.99% (30-day window)

```
Error budget = 0.0001 (0.01%)
Error budget in hours = 0.072 hours (4.32 minutes)
```

| Window | Burn Rate | Error Rate Threshold | Budget Consumed | Time to Exhaustion |
|--------|-----------|---------------------|-----------------|-------------------|
| 1h | 14.4x | 0.1440% | 2.0% | 2.1 days |
| 6h | 6.0x | 0.0600% | 5.0% | 5.0 days |
| 24h | 3.0x | 0.0300% | 10.0% | 10.0 days |
| 72h | 1.0x | 0.0100% | 10.0% | 30.0 days |

### Example 3: SLO 99% (30-day window)

```
Error budget = 0.01 (1.0%)
Error budget in hours = 7.2 hours
```

| Window | Burn Rate | Error Rate Threshold | Budget Consumed | Time to Exhaustion |
|--------|-----------|---------------------|-----------------|-------------------|
| 1h | 14.4x | 14.4000% | 2.0% | 2.1 days |
| 6h | 6.0x | 6.0000% | 5.0% | 5.0 days |
| 24h | 3.0x | 3.0000% | 10.0% | 10.0 days |
| 72h | 1.0x | 1.0000% | 10.0% | 30.0 days |

## Multi-Window Confirmation

Each burn rate alert uses two windows to reduce false positives. Both windows must fire
simultaneously for the alert to trigger.

### Fast Burn Confirmation
- **Long window**: 1h at 14.4x burn rate
- **Short confirmation window**: 5m at 14.4x burn rate
- Both must be true: sustained high error rate over 1h AND currently still elevated in last 5m

### Slow Burn Confirmation
- **Long window**: 24h at 3.0x burn rate
- **Short confirmation window**: 6h at 3.0x burn rate
- Both must be true: sustained elevated error rate over 24h AND still elevated in last 6h

### Why Multi-Window?

Single-window alerting has problems:
- **Short window only** (e.g., 5 min): Too noisy, fires on brief spikes
- **Long window only** (e.g., 24h): Too slow, fires after significant budget is consumed

Multi-window combines:
- Long window catches sustained issues
- Short window confirms the issue is still happening (not a past spike being averaged in)

## FormatBurnRateExplanation Output

The `FormatBurnRateExplanation()` function produces human-readable output like:

```
SLO: 99.900% (Error Budget: 0.1000%)
Burn Rate: 14.4x
Alert Window: 1h
Error Rate Threshold: 1.4400%
At this burn rate, 2.0% of the 30-day error budget would be consumed in 1h
```

## Quick Reference: Common SLO Targets

| SLO Target | Error Budget | Monthly Budget (30d) | Typical Use |
|-----------|-------------|---------------------|-------------|
| 99% | 1.0% | 7.2 hours | Internal tools, batch jobs |
| 99.5% | 0.5% | 3.6 hours | Non-critical services |
| 99.9% | 0.1% | 43.2 minutes | Standard production services |
| 99.95% | 0.05% | 21.6 minutes | Important customer-facing services |
| 99.99% | 0.01% | 4.32 minutes | Critical infrastructure |
| 99.999% | 0.001% | 26 seconds | Payment processing, auth |
