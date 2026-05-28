# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Memory Brain is a **generic standalone memory service** for AI agents. Any agent can integrate by configuring hooks that conform to the input/output format defined in this project.

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
    └─ Storage (~/.memory-brain/)
        ├─ profile.jsonl    # User profile
        ├─ patterns.jsonl   # Behavior patterns
        └─ knowledge.db     # Chromem vector store
```

## Commands

```bash
make build          # Build binary to bin/memory-brain
make run            # Run server on port 12321
make test           # Run tests (verbose)
make deps           # Download and tidy dependencies

go build -o bin/memory-brain ./cmd/main.go
go run ./cmd/main.go server --port 12321
go test -v ./...
```

## Architecture

**Directory Structure:**
- `cmd/main.go` - Entry point (CLI with `server` command)
- `internal/` - Private packages
- `pkg/types/` - Shared type definitions
- `hooks/` - Example shell scripts for agent integration

**Core Packages:**
| Package | Purpose |
|---------|---------|
| `internal/api/server.go` | REST API handlers |
| `internal/search/` | Hybrid search (BM25 + Vector + RRF fusion) |
| `internal/storage/profile.go` | JSONL file storage |
| `internal/decay/ebbinghaus.go` | Memory decay algorithm |
| `internal/dedup/hash.go` | SHA-256 deduplication |
| `internal/privacy/filter.go` | Sensitive data filtering |

**Key Types** (pkg/types/types.go):
- `Observation` - Captured event from Hook
- `ProfileCard` - User profile entry (with Strength/DecayDays)
- `SearchResult` - Search result with source attribution

## Storage Pattern

- JSONL append-only format
- Per-user directory: `~/.memory-brain/users/{userID}/`
- Thread-safe with `sync.RWMutex`

## Hook Integration

Hooks provide a generic integration layer - any agent can use Memory Brain by configuring hooks that call the REST API with the expected JSON format.

**Example hooks** in `hooks/` directory:
- `session-start.sh` - Load user profile at session start
- `post-tool-use.sh` - Capture tool usage (async)
- `post-compact.sh` - Generate session summary
- `session-end.sh` - Batch profile update

Hooks require `jq` for JSON parsing. See `configs/hooks.yaml.example` for configuration format.

## Conventions

- Types centralized in `pkg/types/types.go`
- Import path: `github.com/oneliang/memory-brain`
- Graceful degradation: vector search failure doesn't block BM25
- Thread safety via `sync.RWMutex`