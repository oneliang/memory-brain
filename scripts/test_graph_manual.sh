#!/bin/bash
# Graph RAG 手工测试脚本

BASE_URL="http://localhost:12321"
USER_ID="test-user"

echo "=== Graph RAG 手工测试 ==="
echo ""

# 1. 健康检查
echo "1. 健康检查..."
curl -s "$BASE_URL/memory/health" | jq .
echo ""

# 2. 提取中文输入
echo "2. 提取中文输入..."
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"id\": \"obs-$(date +%s)-1\",
    \"hook_type\": \"user_prompt_submit\",
    \"data\": {
      \"prompt\": \"张三在阿里巴巴工作，他用 Go 和 PostgreSQL 开发微服务\"
    }
  }" | jq .
echo ""

# 3. 提取英文输入
echo "3. 提取英文输入..."
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"id\": \"obs-$(date +%s)-2\",
    \"hook_type\": \"user_prompt_submit\",
    \"data\": {
      \"prompt\": \"John works at Google, he uses Kubernetes and Docker to deploy services\"
    }
  }" | jq .
echo ""

# 4. 提取工具使用
echo "4. 提取工具使用..."
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"id\": \"obs-$(date +%s)-3\",
    \"hook_type\": \"post_tool_use\",
    \"data\": {
      \"tool_name\": \"bash\",
      \"tool_input\": {
        \"command\": \"kubectl get pods -n production\"
      }
    }
  }" | jq .
echo ""

# 等待异步提取完成
echo "等待异步提取完成 (5秒)..."
sleep 5

# 5. 查看统计
echo "5. 查看图统计..."
curl -s "$BASE_URL/memory/graph/stats?user_id=$USER_ID" | jq .
echo ""

# 6. 多跳查询
echo "6. 多跳查询..."
curl -s "$BASE_URL/memory/graph/query?user_id=$USER_ID&max_hops=2&limit=20" | jq .
echo ""

# 7. 获取上下文
echo "7. 获取用户上下文..."
curl -s "$BASE_URL/memory/graph/context?user_id=$USER_ID&limit=10" | jq .
echo ""

# 8. 按类型过滤查询
echo "8. 按类型过滤查询 (只看 tech)..."
curl -s "$BASE_URL/memory/graph/query?user_id=$USER_ID&max_hops=2&limit=10&node_types=tech" | jq .
echo ""

echo "=== 测试完成 ==="
