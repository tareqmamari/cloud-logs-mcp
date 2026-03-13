#!/usr/bin/env bash
# Send sample log entries to IBM Cloud Logs ingestion endpoint
#
# Usage:
#     ./send-test-logs.sh --endpoint ENDPOINT --api-key API_KEY [--count N]
#     ./send-test-logs.sh --endpoint ENDPOINT --api-key API_KEY --json
#     ./send-test-logs.sh --endpoint ENDPOINT --api-key API_KEY --output-file /tmp/result.json
#
# The endpoint should be the ingress URL:
#     https://{instance-id}.ingress.{region}.logs.cloud.ibm.com
#
# Authentication uses an IBM Cloud IAM API key, which is exchanged for a
# bearer token before sending logs.
#
# License: Apache-2.0

set -euo pipefail

# --- Constants ---
IAM_TOKEN_URL="https://iam.cloud.ibm.com/identity/token"
INGESTION_PATH="/logs/v1/singles"

# --- Defaults ---
endpoint=""
api_key=""
count=3
use_json=false
output_file=""

usage() {
    cat <<'USAGE'
Usage: send-test-logs.sh --endpoint ENDPOINT --api-key API_KEY [OPTIONS]

Send test log entries to IBM Cloud Logs ingestion endpoint.

Required:
  --endpoint URL     Ingress endpoint (https://{id}.ingress.{region}.logs.cloud.ibm.com)
  --api-key KEY      IBM Cloud IAM API key

Options:
  --count N          Number of log entries to send (default: 3, max: 1000)
  --json             Output results as structured JSON
  --output-file PATH Write full request/response details to a JSON file
  -h, --help         Show this help message

Examples:
  send-test-logs.sh --endpoint https://abc.ingress.us-south.logs.cloud.ibm.com --api-key mykey
  send-test-logs.sh --endpoint https://abc.ingress.eu-de.logs.cloud.ibm.com --api-key mykey --count 10 --json
USAGE
}

# --- Parse args ---
while [ $# -gt 0 ]; do
    case "$1" in
        --endpoint)    endpoint="$2"; shift 2 ;;
        --api-key)     api_key="$2"; shift 2 ;;
        --count)       count="$2"; shift 2 ;;
        --json)        use_json=true; shift ;;
        --output-file) output_file="$2"; shift 2 ;;
        -h|--help)     usage; exit 0 ;;
        *)             echo "Error: Unknown option: $1" >&2; usage >&2; exit 1 ;;
    esac
done

if [ -z "$endpoint" ]; then
    echo "Error: --endpoint is required" >&2
    usage >&2
    exit 1
fi

if [ -z "$api_key" ]; then
    echo "Error: --api-key is required" >&2
    usage >&2
    exit 1
fi

if [ "$count" -lt 1 ] || [ "$count" -gt 1000 ]; then
    echo "Error: --count must be between 1 and 1000" >&2
    exit 1
fi

# --- Sample log templates ---
# 3 sample logs that cycle for batch generation
sample_log_info() {
    local idx="$1"
    local ts="$2"
    cat <<EOF
{"applicationName":"ingestion-test","subsystemName":"smoke-test","severity":3,"text":"Ingestion pipeline test entry","timestamp":${ts},"json":{"test_id":"test-001","environment":"development","batch_index":${idx}}}
EOF
}

sample_log_warning() {
    local idx="$1"
    local ts="$2"
    cat <<EOF
{"applicationName":"ingestion-test","subsystemName":"smoke-test","severity":4,"text":"Simulated warning: disk usage above 80%","timestamp":${ts},"json":{"test_id":"test-002","disk_usage_pct":82,"batch_index":${idx}}}
EOF
}

sample_log_error() {
    local idx="$1"
    local ts="$2"
    cat <<EOF
{"applicationName":"ingestion-test","subsystemName":"smoke-test","severity":5,"text":"Simulated error: connection timeout to database","timestamp":${ts},"json":{"test_id":"test-003","error_code":"DB_TIMEOUT","db_host":"db-primary.internal","batch_index":${idx}}}
EOF
}

