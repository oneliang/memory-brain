package graph

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store represents the SQLite graph storage
type Store struct {
	db     *sql.DB
	dbPath string
	mu     sync.RWMutex
}

// NewStore creates a new graph store
func NewStore(userID string) (*Store, error) {
	// Default path: ~/.memory-brain/users/{user_id}/graph.db
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	dbDir := filepath.Join(homeDir, ".memory-brain", "users", userID)
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	dbPath := filepath.Join(dbDir, "graph.db")
	return NewStoreWithPath(dbPath)
}

// NewStoreWithPath creates a new graph store with custom path
func NewStoreWithPath(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{
		db:     db,
		dbPath: dbPath,
	}

	if err := store.initTables(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize tables: %w", err)
	}

	return store, nil
}

// initTables creates the necessary tables
func (s *Store) initTables() error {
	schema := `
	CREATE TABLE IF NOT EXISTS nodes (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		type       TEXT NOT NULL,
		name       TEXT NOT NULL,
		properties TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(type, name)
	);

	CREATE TABLE IF NOT EXISTS edges (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		source_id  INTEGER NOT NULL,
		target_id  INTEGER NOT NULL,
		relation   TEXT NOT NULL,
		weight     REAL DEFAULT 1.0,
		timestamp  DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(source_id) REFERENCES nodes(id),
		FOREIGN KEY(target_id) REFERENCES nodes(id)
	);

	CREATE INDEX IF NOT EXISTS idx_edges_source ON edges(source_id);
	CREATE INDEX IF NOT EXISTS idx_edges_target ON edges(target_id);
	CREATE INDEX IF NOT EXISTS idx_nodes_type ON nodes(type);
	CREATE INDEX IF NOT EXISTS idx_nodes_name ON nodes(name);
	`

	_, err := s.db.Exec(schema)
	return err
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// GetOrCreateNode gets an existing node or creates a new one
func (s *Store) GetOrCreateNode(nodeType, name string, properties map[string]interface{}) (*Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Serialize properties to JSON
	var propsJSON string
	if properties != nil {
		propsBytes, err := json.Marshal(properties)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal properties: %w", err)
		}
		propsJSON = string(propsBytes)
	}

	// Try to insert, ignore if exists
	result, err := s.db.Exec(
		`INSERT OR IGNORE INTO nodes (type, name, properties) VALUES (?, ?, ?)`,
		nodeType, name, propsJSON,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to insert node: %w", err)
	}

	// Get the node ID
	var nodeID int64
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		// Node already exists, get its ID
		err = s.db.QueryRow(
			`SELECT id FROM nodes WHERE type = ? AND name = ?`,
			nodeType, name,
		).Scan(&nodeID)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing node: %w", err)
		}
	} else {
		nodeID, _ = result.LastInsertId()
	}

	// Fetch the complete node
	var node Node
	err = s.db.QueryRow(
		`SELECT id, type, name, properties, created_at FROM nodes WHERE id = ?`,
		nodeID,
	).Scan(&node.ID, &node.Type, &node.Name, &node.Properties, &node.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch node: %w", err)
	}

	return &node, nil
}

// CreateEdge creates a new edge between two nodes
func (s *Store) CreateEdge(sourceID, targetID int64, relation string, weight float64) (*Edge, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if weight <= 0 {
		weight = 1.0
	}

	result, err := s.db.Exec(
		`INSERT INTO edges (source_id, target_id, relation, weight) VALUES (?, ?, ?, ?)`,
		sourceID, targetID, relation, weight,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create edge: %w", err)
	}

	edgeID, _ := result.LastInsertId()

	var edge Edge
	err = s.db.QueryRow(
		`SELECT id, source_id, target_id, relation, weight, timestamp FROM edges WHERE id = ?`,
		edgeID,
	).Scan(&edge.ID, &edge.SourceID, &edge.TargetID, &edge.Relation, &edge.Weight, &edge.Timestamp)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch edge: %w", err)
	}

	return &edge, nil
}

