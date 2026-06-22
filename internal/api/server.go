package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/oneliang/memory-brain/internal/config"
	"github.com/oneliang/memory-brain/internal/decay"
	"github.com/oneliang/memory-brain/internal/dedup"
	"github.com/oneliang/memory-brain/internal/graph"
	"github.com/oneliang/memory-brain/internal/profile"
	"github.com/oneliang/memory-brain/internal/privacy"
	"github.com/oneliang/memory-brain/internal/search"
	"github.com/oneliang/memory-brain/internal/summary"
	"github.com/oneliang/memory-brain/internal/storage"
	"github.com/oneliang/memory-brain/pkg/types"
)

// Server represents the Memory Brain REST API server
type Server struct {
	port    int
	server  *http.Server
	handler *Handler
}

// Handler handles all API requests
type Handler struct {
	storage               *storage.Storage
	searchEngines         map[string]*search.HybridSearchEngine // per-user search engines
	searchMu              sync.RWMutex                          // protects searchEngines
	analyzer              *profile.Analyzer
	projectAnalyzer       *profile.ProjectAnalyzer
	generator             *summary.Generator
	sessionDedupManager   *dedup.SessionCacheManager            // session-level dedup (immediate)
	userDedupCaches       map[string]*dedup.HashCache           // user-level dedup (for storage)
	dedupMu               sync.RWMutex                          // protects userDedupCaches
	graphHandler          *GraphHandler                         // graph module handler
	injectGraphToProfile  bool                                  // inject graph context to profile
}

// NewServer creates a new API server
func NewServer(port int, cfg *config.Config) *Server {
	// Initialize storage
	store := storage.NewStorage("")

	// Initialize graph handler with config
	var graphConfig *graph.GraphConfig
	if cfg != nil && cfg.Graph.Enabled {
		graphConfig = &graph.GraphConfig{
			Enabled: cfg.Graph.Enabled,
			Ollama: graph.OllamaConfig{
				Endpoint: cfg.Graph.Ollama.Endpoint,
				Model:    cfg.Graph.Ollama.Model,
				Timeout:  cfg.Graph.Ollama.Timeout,
			},
			Query: graph.QueryConfig{
				MaxHops:      cfg.Graph.Query.MaxHops,
				DefaultLimit: cfg.Graph.Query.DefaultLimit,
			},
		}
	} else {
		// Default config if not provided
		graphConfig = &graph.GraphConfig{
			Enabled: false,
		}
	}
	graphHandler := NewGraphHandler(graphConfig)
	graphHandler.RegisterExtractors()

	return &Server{
		port: port,
		handler: &Handler{
			storage:              store,
			searchEngines:        make(map[string]*search.HybridSearchEngine),
			analyzer:             profile.NewAnalyzer(),
			projectAnalyzer:      profile.NewProjectAnalyzer(),
			generator:            summary.NewGenerator(),
			sessionDedupManager:  dedup.NewSessionCacheManager(dedup.DedupWindow),
			userDedupCaches:      make(map[string]*dedup.HashCache),
			graphHandler:         graphHandler,
			injectGraphToProfile: cfg != nil && cfg.Graph.InjectToProfile,
		},
	}
}

// getSearchEngine returns user-specific search engine
func (h *Handler) getSearchEngine(userID string) *search.HybridSearchEngine {
	h.searchMu.RLock()
	engine, exists := h.searchEngines[userID]
	h.searchMu.RUnlock()

	if exists {
		return engine
	}

	h.searchMu.Lock()
	defer h.searchMu.Unlock()
	// Double-check after acquiring write lock
	if engine, exists = h.searchEngines[userID]; exists {
		return engine
	}

	// Create new search engine for user
	vectorPath := search.DefaultVectorPath(userID)
	newEngine, err := search.NewHybridSearchEngine(vectorPath)
	if err != nil {
		log.Printf("Warning: vector search disabled for user %s: %v", userID, err)
		return nil
	}
	h.searchEngines[userID] = newEngine
	return newEngine
}

// getUserDedupCache returns user-specific dedup cache (for storage-level dedup)
func (h *Handler) getUserDedupCache(userID string) *dedup.HashCache {
	h.dedupMu.RLock()
	cache, exists := h.userDedupCaches[userID]
	h.dedupMu.RUnlock()

	if exists {
		return cache
	}

	h.dedupMu.Lock()
	// Double-check after acquiring write lock
	if cache, exists = h.userDedupCaches[userID]; !exists {
		cache = dedup.NewHashCache(dedup.DedupWindow)
		h.userDedupCaches[userID] = cache
	}
	h.dedupMu.Unlock()

	return cache
}

