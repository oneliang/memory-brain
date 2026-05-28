#!/bin/bash
# user-prompt-submit.sh - UserPromptSubmit Hook for Memory Brain
# Captures user's original message for intent analysis
# IMPORTANT: This hook should capture EVERY user message to build accurate intent profile

# Default configuration
MEMORY_BRAIN_URL="http://localhost:12321"
TIMEOUT=3

# Read stdin JSON from agent
input=$(cat)

# Extract fields from JSON
session_id=$(echo "$input" | jq -r '.session_id // empty')
user_id=$(echo "$input" | jq -r '.user_id // empty')
prompt=$(echo "$input" | jq -r '.prompt // empty')

if [ -z "$session_id" ] || [ -z "$user_id" ] || [ -z "$prompt" ]; then
    exit 0  # Don't block agent
fi

# Generate unique observation ID
obs_id="prompt_$(date +%s)_$(echo "$prompt" | cksum | cut -d' ' -f1)"

# Build request JSON
request_json=$(jq -n \
    --arg id "$obs_id" \
    --arg sid "$session_id" \
    --arg uid "$user_id" \
    --arg hook "user_prompt_submit" \
    --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
    --arg prompt "$prompt" \
    '{
        id: $id,
        session_id: $sid,
        user_id: $uid,
        hook_type: $hook,
        timestamp: $ts,
        data: {
            prompt: $prompt
        }
    }')

# Async call to Memory Brain API (don't wait for response)
curl -s -X POST \
    "${MEMORY_BRAIN_URL}/memory/observe" \
    -H "Content-Type: application/json" \
    -d "$request_json" \
    --max-time "$TIMEOUT" \
    >/dev/null 2>&1 &

# Exit 0 to not block agent
exit 0