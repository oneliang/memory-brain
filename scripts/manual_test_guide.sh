#!/bin/bash
# Memory Brain 手动测试指南

BASE_URL="http://localhost:12321"

echo "=== 1. 健康检查 ==="
curl -s "$BASE_URL/memory/health" | jq .

echo ""
echo "=== 2. 发送单个Observation ==="
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "manual_test_001",
    "session_id": "manual_session_001",
    "user_id": "manual_user_001",
    "hook_type": "post_tool_use",
    "data": {"tool_name": "bash", "tool_result": "success"}
  }' | jq .

echo ""
echo "=== 3. 发送相同数据（验证Session级去重） ==="
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "manual_test_002",
    "session_id": "manual_session_001",
    "user_id": "manual_user_001",
    "hook_type": "post_tool_use",
    "data": {"tool_name": "bash", "tool_result": "success"}
  }' | jq .
echo "# 应返回 dedup=true（同session同数据）"

echo ""
echo "=== 4. 发送不同session相同数据（验证不去重） ==="
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "manual_test_003",
    "session_id": "manual_session_002",
    "user_id": "manual_user_001",
    "hook_type": "post_tool_use",
    "data": {"tool_name": "bash", "tool_result": "success"}
  }' | jq .
echo "# 应返回 dedup=false（不同session）"

echo ""
echo "=== 5. 发送不同工具（验证Pattern去重） ==="
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "manual_test_004",
    "session_id": "manual_session_001",
    "user_id": "manual_user_001",
    "hook_type": "post_tool_use",
    "data": {"tool_name": "read", "tool_result": "success"}
  }' | jq .

curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "manual_test_005",
    "session_id": "manual_session_001",
    "user_id": "manual_user_001",
    "hook_type": "post_tool_use",
    "data": {"tool_name": "read", "tool_result": "success"}
  }' | jq .

echo ""
echo "=== 6. 查看用户画像 ==="
curl -s "$BASE_URL/memory/profile?user=manual_user_001" | jq .

echo ""
echo "=== 7. 查看存储文件 ==="
STORAGE_DIR="$HOME/.memory-brain/users/manual_user_001"
echo "存储目录: $STORAGE_DIR"
ls -la "$STORAGE_DIR"

echo ""
echo "=== patterns.jsonl 内容 ==="
cat "$STORAGE_DIR/patterns.jsonl" | jq -c '{id, pattern, frequency}'

echo ""
echo "# Pattern去重验证:"
echo "# bash应该只有1条记录, frequency=2（来自session_001和session_002各一次）"
echo "# read应该只有1条记录, frequency=2（来自两次调用合并）"
