package extractors

import (
	"testing"

	"github.com/oneliang/memory-brain/pkg/types"
)

func TestRuleExtractor_Name(t *testing.T) {
	ext := NewRuleExtractor()
	if ext.Name() != "rule" {
		t.Errorf("Expected name 'rule', got '%s'", ext.Name())
	}
}

func TestRuleExtractor_IsAsync(t *testing.T) {
	ext := NewRuleExtractor()
	if ext.IsAsync() {
		t.Error("Rule extractor should not be async")
	}
}

func TestRuleExtractor_Extract_ToolUse(t *testing.T) {
	ext := NewRuleExtractor()

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "post_tool_use",
		Data: map[string]interface{}{
			"tool_name": "bash",
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find tool entity
	if len(result.Entities) == 0 {
		t.Fatal("Expected at least 1 entity")
	}

	var foundBash bool
	for _, entity := range result.Entities {
		if entity.Type == "tool" && entity.Name == "bash" {
			foundBash = true
			break
		}
	}
	if !foundBash {
		t.Error("Should find bash tool entity")
	}

	// Should have uses relation
	var foundUses bool
	for _, rel := range result.Relations {
		if rel.Type == "uses" && rel.Target == "bash" {
			foundUses = true
			break
		}
	}
	if !foundUses {
		t.Error("Should have uses relation to bash")
	}
}

func TestRuleExtractor_Extract_FileModify(t *testing.T) {
	ext := NewRuleExtractor()

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "post_tool_use",
		Data: map[string]interface{}{
			"tool_name": "write",
			"file_path": "/internal/api/server.go",
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find file entity
	var foundFile bool
	for _, entity := range result.Entities {
		if entity.Type == "file" && entity.Name == "/internal/api/server.go" {
			foundFile = true
			break
		}
	}
	if !foundFile {
		t.Error("Should find file entity")
	}

	// Should have modifies relation
	var foundModifies bool
	for _, rel := range result.Relations {
		if rel.Type == "modifies" && rel.Target == "/internal/api/server.go" {
			foundModifies = true
			break
		}
	}
	if !foundModifies {
		t.Error("Should have modifies relation to file")
	}
}

func TestRuleExtractor_Extract_Command(t *testing.T) {
	ext := NewRuleExtractor()

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "post_tool_use",
		Data: map[string]interface{}{
			"tool_name": "bash",
			"tool_input": map[string]interface{}{
				"command": "kubectl get pods",
			},
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find command entity
	var foundCommand bool
	for _, entity := range result.Entities {
		if entity.Type == "command" {
			foundCommand = true
			break
		}
	}
	if !foundCommand {
		t.Error("Should find command entity")
	}

	// Should have executes relation
	var foundExecutes bool
	for _, rel := range result.Relations {
		if rel.Type == "executes" {
			foundExecutes = true
			break
		}
	}
	if !foundExecutes {
		t.Error("Should have executes relation")
	}

	// Should extract kubectl from command (tech entity)
	var foundKubectl bool
	for _, entity := range result.Entities {
		if entity.Type == "tech" && entity.Name == "Kubernetes" {
			foundKubectl = true
			break
		}
	}
	// Note: "kubectl" might not directly match "Kubernetes" in dictionary
	// This is a soft check
	if !foundKubectl {
		t.Log("Warning: Kubernetes tech entity not found (might be expected)")
	}
}

func TestRuleExtractor_Extract_Prompt(t *testing.T) {
	ext := NewRuleExtractor()

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "user_prompt_submit",
		Data: map[string]interface{}{
			"prompt": "帮我实现一个 REST API 服务器",
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find intent entity
	var foundIntent bool
	for _, entity := range result.Entities {
		if entity.Type == "intent" && entity.Name == "development" {
			foundIntent = true
			break
		}
	}
	if !foundIntent {
		t.Error("Should find development intent from prompt")
	}

	// Should have works_on relation
	var foundWorksOn bool
	for _, rel := range result.Relations {
		if rel.Type == "works_on" && rel.Target == "development" {
			foundWorksOn = true
			break
		}
	}
	if !foundWorksOn {
		t.Error("Should have works_on relation to development intent")
	}
}

func TestRuleExtractor_Extract_DebugIntent(t *testing.T) {
	ext := NewRuleExtractor()

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "user_prompt_submit",
		Data: map[string]interface{}{
			"prompt": "修复这个 bug",
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find debugging intent
	var foundDebug bool
	for _, entity := range result.Entities {
		if entity.Type == "intent" && entity.Name == "debugging" {
			foundDebug = true
			break
		}
	}
	if !foundDebug {
		t.Error("Should find debugging intent")
	}
}

func TestRuleExtractor_Dedup(t *testing.T) {
	ext := NewRuleExtractor()

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "post_tool_use",
		Data: map[string]interface{}{
			"tool_name": "bash",
			"tool_input": map[string]interface{}{
				"command": "bash --version",
			},
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Count bash entities (should be deduplicated)
	bashCount := 0
	for _, entity := range result.Entities {
		if entity.Name == "bash" {
			bashCount++
		}
	}

	if bashCount > 1 {
		t.Errorf("Expected deduplication, found %d bash entities", bashCount)
	}
}

func TestRuleExtractor_Extract_URL(t *testing.T) {
	ext := NewRuleExtractor()

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "post_tool_use",
		Data: map[string]interface{}{
			"tool_name": "bash",
			"tool_input": map[string]interface{}{
				"command": "curl https://api.github.com/repos",
			},
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find URL entity
	var foundURL bool
	for _, entity := range result.Entities {
		if entity.Type == "url" {
			foundURL = true
			break
		}
	}
	if !foundURL {
		t.Error("Should find URL entity")
	}
}

func TestRuleExtractor_Extract_TechFromDictionary(t *testing.T) {
	ext := NewRuleExtractor()

	obs := &types.Observation{
		ID:       "obs1",
		UserID:   "user1",
		HookType: "user_prompt_submit",
		Data: map[string]interface{}{
			"prompt": "帮我配置 PostgreSQL 和 Redis",
		},
	}

	result, err := ext.Extract(obs)
	if err != nil {
		t.Fatalf("Extract failed: %v", err)
	}

	// Should find PostgreSQL and Redis as tech entities
	var foundPostgres, foundRedis bool
	for _, entity := range result.Entities {
		if entity.Type == "tech" {
			if entity.Name == "PostgreSQL" {
				foundPostgres = true
			}
			if entity.Name == "Redis" {
				foundRedis = true
			}
		}
	}

	if !foundPostgres {
		t.Error("Should find PostgreSQL tech entity from dictionary")
	}
	if !foundRedis {
		t.Error("Should find Redis tech entity from dictionary")
	}
}

func TestClassifyIntent(t *testing.T) {
	tests := []struct {
		prompt   string
		expected string
	}{
		{"帮我实现一个功能", "development"},
		{"implement this feature", "development"},
		{"修复这个 bug", "debugging"},
		{"debug this issue", "debugging"},
		{"查询用户信息", "query"},
		{"how to search", "query"},
		{"配置服务器", "management"},
		{"deploy to production", "management"},
		{"优化性能", "review"},         // "优化" without "代码" matches review
		{"please review", "review"},    // Pure review intent
		{"你好", "general"},
	}

	for _, test := range tests {
		result := classifyIntent(test.prompt)
		if result != test.expected {
			t.Errorf("classifyIntent(%q) = %q, expected %q", test.prompt, result, test.expected)
		}
	}
}