// Start starts the HTTP server
func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()

	// Register routes
	mux.HandleFunc("/memory/observe", s.handler.handleObserve)
	mux.HandleFunc("/memory/profile", s.handler.handleProfile)
	mux.HandleFunc("/memory/search", s.handler.handleSearch)
	mux.HandleFunc("/memory/profile/update", s.handler.handleProfileUpdate)
	mux.HandleFunc("/memory/session/summary", s.handler.handleSessionSummary)
	mux.HandleFunc("/memory/session/profile", s.handler.handleSessionProfile)
	mux.HandleFunc("/memory/project/profile", s.handler.handleProjectProfile)
	mux.HandleFunc("/memory/project/analyze", s.handler.handleProjectAnalyze)
	mux.HandleFunc("/memory/health", s.handler.handleHealth)

	// Register graph routes
	mux.HandleFunc("/memory/graph/extract", s.handler.graphHandler.HandleExtract)
	mux.HandleFunc("/memory/graph/query", s.handler.graphHandler.HandleQuery)
	mux.HandleFunc("/memory/graph/context", s.handler.graphHandler.HandleContext)
	mux.HandleFunc("/memory/graph/stats", s.handler.graphHandler.HandleStats)

	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second, // Increased for graph operations
	}

	// Start server in goroutine
	go func() {
		log.Printf("Server listening on %s", s.server.Addr)
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	// Wait for context cancellation
	<-ctx.Done()

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return s.server.Shutdown(shutdownCtx)
}

// handleObserve handles POST /memory/observe
func (h *Handler) handleObserve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var obs types.Observation
	if err := json.NewDecoder(r.Body).Decode(&obs); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// 1. Session-level deduplication check (immediate, session-focused)
	hash := dedup.ComputeHash(obs.SessionID, obs.HookType, obs.Data)
	if h.sessionDedupManager.CheckAndAdd(obs.SessionID, hash) {
		writeSuccess(w, map[string]interface{}{
			"obs_id": obs.ID,
			"dedup":  true,
		})
		return
	}
	obs.Hash = hash

	// 2. Privacy filtering
	obs.Data = privacy.StripPrivateDataFromMap(obs.Data)

	// 3. Process observation based on HookType
	h.processObservation(&obs)

	// 4. Graph extraction (async, non-blocking)
	if h.graphHandler != nil && obs.UserID != "" {
		go func() {
			// Get or create store
			store, err := h.graphHandler.GetStore(obs.UserID)
			if err != nil {
				log.Printf("Graph: failed to get store for user %s: %v", obs.UserID, err)
				return
			}

			// Run synchronous extractors
			result, err := h.graphHandler.registry.ExtractAll(&obs)
			if err != nil {
				log.Printf("Graph: extraction failed: %v", err)
				return
			}

			if result != nil && (len(result.Entities) > 0 || len(result.Relations) > 0) {
				if err := store.SaveExtractionResult(obs.UserID, result); err != nil {
					log.Printf("Graph: failed to save result: %v", err)
				}
			}

			// Run async extractors (Ollama)
			for _, ext := range h.graphHandler.registry.GetAsync() {
				asyncResult, err := ext.Extract(&obs)
				if err != nil {
					log.Printf("Graph: async extractor %s failed: %v", ext.Name(), err)
					continue
				}
				if asyncResult != nil && (len(asyncResult.Entities) > 0 || len(asyncResult.Relations) > 0) {
					if err := store.SaveExtractionResult(obs.UserID, asyncResult); err != nil {
						log.Printf("Graph: failed to save async result: %v", err)
					}
				}
			}
		}()
	}

	writeSuccess(w, map[string]interface{}{
		"obs_id": obs.ID,
		"dedup":  false,
	})
}

