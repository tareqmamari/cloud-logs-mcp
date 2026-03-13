#!/usr/bin/env bash
# Burn Rate Calculator for SLO-based Multi-Window Alerting
#
# Computes multi-window burn rate thresholds and error budgets based on an SLO
# target and measurement window. Implements the same math as CalculateBurnRate(),
# GetBurnRateThreshold(), and CalculateErrorThreshold() from alerting_engine.go.
#
# Usage:
#     ./calculate-burn-rate.sh --slo 0.999 --window 30
#     ./calculate-burn-rate.sh --slo 0.9999 --window 30 --json
#     ./calculate-burn-rate.sh --slo 0.99 --window 7 --output-file /tmp/burn-rate.json
#
# License: Apache-2.0

set -euo pipefail

# --- Defaults ---
slo=""
window=30
use_json=false
output_file=""

usage() {
    cat <<'USAGE'
Usage: calculate-burn-rate.sh --slo TARGET [OPTIONS]

Calculate multi-window burn rate thresholds for SLO-based alerting.
Based on Google SRE Workbook Chapter 5: Alerting on SLOs.

Required:
  --slo TARGET       SLO target as a decimal (e.g., 0.999 for 99.9%)

Options:
  --window DAYS      SLO measurement window in days (default: 30)
  --json             Output results as structured JSON
  --output-file PATH Write full results to a JSON file; print only a summary to stdout
  -h, --help         Show this help message

Examples:
  calculate-burn-rate.sh --slo 0.999 --window 30
  calculate-burn-rate.sh --slo 0.9999 --window 30 --json
  calculate-burn-rate.sh --slo 0.99 --window 7 --output-file /tmp/burn-rate.json
USAGE
}

# --- Parse args ---
while [ $# -gt 0 ]; do
    case "$1" in
        --slo)         slo="$2"; shift 2 ;;
        --window)      window="$2"; shift 2 ;;
        --json)        use_json=true; shift ;;
        --output-file) output_file="$2"; shift 2 ;;
        -h|--help)     usage; exit 0 ;;
        *)             echo "Error: Unknown option: $1" >&2; usage >&2; exit 1 ;;
    esac
done

if [ -z "$slo" ]; then
    echo "Error: --slo is required" >&2
    usage >&2
    exit 1
fi

# Validate inputs using awk for floating point comparison
if ! awk -v s="$slo" 'BEGIN { if (s+0 < 0.9 || s+0 > 0.99999) exit 1 }'; then
    echo "Error: SLO target must be between 0.9 and 0.99999, got $slo" >&2
    exit 1
fi

if [ "$window" -lt 1 ] || [ "$window" -gt 90 ]; then
    echo "Error: Window must be between 1 and 90 days, got $window" >&2
    exit 1
fi

# --- Multi-window burn rate table data ---
# Format: window_name window_hours burn_rate budget_pct severity alert_type
WINDOWS="1h 1 14.4 2.0 P1_Critical fast_burn
6h 6 6.0 5.0 P1_Critical fast_burn
24h 24 3.0 10.0 P2_Warning slow_burn
72h 72 1.0 10.0 P3_Info slow_burn"

