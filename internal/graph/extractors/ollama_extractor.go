package extractors

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/pkg/types"
)

// OllamaExtractor uses Ollama for NER+RE extraction
type OllamaExtractor struct {
	endpoint   string
	model      string
	timeout    time.Duration
	httpClient *http.Client
}

// NewOllamaExtractor creates a new Ollama-based extractor
func NewOllamaExtractor(endpoint, model string, timeout time.Duration) *OllamaExtractor {
	if endpoint == "" {
		endpoint = "http://localhost:11434"
	}
	if model == "" {
		model = "qwen2.5:3b"
	}
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &OllamaExtractor{
		endpoint: endpoint,
		model:    model,
		timeout:  timeout,
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

// Name returns the name of the extractor
func (e *OllamaExtractor) Name() string {
	return "ollama"
}

// IsAsync returns true (Ollama extraction is slow)
func (e *OllamaExtractor) IsAsync() bool {
	return true
}

const nerRePrompt = `从以下文本中提取实体和关系，返回严格的 JSON 格式：

文本：
%s

返回格式：
{
  "entities": [
    {"type": "person|org|loc|tech|concept|product|event|file|lang", "name": "实体名称", "properties": {}}
  ],
  "relations": [
    {"source": "实体名称", "type": "关系类型", "target": "实体名称"}
  ]
}

实体类型说明：
- person: 人名、用户名
- org: 组织、公司、团队
- loc: 地点、位置
- tech: 技术、工具、框架、中间件
- concept: 概念、方法论、模式
- product: 产品、项目、服务
- event: 事件、会议、活动
- file: 文件、代码文件
- lang: 编程语言

关系类型说明：
【人物相关】
- works_at: 人在组织工作 (张三 --works_at--> 阿里巴巴)
- works_on: 从事某项目/领域 (张三 --works_on--> 机器学习)
- knows: 了解/掌握某技术 (张三 --knows--> Go)
- uses: 使用工具/技术 (张三 --uses--> Python)
- created_by: 被某人创建 (server.go --created_by--> 张三)

【技术相关】
- modifies: 修改文件/代码 (user1 --modifies--> server.go)
- depends_on: 依赖关系 (React --depends_on--> Node.js)
- implements: 实现接口/协议 (UserService --implements--> IRepository)
- integrates_with: 集成/对接 (前端 --integrates_with--> REST API)
- deployed_to: 部署到某环境 (myapp --deployed_to--> Kubernetes)

【结构相关】
- part_of: 部分/属于关系 (handler --part_of--> api模块)
- calls: 调用函数/方法 (main() --calls--> initDB())
- imports: 导入/引用 (main.go --imports--> fmt包)

【内容相关】
- mentions: 提及某实体 (对话 --mentions--> PostgreSQL)
- suggests: 建议某方案 (Claude --suggests--> Redis缓存)
- located_in: 位于某地 (阿里巴巴 --located_in--> 杭州)
- related_to: 相关/关联 (性能优化 --related_to--> 数据库索引)

只返回 JSON，不要其他内容。如果无法提取，返回空数组。`

// Extract extracts entities and relations from an observation using Ollama
func (e *OllamaExtractor) Extract(obs *types.Observation) (*graph.ExtractionResult, error) {
	// Combine text from observation
	text := e.combineText(obs)
	if text == "" {
		return &graph.ExtractionResult{
			Entities:  []graph.Entity{},
			Relations: []graph.Relation{},
		}, nil
	}

	// Call Ollama API
	result, err := e.callOllama(text)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}

	return result, nil
}

// combineText extracts text from observation for NER+RE
func (e *OllamaExtractor) combineText(obs *types.Observation) string {
	var texts []string

	// From prompt
	if prompt, ok := obs.Data["prompt"].(string); ok && prompt != "" {
		texts = append(texts, prompt)
	}

	// From tool input
	if toolInput, ok := obs.Data["tool_input"].(map[string]interface{}); ok {
		if cmd, ok := toolInput["command"].(string); ok && cmd != "" {
			texts = append(texts, cmd)
		}
		if input, ok := toolInput["input"].(string); ok && input != "" {
			texts = append(texts, input)
		}
	} else if cmd, ok := obs.Data["tool_input"].(string); ok && cmd != "" {
		texts = append(texts, cmd)
	}

	// From tool result
	if toolResult, ok := obs.Data["tool_result"].(string); ok && toolResult != "" {
		// Truncate long results
		if len(toolResult) > 500 {
			toolResult = toolResult[:500] + "..."
		}
		texts = append(texts, toolResult)
	}

	return strings.Join(texts, "\n")
}

// OllamaRequest represents the request to Ollama API
type OllamaRequest struct {
	Model  string                 `json:"model"`
	Prompt string                 `json:"prompt"`
	Stream bool                   `json:"stream"`
	Format string                 `json:"format,omitempty"`
	Think  *bool                  `json:"think,omitempty"` // Disable thinking mode for faster response
}

// OllamaResponse represents the response from Ollama API
type OllamaResponse struct {
	Response string `json:"response"`
	Done     bool   `json:"done"`
	Error    string `json:"error,omitempty"`
}

// rawExtractionResult is used for flexible parsing of Ollama response
// (properties might be map or array)
type rawExtractionResult struct {
	Entities []struct {
		Type       string      `json:"type"`
		Name       string      `json:"name"`
		Properties interface{} `json:"properties,omitempty"` // Can be map or array
	} `json:"entities"`
	Relations []graph.Relation `json:"relations"`
}

// callOllama calls the Ollama API for NER+RE
func (e *OllamaExtractor) callOllama(text string) (*graph.ExtractionResult, error) {
	prompt := fmt.Sprintf(nerRePrompt, text)

	// Disable thinking mode for faster response (qwen models support this)
	thinkDisabled := false

	reqBody := OllamaRequest{
		Model:  e.model,
		Prompt: prompt,
		Stream: false,
		Format: "json",   // Force JSON output
		Think:  &thinkDisabled, // Disable thinking mode
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := e.endpoint + "/api/generate"
	resp, err := e.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama API returned status %d: %s", resp.StatusCode, string(body))
	}

	var ollamaResp OllamaResponse
	if err := json.Unmarshal(body, &ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if ollamaResp.Error != "" {
		return nil, fmt.Errorf("Ollama error: %s", ollamaResp.Error)
	}

	// Parse the JSON response from Ollama (use flexible parser for properties)
	var rawResult rawExtractionResult
	var parseErr error

	if err := json.Unmarshal([]byte(ollamaResp.Response), &rawResult); err != nil {
		// Try to extract JSON from response (Ollama might add extra text)
		jsonStr := extractJSON(ollamaResp.Response)
		if jsonStr == "" {
			return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
		}
		if err := json.Unmarshal([]byte(jsonStr), &rawResult); err != nil {
			return nil, fmt.Errorf("failed to parse extracted JSON: %w", err)
		}
	}

	// Convert to proper ExtractionResult
	result := graph.ExtractionResult{
		Entities:  make([]graph.Entity, 0, len(rawResult.Entities)),
		Relations: rawResult.Relations,
	}

	for _, raw := range rawResult.Entities {
		entity := graph.Entity{
			Type: raw.Type,
			Name: raw.Name,
		}
		// Handle properties: convert to map if it's a map, ignore if array
		if props, ok := raw.Properties.(map[string]interface{}); ok {
			entity.Properties = props
		}
		// If properties is array or other type, just leave it nil
		result.Entities = append(result.Entities, entity)
	}

	_ = parseErr // avoid unused variable warning

	// Ensure non-nil slices
	if result.Entities == nil {
		result.Entities = []graph.Entity{}
	}
	if result.Relations == nil {
		result.Relations = []graph.Relation{}
	}

	return &result, nil
}

// extractJSON tries to extract JSON from a string
func extractJSON(s string) string {
	// Find first { and last }
	start := strings.Index(s, "{")
	end := strings.LastIndex(s, "}")

	if start == -1 || end == -1 || end <= start {
		return ""
	}

	return s[start : end+1]
}

// CheckHealth checks if Ollama is available
func (e *OllamaExtractor) CheckHealth() error {
	url := e.endpoint + "/api/tags"
	resp, err := e.httpClient.Get(url)
	if err != nil {
		return fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	return nil
}

// GetModelInfo returns information about the model
func (e *OllamaExtractor) GetModelInfo() (map[string]interface{}, error) {
	url := e.endpoint + "/api/tags"
	resp, err := e.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ollama: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	return result, nil
}