// processObservation handles observation storage based on HookType
func (h *Handler) processObservation(obs *types.Observation) {
	switch obs.HookType {
	case "post_tool_use":
		// Extract tool info and update patterns
		toolName, ok := obs.Data["tool_name"].(string)
		if !ok {
			return
		}

		// Pattern dedup: read existing patterns, merge if exists
		patterns, err := h.storage.ReadPatterns(obs.UserID)
		if err != nil {
			log.Printf("Failed to read patterns for dedup: %v", err)
			patterns = []types.PatternCard{}
		}

		// Find matching pattern (same type + same pattern string)
		var existingPattern *types.PatternCard
		for i := range patterns {
			if patterns[i].Type == "tool_usage" && patterns[i].Pattern == toolName {
				existingPattern = &patterns[i]
				break
			}
		}

		now := time.Now()
		if existingPattern != nil {
			// Update existing pattern: increment frequency, update last seen
			existingPattern.Frequency++
			existingPattern.LastSeen = now
			// Write back all patterns (merged)
			if err := h.storage.WritePatterns(obs.UserID, patterns); err != nil {
				log.Printf("Failed to write merged patterns: %v", err)
			}
		} else {
			// Create new PatternCard with unique ID (include toolName for uniqueness)
			pattern := &types.PatternCard{
				ID:         fmt.Sprintf("pattern_%s_%s_%d", obs.UserID, toolName, now.UnixNano()),
				UserID:     obs.UserID,
				Type:       "tool_usage",
				Pattern:    toolName,
				Frequency:  1,
				Confidence: 1.0,
				LastSeen:   now,
				CreatedAt:  now,
			}

			if err := h.storage.AppendPattern(obs.UserID, pattern); err != nil {
				log.Printf("Failed to append pattern: %v", err)
			}
		}

		// Write to session-level storage: append the new pattern
		if obs.SessionID != "" {
			// Find the tool pattern we just created/updated
			userPatterns, _ := h.storage.ReadPatterns(obs.UserID)
			for i := len(userPatterns) - 1; i >= 0; i-- {
				if userPatterns[i].Type == "tool_usage" && userPatterns[i].Pattern == toolName {
					h.storage.AppendSessionPattern(obs.UserID, obs.SessionID, &userPatterns[i])
					break
				}
			}
		}

	case "user_prompt_submit":
		// Extract user prompt for intent analysis
		prompt, ok := obs.Data["prompt"].(string)
		if !ok || prompt == "" {
			return
		}

		// Classify intent from user's original message
		intent := classifyIntent(prompt)
		now := time.Now()

		// Create or update intent profile card
		profiles, err := h.storage.ReadProfiles(obs.UserID)
		if err != nil {
			log.Printf("Failed to read profiles for intent: %v", err)
			profiles = []types.ProfileCard{}
		}

		// Find existing intent profile
		var existingIntent *types.ProfileCard
		for i := range profiles {
			if profiles[i].Type == "IntentCard" {
				existingIntent = &profiles[i]
				break
			}
		}

		if existingIntent != nil {
			// Update intent frequency
			intentFreq, ok := existingIntent.Content["intent_freq"].(map[string]interface{})
			if !ok {
				intentFreq = make(map[string]interface{})
			}
			if freq, exists := intentFreq[intent]; exists {
				// JSON numbers are parsed as float64
				switch v := freq.(type) {
				case int:
					intentFreq[intent] = v + 1
				case float64:
					intentFreq[intent] = int(v) + 1
				default:
					intentFreq[intent] = 1
				}
			} else {
				intentFreq[intent] = 1
			}
			existingIntent.Content["intent_freq"] = intentFreq
			existingIntent.Content["last_prompt"] = prompt
			existingIntent.Metadata.Timestamp = now

			if err := h.storage.WriteProfiles(obs.UserID, profiles); err != nil {
				log.Printf("Failed to write intent profile: %v", err)
			}

			// Write to session-level storage: replace the IntentCard (singleton)
			if obs.SessionID != "" {
				// Read existing session profiles
				sessionProfiles, _ := h.storage.ReadSessionProfiles(obs.UserID, obs.SessionID)
				if sessionProfiles == nil {
					sessionProfiles = []types.ProfileCard{}
				}

				// Find and update or add the IntentCard
				found := false
				for i := range sessionProfiles {
					if sessionProfiles[i].Type == "IntentCard" {
						sessionProfiles[i] = *existingIntent
						found = true
						break
					}
				}
				if !found {
					sessionProfiles = append(sessionProfiles, *existingIntent)
				}

				h.storage.WriteSessionProfiles(obs.UserID, obs.SessionID, sessionProfiles)
			}

			// Write to project-level storage if Directory is provided
			if obs.Directory != "" {
				// Read existing project profiles
				projectProfiles, _ := h.storage.ReadProjectProfiles(obs.Directory)
				if projectProfiles == nil {
					projectProfiles = []types.ProfileCard{}
				}

				// Find and update or add the IntentCard
				found := false
				for i := range projectProfiles {
					if projectProfiles[i].Type == "IntentCard" {
						projectProfiles[i] = *existingIntent
						found = true
						break
					}
				}
				if !found {
					projectProfiles = append(projectProfiles, *existingIntent)
				}

				h.storage.WriteProjectProfiles(obs.Directory, projectProfiles)
			}
		} else {
			// Create new intent profile card
			intentCard := &types.ProfileCard{
				ID:      fmt.Sprintf("intent_%s_%d", obs.UserID, now.UnixNano()),
				Type:    "IntentCard",
				Content: map[string]interface{}{
					"intent_freq":   map[string]interface{}{intent: 1},
					"primary_intent": intent,
					"last_prompt":    prompt,
				},
				Metadata: types.CardMetadata{
					UserID:     obs.UserID,
					SessionID:  obs.SessionID,
					Timestamp:  now,
					Importance: 0.8,
					Strength:   1.0,
					DecayDays:  7,
				},
			}

			if err := h.storage.AppendProfile(obs.UserID, intentCard); err != nil {
				log.Printf("Failed to append intent profile: %v", err)
			}

			// Write to session-level storage
			if obs.SessionID != "" {
				h.storage.AppendSessionProfile(obs.UserID, obs.SessionID, intentCard)
			}

			// Write to project-level storage if Directory is provided
			if obs.Directory != "" {
				h.storage.AppendProjectProfile(obs.Directory, intentCard)
			}
		}

		// Also store prompt as interaction pattern for session summary
		promptPattern := &types.PatternCard{
			ID:         fmt.Sprintf("prompt_%s_%d", obs.UserID, now.UnixNano()),
			UserID:     obs.UserID,
			Type:       "user_prompt",
			Pattern:    prompt,
			Frequency:  1,
			Confidence: 1.0,
			LastSeen:   now,
			CreatedAt:  now,
		}
		if err := h.storage.AppendPattern(obs.UserID, promptPattern); err != nil {
			log.Printf("Failed to append prompt pattern: %v", err)
		}

		// Write prompt pattern to session-level storage
		if obs.SessionID != "" {
			h.storage.AppendSessionPattern(obs.UserID, obs.SessionID, promptPattern)
		}

	default:
		// Other hook types - future implementation
		log.Printf("Observation received: hook=%s, session=%s", obs.HookType, obs.SessionID)
	}
}

