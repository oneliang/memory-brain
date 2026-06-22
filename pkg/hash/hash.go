package hash

import (
	"crypto/sha256"
	"encoding/hex"
)

// ProjectHash computes SHA256 hash of directory path for project identification
// Uses first 16 bytes (32 hex chars) for a balance of uniqueness and brevity
func ProjectHash(directory string) string {
	hash := sha256.Sum256([]byte(directory))
	return hex.EncodeToString(hash[:16])
}
