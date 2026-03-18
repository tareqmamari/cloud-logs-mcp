#!/usr/bin/env bash
# Offline DataPrime query validator for IBM Cloud Logs
# Validates DataPrime queries without requiring IBM Cloud Logs connectivity.
# Checks for common syntax errors, invalid field references, and anti-patterns.
#
# Usage:
#     ./validate-query.sh "source logs | filter \$m.severity >= ERROR"
#     ./validate-query.sh --fix "source logs | filter \$d.message.contains('error')"
#     echo "source logs | filter \$m.severity >= 5" | ./validate-query.sh --stdin
#     ./validate-query.sh --json "source logs | filter \$l.namespace == 'prod'"
#     ./validate-query.sh --output-file /tmp/result.json "source logs | filter ..."
#
# License: Apache-2.0

set -euo pipefail

# --- Valid fields ---
VALID_LABEL_FIELDS="applicationname subsystemname computername ipaddress threadid processid classname methodname category"
VALID_METADATA_FIELDS="severity timestamp priority"
VALID_SEVERITIES="VERBOSE DEBUG INFO WARNING ERROR CRITICAL"
MIXED_TYPE_FIELDS="message msg log text body content payload data error err reason details"
STRING_METHODS="contains startsWith endsWith matches toLowerCase toUpperCase trim"

# --- State ---
errors=()
warnings=()
corrections_from=()
corrections_to=()
is_valid=true

add_error() {
    is_valid=false
    errors+=("$1")
}

add_warning() {
    warnings+=("$1")
}

add_correction() {
    corrections_from+=("$1")
    corrections_to+=("$2")
}

# --- Validation functions ---

validate_no_tilde() {
    local q="$1"
    if echo "$q" | grep -qE '!?~~'; then
        add_error "~~ operator is NOT supported. Use matches() for regex or contains() for substring."
    fi
}

validate_label_fields() {
    local q="$1"
    local field
    while IFS= read -r field; do
        [ -z "$field" ] && continue
        local lower
        lower=$(echo "$field" | tr '[:upper:]' '[:lower:]')
        local found=false
        for valid in $VALID_LABEL_FIELDS; do
            if [ "$lower" = "$valid" ]; then
                found=true
                break
            fi
        done
        if [ "$found" = "false" ]; then
            case "$lower" in
                namespace) add_error "Unknown label field: \$l.$field. Use \$l.applicationname — K8s namespace maps to applicationname" ;;
                service)   add_error "Unknown label field: \$l.$field. Use \$l.applicationname or \$l.subsystemname" ;;
                *)         add_error "Unknown label field: \$l.$field. Valid: $VALID_LABEL_FIELDS" ;;
            esac
        fi
    done < <(echo "$q" | grep -oE '\$l\.([a-zA-Z_]+)' | sed 's/\$l\.//')
}

validate_metadata_fields() {
    local q="$1"
    local field
    while IFS= read -r field; do
        [ -z "$field" ] && continue
        local lower
        lower=$(echo "$field" | tr '[:upper:]' '[:lower:]')
        local found=false
        for valid in $VALID_METADATA_FIELDS; do
            if [ "$lower" = "$valid" ]; then
                found=true
                break
            fi
        done
        if [ "$found" = "false" ]; then
            case "$lower" in
                level|loglevel|log_level) add_error "Unknown metadata field: \$m.$field. Use \$m.severity (values: VERBOSE, DEBUG, INFO, WARNING, ERROR, CRITICAL)" ;;
                *)                        add_error "Unknown metadata field: \$m.$field. Valid: $VALID_METADATA_FIELDS" ;;
            esac
        fi
    done < <(echo "$q" | grep -oE '\$m\.([a-zA-Z_]+)' | sed 's/\$m\.//')
}

