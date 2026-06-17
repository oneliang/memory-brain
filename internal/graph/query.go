package graph

import (
	"encoding/json"
	"sort"
)

// QueryEngine handles multi-hop graph queries
type QueryEngine struct {
	store *Store
}

// NewQueryEngine creates a new query engine
func NewQueryEngine(store *Store) *QueryEngine {
	return &QueryEngine{store: store}
}

// QueryOptions represents options for graph queries
type QueryOptions struct {
	MaxHops    int     // Maximum number of hops (default: 2)
	Limit      int     // Maximum number of results (default: 10)
	MinScore   float64 // Minimum score threshold (default: 0.1)
	DecayRate  float64 // Score decay per hop (default: 0.7)
	NodeTypes  []string // Filter by node types (empty = all)
	Relations  []string // Filter by relation types (empty = all)
}

// DefaultQueryOptions returns default query options
func DefaultQueryOptions() *QueryOptions {
	return &QueryOptions{
		MaxHops:   2,
		Limit:     10,
		MinScore:  0.1,
		DecayRate: 0.7,
	}
}

// MultiHopQuery performs multi-hop traversal from a user node
func (q *QueryEngine) MultiHopQuery(userID string, opts *QueryOptions) ([]QueryResult, error) {
	if opts == nil {
		opts = DefaultQueryOptions()
	}

	// Get or create user node
	userNode, err := q.store.GetNodeByTypeAndName("user", userID)
	if err != nil {
		return []QueryResult{}, nil // User not found, return empty
	}

	// BFS traversal
	visited := make(map[int64]bool)
	scoreMap := make(map[int64]float64)
	hopsMap := make(map[int64]int)

	// Start from user node
	queue := []int64{userNode.ID}
	visited[userNode.ID] = true
	scoreMap[userNode.ID] = 1.0
	hopsMap[userNode.ID] = 0

	for len(queue) > 0 {
		// Dequeue
		currentID := queue[0]
		queue = queue[1:]

		currentHops := hopsMap[currentID]
		if currentHops >= opts.MaxHops {
			continue
		}

		// Get outgoing edges
		edges, err := q.store.GetOutgoingEdges(currentID)
		if err != nil {
			continue
		}

		for _, edge := range edges {
			// Filter by relation type if specified
			if len(opts.Relations) > 0 && !contains(opts.Relations, edge.Relation) {
				continue
			}

			// Calculate new score with decay
			newScore := scoreMap[currentID] * edge.Weight * opts.DecayRate

			// Update score if better
			if newScore > scoreMap[edge.TargetID] {
				scoreMap[edge.TargetID] = newScore
				hopsMap[edge.TargetID] = currentHops + 1
			}

			// Enqueue if not visited
			if !visited[edge.TargetID] {
				visited[edge.TargetID] = true
				queue = append(queue, edge.TargetID)
			}
		}
	}

	// Collect results (exclude user node itself)
	var results []QueryResult
	for nodeID, score := range scoreMap {
		if nodeID == userNode.ID {
			continue
		}
		if score < opts.MinScore {
			continue
		}

		node, err := q.store.GetNode(nodeID)
		if err != nil {
			continue
		}

		// Filter by node type if specified
		if len(opts.NodeTypes) > 0 && !contains(opts.NodeTypes, node.Type) {
			continue
		}

		results = append(results, QueryResult{
			Node:  *node,
			Score: score,
			Hops:  hopsMap[nodeID],
		})
	}

	// Sort by score (descending)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	// Apply limit
	if len(results) > opts.Limit {
		results = results[:opts.Limit]
	}

	return results, nil
}

// GetNeighbors returns immediate neighbors of a node
func (q *QueryEngine) GetNeighbors(nodeID int64, relation string) ([]QueryResult, error) {
	var edges []Edge
	var err error

	if relation == "" {
		// Get all outgoing edges
		edges, err = q.store.GetOutgoingEdges(nodeID)
	} else {
		// Get filtered edges (need to query with relation filter)
		allEdges, queryErr := q.store.GetOutgoingEdges(nodeID)
		if queryErr != nil {
			return nil, queryErr
		}
		for _, edge := range allEdges {
			if edge.Relation == relation {
				edges = append(edges, edge)
			}
		}
		err = nil
	}

	if err != nil {
		return nil, err
	}

	var results []QueryResult
	for _, edge := range edges {
		node, err := q.store.GetNode(edge.TargetID)
		if err != nil {
			continue
		}

		results = append(results, QueryResult{
			Node:  *node,
			Score: edge.Weight,
			Hops:  1,
		})
	}

	// Sort by score
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})

	return results, nil
}

// GetContext returns contextual information for a user (high-level API)
func (q *QueryEngine) GetContext(userID string, maxEntities int) (map[string]interface{}, error) {
	if maxEntities <= 0 {
		maxEntities = 20
	}

	// Get multi-hop results
	opts := &QueryOptions{
		MaxHops:   2,
		Limit:     maxEntities,
		MinScore:  0.2,
		DecayRate: 0.7,
	}

	results, err := q.MultiHopQuery(userID, opts)
	if err != nil {
		return nil, err
	}

	// Group by entity type
	entitiesByType := make(map[string][]map[string]interface{})
	for _, result := range results {
		entity := map[string]interface{}{
			"name":  result.Node.Name,
			"score": result.Score,
			"hops":  result.Hops,
		}

		// Parse properties if present
		if result.Node.Properties != "" {
			var props map[string]interface{}
			if err := json.Unmarshal([]byte(result.Node.Properties), &props); err == nil {
				entity["properties"] = props
			}
		}

		entitiesByType[result.Node.Type] = append(entitiesByType[result.Node.Type], entity)
	}

	// Get top tools, technologies, files
	topTools := getTopEntities(entitiesByType["tool"], 5)
	topTechs := getTopEntities(entitiesByType["tech"], 5)
	topFiles := getTopEntities(entitiesByType["file"], 5)
	topConcepts := getTopEntities(entitiesByType["concept"], 5)

	return map[string]interface{}{
		"user_id":    userID,
		"total":      len(results),
		"tools":      topTools,
		"techs":      topTechs,
		"files":      topFiles,
		"concepts":   topConcepts,
		"all":        entitiesByType,
	}, nil
}

// getTopEntities returns top N entities by score
func getTopEntities(entities []map[string]interface{}, limit int) []map[string]interface{} {
	if len(entities) <= limit {
		return entities
	}
	return entities[:limit]
}

// FindPath finds the shortest path between two nodes
func (q *QueryEngine) FindPath(sourceID, targetID int64, maxHops int) ([]int64, error) {
	if maxHops <= 0 {
		maxHops = 5
	}

	// BFS to find shortest path
	visited := make(map[int64]bool)
	parent := make(map[int64]int64)
	queue := []int64{sourceID}
	visited[sourceID] = true

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		if current == targetID {
			// Reconstruct path
			var path []int64
			for node := targetID; node != sourceID; node = parent[node] {
				path = append([]int64{node}, path...)
			}
			path = append([]int64{sourceID}, path...)
			return path, nil
		}

		// Check hops
		hops := 0
		for node := current; node != sourceID; node = parent[node] {
			hops++
		}
		if hops >= maxHops {
			continue
		}

		// Get neighbors
		edges, err := q.store.GetOutgoingEdges(current)
		if err != nil {
			continue
		}

		for _, edge := range edges {
			if !visited[edge.TargetID] {
				visited[edge.TargetID] = true
				parent[edge.TargetID] = current
				queue = append(queue, edge.TargetID)
			}
		}
	}

	return nil, nil // No path found
}

// Helper function
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