// GetNode gets a node by ID
func (s *Store) GetNode(nodeID int64) (*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var node Node
	err := s.db.QueryRow(
		`SELECT id, type, name, properties, created_at FROM nodes WHERE id = ?`,
		nodeID,
	).Scan(&node.ID, &node.Type, &node.Name, &node.Properties, &node.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return &node, nil
}

// GetNodeByTypeAndName gets a node by type and name
func (s *Store) GetNodeByTypeAndName(nodeType, name string) (*Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var node Node
	err := s.db.QueryRow(
		`SELECT id, type, name, properties, created_at FROM nodes WHERE type = ? AND name = ?`,
		nodeType, name,
	).Scan(&node.ID, &node.Type, &node.Name, &node.Properties, &node.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	return &node, nil
}

// GetOutgoingEdges gets all edges from a node
func (s *Store) GetOutgoingEdges(nodeID int64) ([]Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		`SELECT id, source_id, target_id, relation, weight, timestamp FROM edges WHERE source_id = ?`,
		nodeID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges: %w", err)
	}
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var edge Edge
		if err := rows.Scan(&edge.ID, &edge.SourceID, &edge.TargetID, &edge.Relation, &edge.Weight, &edge.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}
		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

// GetIncomingEdges gets all edges to a node
func (s *Store) GetIncomingEdges(nodeID int64) ([]Edge, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		`SELECT id, source_id, target_id, relation, weight, timestamp FROM edges WHERE target_id = ?`,
		nodeID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query edges: %w", err)
	}
	defer rows.Close()

	var edges []Edge
	for rows.Next() {
		var edge Edge
		if err := rows.Scan(&edge.ID, &edge.SourceID, &edge.TargetID, &edge.Relation, &edge.Weight, &edge.Timestamp); err != nil {
			return nil, fmt.Errorf("failed to scan edge: %w", err)
		}
		edges = append(edges, edge)
	}

	return edges, rows.Err()
}

// UpdateEdgeWeight updates the weight of an edge (adds to existing weight)
func (s *Store) UpdateEdgeWeight(sourceID, targetID int64, relation string, weightDelta float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(
		`UPDATE edges SET weight = weight + ? WHERE source_id = ? AND target_id = ? AND relation = ?`,
		weightDelta, sourceID, targetID, relation,
	)
	return err
}

// SaveExtractionResult saves entities and relations from an ExtractionResult
func (s *Store) SaveExtractionResult(userID string, result *ExtractionResult) error {
	if result == nil {
		return nil
	}

	// Create user node
	userNode, err := s.GetOrCreateNode("user", userID, nil)
	if err != nil {
		return fmt.Errorf("failed to create user node: %w", err)
	}

	// Save entities
	entityMap := make(map[string]*Node) // name -> node
	for _, entity := range result.Entities {
		node, err := s.GetOrCreateNode(entity.Type, entity.Name, entity.Properties)
		if err != nil {
			return fmt.Errorf("failed to save entity %s: %w", entity.Name, err)
		}
		entityMap[entity.Name] = node
	}

	// Save relations
	for _, relation := range result.Relations {
		var sourceNode, targetNode *Node
		var ok bool

		// Get source node
		if relation.Source == userID {
			sourceNode = userNode
		} else if sourceNode, ok = entityMap[relation.Source]; !ok {
			// Source not in entities, try to find it
			sourceNode, err = s.GetNodeByTypeAndName("", relation.Source)
			if err != nil {
				// Create a generic node
				sourceNode, err = s.GetOrCreateNode("unknown", relation.Source, nil)
				if err != nil {
					continue
				}
			}
		}

		// Get target node
		if targetNode, ok = entityMap[relation.Target]; !ok {
			// Target not in entities, try to find it
			targetNode, err = s.GetNodeByTypeAndName("", relation.Target)
			if err != nil {
				// Create a generic node
				targetNode, err = s.GetOrCreateNode("unknown", relation.Target, nil)
				if err != nil {
					continue
				}
			}
		}

		weight := relation.Weight
		if weight <= 0 {
			weight = 1.0
		}

		// Create edge
		_, err = s.CreateEdge(sourceNode.ID, targetNode.ID, relation.Type, weight)
		if err != nil {
			// Edge might already exist, try to update weight
			s.UpdateEdgeWeight(sourceNode.ID, targetNode.ID, relation.Type, weight)
		}
	}

	return nil
}

// GetNodeCount returns the total number of nodes
func (s *Store) GetNodeCount() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM nodes`).Scan(&count)
	return count, err
}

// GetEdgeCount returns the total number of edges
func (s *Store) GetEdgeCount() (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM edges`).Scan(&count)
	return count, err
}

// GetNodesByType returns all nodes of a specific type
func (s *Store) GetNodesByType(nodeType string) ([]Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(
		`SELECT id, type, name, properties, created_at FROM nodes WHERE type = ?`,
		nodeType,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		if err := rows.Scan(&node.ID, &node.Type, &node.Name, &node.Properties, &node.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// SearchNodes searches nodes by name (partial match)
func (s *Store) SearchNodes(query string, limit int) ([]Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(
		`SELECT id, type, name, properties, created_at FROM nodes WHERE name LIKE ? LIMIT ?`,
		"%"+query+"%", limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search nodes: %w", err)
	}
	defer rows.Close()

	var nodes []Node
	for rows.Next() {
		var node Node
		if err := rows.Scan(&node.ID, &node.Type, &node.Name, &node.Properties, &node.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan node: %w", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// Truncate removes all nodes and edges (for testing)
func (s *Store) Truncate() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM edges`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM nodes`); err != nil {
		return err
	}

	return tx.Commit()
}

// GetStats returns statistics about the graph
func (s *Store) GetStats() (map[string]interface{}, error) {
	nodeCount, err := s.GetNodeCount()
	if err != nil {
		return nil, err
	}

	edgeCount, err := s.GetEdgeCount()
	if err != nil {
		return nil, err
	}

	// Count by node type
	rows, err := s.db.Query(`SELECT type, COUNT(*) FROM nodes GROUP BY type`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	nodeTypes := make(map[string]int)
	for rows.Next() {
		var nodeType string
		var count int
		if err := rows.Scan(&nodeType, &count); err != nil {
			return nil, err
		}
		nodeTypes[nodeType] = count
	}

	// Count by relation type
	rows2, err := s.db.Query(`SELECT relation, COUNT(*) FROM edges GROUP BY relation`)
	if err != nil {
		return nil, err
	}
	defer rows2.Close()

	relationTypes := make(map[string]int)
	for rows2.Next() {
		var relation string
		var count int
		if err := rows2.Scan(&relation, &count); err != nil {
			return nil, err
		}
		relationTypes[relation] = count
	}

	return map[string]interface{}{
		"node_count":     nodeCount,
		"edge_count":     edgeCount,
		"node_types":     nodeTypes,
		"relation_types": relationTypes,
		"db_path":        s.dbPath,
		"timestamp":      time.Now(),
	}, nil
}
