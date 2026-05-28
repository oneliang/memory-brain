# Memory Brain

A standalone memory service for AI agents. Any agent can integrate by sending HTTP requests to the REST API defined in this project.

## Overview

Memory Brain is an independent server that provides memory capabilities for AI agents. It captures user behavior through HTTP API calls and builds user profiles for personalized interactions.

## Architecture

```
Memory Brain Server (Go, port 12321)
    │
    ├─ REST API
    │   ├─ POST /memory/observe         # Capture observations
    │   ├─ GET  /memory/profile         # Load user profile
    │   ├─ GET  /memory/search          # Hybrid search
    │   ├─ PUT  /memory/profile/update  # Update profile
    │   ├─ POST /memory/session/summary # Session summary
    │   └─ GET  /memory/health          # Health check
    │
    └─ Storage (~/.memory-brain/users/{user_id}/)
        ├─ profile.jsonl        # User profile cards
        ├─ patterns.jsonl       # Behavior patterns (tool usage)
        ├─ sessions_archive/    # Session summaries
        └─ knowledge.db/        # Chromem vector store
```

## Installation

```bash
# Clone the repository
git clone https://github.com/oneliang/memory-brain.git
cd memory-brain

# Build
make build

# Run (default port 12321)
make run
# Or specify port:
./bin/memory-brain server --port 12321
```

## Integration Guide

### Integration Principle

Memory Brain is a **generic service** - any agent can integrate by calling the REST API. Each agent has its own hook system, so you need to:

1. Understand your agent's hook mechanism
2. Call Memory Brain API from your agent's hooks
3. Pass the correct JSON format

### Hook Script Reference

The `hooks/*.sh` files are **reference examples** showing how to call the API. Adapt them to your agent's hook format.

**stdin JSON format** (from agent to hook):
```json
{
  "user_id": "user_123",
  "session_id": "session_abc",
  "tool_name": "bash",
  "tool_input": {...},
  "tool_result": {...}
}
```

### Integration Steps

1. **Start Memory Brain server**:
   ```bash
   memory-brain server --port 12321
   ```

2. **Configure your agent's hooks** to call the API (adapt from `hooks/*.sh`)

3. **Verify**:
   ```bash
   curl http://localhost:12321/memory/health
   ```

## API Reference

### POST /memory/observe

Capture an observation from a hook event. Supports multiple hook types.

#### Hook Type: post_tool_use

Capture tool usage for behavior pattern analysis.

**Request**:
```json
{
  "id": "obs_001",
  "session_id": "session_abc",
  "user_id": "user_123",
  "hook_type": "post_tool_use",
  "timestamp": "2026-05-26T10:00:00Z",
  "data": {
    "tool_name": "bash",
    "tool_input": {"command": "ls -la"},
    "tool_result": "success"
  }
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "obs_id": "obs_001",
    "dedup": false
  }
}
```

**Storage**:
- Creates/updates `patterns.jsonl` with tool_usage pattern (frequency merged)
- Pattern ID format: `pattern_{user_id}_{tool_name}_{nanosecond}`

#### Hook Type: user_prompt_submit

Capture user's original message for intent analysis. **Recommended to capture every user message.**

**Request**:
```json
{
  "id": "obs_002",
  "session_id": "session_abc",
  "user_id": "user_123",
  "hook_type": "user_prompt_submit",
  "timestamp": "2026-05-26T10:01:00Z",
  "data": {
    "prompt": "帮我实现一个REST API服务器"
  }
}
```

**Response**:
```json
{
  "success": true,
  "data": {
    "obs_id": "obs_002",
    "dedup": false
  }
}
```

**Storage**:
- Creates/updates `profile.jsonl` with IntentCard (intent frequency merged)
- Creates `patterns.jsonl` with user_prompt pattern (original text for summary generation)

**Intent Classification**:

| Intent | Keywords (CN/EN) |
|--------|------------------|
| development | 实现/开发/代码/编写/创建/添加/implement/code/create/add/build |
| debugging | 调试/修复/错误/bug/问题/debug/fix/error/issue |
| query | 查询/搜索/查找/怎么/如何/解释/query/search/find/how/explain |
| management | 管理/配置/部署/设置/manage/config/deploy/setup/install |
| review | 检查/审查/优化/重构/测试/review/optimize/refactor/test |
| general | (default, no matching keywords) |

**Deduplication**: Returns `dedup: true` if same session + same data was already received within 5 minutes.

### GET /memory/profile

Load user profile. Use at session start.

**Query params**:
- `user` - User ID (required)
- `inject` - Return system message for injection (optional, default: false)

**Response**:
```json
{
  "success": true,
  "data": {
    "profile_summary": {
      "user_id": "user_123",
      "profile_count": 5,
      "pattern_count": 3
    },
    "systemMessage": "用户画像摘要:\n- 工具偏好: bash(10), read(5)\n- 主要意图: development"
  }
}
```

### GET /memory/search

Hybrid search across sessions (BM25 + Vector + RRF fusion).

**Query params**:
- `query` - Search query (required)
- `user` - User ID (required)
- `limit` - Max results (optional, default: 5)

### PUT /memory/profile/update

Batch update user profile. Use at session end.

**Request**:
```json
{
  "user_id": "user_123",
  "session_id": "session_abc",
  "observations": [...]
}
```

### POST /memory/session/summary

Generate and save session summary.

**Request**:
```json
{
  "user_id": "user_123",
  "session_id": "session_abc",
  "generate": true,
  "observations": [...]
}
```

## Features

- **Four-layer memory**: Working → Episodic → Semantic → Procedural
- **Hybrid search**: BM25 + Vector + RRF fusion
- **Hybrid dedup**: Session-level (immediate) + Pattern-level (frequency merge)
- **Intent analysis**: Automatic classification of user prompts
- **Privacy filtering**: Automatic removal of API keys and secrets
- **Decay algorithm**: Ebbinghaus curve for memory strength

## Hook Events

| Event | When | API Call | Hook Type | Purpose |
|-------|------|----------|-----------|---------|
| SessionStart | Session begins | GET /memory/profile | - | Load user profile |
| UserPromptSubmit | User sends message | POST /memory/observe | `user_prompt_submit` | Capture user intent |
| PostToolUse | After tool call | POST /memory/observe | `post_tool_use` | Capture tool usage |
| PostCompact | After compression | POST /memory/session/summary | - | Generate summary |
| SessionEnd | Session ends | PUT /memory/profile/update | - | Batch update profile |

**Recommended**: Capture `UserPromptSubmit` for every user message to build accurate intent profile.

## Storage Files

| File | Format | Content |
|------|--------|---------|
| `profile.jsonl` | JSONL (append) | IntentCard (intent frequency), PreferenceCard |
| `patterns.jsonl` | JSONL (append) | tool_usage patterns, user_prompt patterns |
| `sessions_archive/*.json` | JSON (single) | Session summaries |

## License

Apache 2.0