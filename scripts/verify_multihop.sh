#!/bin/bash
# Graph RAG 多跳查询验证脚本

BASE_URL="http://localhost:12321"
USER_ID="verify-user"

echo "=== 多跳查询验证 ==="
echo ""

# 清理旧数据（使用新用户）
echo "使用新用户: $USER_ID"
echo ""

# 1. 创建明确的图结构
echo "=== 步骤 1: 创建测试数据 ==="
echo ""

# 观察 1: user -> uses -> bash
echo "1.1 创建: user --uses--> bash"
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"id\": \"v-1\",
    \"hook_type\": \"post_tool_use\",
    \"data\": {
      \"tool_name\": \"bash\"
    }
  }" | jq '{success, entities, relations}'
echo ""

# 观察 2: user -> uses -> git
echo "1.2 创建: user --uses--> git"
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"id\": \"v-2\",
    \"hook_type\": \"post_tool_use\",
    \"data\": {
      \"tool_name\": \"git\"
    }
  }" | jq '{success, entities, relations}'
echo ""

# 观察 3: user -> modifies -> server.go
echo "1.3 创建: user --modifies--> server.go"
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"id\": \"v-3\",
    \"hook_type\": \"post_tool_use\",
    \"data\": {
      \"tool_name\": \"write\",
      \"file_path\": \"/internal/server.go\"
    }
  }" | jq '{success, entities, relations}'
echo ""

# 观察 4: bash -> executes -> kubectl (通过命令)
echo "1.4 创建: user --executes--> kubectl (通过 bash)"
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"id\": \"v-4\",
    \"hook_type\": \"post_tool_use\",
    \"data\": {
      \"tool_name\": \"bash\",
      \"tool_input\": {
        \"command\": \"kubectl get pods\"
      }
    }
  }" | jq '{success, entities, relations}'
echo ""

# 观察 5: server.go -> uses -> PostgreSQL
echo "1.5 创建: server.go --uses--> PostgreSQL (通过 prompt)"
curl -s -X POST "$BASE_URL/memory/graph/extract" \
  -H "Content-Type: application/json" \
  -d "{
    \"user_id\": \"$USER_ID\",
    \"id\": \"v-5\",
    \"hook_type\": \"user_prompt_submit\",
    \"data\": {
      \"prompt\": \"server.go 使用 PostgreSQL 数据库\"
    }
  }" | jq '{success, entities, relations}'
echo ""

# 等待异步处理
echo "等待异步提取 (10秒)..."
sleep 10

# 2. 查看图统计
echo ""
echo "=== 步骤 2: 图统计 ==="
curl -s "$BASE_URL/memory/graph/stats?user_id=$USER_ID" | jq .
echo ""

# 3. 1-hop 查询
echo ""
echo "=== 步骤 3: 1-hop 查询 (直接邻居) ==="
echo "预期: bash, git, server.go, kubectl, PostgreSQL 等直接关联的实体"
echo ""
curl -s "$BASE_URL/memory/graph/query?user_id=$USER_ID&max_hops=1&limit=20" | jq '.results[] | {type: .node.type, name: .node.name, hops: .hops, score: .score}'
echo ""

# 4. 2-hop 查询
echo ""
echo "=== 步骤 4: 2-hop 查询 (二跳邻居) ==="
echo "预期: 包含 1-hop 的所有实体 + 通过它们连接的其他实体"
echo ""
curl -s "$BASE_URL/memory/graph/query?user_id=$USER_ID&max_hops=2&limit=20" | jq '.results[] | {type: .node.type, name: .node.name, hops: .hops, score: .score}'
echo ""

# 5. 验证分数衰减
echo ""
echo "=== 步骤 5: 验证分数衰减 ==="
echo "1-hop 的分数应该 > 2-hop 的分数"
echo ""
curl -s "$BASE_URL/memory/graph/query?user_id=$USER_ID&max_hops=2&limit=20" | jq '
  .results |
  group_by(.hops) |
  map({
    hops: .[0].hops,
    count: length,
    max_score: (map(.score) | max),
    min_score: (map(.score) | min)
  })
'
echo ""

# 6. 按类型过滤
echo ""
echo "=== 步骤 6: 按类型过滤 (只看 tool) ==="
curl -s "$BASE_URL/memory/graph/query?user_id=$USER_ID&max_hops=2&limit=20&node_types=tool" | jq '.results[] | {type: .node.type, name: .node.name, hops: .hops}'
echo ""

# 7. 获取上下文
echo ""
echo "=== 步骤 7: 用户上下文 ==="
curl -s "$BASE_URL/memory/graph/context?user_id=$USER_ID&limit=10" | jq .
echo ""

# 8. 预期图结构
echo ""
echo "=== 预期图结构 ==="
echo ""
echo "  $USER_ID --uses--> bash"
echo "  $USER_ID --uses--> git"
echo "  $USER_ID --modifies--> /internal/server.go"
echo "  $USER_ID --executes--> kubectl get pods"
echo "  $USER_ID --mentions--> PostgreSQL"
echo ""
echo "如果查询结果符合上述结构，说明多跳查询实现正确！"
echo ""

echo "=== 验证完成 ==="
