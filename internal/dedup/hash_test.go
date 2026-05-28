package dedup

import (
	"fmt"
	"testing"
	"time"
)

// TestComputeHash_Deterministic tests hash determinism
func TestComputeHash_Deterministic(t *testing.T) {
	data := map[string]interface{}{
		"tool": "bash",
		"args": "ls -la",
	}

	hash1 := ComputeHash("session_1", "post_tool_use", data)
	hash2 := ComputeHash("session_1", "post_tool_use", data)

	if hash1 != hash2 {
		t.Errorf("same input should produce same hash: %s vs %s", hash1, hash2)
	}

	// Hash should be 64 characters (SHA-256 hex)
	if len(hash1) != 64 {
		t.Errorf("SHA-256 hash should be 64 hex characters, got %d", len(hash1))
	}
}

// TestComputeHash_DifferentInputs tests hash uniqueness
func TestComputeHash_DifferentInputs(t *testing.T) {
	data := map[string]interface{}{"tool": "bash"}

	hash1 := ComputeHash("session_1", "post_tool_use", data)
	hash2 := ComputeHash("session_2", "post_tool_use", data)
	hash3 := ComputeHash("session_1", "pre_tool_use", data)

	if hash1 == hash2 {
		t.Error("different session_id should produce different hash")
	}

	if hash1 == hash3 {
		t.Error("different hook_type should produce different hash")
	}
}

// TestComputeHash_DifferentData tests data-based hash uniqueness
func TestComputeHash_DifferentData(t *testing.T) {
	data1 := map[string]interface{}{"tool": "bash", "args": "ls"}
	data2 := map[string]interface{}{"tool": "bash", "args": "cat"}

	hash1 := ComputeHash("session_1", "post_tool_use", data1)
	hash2 := ComputeHash("session_1", "post_tool_use", data2)

	if hash1 == hash2 {
		t.Error("different data should produce different hash")
	}
}

// TestComputeHashFromString tests string hash
func TestComputeHashFromString(t *testing.T) {
	input := "test string for hashing"

	hash1 := ComputeHashFromString(input)
	hash2 := ComputeHashFromString(input)

	if hash1 != hash2 {
		t.Error("same string should produce same hash")
	}

	if len(hash1) != 64 {
		t.Errorf("SHA-256 hash should be 64 hex characters, got %d", len(hash1))
	}
}

// TestComputeHashFromString_Different tests different strings
func TestComputeHashFromString_Different(t *testing.T) {
	hash1 := ComputeHashFromString("string1")
	hash2 := ComputeHashFromString("string2")

	if hash1 == hash2 {
		t.Error("different strings should produce different hashes")
	}
}

// TestHashCache_Basic tests cache operations
func TestHashCache_Basic(t *testing.T) {
	cache := NewHashCache(5 * time.Minute)

	hash := "test_hash_123"

	// First check - should not be duplicate
	if cache.IsDuplicate(hash) {
		t.Error("first check should not be duplicate")
	}

	// Add to cache
	cache.Add(hash)

	// Second check - should be duplicate
	if !cache.IsDuplicate(hash) {
		t.Error("second check should be duplicate")
	}
}

// TestHashCache_WindowExpiry tests time window expiry
func TestHashCache_WindowExpiry(t *testing.T) {
	// Use very short window for testing
	cache := NewHashCache(100 * time.Millisecond)

	hash := "expiry_test_hash"

	cache.Add(hash)

	// Should be duplicate immediately
	if !cache.IsDuplicate(hash) {
		t.Error("should be duplicate within window")
	}

	// Wait for window to expire
	time.Sleep(150 * time.Millisecond)

	// Should no longer be duplicate
	if cache.IsDuplicate(hash) {
		t.Error("should not be duplicate after window expiry")
	}
}

// TestHashCache_Cleanup tests automatic cleanup
func TestHashCache_Cleanup(t *testing.T) {
	cache := NewHashCache(50 * time.Millisecond)

	// Add multiple hashes
	for i := 0; i < 10; i++ {
		cache.Add(ComputeHashFromString(string(rune(i))))
	}

	// Wait for expiry
	time.Sleep(100 * time.Millisecond)

	// Add another hash to trigger cleanup
	cache.Add("trigger_cleanup")

	// Old hashes should be cleaned up
	if len(cache.hashes) > 2 {
		t.Errorf("cache should have cleaned up old entries, got %d", len(cache.hashes))
	}
}

