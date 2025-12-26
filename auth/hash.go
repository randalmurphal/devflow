package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

// HashToken creates a SHA-256 hash of a token for secure storage.
// Use this to store refresh tokens or API keys in the database.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
