#!/bin/bash
# session-end.sh - SessionEnd Hook for Memory Brain
# Batch updates user profile after session ends

# Default configuration
MEMORY_BRAIN_URL="http://localhost:12321"
TIMEOUT=3

# Read stdin JSON from agent
input=$(cat)

# Extract fields from JSON
session_id=$(echo "$input" | jq -r '.session_id // empty')
user_id=$(echo "$input" | jq -r '.user_id // empty')

if [ -z "$session_id" ] || [ -z "$user_id" ]; then
    exit 0
fi

# Build request JSON
request_json=$(jq -n \
    --arg sid "$session_id" \
    --arg uid "$user_id" \
    --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
    '{
        session_id: $sid,
        user_id: $uid,
        timestamp: $ts
    }')

# Call Memory Brain API to update profile
curl -s -X PUT \
    "${MEMORY_BRAIN_URL}/memory/profile/update" \
    -H "Content-Type: application/json" \
    -d "$request_json" \
    --max-time "$TIMEOUT" \
    >/dev/null 2>&1

exit 0