// classifyIntent classifies user intent from prompt text
func classifyIntent(prompt string) string {
	promptLower := prompt

	// Development keywords
	if containsAny(promptLower, []string{"实现", "开发", "代码", "编写", "创建", "添加", "修改", "implement", "code", "create", "add", "modify", "build"}) {
		return "development"
	}

	// Debug keywords
	if containsAny(promptLower, []string{"调试", "修复", "错误", "bug", "问题", "debug", "fix", "error", "issue", "解决"}) {
		return "debugging"
	}

	// Query keywords
	if containsAny(promptLower, []string{"查询", "搜索", "查找", "是什么", "怎么", "如何", "query", "search", "find", "what", "how", "explain", "解释"}) {
		return "query"
	}

	// Management keywords
	if containsAny(promptLower, []string{"管理", "配置", "部署", "设置", "manage", "config", "deploy", "setup", "install"}) {
		return "management"
	}

	// Review keywords
	if containsAny(promptLower, []string{"检查", "审查", "优化", "重构", "review", "optimize", "refactor", "测试", "test"}) {
		return "review"
	}

	return "general"
}

// containsAny checks if string contains any of the keywords
func containsAny(s string, keywords []string) bool {
	for _, kw := range keywords {
		if len(s) >= len(kw) {
			for i := 0; i <= len(s)-len(kw); i++ {
				if s[i:i+len(kw)] == kw {
					return true
				}
			}
		}
	}
	return false
}

// handleProfile handles GET /memory/profile
func (h *Handler) handleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		writeError(w, "Missing user_id parameter", http.StatusBadRequest)
		return
	}

	inject := r.URL.Query().Get("inject") == "true"

	// 1. Load profiles and patterns
	profiles, err := h.storage.ReadProfiles(userID)
	if err != nil {
		log.Printf("Failed to read profiles: %v", err)
		profiles = []types.ProfileCard{}
	}

	patterns, err := h.storage.ReadPatterns(userID)
	if err != nil {
		log.Printf("Failed to read patterns: %v", err)
		patterns = []types.PatternCard{}
	}

	// 2. Apply decay
	profiles = decay.ApplyDecay(profiles, decay.DefaultDecayDays)
	patterns = decay.ApplyDecayToPatterns(patterns, decay.DefaultDecayDays)

	// 3. Build response
	resp := types.ProfileResponse{
		ProfileSummary: map[string]interface{}{
			"user_id":      userID,
			"profile_count": len(profiles),
			"pattern_count": len(patterns),
		},
	}

	// 4. Generate system message if inject=true
	if inject {
		var messages []string

		// Profile message
		if len(profiles) > 0 || len(patterns) > 0 {
			if msg := h.buildProfileMessage(profiles, patterns); msg != "" {
				messages = append(messages, msg)
			}
		}

		// Graph context message
		if h.injectGraphToProfile && h.graphHandler != nil {
			if msg := h.buildGraphContextMessage(userID); msg != "" {
				messages = append(messages, msg)
			}
		}

		if len(messages) > 0 {
			resp.SystemMessage = strings.Join(messages, "\n\n")
		}
	}

	writeSuccess(w, resp)
}

