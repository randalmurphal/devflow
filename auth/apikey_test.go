package auth

import (
	"strings"
	"testing"
)

func TestGenerateAPIKey(t *testing.T) {
	cfg := APIKeyConfig{
		Prefix:       "test_live_",
		RandomLength: 32,
		PrefixLength: 14,
	}

	t.Run("basic generation", func(t *testing.T) {
		key, err := GenerateAPIKey(cfg)
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}
		if key.ID == "" {
			t.Error("ID is empty")
		}
		if key.Secret == "" {
			t.Error("Secret is empty")
		}
		if key.Prefix == "" {
			t.Error("Prefix is empty")
		}
		if key.Hash == "" {
			t.Error("Hash is empty")
		}

		// Verify secret has correct prefix
		if !strings.HasPrefix(key.Secret, "test_live_") {
			t.Errorf("Secret %q should start with 'test_live_'", key.Secret)
		}

		// Verify format
		if !ValidateAPIKeyFormat(key.Secret, cfg) {
			t.Errorf("Secret %q does not match expected format", key.Secret)
		}

		// Verify hash
		if HashToken(key.Secret) != key.Hash {
			t.Error("hash mismatch")
		}
	})

	t.Run("default config", func(t *testing.T) {
		key, err := GenerateAPIKey(APIKeyConfig{})
		if err != nil {
			t.Fatalf("GenerateAPIKey() error = %v", err)
		}
		if !strings.HasPrefix(key.Secret, DefaultAPIKeyPrefix) {
			t.Errorf("Secret %q should start with %q", key.Secret, DefaultAPIKeyPrefix)
		}
	})

	t.Run("uniqueness", func(t *testing.T) {
		keys := make(map[string]bool)
		for i := 0; i < 10; i++ {
			key, err := GenerateAPIKey(cfg)
			if err != nil {
				t.Fatalf("GenerateAPIKey() error = %v", err)
			}
			if keys[key.Secret] {
				t.Errorf("duplicate key generated: %s", key.Secret)
			}
			keys[key.Secret] = true
		}
	})
}

func TestValidateAPIKeyFormat(t *testing.T) {
	cfg := APIKeyConfig{
		Prefix:       "tk_live_",
		RandomLength: 32,
	}

	tests := []struct {
		key  string
		want bool
	}{
		{"tk_live_12345678901234567890123456789012", true},
		{"tk_live_short", false},
		{"wrong_prefix_1234567890123456789012", false},
		{"", false},
		{"tk_live_", false},
		{"tk_live_123456789012345678901234567890123", false}, // too long
	}

	for _, tt := range tests {
		got := ValidateAPIKeyFormat(tt.key, cfg)
		if got != tt.want {
			t.Errorf("ValidateAPIKeyFormat(%q) = %v, want %v", tt.key, got, tt.want)
		}
	}
}

func TestExtractAPIKeyPrefix(t *testing.T) {
	cfg := APIKeyConfig{
		Prefix:       "tk_live_",
		PrefixLength: 12,
	}

	tests := []struct {
		name string
		key  string
		want string
	}{
		{"valid key", "tk_live_abcd1234567890123456789012345678", "tk_live_abcd..."},
		{"short key", "tk_live_abc", "tk_live_abc"},
		{"exact length", "tk_live_abcd", "tk_live_abcd"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractAPIKeyPrefix(tt.key, cfg)
			if got != tt.want {
				t.Errorf("ExtractAPIKeyPrefix(%q) = %q, want %q", tt.key, got, tt.want)
			}
		})
	}
}

func TestAPIKeyConfig_Defaults(t *testing.T) {
	cfg := APIKeyConfig{}

	if cfg.prefix() != DefaultAPIKeyPrefix {
		t.Errorf("prefix() = %q, want %q", cfg.prefix(), DefaultAPIKeyPrefix)
	}
	if cfg.randomLength() != DefaultAPIKeyLength {
		t.Errorf("randomLength() = %d, want %d", cfg.randomLength(), DefaultAPIKeyLength)
	}
	if cfg.prefixLength() != DefaultAPIKeyPrefixLength {
		t.Errorf("prefixLength() = %d, want %d", cfg.prefixLength(), DefaultAPIKeyPrefixLength)
	}
}
