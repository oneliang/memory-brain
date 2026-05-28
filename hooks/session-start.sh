#!/bin/bash
# session-start.sh - SessionStart Hook for Memory Brain
# Reads stdin JSON from agent, calls Memory Brain API

# Default configuration
MEMORY_BRAIN_URL="http://localhost:12321"
INJECT_CONTEXT="${INJECT_CONTEXT:-false}"  # Default: false (silent accumulation)
TIMEOUT=3

# Read stdin JSON from agent
input=$(cat)

# Extract user_id from JSON
user_id=$(echo "$input" | jq -r '.user_id // empty')
session_id=$(echo "$input" | jq -r '.session_id // empty')

if [ -z "$user_id" ]; then
    exit 0  # Don't block agent
fi

# Call Memory Brain API to load profile
response=$(curl -s -X GET \
    "${MEMORY_BRAIN_URL}/memory/profile?user=${user_id}&inject=${INJECT_CONTEXT}" \
    --max-time "$TIMEOUT" \
    2>/dev/null)

# If inject enabled and response contains systemMessage, output to stdout
if [ "$INJECT_CONTEXT" = "true" ] && [ -n "$response" ]; then
    system_message=$(echo "$response" | jq -r '.data.systemMessage // empty')
    if [ -n "$system_message" ]; then
        echo "$system_message"
    fi
fi

# Exit 0 to not block agent
exit 0