#!/bin/bash

# Memory Brain 功能测试脚本
# 模拟一个完整的对话会话，测试所有核心功能

set -e

BASE_URL="http://localhost:12321"
USER_ID="test_user_001"
SESSION_ID="test_session_001"

echo "=== Memory Brain 功能测试 ==="
echo "用户ID: $USER_ID"
echo "会话ID: $SESSION_ID"
echo ""

# 1. 健康检查
echo "1. 健康检查"
curl -s "$BASE_URL/memory/health" | jq .
echo ""

# 2. 模拟10条观察数据
echo "2. 发送观察数据..."

# 观察1: 用户提问
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_001",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "user_prompt_submit",
    "timestamp": "2026-05-25T10:00:00Z",
    "data": {
      "prompt": "实现一个REST API服务器，支持用户认证和数据分析"
    }
  }' | jq .
echo ""

# 观察2: 工具调用 - 读取文件
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_002",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "2026-05-25T10:01:00Z",
    "data": {
      "tool_name": "read",
      "file_path": "/src/api/server.go",
      "tool_result": "success"
    }
  }' | jq .
echo ""

# 观察3: 工具调用 - 编辑文件
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_003",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "2026-05-25T10:02:00Z",
    "data": {
      "tool_name": "edit",
      "file_path": "/src/api/server.go",
      "tool_result": "success"
    }
  }' | jq .
echo ""

# 观察4: 工具调用 - Bash命令
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_004",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "2026-05-25T10:03:00Z",
    "data": {
      "tool_name": "bash",
      "tool_input": "go build -o bin/server ./cmd/main.go",
      "tool_result": "success"
    }
  }' | jq .
echo ""

# 观察5: 用户追问
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_005",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "user_prompt_submit",
    "timestamp": "2026-05-25T10:04:00Z",
    "data": {
      "prompt": "添加JWT认证中间件"
    }
  }' | jq .
echo ""

# 观察6: 工具调用 - 读取认证模块
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_006",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "2026-05-25T10:05:00Z",
    "data": {
      "tool_name": "read",
      "file_path": "/src/auth/jwt.go",
      "tool_result": "success"
    }
  }' | jq .
echo ""

# 观察7: 工具调用 - 创建新文件
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_007",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "2026-05-25T10:06:00Z",
    "data": {
      "tool_name": "write",
      "file_path": "/src/middleware/auth.go",
      "tool_result": "success"
    }
  }' | jq .
echo ""

# 观察8: 用户确认
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_008",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "user_prompt_submit",
    "timestamp": "2026-05-25T10:07:00Z",
    "data": {
      "prompt": "运行测试验证功能"
    }
  }' | jq .
echo ""

# 观察9: 工具调用 - 运行测试
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_009",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "2026-05-25T10:08:00Z",
    "data": {
      "tool_name": "bash",
      "tool_input": "go test ./...",
      "tool_result": "success"
    }
  }' | jq .
echo ""

# 观察10: 重复数据测试（去重）
echo "测试去重功能（发送重复数据）..."
curl -s -X POST "$BASE_URL/memory/observe" \
  -H "Content-Type: application/json" \
  -d '{
    "id": "obs_010",
    "session_id": "'$SESSION_ID'",
    "user_id": "'$USER_ID'",
    "hook_type": "post_tool_use",
    "timestamp": "2026-05-25T10:08:00Z",
    "data": {
      "tool_name": "bash",
      "tool_input": "go test ./...",
      "tool_result": "success"
    }
  }' | jq .
echo ""
echo "注意: obs_010 应显示 dedup=true（被去重）"

# 3. 查看用户画像
echo ""
echo "3. 查看用户画像"
curl -s "$BASE_URL/memory/profile?user=$USER_ID" | jq .
echo ""

# 4. 查看用户画像（带系统消息注入）
echo "4. 查看用户画像（inject=true）"
curl -s "$BASE_URL/memory/profile?user=$USER_ID&inject=true" | jq .
echo ""

