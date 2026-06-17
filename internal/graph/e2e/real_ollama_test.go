package e2e

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/internal/graph/extractors"
	"github.com/oneliang/memory-brain/pkg/types"
)

// setupTestStore creates a temporary store for testing
func setupRealTestStore(t *testing.T) (*graph.Store, func()) {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "graph_real_e2e_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	store, err := graph.NewStoreWithPath(dbPath)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create store: %v", err)
	}

	cleanup := func() {
		store.Close()
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

// checkOllamaAvailable checks if Ollama is available
func checkOllamaAvailable(t *testing.T) bool {
	ext := extractors.NewOllamaExtractor("http://localhost:11434", "qwen3.5:2b", 30*time.Second)
	err := ext.CheckHealth()
	if err != nil {
		t.Skipf("Ollama not available: %v", err)
		return false
	}
	return true
}

// TestRealOllama_Chinese tests NER+RE with Chinese input
func TestRealOllama_Chinese(t *testing.T) {
	if !checkOllamaAvailable(t) {
		return
	}

	store, cleanup := setupRealTestStore(t)
	defer cleanup()

	// Create Ollama extractor with real endpoint
	ollamaExt := extractors.NewOllamaExtractor("http://localhost:11434", "qwen3.5:2b", 120*time.Second)

	// Test cases with Chinese input
	testCases := []struct {
		name     string
		prompt   string
		expected []string // expected entity names
	}{
		{
			name:     "人物与组织",
			prompt:   "张三在阿里巴巴工作，他用 Python 开发机器学习模型",
			expected: []string{"张三", "阿里巴巴", "Python"},
		},
		{
			name:     "技术与框架",
			prompt:   "帮我配置 PostgreSQL 和 Redis，使用 Docker 部署到 Kubernetes",
			expected: []string{"PostgreSQL", "Redis", "Docker", "Kubernetes"},
		},
		{
			name:     "项目与地点",
			prompt:   "李四在北京的字节跳动负责 TikTok 项目",
			expected: []string{"李四", "北京", "字节跳动", "TikTok"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obs := &types.Observation{
				ID:       "obs-" + tc.name,
				UserID:   "user1",
				HookType: "user_prompt_submit",
				Data: map[string]interface{}{
					"prompt": tc.prompt,
				},
				Timestamp: time.Now(),
			}

			t.Logf("输入: %s", tc.prompt)

			result, err := ollamaExt.Extract(obs)
			if err != nil {
				t.Fatalf("Extract failed: %v", err)
			}

			t.Logf("提取结果:")
			t.Logf("  实体 (%d):", len(result.Entities))
			for _, entity := range result.Entities {
				t.Logf("    - %s: %s", entity.Type, entity.Name)
			}
			t.Logf("  关系 (%d):", len(result.Relations))
			for _, rel := range result.Relations {
				t.Logf("    - %s --[%s]--> %s", rel.Source, rel.Type, rel.Target)
			}

			// Verify expected entities
			foundEntities := make(map[string]bool)
			for _, entity := range result.Entities {
				foundEntities[entity.Name] = true
			}

			for _, expected := range tc.expected {
				if !foundEntities[expected] {
					t.Errorf("未找到期望的实体: %s", expected)
				} else {
					t.Logf("  ✓ 找到: %s", expected)
				}
			}

			// Save to store
			if err := store.SaveExtractionResult(obs.UserID, result); err != nil {
				t.Fatalf("SaveExtractionResult failed: %v", err)
			}
		})
	}

	// Verify graph statistics
	stats, _ := store.GetStats()
	t.Logf("\n图统计:")
	t.Logf("  节点数: %v", stats["node_count"])
	t.Logf("  边数: %v", stats["edge_count"])
	t.Logf("  节点类型: %v", stats["node_types"])

	// Query and verify
	engine := graph.NewQueryEngine(store)
	results, err := engine.MultiHopQuery("user1", &graph.QueryOptions{
		MaxHops:   2,
		Limit:     50,
		MinScore:  0.1,
		DecayRate: 0.7,
	})
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	t.Logf("\n多跳查询结果 (共 %d 个实体):", len(results))
	for i, result := range results {
		if i < 20 {
			t.Logf("  [%d] %s:%s (hops=%d, score=%.2f)",
				i+1, result.Node.Type, result.Node.Name, result.Hops, result.Score)
		}
	}
}

// TestRealOllama_English tests NER+RE with English input
func TestRealOllama_English(t *testing.T) {
	if !checkOllamaAvailable(t) {
		return
	}

	store, cleanup := setupRealTestStore(t)
	defer cleanup()

	// Create Ollama extractor with real endpoint
	ollamaExt := extractors.NewOllamaExtractor("http://localhost:11434", "qwen3.5:2b", 120*time.Second)

	// Test cases with English input
	testCases := []struct {
		name     string
		prompt   string
		expected []string // expected entity names
	}{
		{
			name:     "Person and Organization",
			prompt:   "John Smith works at Google as a software engineer, he uses Go and Kubernetes",
			expected: []string{"John Smith", "Google", "Go", "Kubernetes"},
		},
		{
			name:     "Technology Stack",
			prompt:   "Deploy the React application to AWS using Docker and Terraform",
			expected: []string{"React", "AWS", "Docker", "Terraform"},
		},
		{
			name:     "Project Context",
			prompt:   "Sarah from Microsoft is working on the Azure cloud platform in Seattle",
			expected: []string{"Sarah", "Microsoft", "Azure", "Seattle"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obs := &types.Observation{
				ID:       "obs-" + tc.name,
				UserID:   "user1",
				HookType: "user_prompt_submit",
				Data: map[string]interface{}{
					"prompt": tc.prompt,
				},
				Timestamp: time.Now(),
			}

			t.Logf("Input: %s", tc.prompt)

			result, err := ollamaExt.Extract(obs)
			if err != nil {
				t.Fatalf("Extract failed: %v", err)
			}

			t.Logf("Extraction results:")
			t.Logf("  Entities (%d):", len(result.Entities))
			for _, entity := range result.Entities {
				t.Logf("    - %s: %s", entity.Type, entity.Name)
			}
			t.Logf("  Relations (%d):", len(result.Relations))
			for _, rel := range result.Relations {
				t.Logf("    - %s --[%s]--> %s", rel.Source, rel.Type, rel.Target)
			}

			// Verify expected entities
			foundEntities := make(map[string]bool)
			for _, entity := range result.Entities {
				foundEntities[entity.Name] = true
			}

			for _, expected := range tc.expected {
				if !foundEntities[expected] {
					t.Errorf("Expected entity not found: %s", expected)
				} else {
					t.Logf("  ✓ Found: %s", expected)
				}
			}

			// Save to store
			if err := store.SaveExtractionResult(obs.UserID, result); err != nil {
				t.Fatalf("SaveExtractionResult failed: %v", err)
			}
		})
	}

	// Verify graph statistics
	stats, _ := store.GetStats()
	t.Logf("\nGraph statistics:")
	t.Logf("  Nodes: %v", stats["node_count"])
	t.Logf("  Edges: %v", stats["edge_count"])
	t.Logf("  Node types: %v", stats["node_types"])

	// Query and verify
	engine := graph.NewQueryEngine(store)
	results, err := engine.MultiHopQuery("user1", &graph.QueryOptions{
		MaxHops:   2,
		Limit:     50,
		MinScore:  0.1,
		DecayRate: 0.7,
	})
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	t.Logf("\nMulti-hop query results (%d entities):", len(results))
	for i, result := range results {
		if i < 20 {
			t.Logf("  [%d] %s:%s (hops=%d, score=%.2f)",
				i+1, result.Node.Type, result.Node.Name, result.Hops, result.Score)
		}
	}
}

// TestRealOllama_MixedLanguage tests NER+RE with mixed Chinese/English input
func TestRealOllama_MixedLanguage(t *testing.T) {
	if !checkOllamaAvailable(t) {
		return
	}

	store, cleanup := setupRealTestStore(t)
	defer cleanup()

	// Create Ollama extractor with real endpoint
	ollamaExt := extractors.NewOllamaExtractor("http://localhost:11434", "qwen3.5:2b", 120*time.Second)

	// Test cases with mixed language input
	testCases := []struct {
		name     string
		prompt   string
		expected []string // expected entity names
	}{
		{
			name:     "中英混合1",
			prompt:   "王五在 Microsoft 工作，他用 Go 语言开发 Azure 服务",
			expected: []string{"王五", "Microsoft", "Go", "Azure"},
		},
		{
			name:     "中英混合2",
			prompt:   "部署 React 应用到阿里云 ECS，使用 Docker 容器化",
			expected: []string{"React", "阿里云", "Docker"},
		},
		{
			name:     "技术术语",
			prompt:   "优化 PostgreSQL 查询性能，使用 Redis 做缓存，部署到 K8s 集群",
			expected: []string{"PostgreSQL", "Redis", "K8s"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			obs := &types.Observation{
				ID:       "obs-" + tc.name,
				UserID:   "user1",
				HookType: "user_prompt_submit",
				Data: map[string]interface{}{
					"prompt": tc.prompt,
				},
				Timestamp: time.Now(),
			}

			t.Logf("输入: %s", tc.prompt)

			result, err := ollamaExt.Extract(obs)
			if err != nil {
				t.Fatalf("Extract failed: %v", err)
			}

			t.Logf("提取结果:")
			t.Logf("  实体 (%d):", len(result.Entities))
			for _, entity := range result.Entities {
				t.Logf("    - %s: %s", entity.Type, entity.Name)
			}
			t.Logf("  关系 (%d):", len(result.Relations))
			for _, rel := range result.Relations {
				t.Logf("    - %s --[%s]--> %s", rel.Source, rel.Type, rel.Target)
			}

			// Verify expected entities
			foundEntities := make(map[string]bool)
			for _, entity := range result.Entities {
				foundEntities[entity.Name] = true
			}

			for _, expected := range tc.expected {
				if !foundEntities[expected] {
					t.Errorf("未找到期望的实体: %s", expected)
				} else {
					t.Logf("  ✓ 找到: %s", expected)
				}
			}

			// Save to store
			if err := store.SaveExtractionResult(obs.UserID, result); err != nil {
				t.Fatalf("SaveExtractionResult failed: %v", err)
			}
		})
	}

	// Verify graph statistics
	stats, _ := store.GetStats()
	t.Logf("\n图统计:")
	t.Logf("  节点数: %v", stats["node_count"])
	t.Logf("  边数: %v", stats["edge_count"])
	t.Logf("  节点类型: %v", stats["node_types"])

	// Query and verify
	engine := graph.NewQueryEngine(store)
	results, err := engine.MultiHopQuery("user1", &graph.QueryOptions{
		MaxHops:   2,
		Limit:     50,
		MinScore:  0.1,
		DecayRate: 0.7,
	})
	if err != nil {
		t.Fatalf("MultiHopQuery failed: %v", err)
	}

	t.Logf("\n多跳查询结果 (共 %d 个实体):", len(results))
	for i, result := range results {
		if i < 20 {
			t.Logf("  [%d] %s:%s (hops=%d, score=%.2f)",
				i+1, result.Node.Type, result.Node.Name, result.Hops, result.Score)
		}
	}
}

// TestRealOllama_ComplexScenario tests a complex multi-observation scenario
func TestRealOllama_ComplexScenario(t *testing.T) {
	if !checkOllamaAvailable(t) {
		return
	}

	store, cleanup := setupRealTestStore(t)
	defer cleanup()

	// Create extractors
	ruleExt := extractors.NewRuleExtractor()
	ollamaExt := extractors.NewOllamaExtractor("http://localhost:11434", "qwen3.5:2b", 60*time.Second)

	// Simulate a coding session with multiple observations
	observations := []struct {
		hookType string
		data     map[string]interface{}
	}{
		{
			hookType: "user_prompt_submit",
			data: map[string]interface{}{
				"prompt": "帮我用 Go 开发一个 REST API 服务，使用 PostgreSQL 存储数据",
			},
		},
		{
			hookType: "post_tool_use",
			data: map[string]interface{}{
				"tool_name": "bash",
				"tool_input": map[string]interface{}{
					"command": "go mod init myapi && go get github.com/gin-gonic/gin",
				},
			},
		},
		{
			hookType: "post_tool_use",
			data: map[string]interface{}{
				"tool_name": "write",
				"file_path": "/internal/handler/user.go",
			},
		},
		{
			hookType: "user_prompt_submit",
			data: map[string]interface{}{
				"prompt": "添加 Redis 缓存层，优化数据库查询性能",
			},
		},
		{
			hookType: "post_tool_use",
			data: map[string]interface{}{
				"tool_name": "bash",
				"tool_input": map[string]interface{}{
					"command": "docker build -t myapi:latest .",
				},
			},
		},
	}

	t.Log("=== 模拟编码会话 ===")

	for i, obsData := range observations {
		obs := &types.Observation{
			ID:        fmt.Sprintf("obs-%d", i+1),
			UserID:    "user1",
			HookType:  obsData.hookType,
			Data:      obsData.data,
			Timestamp: time.Now(),
		}

		t.Logf("\n--- 观察 %d (%s) ---", i+1, obsData.hookType)

		// Run rule extractor (sync)
		ruleResult, err := ruleExt.Extract(obs)
		if err != nil {
			t.Logf("Rule extractor error: %v", err)
		} else {
			t.Logf("Rule 提取: %d 实体, %d 关系", len(ruleResult.Entities), len(ruleResult.Relations))
			if err := store.SaveExtractionResult(obs.UserID, ruleResult); err != nil {
				t.Fatalf("SaveExtractionResult (rule) failed: %v", err)
			}
		}

		// Run Ollama extractor (async) for text content
		if obsData.hookType == "user_prompt_submit" {
			ollamaResult, err := ollamaExt.Extract(obs)
			if err != nil {
				t.Logf("Ollama extractor error: %v", err)
			} else {
				t.Logf("Ollama 提取: %d 实体, %d 关系", len(ollamaResult.Entities), len(ollamaResult.Relations))
				for _, entity := range ollamaResult.Entities {
					t.Logf("  - %s: %s", entity.Type, entity.Name)
				}
				for _, rel := range ollamaResult.Relations {
					t.Logf("  - %s --[%s]--> %s", rel.Source, rel.Type, rel.Target)
				}
				if err := store.SaveExtractionResult(obs.UserID, ollamaResult); err != nil {
					t.Fatalf("SaveExtractionResult (ollama) failed: %v", err)
				}
			}
		}
	}

	// Verify graph statistics
	stats, _ := store.GetStats()
	t.Logf("\n=== 最终图统计 ===")
	t.Logf("节点数: %v", stats["node_count"])
	t.Logf("边数: %v", stats["edge_count"])
	t.Logf("节点类型: %v", stats["node_types"])

	// Get context
	engine := graph.NewQueryEngine(store)
	context, err := engine.GetContext("user1", 20)
	if err != nil {
		t.Fatalf("GetContext failed: %v", err)
	}

	t.Logf("\n=== 用户上下文 ===")
	t.Logf("User ID: %v", context["user_id"])
	t.Logf("Total entities: %v", context["total"])

	if tools, ok := context["tools"].([]map[string]interface{}); ok {
		t.Logf("Tools (%d):", len(tools))
		for _, tool := range tools {
			t.Logf("  - %s (score: %.2f)", tool["name"], tool["score"])
		}
	}

	if techs, ok := context["techs"].([]map[string]interface{}); ok {
		t.Logf("Techs (%d):", len(techs))
		for _, tech := range techs {
			t.Logf("  - %s (score: %.2f)", tech["name"], tech["score"])
		}
	}

	if files, ok := context["files"].([]map[string]interface{}); ok {
		t.Logf("Files (%d):", len(files))
		for _, file := range files {
			t.Logf("  - %s (score: %.2f)", file["name"], file["score"])
		}
	}
}