validate_operators() {
    local q="$1"
    if echo "$q" | grep -qiE '[[:space:]]+AND[[:space:]]+'; then
        add_error "Use && instead of AND for logical AND."
    fi
    if echo "$q" | grep -qiE '[[:space:]]+OR[[:space:]]+'; then
        add_error "Use || instead of OR for logical OR."
    fi
    if echo "$q" | grep -qE '\$[dlmp]\.[a-zA-Z_]+[[:space:]]*=[^=]'; then
        add_error "Use == for equality comparison, not single =."
    fi
    if echo "$q" | grep -qiE '[[:space:]]+LIKE[[:space:]]+'; then
        add_error "LIKE is not valid. Use contains(), startsWith(), endsWith(), or matches()."
    fi
    if echo "$q" | grep -qiE '[[:space:]]+IN[[:space:]]*\('; then
        add_error "IN is not valid. Use multiple || conditions instead."
    fi
}

validate_quotes() {
    local q="$1"
    if echo "$q" | grep -qE '\$[dlmp]\.[a-zA-Z_]+\s*==\s*"[^"]*"'; then
        add_error "Use single quotes for strings, not double quotes. Example: 'value'"
    fi
}

validate_severity() {
    local q="$1"
    local sev
    while IFS= read -r sev; do
        [ -z "$sev" ] && continue
        local upper
        upper=$(echo "$sev" | tr '[:lower:]' '[:upper:]')
        local found=false
        for valid in $VALID_SEVERITIES; do
            if [ "$upper" = "$valid" ]; then
                found=true
                break
            fi
        done
        if [ "$found" = "false" ]; then
            add_error "Invalid severity: $sev. Valid: VERBOSE, DEBUG, INFO, WARNING, ERROR, CRITICAL"
        fi
    done < <(echo "$q" | grep -oE '\$m\.severity\s*[!=><]+\s*['\''"]?([a-zA-Z]+)' | sed -E "s/.*[!=><]+\s*['\"]?//" | grep -E '^[a-zA-Z]+$')
}

validate_sort() {
    local q="$1"
    if echo "$q" | grep -qE '\|[[:space:]]*sort[[:space:]]+'; then
        add_warning "Use 'orderby' instead of 'sort'. DataPrime does not have a 'sort' command."
    fi
}

validate_timestamp_ordering() {
    local q="$1"
    if echo "$q" | grep -qE 'orderby[[:space:]]+-\$m\.timestamp'; then
        add_error "Use 'orderby \$m.timestamp desc' instead of 'orderby -\$m.timestamp'. The '-' prefix only works for numeric fields."
    fi
}

# --- Auto-correction ---

auto_correct() {
    local q="$1"
    corrected="$q"

    # Fix mixed-type string methods: $d.message.contains() -> $d.message:string.contains()
    for f in $MIXED_TYPE_FIELDS; do
        for m in $STRING_METHODS; do
            if echo "$corrected" | grep -qF "\$d.${f}.${m}("; then
                corrected=$(echo "$corrected" | sed "s/\$d\\.${f}\\.${m}(/\$d.${f}:string.${m}(/g")
                add_correction "\$d.${f}.${m}()" "\$d.${f}:string.${m}()"
            fi
        done
    done

    # Fix numeric severity: $m.severity >= 5 -> $m.severity >= CRITICAL
    local sev_num sev_name
    while IFS= read -r match; do
        [ -z "$match" ] && continue
        sev_num=$(echo "$match" | grep -oE '[0-9]+$')
        case "$sev_num" in
            0) sev_name="VERBOSE" ;;
            1) sev_name="DEBUG" ;;
            2) sev_name="INFO" ;;
            3) sev_name="WARNING" ;;
            4) sev_name="ERROR" ;;
            5) sev_name="CRITICAL" ;;
            *) sev_name="ERROR" ;;
        esac
        corrected=$(echo "$corrected" | sed -E 's/(\$m\.severity[[:space:]]*[!=><]+[[:space:]]*)'"${sev_num}"'/\1'"${sev_name}"'/')
        add_correction "severity $sev_num" "severity $sev_name"
    done < <(echo "$corrected" | grep -oE '\$m\.severity[[:space:]]*[!=><]+[[:space:]]*[0-9]+')

    # Fix sort -> orderby
    if echo "$corrected" | grep -qE '\|\s*sort\s+'; then
        local sort_match
        sort_match=$(echo "$corrected" | grep -oE '\|\s*sort\s+-?[a-zA-Z_$][a-zA-Z0-9_.]*')
        if [ -n "$sort_match" ]; then
            local field_part
            field_part=$(echo "$sort_match" | sed -E 's/\|\s*sort\s+//')
            local new_val
            if echo "$field_part" | grep -qE '^-'; then
                field_part=$(echo "$field_part" | sed 's/^-//')
                new_val="| orderby ${field_part} desc"
            else
                new_val="| orderby ${field_part}"
            fi
            corrected=$(echo "$corrected" | sed "s|${sort_match}|${new_val}|")
            add_correction "$(echo "$sort_match" | sed 's/^\s*//')" "$(echo "$new_val" | sed 's/^\s*//')"
        fi
    fi

    # Fix bucket() -> roundTime()
    if echo "$corrected" | grep -qE 'bucket\s*\('; then
        corrected=$(echo "$corrected" | sed -E 's/bucket\s*\(\s*(\$m\.timestamp)\s*,\s*([0-9]+[smhd])\s*\)/roundTime(\1, \2)/g')
        add_correction "bucket()" "roundTime()"
    fi

    # Fix count_distinct -> approx_count_distinct
    if echo "$corrected" | grep -qE 'count_distinct\(|distinct_count\('; then
        corrected=$(echo "$corrected" | sed -E 's/count_distinct\(/approx_count_distinct(/g; s/distinct_count\(/approx_count_distinct(/g')
        add_correction "count_distinct()" "approx_count_distinct()"
    fi

    # Fix orderby -$m.timestamp -> orderby $m.timestamp desc
    if echo "$corrected" | grep -qE 'orderby[[:space:]]+-\$m\.timestamp'; then
        corrected=$(echo "$corrected" | sed -E 's/orderby[[:space:]]+-\$m\.timestamp/orderby $m.timestamp desc/g')
        add_correction "orderby -\$m.timestamp" "orderby \$m.timestamp desc"
    fi

    echo "$corrected"
}