# --- Build log batch ---
build_batch() {
    local n="$1"
    local now
    now=$(date +%s)
    local batch="["
    local i=0
    while [ "$i" -lt "$n" ]; do
        [ "$i" -gt 0 ] && batch="${batch},"
        local ts
        ts=$(awk -v base="$now" -v idx="$i" 'BEGIN{printf "%.3f", base + idx * 0.001}')
        local mod=$((i % 3))
        case "$mod" in
            0) batch="${batch}$(sample_log_info "$i" "$ts")" ;;
            1) batch="${batch}$(sample_log_warning "$i" "$ts")" ;;
            2) batch="${batch}$(sample_log_error "$i" "$ts")" ;;
        esac
        i=$((i + 1))
    done
    batch="${batch}]"
    echo "$batch"
}

# --- Get IAM token ---
get_iam_token() {
    local response
    response=$(curl -s -X POST "$IAM_TOKEN_URL" \
        -H "Content-Type: application/x-www-form-urlencoded" \
        -d "grant_type=urn:ibm:params:oauth:grant-type:apikey&apikey=${api_key}" \
        --max-time 30)

    local token
    # Extract access_token using sed (no jq dependency)
    token=$(echo "$response" | sed -n 's/.*"access_token"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

    if [ -z "$token" ]; then
        echo "Error: Failed to get IAM token. Response: $response" >&2
        exit 1
    fi
    echo "$token"
}

# --- Send logs ---
send_logs() {
    local token="$1"
    local batch="$2"
    local url="${endpoint%/}${INGESTION_PATH}"

    local http_response
    http_response=$(curl -s -w "\n%{http_code}" -X POST "$url" \
        -H "Content-Type: application/json" \
        -H "Authorization: Bearer ${token}" \
        -d "$batch" \
        --max-time 30)

    # Split response body and status code
    local status_code
    status_code=$(echo "$http_response" | tail -1)
    local response_body
    response_body=$(echo "$http_response" | sed '$d')

    # Export for use in output
    SEND_URL="$url"
    SEND_STATUS="$status_code"
    SEND_BODY="$response_body"
}

# --- Main ---

# Suppress verbose output when using structured output
verbose=true
if [ "$use_json" = "true" ] || [ -n "$output_file" ]; then
    verbose=false
fi

[ "$verbose" = "true" ] && echo "Authenticating with IBM Cloud IAM..."
token=$(get_iam_token)
[ "$verbose" = "true" ] && echo "Authentication successful."

[ "$verbose" = "true" ] && echo ""
[ "$verbose" = "true" ] && echo "Building batch of ${count} log entries..."
batch=$(build_batch "$count")

[ "$verbose" = "true" ] && echo "Sending ${count} log entries to ${endpoint}..."
SEND_URL="" SEND_STATUS="" SEND_BODY=""
send_logs "$token" "$batch"

[ "$verbose" = "true" ] && echo "POST ${SEND_URL}"
[ "$verbose" = "true" ] && echo "Status: ${SEND_STATUS}"
[ "$verbose" = "true" ] && [ -n "$SEND_BODY" ] && echo "Response: ${SEND_BODY}"
[ "$verbose" = "true" ] && echo ""
[ "$verbose" = "true" ] && echo "Done. Logs should appear in IBM Cloud Logs within 10-30 seconds."
[ "$verbose" = "true" ] && echo "Verify with: source logs | filter \$l.applicationname == 'ingestion-test' | limit 10"

# Check for HTTP errors
if [ "${SEND_STATUS:-0}" -ge 400 ]; then
    echo "Error: HTTP ${SEND_STATUS} from ${SEND_URL}" >&2
    [ -n "$SEND_BODY" ] && echo "Response: ${SEND_BODY}" >&2
    exit 1
fi

# Structured output
if [ -n "$output_file" ]; then
    # Escape JSON strings
    escaped_body=$(echo "$SEND_BODY" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')
    cat > "$output_file" <<EOF
{
  "endpoint": "${endpoint}",
  "logs_sent": ${count},
  "status_code": ${SEND_STATUS},
  "response_body": "${escaped_body}",
  "request_url": "${SEND_URL}",
  "request_log_count": ${count}
}
EOF
    echo "Sent ${count} log entries to ${endpoint} (status ${SEND_STATUS}). Details: ${output_file}"
elif [ "$use_json" = "true" ]; then
    escaped_body=$(echo "$SEND_BODY" | sed 's/\\/\\\\/g; s/"/\\"/g' | tr '\n' ' ')
    cat <<EOF
{
  "endpoint": "${endpoint}",
  "logs_sent": ${count},
  "status_code": ${SEND_STATUS},
  "response_body": "${escaped_body}",
  "request_url": "${SEND_URL}",
  "request_log_count": ${count}
}
EOF
fi