// buildProfileMessage generates a system message from profiles and patterns
func (h *Handler) buildProfileMessage(profiles []types.ProfileCard, patterns []types.PatternCard) string {
	var parts []string

	// Summarize patterns (tool preferences)
	toolFreq := make(map[string]int)
	for _, p := range patterns {
		if p.Type == "tool_usage" {
			toolFreq[p.Pattern] += p.Frequency
		}
	}

	if len(toolFreq) > 0 {
		var toolList []string
		for tool, freq := range toolFreq {
			toolList = append(toolList, fmt.Sprintf("%s(%d)", tool, freq))
		}
		parts = append(parts, "工具偏好: "+strings.Join(toolList, ", "))
	}

	// Summarize profiles
	for _, card := range profiles {
		if card.Type == "PreferenceCard" {
			if pref, ok := card.Content["preference"].(string); ok {
				parts = append(parts, pref)
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return "用户画像摘要:\n- " + strings.Join(parts, "\n- ")
}

// buildPathText converts a path of node IDs to readable text with relations
// Example: [user, bash, kubectl] -> "user → bash (uses) → kubectl (executes)"
func buildPathText(path []int64, store *graph.Store) string {
	if len(path) == 0 {
		return ""
	}

	var parts []string
	for i := 0; i < len(path)-1; i++ {
		// Get current node
		node, err := store.GetNode(path[i])
		if err != nil || node == nil {
			continue
		}

		// Get edges to find relation to next node
		edges, err := store.GetOutgoingEdges(path[i])
		if err != nil {
			parts = append(parts, node.Name)
			continue
		}

		// Find edge to next node
		relation := ""
		for _, edge := range edges {
			if edge.TargetID == path[i+1] {
				relation = edge.Relation
				break
			}
		}

		if relation != "" {
			parts = append(parts, fmt.Sprintf("%s (%s)", node.Name, relation))
		} else {
			parts = append(parts, node.Name)
		}
	}

	// Add last node
	if len(path) > 0 {
		lastNode, err := store.GetNode(path[len(path)-1])
		if err == nil && lastNode != nil {
			parts = append(parts, lastNode.Name)
		}
	}

	return strings.Join(parts, " → ")
}

// typeLabel returns a human-readable label for entity types
func typeLabel(typ string) string {
	labels := map[string]string{
		"user":    "用户",
		"tool":    "工具",
		"tech":    "技术",
		"file":    "文件",
		"command": "命令",
		"person":  "人物",
		"org":     "组织",
		"loc":     "地点",
		"intent":  "意图",
		"lang":    "语言",
		"concept": "概念",
		"product": "产品",
	}
	if label, ok := labels[typ]; ok {
		return label
	}
	return typ
}

// buildGraphContextMessage generates context message from graph data with path format
func (h *Handler) buildGraphContextMessage(userID string) string {
	if h.graphHandler == nil {
		return ""
	}

	// Get graph store
	store, err := h.graphHandler.GetStore(userID)
	if err != nil {
		log.Printf("Failed to get graph store for user %s: %v", userID, err)
		return ""
	}

	// Execute multi-hop query to get paths
	engine := graph.NewQueryEngine(store)
	opts := &graph.QueryOptions{
		MaxHops:   2,
		Limit:     20,
		MinScore:  0.1,
		DecayRate: 0.7,
	}

	results, err := engine.MultiHopQuery(userID, opts)
	if err != nil {
		log.Printf("Failed to execute multi-hop query for user %s: %v", userID, err)
		return ""
	}

	if len(results) == 0 {
		return ""
	}

	// Build path texts
	var paths []string
	seen := make(map[string]bool)
	for _, result := range results {
		if len(result.Path) > 1 {
			pathText := buildPathText(result.Path, store)
			if pathText != "" && !seen[pathText] {
				paths = append(paths, pathText)
				seen[pathText] = true
				if len(paths) >= 5 { // Limit to 5 paths
					break
				}
			}
		}
	}

	// Build entity type statistics
	typeCount := make(map[string][]string)
	seenEntities := make(map[string]bool)
	for _, result := range results {
		entityKey := result.Node.Type + ":" + result.Node.Name
		if !seenEntities[entityKey] {
			typeCount[result.Node.Type] = append(typeCount[result.Node.Type], result.Node.Name)
			seenEntities[entityKey] = true
		}
	}

	// Format output
	var parts []string

	// Paths section
	if len(paths) > 0 {
		parts = append(parts, "用户知识图谱:")
		for i, path := range paths {
			parts = append(parts, fmt.Sprintf("路径%d: %s", i+1, path))
		}
	}

	// Entity statistics section
	if len(typeCount) > 0 {
		parts = append(parts, "\n实体统计:")
		// Order types for consistent output
		typeOrder := []string{"tool", "tech", "file", "command", "person", "org", "loc", "intent", "lang", "concept", "product"}
		for _, typ := range typeOrder {
			if names, ok := typeCount[typ]; ok && len(names) > 0 {
				// Limit to 5 entities per type
				displayNames := names
				if len(names) > 5 {
					displayNames = names[:5]
				}
				parts = append(parts, fmt.Sprintf("- %s: %s", typeLabel(typ), strings.Join(displayNames, ", ")))
			}
		}
		// Add any remaining types not in the order list
		for typ, names := range typeCount {
			found := false
			for _, t := range typeOrder {
				if t == typ {
					found = true
					break
				}
			}
			if !found && len(names) > 0 {
				displayNames := names
				if len(names) > 5 {
					displayNames = names[:5]
				}
				parts = append(parts, fmt.Sprintf("- %s: %s", typeLabel(typ), strings.Join(displayNames, ", ")))
			}
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, "\n")
}

// handleSearch handles GET /memory/search
func (h *Handler) handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	query := r.URL.Query().Get("query")
	userID := r.URL.Query().Get("user_id")
	if query == "" || userID == "" {
		writeError(w, "Missing query or user_id parameter", http.StatusBadRequest)
		return
	}

	limit := 5
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
			limit = parsed
		}
	}

	// Get user-specific search engine
	searchEngine := h.getSearchEngine(userID)
	if searchEngine == nil {
		writeError(w, "Search engine not available", http.StatusInternalServerError)
		return
	}

	// Perform hybrid search
	ctx := r.Context()
	results, err := searchEngine.Search(ctx, query, limit)
	if err != nil {
		writeError(w, "Search failed", http.StatusInternalServerError)
		return
	}

	resp := types.SearchResponse{
		Results: results,
	}

	writeSuccess(w, resp)
}

// handleProfileUpdate handles PUT /memory/profile/update
func (h *Handler) handleProfileUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		UserID       string              `json:"user_id"`
		SessionID    string              `json:"session_id"`
		Observations []types.Observation `json:"observations,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		writeError(w, "Missing user_id", http.StatusBadRequest)
		return
	}

	// 1. Apply decay to existing profiles and patterns
	existingProfiles, err := h.storage.ReadProfiles(req.UserID)
	if err != nil {
		log.Printf("Failed to read profiles for update: %v", err)
	}

	existingPatterns, err := h.storage.ReadPatterns(req.UserID)
	if err != nil {
		log.Printf("Failed to read patterns for update: %v", err)
	}

	// 2. Analyze observations if provided
	var newProfiles []types.ProfileCard
	var newPatterns []types.PatternCard

	if len(req.Observations) > 0 {
		// Extract tool preference
		if prefCard := h.analyzer.AnalyzeToolPreference(req.Observations); prefCard != nil {
			prefCard.Metadata.UserID = req.UserID
			prefCard.Metadata.SessionID = req.SessionID
			newProfiles = append(newProfiles, *prefCard)
		}

		// Extract work patterns
		if patternCard := h.analyzer.AnalyzeWorkPatterns(req.Observations); patternCard != nil {
			patternCard.UserID = req.UserID
			newPatterns = append(newPatterns, *patternCard)
		}

		// Extract intent preference
		if intentCard := h.analyzer.AnalyzeIntent(req.Observations); intentCard != nil {
			intentCard.Metadata.UserID = req.UserID
			intentCard.Metadata.SessionID = req.SessionID
			newProfiles = append(newProfiles, *intentCard)
		}

		// Save new profiles
		for _, card := range newProfiles {
			if err := h.storage.AppendProfile(req.UserID, &card); err != nil {
				log.Printf("Failed to append profile: %v", err)
			}
		}

		// Save new patterns
		for _, pattern := range newPatterns {
			if err := h.storage.AppendPattern(req.UserID, &pattern); err != nil {
				log.Printf("Failed to append pattern: %v", err)
			}
		}
	}

	// 3. Apply decay to all profiles and patterns
	allProfiles := append(existingProfiles, newProfiles...)
	allPatterns := append(existingPatterns, newPatterns...)

	remainingProfiles := decay.ApplyDecay(allProfiles, decay.DefaultDecayDays)
	remainingPatterns := decay.ApplyDecayToPatterns(allPatterns, decay.DefaultDecayDays)

	// 4. Persist decayed profiles and patterns
	if err := h.storage.WriteProfiles(req.UserID, remainingProfiles); err != nil {
		log.Printf("Failed to write decayed profiles: %v", err)
	}
	if err := h.storage.WritePatterns(req.UserID, remainingPatterns); err != nil {
		log.Printf("Failed to write decayed patterns: %v", err)
	}

	writeSuccess(w, map[string]interface{}{
		"updated":           true,
		"profile_count":     len(remainingProfiles),
		"pattern_count":     len(remainingPatterns),
		"new_profiles":      len(newProfiles),
		"new_patterns":      len(newPatterns),
		"decayed_profiles":  len(allProfiles) - len(remainingProfiles),
		"decayed_patterns":  len(allPatterns) - len(remainingPatterns),
	})
}

// handleSessionSummary handles POST /memory/session/summary
func (h *Handler) handleSessionSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Summary     *types.SessionSummary  `json:"summary,omitempty"`
		Observations []types.Observation   `json:"observations,omitempty"`
		UserID      string                 `json:"user_id"`
		SessionID   string                 `json:"session_id"`
		Generate    bool                   `json:"generate"` // If true, generate from observations
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	var summary *types.SessionSummary

	// Generate summary from observations if requested
	if req.Generate && len(req.Observations) > 0 {
		summary = h.generator.BuildSyntheticSummary(req.Observations)
		if summary != nil {
			// Override user_id and session_id if provided
			if req.UserID != "" {
				summary.UserID = req.UserID
			}
			if req.SessionID != "" {
				summary.SessionID = req.SessionID
			}
		}
	} else if req.Summary != nil {
		// Use provided summary
		summary = req.Summary
	} else {
		writeError(w, "Missing summary or observations", http.StatusBadRequest)
		return
	}

	if summary == nil {
		writeError(w, "Failed to generate summary", http.StatusInternalServerError)
		return
	}

	// Save session summary to storage
	if err := h.storage.SaveSessionSummary(summary.UserID, summary.SessionID, summary); err != nil {
		log.Printf("Failed to save session summary: %v", err)
		writeError(w, "Failed to save summary", http.StatusInternalServerError)
		return
	}

	// Index summary for search (add to HybridSearchEngine)
	searchEngine := h.getSearchEngine(summary.UserID)
	if searchEngine != nil {
		doc := &search.IndexDoc{
			ID:        summary.ID,
			Title:     summary.Title,
			Content:   summary.Narrative,
			SessionID: summary.SessionID,
			Timestamp: summary.CreatedAt.Unix(),
		}
		searchEngine.AddDocument(r.Context(), doc)
	}

	writeSuccess(w, map[string]interface{}{
		"summary_id": summary.ID,
		"saved":      true,
		"generated":  req.Generate,
		"title":      summary.Title,
	})
}

// handleHealth handles GET /memory/health
func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	writeSuccess(w, map[string]interface{}{
		"status":  "healthy",
		"version": "0.1.0",
	})
}

// handleSessionProfile handles GET /memory/session/profile
func (h *Handler) handleSessionProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	userID := r.URL.Query().Get("user_id")
	sessionID := r.URL.Query().Get("session_id")
	if userID == "" || sessionID == "" {
		writeError(w, "Missing user_id or session_id parameter", http.StatusBadRequest)
		return
	}

	inject := r.URL.Query().Get("inject") == "true"

	// 1. Load session profiles and patterns
	profiles, err := h.storage.ReadSessionProfiles(userID, sessionID)
	if err != nil {
		log.Printf("Failed to read session profiles: %v", err)
		profiles = []types.ProfileCard{}
	}

	patterns, err := h.storage.ReadSessionPatterns(userID, sessionID)
	if err != nil {
		log.Printf("Failed to read session patterns: %v", err)
		patterns = []types.PatternCard{}
	}

	// 2. Build response
	resp := map[string]interface{}{
		"user_id":   userID,
		"session_id": sessionID,
		"profile_count": len(profiles),
		"pattern_count": len(patterns),
		"profiles":  profiles,
		"patterns":  patterns,
	}

	// 3. Generate system message if inject=true
	if inject && (len(profiles) > 0 || len(patterns) > 0) {
		resp["systemMessage"] = h.buildSessionProfileMessage(profiles, patterns)
	}

	writeSuccess(w, resp)
}

// handleProjectProfile handles GET /memory/project/profile
func (h *Handler) handleProjectProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	directory := r.URL.Query().Get("directory")
	if directory == "" {
		writeError(w, "Missing directory parameter", http.StatusBadRequest)
		return
	}

	inject := r.URL.Query().Get("inject") == "true"

	// 1. Load project profiles
	profiles, err := h.storage.ReadProjectProfiles(directory)
	if err != nil {
		log.Printf("Failed to read project profiles: %v", err)
		profiles = []types.ProfileCard{}
	}

	// 2. Load project metadata
	meta, err := h.storage.ReadProjectMeta(directory)
	if err != nil {
		log.Printf("Failed to read project meta: %v", err)
		meta = nil
	}

	// 3. Build response
	resp := map[string]interface{}{
		"directory":     directory,
		"profile_count": len(profiles),
		"profiles":      profiles,
	}

	if meta != nil {
		resp["summary"] = meta.Summary
		resp["category"] = meta.Category
		resp["stats"] = meta.Stats
		resp["highlights"] = meta.Highlights
		resp["last_accessed"] = meta.LastAccessed
	}

	// 4. Generate system message if inject=true
	if inject && len(profiles) > 0 {
		resp["systemMessage"] = h.buildProjectProfileMessage(profiles, meta)
	}

	writeSuccess(w, resp)
}

// handleProjectAnalyze handles POST /memory/project/analyze
func (h *Handler) handleProjectAnalyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Directory string `json:"directory"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Directory == "" {
		writeError(w, "Missing directory parameter", http.StatusBadRequest)
		return
	}

	// Analyze project
	profile, err := h.projectAnalyzer.AnalyzeProject(req.Directory)
	if err != nil {
		writeError(w, fmt.Sprintf("Failed to analyze project: %v", err), http.StatusInternalServerError)
		return
	}

	// Save project metadata
	if err := h.storage.SaveProjectMeta(req.Directory, profile); err != nil {
		log.Printf("Failed to save project meta: %v", err)
	}

	writeSuccess(w, map[string]interface{}{
		"analyzed":     true,
		"directory":    req.Directory,
		"project_hash": profile.ProjectHash,
		"category":     profile.Category,
		"summary":      profile.Summary,
		"stats":        profile.Stats,
		"highlights":   profile.Highlights,
	})
}