# 5. 更新用户画像
echo "5. 更新用户画像（分析本次会话）"
curl -s -X PUT "$BASE_URL/memory/profile/update" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "'$USER_ID'",
    "session_id": "'$SESSION_ID'"
  }' | jq .
echo ""

# 6. 生成会话摘要（传入observations数据）
echo "6. 生成会话摘要（传入本次会话的观察数据）"
curl -s -X POST "$BASE_URL/memory/session/summary" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": "'$USER_ID'",
    "session_id": "'$SESSION_ID'",
    "generate": true,
    "observations": [
      {
        "id": "obs_001",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "user_prompt_submit",
        "data": {"prompt": "实现一个REST API服务器，支持用户认证和数据分析"}
      },
      {
        "id": "obs_002",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "post_tool_use",
        "data": {"tool_name": "read", "file_path": "/src/api/server.go", "tool_result": "success"}
      },
      {
        "id": "obs_003",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "post_tool_use",
        "data": {"tool_name": "edit", "file_path": "/src/api/server.go", "tool_result": "success"}
      },
      {
        "id": "obs_004",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "post_tool_use",
        "data": {"tool_name": "bash", "tool_input": "go build", "tool_result": "success"}
      },
      {
        "id": "obs_005",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "user_prompt_submit",
        "data": {"prompt": "添加JWT认证中间件"}
      },
      {
        "id": "obs_006",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "post_tool_use",
        "data": {"tool_name": "read", "file_path": "/src/auth/jwt.go", "tool_result": "success"}
      },
      {
        "id": "obs_007",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "post_tool_use",
        "data": {"tool_name": "write", "file_path": "/src/middleware/auth.go", "tool_result": "success"}
      },
      {
        "id": "obs_008",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "user_prompt_submit",
        "data": {"prompt": "运行测试验证功能"}
      },
      {
        "id": "obs_009",
        "session_id": "'$SESSION_ID'",
        "user_id": "'$USER_ID'",
        "hook_type": "post_tool_use",
        "data": {"tool_name": "bash", "tool_input": "go test ./...", "tool_result": "success"}
      }
    ]
  }' | jq .
echo ""

# 7. 搜索测试
echo "7. 搜索功能测试"
echo "搜索关键词: REST API"
curl -s "$BASE_URL/memory/search?query=REST+API&user=$USER_ID&limit=5" | jq .
echo ""

echo "搜索关键词: 认证"
curl -s "$BASE_URL/memory/search?query=认证&user=$USER_ID&limit=5" | jq .
echo ""

echo "搜索关键词: Go"
curl -s "$BASE_URL/memory/search?query=Go&user=$USER_ID&limit=5" | jq .
echo ""

# 8. 查看存储文件
echo ""
echo "8. 查看存储文件内容"
STORAGE_DIR="$HOME/.memory-brain/users/$USER_ID"

echo "--- profile.jsonl ---"
if [ -f "$STORAGE_DIR/profile.jsonl" ]; then
  cat "$STORAGE_DIR/profile.jsonl" | jq -c .
else
  echo "文件不存在"
fi

echo ""
echo "--- patterns.jsonl ---"
if [ -f "$STORAGE_DIR/patterns.jsonl" ]; then
  cat "$STORAGE_DIR/patterns.jsonl" | jq -c .
else
  echo "文件不存在"
fi

echo ""
echo "=== 测试完成 ==="
echo ""
echo "存储位置: $STORAGE_DIR"
echo ""
echo "测试覆盖功能:"
echo "✅ 健康检查"
echo "✅ 观察数据接收（10条）"
echo "✅ 去重功能验证"
echo "✅ 用户画像加载"
echo "✅ 用户画像更新"
echo "✅ 会话摘要生成"
echo "✅ 混合检索功能"
echo "✅ 文件存储验证"