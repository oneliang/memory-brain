#!/bin/bash
# test_graph.sh - End-to-end test for Graph RAG functionality
# Usage: ./scripts/test_graph.sh

set -e

BASE_URL="http://localhost:12321"
USER_ID="test_user_$(date +%s)"

echo "=== Memory Brain Graph RAG Test ==="
echo "User ID: $USER_ID"
echo ""

# Color codes
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print success
success() {
    echo -e "${GREEN}✓ $1${NC}"
}

# Function to print error
error() {
    echo -e "${RED}✗ $1${NC}"
    exit 1
}

# Function to print info
info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Check if server is running
info "Checking if Memory Brain server is running..."
if curl -s "$BASE_URL/memory/health" > /dev/null 2>&1; then
    success "Server is running"
else
    error "Server is not running. Please start it with: make run"
fi

echo ""
info "=== Test 1: Health Check ==="
curl -s "$BASE_URL/memory/health" | jq .

echo ""
info "=== Test 2: Send Observation (user_prompt_submit) ==="
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"obs_001\",
    \"user_id\": \"$USER_ID\",
    \"session_id\": \"session_001\",
    \"hook_type\": \"user_prompt_submit\",
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"data\": {
      \"prompt\": \"帮我优化 PostgreSQL 查询，张三说要用索引\"
    }
  }" | jq .

echo ""
info "=== Test 3: Send Observation (post_tool_use) ==="
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"obs_002\",
    \"user_id\": \"$USER_ID\",
    \"session_id\": \"session_001\",
    \"hook_type\": \"post_tool_use\",
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"data\": {
      \"tool_name\": \"bash\",
      \"tool_input\": {\"command\": \"psql -U postgres -c 'SELECT * FROM users'\"},
      \"tool_result\": \"success\"
    }
  }" | jq .

echo ""
info "=== Test 4: Send Another Observation ==="
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"obs_003\",
    \"user_id\": \"$USER_ID\",
    \"session_id\": \"session_001\",
    \"hook_type\": \"post_tool_use\",
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"data\": {
      \"tool_name\": \"read\",
      \"file_path\": \"/internal/api/server.go\",
      \"tool_result\": \"success\"
    }
  }" | jq .

echo ""
info "Waiting 3 seconds for async extraction..."
sleep 3

echo ""
info "=== Test 5: Query Graph (2-hop) ==="
curl -s "$BASE_URL/memory/graph/query?user=$USER_ID&hops=2&limit=10" | jq .

echo ""
info "=== Test 6: Get Context ==="
curl -s "$BASE_URL/memory/graph/context?user=$USER_ID&limit=20" | jq .

echo ""
info "=== Test 7: Get Stats ==="
curl -s "$BASE_URL/memory/graph/stats?user=$USER_ID" | jq .

echo ""
info "=== Test 8: Manual Extract ==="
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"id\": \"obs_manual\",
    \"user_id\": \"$USER_ID\",
    \"session_id\": \"session_002\",
    \"hook_type\": \"user_prompt_submit\",
    \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\",
    \"data\": {
      \"prompt\": \"帮我查看 Redis 缓存的配置，李四负责这个项目\"
    }
  }" | jq .

echo ""
info "Waiting 2 seconds..."
sleep 2

echo ""
info "=== Test 9: Query Updated Graph ==="
curl -s "$BASE_URL/memory/graph/query?user=$USER_ID&hops=2&limit=15" | jq .

echo ""
info "=== Test 10: Get Updated Context ==="
curl -s "$BASE_URL/memory/graph/context?user=$USER_ID&limit=20" | jq .

echo ""
success "All tests completed!"
echo ""
echo "Note: For full NER+RE with Ollama, ensure Ollama is running and the model is downloaded:"
echo "  ollama pull qwen2.5:3b"