// buildSessionProfileMessage generates a system message from session profile
func (h *Handler) buildSessionProfileMessage(profiles []types.ProfileCard, patterns []types.PatternCard) string {
	var parts []string

	// Extract intent info
	for _, card := range profiles {
		if card.Type == "IntentCard" {
			if intentFreq, ok := card.Content["intent_freq"].(map[string]interface{}); ok {
				// Sort keys for deterministic output
				var keys []string
				for intent := range intentFreq {
					keys = append(keys, intent)
				}
				sort.Strings(keys)

				var intents []string
				for _, intent := range keys {
					freq := intentFreq[intent]
					intents = append(intents, fmt.Sprintf("%s(%v)", intent, freq))
				}
				if len(intents) > 0 {
					parts = append(parts, "Session意图: "+strings.Join(intents, ", "))
				}
			}
			if prompt, ok := card.Content["last_prompt"].(string); ok && prompt != "" {
				parts = append(parts, "最近问题: "+prompt)
			}
		}
	}

	// Extract tool patterns
	toolFreq := make(map[string]int)
	for _, p := range patterns {
		if p.Type == "tool_usage" {
			toolFreq[p.Pattern] += p.Frequency
		}
	}
	if len(toolFreq) > 0 {
		var toolList []string
		for tool, freq := range toolFreq {
			toolList = append(toolList, fmt.Sprintf("%s(%d)", tool, freq))
		}
		parts = append(parts, "工具使用: "+strings.Join(toolList, ", "))
	}

	if len(parts) == 0 {
		return ""
	}
	return "Session画像摘要:\n- " + strings.Join(parts, "\n- ")
}