// TestHashCache_Concurrent tests concurrent access safety
func TestHashCache_Concurrent(t *testing.T) {
	cache := NewHashCache(5 * time.Minute)

	// Concurrent adds and checks
	done := make(chan bool)

	for i := 0; i < 100; i++ {
		go func(n int) {
			hash := ComputeHashFromString(string(rune(n)))
			cache.Add(hash)
			_ = cache.IsDuplicate(hash)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}

	// Should have ~100 unique hashes
	if len(cache.hashes) < 95 {
		t.Errorf("expected ~100 hashes, got %d", len(cache.hashes))
	}
}

// TestCheckAndAdd tests global cache function
func TestCheckAndAdd(t *testing.T) {
	// Note: This test uses the global DefaultHashCache
	// It may affect other tests if run in parallel

	hash := ComputeHashFromString("check_and_add_test")

	// First call - should return false (not duplicate, added)
	isDup1 := CheckAndAdd(hash)
	if isDup1 {
		t.Error("first CheckAndAdd should return false")
	}

	// Second call with same hash - should return true (duplicate)
	isDup2 := CheckAndAdd(hash)
	if !isDup2 {
		t.Error("second CheckAndAdd should return true")
	}
}

// TestDedupWindow tests constant value
func TestDedupWindow(t *testing.T) {
	if DedupWindow <= 0 {
		t.Error("DedupWindow should be positive")
	}

	if DedupWindow > 10*time.Minute {
		t.Error("DedupWindow should be reasonable (not too long)")
	}
}

// TestNewHashCache tests cache creation
func TestNewHashCache(t *testing.T) {
	window := 10 * time.Second
	cache := NewHashCache(window)

	if cache == nil {
		t.Fatal("cache should not be nil")
	}

	if cache.window != window {
		t.Errorf("expected window %v, got %v", window, cache.window)
	}

	if cache.hashes == nil {
		t.Error("hashes map should be initialized")
	}
}

// === SessionCacheManager Tests ===

// TestSessionCacheManager_New tests manager creation
func TestSessionCacheManager_New(t *testing.T) {
	manager := NewSessionCacheManager(5 * time.Minute)
	if manager == nil {
		t.Fatal("manager should not be nil")
	}
	if manager.GetSessionCount() != 0 {
		t.Error("new manager should have 0 sessions")
	}
}

// TestSessionCacheManager_GetOrCreate tests cache creation per session
func TestSessionCacheManager_GetOrCreate(t *testing.T) {
	manager := NewSessionCacheManager(5 * time.Minute)

	cache1 := manager.GetOrCreate("session_1")
	cache2 := manager.GetOrCreate("session_2")
	cache1Again := manager.GetOrCreate("session_1")

	if cache1 == nil || cache2 == nil {
		t.Fatal("caches should not be nil")
	}

	// Same session should return same cache
	if cache1 != cache1Again {
		t.Error("same session should return same cache")
	}

	// Different sessions should have different caches
	if cache1 == cache2 {
		t.Error("different sessions should have different caches")
	}

	if manager.GetSessionCount() != 2 {
		t.Errorf("should have 2 session caches, got %d", manager.GetSessionCount())
	}
}

// TestSessionCacheManager_CheckAndAdd tests dedup logic
func TestSessionCacheManager_CheckAndAdd(t *testing.T) {
	manager := NewSessionCacheManager(5 * time.Minute)

	hash := "test_hash"

	// First check - should not be duplicate
	isDup1 := manager.CheckAndAdd("session_1", hash)
	if isDup1 {
		t.Error("first CheckAndAdd should return false")
	}

	// Same hash, same session - should be duplicate
	isDup2 := manager.CheckAndAdd("session_1", hash)
	if !isDup2 {
		t.Error("second CheckAndAdd same session should return true")
	}

	// Same hash, different session - should NOT be duplicate
	isDup3 := manager.CheckAndAdd("session_2", hash)
	if isDup3 {
		t.Error("different session should not be duplicate")
	}
}

// TestSessionCacheManager_ClearSession tests session clearing
func TestSessionCacheManager_ClearSession(t *testing.T) {
	manager := NewSessionCacheManager(5 * time.Minute)

	manager.GetOrCreate("session_1")
	manager.GetOrCreate("session_2")

	if manager.GetSessionCount() != 2 {
		t.Fatal("should have 2 sessions before clear")
	}

	manager.ClearSession("session_1")

	if manager.GetSessionCount() != 1 {
		t.Errorf("should have 1 session after clear, got %d", manager.GetSessionCount())
	}
}

// TestSessionCacheManager_ClearSession_RestartsFresh tests that cleared session starts fresh
func TestSessionCacheManager_ClearSession_RestartsFresh(t *testing.T) {
	manager := NewSessionCacheManager(5 * time.Minute)

	hash := "test_hash"

	// Add hash to session_1
	manager.CheckAndAdd("session_1", hash)

	// Clear session
	manager.ClearSession("session_1")

	// Same session should now accept same hash
	isDup := manager.CheckAndAdd("session_1", hash)
	if isDup {
		t.Error("cleared session should not have duplicate hash")
	}
}

// TestSessionCacheManager_Concurrent tests concurrent session access
func TestSessionCacheManager_Concurrent(t *testing.T) {
	manager := NewSessionCacheManager(5 * time.Minute)

	done := make(chan bool)

	// Concurrent operations on different sessions
	for i := 0; i < 50; i++ {
		go func(n int) {
			sessionID := fmt.Sprintf("session_%d", n)
			hash := "test_hash"
			manager.CheckAndAdd(sessionID, hash)
			manager.GetOrCreate(sessionID)
			done <- true
		}(i)
	}

	for i := 0; i < 50; i++ {
		<-done
	}

	if manager.GetSessionCount() < 45 {
		t.Errorf("should have ~50 sessions, got %d", manager.GetSessionCount())
	}
}