# --- Generate human-readable table ---
generate_table() {
    awk -v slo="$slo" -v win="$window" '
    BEGIN {
        error_budget = 1.0 - slo
        window_hours = win * 24
        error_budget_hours = error_budget * window_hours
        error_budget_minutes = error_budget_hours * 60

        print "========================================================================"
        print "Multi-Window Burn Rate Alerting Table"
        print "========================================================================"
        print ""
        printf "  SLO Target:        %.4f%%\n", slo * 100
        printf "  Error Budget:      %.4f%%\n", error_budget * 100
        printf "  SLO Window:        %d days (%d hours)\n", win, window_hours
        printf "  Error Budget Time: %.2f hours (%.1f minutes)\n", error_budget_hours, error_budget_minutes
        print ""
        print "------------------------------------------------------------------------"
        printf "%-8s %10s %14s %12s %-12s %-10s\n", "Window", "Burn Rate", "Err Threshold", "Budget Used", "Severity", "Type"
        print "------------------------------------------------------------------------"
    }
    {
        wname = $1; whours = $2; burn_rate = $3; budget_pct = $4
        severity = $5; alert_type = $6
        gsub(/_/, " ", severity)

        threshold = error_budget * burn_rate
        threshold_pct = threshold * 100
        budget_consumed = (whours / window_hours) * burn_rate * 100

        printf "%-8s %9.1fx %13.4f%% %11.1f%% %-12s %-10s\n", \
            wname, burn_rate, threshold_pct, budget_consumed, severity, alert_type
    }
    END {
        print "------------------------------------------------------------------------"
        print ""
        print "Detailed Breakdown:"
        print ""
    }
    ' <<< "$WINDOWS"

    # Detailed breakdown
    echo "$WINDOWS" | while read -r wname whours burn_rate budget_pct severity alert_type; do
        severity_display=$(echo "$severity" | tr '_' ' ')
        awk -v slo="$slo" -v win="$window" -v wn="$wname" -v wh="$whours" \
            -v br="$burn_rate" -v sev="$severity_display" -v at="$alert_type" '
        BEGIN {
            error_budget = 1.0 - slo
            window_hours = win * 24
            threshold = error_budget * br
            exhaustion_days = win / br
            budget_consumed = (wh / window_hours) * br * 100

            if (index(sev, "P1") > 0) action = "Page on-call"
            else if (index(sev, "P2") > 0) action = "Create ticket"
            else action = "Informational"

            printf "  %s window (%s):\n", wn, at
            printf "    Burn Rate:           %.1fx sustainable rate\n", br
            printf "    Error Rate Threshold: %.4f%%\n", threshold * 100
            printf "    Budget Consumed:     %.1f%% in %s\n", budget_consumed, wn
            printf "    Time to Exhaustion:  %.1f days\n", exhaustion_days
            printf "    Severity:            %s\n", sev
            printf "    Action:              %s\n", action
            print ""
        }' /dev/null
    done

    # Multi-window confirmation pairs
    cat <<'PAIRS'
Multi-Window Confirmation Pairs:

  Fast Burn (Page):
    Long window:  1h at 14.4x burn rate
    Short window: 5m at 14.4x burn rate (confirmation)
    Both must fire simultaneously to trigger alert

  Slow Burn (Ticket):
    Long window:  24h at 3.0x burn rate
    Short window: 6h at 3.0x burn rate (confirmation)
    Both must fire simultaneously to trigger alert

PAIRS

    # CalculateErrorThreshold examples
    echo "CalculateErrorThreshold Examples:"
    echo "  Formula: (budget_consumption_% / 100) * error_budget * (slo_window_hours / alert_window_hours)"
    echo ""

    awk -v slo="$slo" -v win="$window" '
    BEGIN {
        error_budget = 1.0 - slo
        window_hours = win * 24

        split("2.0 5.0 10.0 10.0", bpcts)
        split("1 6 24 72", ahours)

        for (i = 1; i <= 4; i++) {
            threshold = (bpcts[i] / 100) * error_budget * (window_hours / ahours[i])
            printf "  %.0f%% budget in %sh: threshold = (%.0f/100) * %s * (%d/%s) = %.4f%%\n", \
                bpcts[i], ahours[i], bpcts[i], error_budget, window_hours, ahours[i], threshold * 100
        }
        print ""
    }' /dev/null
}

# --- Generate JSON output ---
generate_json() {
    awk -v slo="$slo" -v win="$window" '
    BEGIN {
        error_budget = 1.0 - slo
        window_hours = win * 24

        printf "{\n"
        printf "  \"slo_target\": %s,\n", slo
        printf "  \"slo_target_pct\": %.4f,\n", slo * 100
        printf "  \"error_budget\": %.6f,\n", error_budget
        printf "  \"window_days\": %d,\n", win
        printf "  \"window_hours\": %d,\n", window_hours
        printf "  \"windows\": [\n"
    }
    {
        wname = $1; whours = $2 + 0; burn_rate = $3 + 0; budget_pct = $4 + 0
        severity = $5; alert_type = $6
        gsub(/_/, " ", severity)

        threshold = error_budget * burn_rate
        threshold_pct = threshold * 100
        budget_consumed = (whours / (win * 24)) * burn_rate * 100
        exhaustion_days = win / burn_rate

        if (NR > 1) printf ",\n"
        printf "    {\n"
        printf "      \"window\": \"%s\",\n", wname
        printf "      \"window_hours\": %d,\n", whours
        printf "      \"burn_rate\": %.1f,\n", burn_rate
        printf "      \"error_threshold_pct\": %.6f,\n", threshold_pct
        printf "      \"budget_consumed_pct\": %.2f,\n", budget_consumed
        printf "      \"exhaustion_days\": %.2f,\n", exhaustion_days
        printf "      \"severity\": \"%s\",\n", severity
        printf "      \"alert_type\": \"%s\"\n", alert_type
        printf "    }"
    }
    END {
        printf "\n  ]\n"
        printf "}\n"
    }
    ' <<< "$WINDOWS"
}

# --- Main ---
if [ -n "$output_file" ]; then
    # Write JSON with embedded formatted table
    json_output=$(generate_json)
    table_output=$(generate_table)
    # Embed the table as a JSON string field
    escaped_table=$(echo "$table_output" | awk '{gsub(/\\/, "\\\\"); gsub(/"/, "\\\""); gsub(/\t/, "\\t"); printf "%s\\n", $0}')
    # Insert formatted_table before the closing brace
    echo "$json_output" | sed '$d' > "$output_file"
    printf ',\n  "formatted_table": "%s"\n}\n' "$escaped_table" >> "$output_file"
    echo "Burn rate table written to ${output_file} (4 windows, SLO $(awk -v s="$slo" 'BEGIN{printf "%.4g%%", s*100}'))"
elif [ "$use_json" = "true" ]; then
    generate_json
else
    generate_table
fi