// buildProjectProfileMessage generates a system message from project profile
func (h *Handler) buildProjectProfileMessage(profiles []types.ProfileCard, meta *types.ProjectProfile) string {
	var parts []string

	// Add directory summary from meta
	if meta != nil {
		// Category-specific message
		switch meta.Category {
		case "development":
			parts = append(parts, "这是一个开发项目目录。")
		case "documentation":
			parts = append(parts, "这是一个文档库目录。")
		case "operations":
			parts = append(parts, "这是一个数据/运营目录。")
		case "design":
			parts = append(parts, "这是一个设计资源目录。")
		case "media":
			parts = append(parts, "这是一个媒体文件目录。")
		case "mixed":
			parts = append(parts, "这是一个混合用途目录。")
		default:
			parts = append(parts, "这是一个未分类目录。")
		}

		if meta.Summary != "" {
			parts = append(parts, meta.Summary)
		}

		// Add highlights
		if len(meta.Highlights) > 0 {
			parts = append(parts, "主要特征: "+strings.Join(meta.Highlights, ", "))
		}
	}

	// Add patterns from profiles
	for _, card := range profiles {
		if pattern, ok := card.Content["pattern"].(string); ok && pattern != "" {
			parts = append(parts, pattern)
		}
	}

	if len(parts) == 0 {
		return ""
	}
	return "目录画像摘要:\n- " + strings.Join(parts, "\n- ")
}

// Helper functions
func writeSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(types.APIResponse{
		Success: true,
		Data:    data,
	})
}

func writeError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(types.APIResponse{
		Success: false,
		Error:   message,
	})
}