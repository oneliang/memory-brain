#!/bin/bash
# 简单的 curl 测试命令集合

USER_ID="curl-test-user"
BASE_URL="http://localhost:12321"

echo "========================================="
echo "Curl 测试命令集合"
echo "========================================="
echo ""

# 1. 添加测试数据
echo "1. 添加测试数据（复制这些命令执行）："
echo ""
echo "# 观察1: 使用 bash 执行 kubectl"
cat << 'EOF'
curl -X POST http://localhost:12321/memory/observe \
  -H "Content-Type: application/json" \
  -d '{
    "id": "curl_test_001",
    "session_id": "session_curl",
    "user_id": "curl-test-user",
    "hook_type": "post_tool_use",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "data": {
      "tool_name": "bash",
      "args": "kubectl get pods -n production",
      "success": true
    }
  }'
EOF

echo ""
echo "# 观察2: 修改文件涉及 PostgreSQL"
cat << 'EOF'
curl -X POST http://localhost:12321/memory/observe \
  -H "Content-Type: application/json" \
  -d '{
    "id": "curl_test_002",
    "session_id": "session_curl",
    "user_id": "curl-test-user",
    "hook_type": "post_tool_use",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "data": {
      "tool_name": "write",
      "file_path": "/internal/server.go",
      "mentions": ["PostgreSQL", "Redis"],
      "success": true
    }
  }'
EOF

echo ""
echo "========================================="
echo "2. 等待异步提取（8秒）"
echo "========================================="
for i in {1..8}; do
    sleep 1
    echo "   $i..."
done
echo "✅ 提取完成"
echo ""

# 3. 查看图谱数据
echo "========================================="
echo "3. 查看图谱数据"
echo "========================================="
echo ""
echo "命令："
echo "curl 'http://localhost:12321/memory/graph/context?user_id=$USER_ID' | jq ."
echo ""
echo "输出："
curl -s "$BASE_URL/memory/graph/context?user_id=$USER_ID" | jq .
echo ""

# 4. 查看最终输出（核心）
echo "========================================="
echo "4. 查看最终输出（路径格式）"
echo "========================================="
echo ""

echo "4.1 完整 JSON 响应："
echo "命令："
echo "curl 'http://localhost:12321/memory/profile?user=$USER_ID&inject=true' | jq ."
echo ""
echo "输出："
curl -s "$BASE_URL/memory/profile?user=$USER_ID&inject=true" | jq .
echo ""

echo "========================================="
echo "4.2 只看 systemMessage（格式化后）"
echo "========================================="
echo ""
echo "命令："
echo "curl 'http://localhost:12321/memory/profile?user=$USER_ID&inject=true' | jq -r '.data.systemMessage'"
echo ""
echo "输出："
curl -s "$BASE_URL/memory/profile?user=$USER_ID&inject=true" | jq -r '.data.systemMessage'
echo ""

echo "========================================="
echo "4.3 查看图谱统计"
echo "========================================="
echo ""
echo "命令："
echo "curl 'http://localhost:12321/memory/graph/stats?user_id=$USER_ID' | jq ."
echo ""
echo "输出："
curl -s "$BASE_URL/memory/graph/stats?user_id=$USER_ID" | jq .
echo ""

echo "========================================="
echo "快速测试命令汇总"
echo "========================================="
echo ""
echo "# 1. 添加数据"
echo "curl -X POST http://localhost:12321/memory/observe \\"
echo "  -H 'Content-Type: application/json' \\"
echo "  -d '{\"id\":\"test_001\",\"session_id\":\"s1\",\"user_id\":\"curl-test-user\",\"hook_type\":\"post_tool_use\",\"timestamp\":\"'$(date -u +%Y-%m-%dT%H:%M:%SZ)'\",\"data\":{\"tool_name\":\"bash\",\"args\":\"kubectl get pods\",\"success\":true}}'"
echo ""
echo "# 2. 等待 8 秒"
echo "sleep 8"
echo ""
echo "# 3. 查看最终输出"
echo "curl 'http://localhost:12321/memory/profile?user=curl-test-user&inject=true' | jq -r '.data.systemMessage'"
echo ""
