package dedup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// DedupWindow defines the time window for deduplication (5 minutes)
const DedupWindow = 5 * time.Minute

// HashCache stores recent hashes for deduplication
type HashCache struct {
	hashes map[string]time.Time
	window time.Duration
	mu     sync.RWMutex // protects hashes
}

// NewHashCache creates a new hash cache
func NewHashCache(window time.Duration) *HashCache {
	return &HashCache{
		hashes: make(map[string]time.Time),
		window: window,
	}
}

// ComputeHash computes SHA-256 hash of observation data
func ComputeHash(sessionID, hookType string, data map[string]interface{}) string {
	// Create a deterministic JSON representation
	dataBytes, err := json.Marshal(data)
	if err != nil {
		// Fallback to string representation
		dataBytes = []byte(fmt.Sprintf("%v", data))
	}

	// Combine session_id, hook_type, and data for unique hash
	input := fmt.Sprintf("%s:%s:%s", sessionID, hookType, string(dataBytes))
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// ComputeHashFromString computes SHA-256 hash from a string
func ComputeHashFromString(input string) string {
	hash := sha256.Sum256([]byte(input))
	return hex.EncodeToString(hash[:])
}

// IsDuplicate checks if a hash is duplicate within the time window
func (c *HashCache) IsDuplicate(hash string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if seenAt, exists := c.hashes[hash]; exists {
		return time.Since(seenAt) < c.window
	}
	return false
}

// Add adds a hash to the cache
func (c *HashCache) Add(hash string) {
	c.mu.Lock()
	c.hashes[hash] = time.Now()
	c.mu.Unlock()

	// Cleanup old entries
	c.cleanup()
}

// cleanup removes expired entries
func (c *HashCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for hash, seenAt := range c.hashes {
		if now.Sub(seenAt) > c.window {
			delete(c.hashes, hash)
		}
	}
}

// DefaultHashCache is the global dedup cache
var DefaultHashCache = NewHashCache(DedupWindow)

// CheckAndAdd checks if duplicate and adds if not
func CheckAndAdd(hash string) bool {
	if DefaultHashCache.IsDuplicate(hash) {
		return true // Is duplicate
	}
	DefaultHashCache.Add(hash)
	return false // Not duplicate, added
}

// SessionCacheManager manages per-session dedup caches
// Used for immediate observation dedup (session-focused)
type SessionCacheManager struct {
	caches map[string]*HashCache // sessionID -> cache
	mu     sync.RWMutex
	window time.Duration
}

// NewSessionCacheManager creates a new session cache manager
func NewSessionCacheManager(window time.Duration) *SessionCacheManager {
	return &SessionCacheManager{
		caches: make(map[string]*HashCache),
		window: window,
	}
}

// GetOrCreate returns the cache for a session, creating if needed
func (m *SessionCacheManager) GetOrCreate(sessionID string) *HashCache {
	m.mu.RLock()
	cache, exists := m.caches[sessionID]
	m.mu.RUnlock()

	if exists {
		return cache
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	// Double-check after acquiring write lock
	if cache, exists = m.caches[sessionID]; !exists {
		cache = NewHashCache(m.window)
		m.caches[sessionID] = cache
	}
	return cache
}

// CheckAndAdd checks for duplicates and adds hash to session cache
// Returns true if duplicate, false if not (and added)
func (m *SessionCacheManager) CheckAndAdd(sessionID, hash string) bool {
	cache := m.GetOrCreate(sessionID)
	if cache.IsDuplicate(hash) {
		return true // Is duplicate within session
	}
	cache.Add(hash)
	return false // Not duplicate, added to session cache
}

// ClearSession removes a session's cache (call when session ends)
func (m *SessionCacheManager) ClearSession(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.caches, sessionID)
}

// CleanupExpired removes all expired session caches
func (m *SessionCacheManager) CleanupExpired() {
	m.mu.Lock()
	defer m.mu.Unlock()
	now := time.Now()
	for sessionID, cache := range m.caches {
		cache.mu.RLock()
		if len(cache.hashes) > 0 {
			// Check if all entries are expired
			allExpired := true
			for _, seenAt := range cache.hashes {
				if now.Sub(seenAt) < cache.window {
					allExpired = false
					break
				}
			}
			if allExpired {
				delete(m.caches, sessionID)
			}
		} else {
			// Empty cache, remove it
			delete(m.caches, sessionID)
		}
		cache.mu.RUnlock()
	}
}

// GetSessionCount returns the number of active session caches
func (m *SessionCacheManager) GetSessionCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.caches)
}