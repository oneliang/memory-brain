#!/bin/bash
# post-compact.sh - PostCompact Hook for Memory Brain
# Generates session summary after compression

# Default configuration
MEMORY_BRAIN_URL="http://localhost:12321"
TIMEOUT=3

# Read stdin JSON from agent
input=$(cat)

# Extract fields from JSON
session_id=$(echo "$input" | jq -r '.session_id // empty')
user_id=$(echo "$input" | jq -r '.user_id // empty')
pre_tokens=$(echo "$input" | jq -r '.pre_tokens // 0')
post_tokens=$(echo "$input" | jq -r '.post_tokens // 0')
strategy=$(echo "$input" | jq -r '.strategy // "selective"')

if [ -z "$session_id" ] || [ -z "$user_id" ]; then
    exit 0
fi

# Generate summary ID
summary_id="sum_$(date +%s)_$(echo "$session_id" | cksum | cut -d' ' -f1)"

# Build request JSON
request_json=$(jq -n \
    --arg id "$summary_id" \
    --arg sid "$session_id" \
    --arg uid "$user_id" \
    --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
    --argjson pre "$pre_tokens" \
    --argjson post "$post_tokens" \
    --arg strat "$strategy" \
    '{
        id: $id,
        session_id: $sid,
        user_id: $uid,
        pre_tokens: $pre,
        post_tokens: $post,
        strategy: $strat,
        timestamp: $ts
    }')

# Call Memory Brain API to generate summary
curl -s -X POST \
    "${MEMORY_BRAIN_URL}/memory/session/summary" \
    -H "Content-Type: application/json" \
    -d "$request_json" \
    --max-time "$TIMEOUT" \
    >/dev/null 2>&1

exit 0