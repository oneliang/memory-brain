#!/bin/bash
# post-tool-use.sh - PostToolUse Hook for Memory Brain
# Captures tool usage and sends to Memory Brain

# Default configuration
MEMORY_BRAIN_URL="http://localhost:12321"
TIMEOUT=3

# Read stdin JSON from agent
input=$(cat)

# Extract fields from JSON
session_id=$(echo "$input" | jq -r '.session_id // empty')
user_id=$(echo "$input" | jq -r '.user_id // empty')
tool_name=$(echo "$input" | jq -r '.tool_name // empty')
tool_input=$(echo "$input" | jq '.tool_input // {}')
tool_result=$(echo "$input" | jq '.tool_result // {}')
directory=$(echo "$input" | jq -r '.directory // empty')

# If directory not provided, use current working directory
if [ -z "$directory" ]; then
    directory=$(pwd)
fi

if [ -z "$session_id" ] || [ -z "$user_id" ] || [ -z "$tool_name" ]; then
    exit 0  # Don't block agent
fi

# Generate unique observation ID
obs_id="obs_$(date +%s)_$(echo "$tool_name" | cksum | cut -d' ' -f1)"

# Build request JSON
request_json=$(jq -n \
    --arg id "$obs_id" \
    --arg sid "$session_id" \
    --arg uid "$user_id" \
    --arg hook "post_tool_use" \
    --arg ts "$(date -u +"%Y-%m-%dT%H:%M:%SZ")" \
    --arg tool "$tool_name" \
    --argjson input "$tool_input" \
    --argjson result "$tool_result" \
    --arg dir "$directory" \
    '{
        id: $id,
        session_id: $sid,
        user_id: $uid,
        hook_type: $hook,
        timestamp: $ts,
        directory: $dir,
        data: {
            tool_name: $tool,
            tool_input: $input,
            tool_result: $result
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