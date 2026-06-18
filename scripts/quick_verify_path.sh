#!/bin/bash
# 快速验证路径格式输出

set -e

BASE_URL="http://localhost:12321"
USER_ID="quick-test-user"

echo "========================================="
echo "路径格式快速验证"
echo "========================================="
echo ""

# 检查服务
if ! curl -s "$BASE_URL/memory/health" > /dev/null 2>&1; then
    echo "❌ 服务未运行，请先启动: make run"
    exit 1
fi

# 清理并添加测试数据
echo "1. 添加测试数据..."
rm -rf ~/.memory-brain/users/$USER_ID 2>/dev/null || true

curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_001",
    "session_id": "session_001",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "data": {
      "tool_name": "bash",
      "args": "kubectl get pods",
      "success": true
    }
  }' > /dev/null

curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_002",
    "session_id": "session_001",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'",
    "data": {
      "tool_name": "write",
      "file_path": "/server.go",
      "mentions": ["PostgreSQL"],
      "success": true
    }
  }' > /dev/null

echo "✅ 已添加 2 条观察数据"
echo ""

# 等待提取
echo "2. 等待 NER+RE 提取（8秒）..."
for i in {1..8}; do
    sleep 1
    echo "   $i..."
done
echo ""

# 验证图谱数据
echo "3. 验证图谱数据..."
GRAPH_DATA=$(curl -s "$BASE_URL/memory/graph/context?user_id=$USER_ID")
TOTAL=$(echo "$GRAPH_DATA" | jq -r '.data.total // 0')

if [ "$TOTAL" = "0" ] || [ "$TOTAL" = "null" ]; then
    echo "⚠️  图谱数据为空，尝试手动触发提取..."
    curl -s -X POST "$BASE_URL/memory/graph/extract" \
      -H "Content-Type: application/json" \
      -d '{"user_id": "'$USER_ID'"}' > /dev/null
    sleep 5
    GRAPH_DATA=$(curl -s "$BASE_URL/memory/graph/context?user_id=$USER_ID")
    TOTAL=$(echo "$GRAPH_DATA" | jq -r '.data.total // 0')
fi

echo "   实体总数: $TOTAL"
if [ "$TOTAL" -eq 0 ] 2>/dev/null; then
    echo "❌ 提取失败，请检查 Ollama 是否运行"
    echo ""
    echo "调试信息："
    echo "  Ollama 状态: $(ollama list 2>&1 | head -3)"
    echo "  图谱响应: $GRAPH_DATA"
    exit 1
fi
echo "✅ 图谱数据已生成"
echo ""

# 显示路径格式输出
echo "========================================="
echo "4. 路径格式输出（核心验证）"
echo "========================================="
echo ""

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

# 验证格式特征
echo "========================================="
echo "5. 格式验证"
echo "========================================="

PASS=0
TOTAL=0

TOTAL=$((TOTAL + 1))
if echo "$SYSTEM_MSG" | grep -q "用户知识图谱:"; then
    echo "✅ 包含 '用户知识图谱:' 标题"
    PASS=$((PASS + 1))
else
    echo "❌ 缺少 '用户知识图谱:' 标题"
fi

TOTAL=$((TOTAL + 1))
if echo "$SYSTEM_MSG" | grep -q "路径[0-9]:"; then
    echo "✅ 包含路径编号"
    PASS=$((PASS + 1))
else
    echo "❌ 缺少路径编号"
fi

TOTAL=$((TOTAL + 1))
if echo "$SYSTEM_MSG" | grep -q "→"; then
    echo "✅ 包含关系箭头（→）"
    PASS=$((PASS + 1))
else
    echo "❌ 缺少关系箭头"
fi

TOTAL=$((TOTAL + 1))
if echo "$SYSTEM_MSG" | grep -qE "\([a-z_]+\)"; then
    echo "✅ 包含关系类型（uses, executes 等）"
    PASS=$((PASS + 1))
else
    echo "❌ 缺少关系类型"
fi

TOTAL=$((TOTAL + 1))
if echo "$SYSTEM_MSG" | grep -q "实体统计:"; then
    echo "✅ 包含 '实体统计:' 部分"
    PASS=$((PASS + 1))
else
    echo "❌ 缺少 '实体统计:' 部分"
fi

echo ""
echo "========================================="
echo "验证结果: $PASS/$TOTAL 通过"
echo "========================================="

if [ $PASS -eq $TOTAL ]; then
    echo ""
    echo "✅ 所有验证通过！"
    echo ""
    echo "关键改进："
    echo "  旧格式: 扁平实体列表，看不到关系"
    echo "  新格式: 路径格式，显示完整关系链"
    echo ""
    echo "示例："
    echo "  路径1: user (uses) → bash (executes) → kubectl"
    echo "  路径2: user (modifies) → server.go (mentions) → PostgreSQL"
    echo ""
    exit 0
else
    echo ""
    echo "❌ 部分验证失败"
    exit 1
fi
