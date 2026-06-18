#!/bin/bash
# 验证路径格式输出的手工测试脚本

set -e

BASE_URL="http://localhost:12321"
USER_ID="path-test-user"

echo "========================================="
echo "路径格式输出验证测试"
echo "========================================="
echo ""

# 1. 检查服务是否运行
echo "1. 检查服务状态..."
if ! curl -s "$BASE_URL/memory/health" > /dev/null 2>&1; then
    echo "❌ 服务未运行，请先启动: make run"
    exit 1
fi
echo "✅ 服务正常运行"
echo ""

# 2. 清理旧数据（可选）
echo "2. 清理旧测试数据..."
rm -rf ~/.memory-brain/users/$USER_ID
echo "✅ 已清理"
echo ""

# 3. 添加测试观察数据
echo "3. 添加测试数据（创建多跳关系）..."

# 观察1: 用户使用 bash 执行 kubectl 命令
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_path_001",
    "session_id": "session_path_001",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "data": {
      "tool_name": "bash",
      "args": "kubectl get pods -n production",
      "success": true
    }
  }' > /dev/null
echo "   ✓ 观察1: bash → kubectl"

# 观察2: 用户使用 git 提交代码
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_path_002",
    "session_id": "session_path_001",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "data": {
      "tool_name": "bash",
      "args": "git commit -m 'update server'",
      "success": true
    }
  }' > /dev/null
echo "   ✓ 观察2: bash → git"

# 观察3: 用户修改文件，涉及 PostgreSQL
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_path_003",
    "session_id": "session_path_001",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "data": {
      "tool_name": "write",
      "file_path": "/internal/db/connection.go",
      "mentions": ["PostgreSQL", "connection pool"],
      "success": true
    }
  }' > /dev/null
echo "   ✓ 观察3: write → connection.go → PostgreSQL"

# 观察4: 用户使用 Docker 部署
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_path_004",
    "session_id": "session_path_001",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "data": {
      "tool_name": "bash",
      "args": "docker build -t myapp .",
      "success": true
    }
  }' > /dev/null
echo "   ✓ 观察4: bash → docker"

echo ""

# 4. 等待异步提取
echo "4. 等待 NER+RE 异步提取..."
echo "   （预计需要 5-10 秒）"
for i in {1..10}; do
    sleep 1
    echo "   $i..."
done
echo "✅ 提取完成"
echo ""

# 5. 验证图谱数据
echo "5. 验证图谱数据..."
GRAPH_DATA=$(curl -s "$BASE_URL/memory/graph/context?user_id=$USER_ID")
TOTAL=$(echo "$GRAPH_DATA" | jq -r '.data.total // 0')

if [ "$TOTAL" = "0" ] || [ "$TOTAL" = "null" ]; then
    echo "⚠️  警告: 图谱数据为空，可能提取失败"
    echo "   请检查 Ollama 是否运行: ollama list"
    echo ""
    echo "   尝试手动提取..."
    curl -s -X POST "$BASE_URL/memory/graph/extract" \
      -H "Content-Type: application/json" \
      -d '{"user_id": "'$USER_ID'"}' > /dev/null
    sleep 8
    GRAPH_DATA=$(curl -s "$BASE_URL/memory/graph/context?user_id=$USER_ID")
    TOTAL=$(echo "$GRAPH_DATA" | jq -r '.data.total // 0')
fi

echo "   图谱实体总数: $TOTAL"
if [ "$TOTAL" -gt 0 ]; then
    echo "✅ 图谱数据已生成"
else
    echo "❌ 图谱数据为空，请检查 Ollama"
    exit 1
fi
echo ""

# 6. 验证路径格式输出
echo "6. 验证路径格式输出..."
echo ""
echo "========================================="
echo "新的路径格式输出（/memory/profile?inject=true）"
echo "========================================="
PROFILE_RESP=$(curl -s "$BASE_URL/memory/profile?user=$USER_ID&inject=true")
SYSTEM_MSG=$(echo "$PROFILE_RESP" | jq -r '.data.systemMessage // empty')

if [ -z "$SYSTEM_MSG" ]; then
    echo "❌ 未获取到系统消息"
    echo ""
    echo "完整响应："
    echo "$PROFILE_RESP" | jq .
    exit 1
fi

echo "$SYSTEM_MSG"
echo ""

# 7. 验证路径格式特征
echo "========================================="
echo "7. 验证路径格式特征"
echo "========================================="

if echo "$SYSTEM_MSG" | grep -q "用户知识图谱:"; then
    echo "✅ 包含 '用户知识图谱:' 标题"
else
    echo "❌ 缺少 '用户知识图谱:' 标题"
fi

if echo "$SYSTEM_MSG" | grep -q "路径[0-9]:"; then
    echo "✅ 包含路径编号（路径1:, 路径2:, ...）"
else
    echo "❌ 缺少路径编号"
fi

if echo "$SYSTEM_MSG" | grep -q "→"; then
    echo "✅ 包含关系箭头（→）"
else
    echo "❌ 缺少关系箭头"
fi

if echo "$SYSTEM_MSG" | grep -qE "\([a-z_]+\)"; then
    echo "✅ 包含关系类型（如 uses, executes, modifies）"
else
    echo "❌ 缺少关系类型"
fi

if echo "$SYSTEM_MSG" | grep -q "实体统计:"; then
    echo "✅ 包含 '实体统计:' 部分"
else
    echo "❌ 缺少 '实体统计:' 部分"
fi

echo ""

# 8. 显示图谱统计
echo "========================================="
echo "8. 图谱统计信息"
echo "========================================="
curl -s "$BASE_URL/memory/graph/stats?user_id=$USER_ID" | jq -r '
  .data |
  "节点数: \(.total_nodes)",
  "边数: \(.total_edges)",
  "节点类型:",
  (.node_types | to_entries[] | "  - \(.key): \(.value)"),
  "关系类型:",
  (.relation_types | to_entries[] | "  - \(.key): \(.value)")
'

echo ""
echo "========================================="
echo "✅ 验证完成！"
echo "========================================="
echo ""
echo "关键改进："
echo "  旧格式: 扁平实体列表，LLM 看不到关系"
echo "  新格式: 路径格式，显示完整关系链"
echo ""
echo "示例对比："
echo "  旧: - 常用工具: bash, git"
echo "  新: 路径1: test-user (uses) → bash (executes) → kubectl"
echo ""