# --- Output ---

output_json() {
    local query="$1"
    local corrected_query="$2"
    local fix="$3"

    printf '{\n'
    printf '  "valid": %s,\n' "$is_valid"

    # errors array
    printf '  "errors": ['
    local first=true
    for e in "${errors[@]+"${errors[@]}"}"; do
        [ "$first" = "true" ] && first=false || printf ','
        printf '\n    "%s"' "$(echo "$e" | sed 's/"/\\"/g')"
    done
    [ ${#errors[@]} -gt 0 ] && printf '\n  '
    printf '],\n'

    # warnings array
    printf '  "warnings": ['
    first=true
    for w in "${warnings[@]+"${warnings[@]}"}"; do
        [ "$first" = "true" ] && first=false || printf ','
        printf '\n    "%s"' "$(echo "$w" | sed 's/"/\\"/g')"
    done
    [ ${#warnings[@]} -gt 0 ] && printf '\n  '
    printf '],\n'

    # corrections array
    printf '  "corrections": ['
    first=true
    for i in "${!corrections_from[@]}"; do
        [ "$first" = "true" ] && first=false || printf ','
        printf '\n    {"from": "%s", "to": "%s"}' \
            "$(echo "${corrections_from[$i]}" | sed 's/"/\\"/g')" \
            "$(echo "${corrections_to[$i]}" | sed 's/"/\\"/g')"
    done
    [ ${#corrections_from[@]} -gt 0 ] && printf '\n  '
    printf '],\n'

    printf '  "original_query": "%s",\n' "$(echo "$query" | sed 's/"/\\"/g')"
    if [ "$fix" = "true" ] && [ "$corrected_query" != "$query" ]; then
        printf '  "corrected_query": "%s"\n' "$(echo "$corrected_query" | sed 's/"/\\"/g')"
    else
        printf '  "corrected_query": null\n'
    fi
    printf '}\n'
}

output_text() {
    local query="$1"
    local corrected_query="$2"
    local fix="$3"

    if [ ${#corrections_from[@]} -gt 0 ]; then
        echo "Auto-corrections applied:"
        for i in "${!corrections_from[@]}"; do
            echo "  ${corrections_from[$i]} -> ${corrections_to[$i]}"
        done
        echo
    fi

    if [ ${#errors[@]} -gt 0 ]; then
        echo "ERRORS:"
        for e in "${errors[@]}"; do
            echo "  - $e"
        done
        echo
    fi

    if [ ${#warnings[@]} -gt 0 ]; then
        echo "WARNINGS:"
        for w in "${warnings[@]}"; do
            echo "  - $w"
        done
        echo
    fi

    if [ "$is_valid" = "true" ]; then
        if [ ${#corrections_from[@]} -gt 0 ]; then
            echo "VALID (with corrections)"
        else
            echo "VALID"
        fi
    else
        echo "INVALID"
    fi

    if [ "$fix" = "true" ] && [ "$corrected_query" != "$query" ]; then
        echo
        echo "Corrected query:"
        echo "  $corrected_query"
    fi
}

# --- Main ---

usage() {
    cat <<'USAGE'
Usage: validate-query.sh [OPTIONS] [QUERY]

Validate DataPrime queries for IBM Cloud Logs.

Options:
  --fix           Auto-correct common issues
  --stdin         Read query from stdin
  --json          Output results as JSON
  --output-file PATH  Write full results to a JSON file; print only a summary to stdout
  -h, --help      Show this help message

Examples:
  validate-query.sh "source logs | filter \$m.severity >= ERROR"
  validate-query.sh --fix "source logs | filter \$d.message.contains('err')"
  echo "source logs | filter \$m.severity >= 5" | validate-query.sh --stdin --fix
  validate-query.sh --json "source logs | filter \$l.namespace == 'prod'"
USAGE
}

fix=false
use_stdin=false
use_json=false
output_file=""
query=""

while [ $# -gt 0 ]; do
    case "$1" in
        --fix)        fix=true; shift ;;
        --stdin)      use_stdin=true; shift ;;
        --json)       use_json=true; shift ;;
        --output-file) output_file="$2"; shift 2 ;;
        -h|--help)    usage; exit 0 ;;
        -*)           echo "Unknown option: $1" >&2; usage >&2; exit 1 ;;
        *)            query="$1"; shift ;;
    esac
done

if [ "$use_stdin" = "true" ]; then
    query=$(cat)
elif [ -z "$query" ]; then
    usage
    exit 1
fi

# Run auto-correction if requested
corrected="$query"
if [ "$fix" = "true" ]; then
    corrected=$(auto_correct "$query")
fi

# Run all validations on the (possibly corrected) query
validate_no_tilde "$corrected"
validate_label_fields "$corrected"
validate_metadata_fields "$corrected"
validate_operators "$corrected"
validate_quotes "$corrected"
validate_severity "$corrected"
validate_sort "$corrected"
validate_timestamp_ordering "$corrected"

# Output
if [ -n "$output_file" ]; then
    output_json "$query" "$corrected" "$fix" > "$output_file"
    n_errors=${#errors[@]}
    n_warnings=${#warnings[@]}
    echo "Validation complete: ${n_errors} error(s), ${n_warnings} warning(s). Details: ${output_file}"
elif [ "$use_json" = "true" ]; then
    output_json "$query" "$corrected" "$fix"
else
    output_text "$query" "$corrected" "$fix"
fi

if [ "$is_valid" = "true" ]; then
    exit 0
else
    exit 1
fi
