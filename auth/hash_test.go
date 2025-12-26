package auth

import "testing"

func TestHashToken(t *testing.T) {
	t.Run("deterministic", func(t *testing.T) {
		token := "test-token-12345"
		hash1 := HashToken(token)
		hash2 := HashToken(token)

		if hash1 != hash2 {
			t.Errorf("HashToken not deterministic: %q != %q", hash1, hash2)
		}
	})

	t.Run("different inputs different hashes", func(t *testing.T) {
		hash1 := HashToken("token-a")
		hash2 := HashToken("token-b")

		if hash1 == hash2 {
			t.Error("different tokens should have different hashes")
		}
	})

	t.Run("empty input", func(t *testing.T) {
		hash := HashToken("")
		if hash == "" {
			t.Error("hash of empty string should not be empty")
		}
	})

	t.Run("hash length", func(t *testing.T) {
		hash := HashToken("test")
		// SHA-256 produces 32 bytes = 64 hex characters
		if len(hash) != 64 {
			t.Errorf("hash length = %d, want 64", len(hash))
		}
	})
}
