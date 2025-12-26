package auth

import (
	"fmt"
	"strings"

	nanoid "github.com/matoous/go-nanoid/v2"
)

// Default API key configuration.
const (
	DefaultAPIKeyPrefix       = "key_"
	DefaultAPIKeyLength       = 32
	DefaultAPIKeyPrefixLength = 12
)

// APIKeyConfig holds configuration for API key generation.
type APIKeyConfig struct {
	// Prefix is prepended to all keys (e.g., "myapp_live_").
	// Defaults to "key_" if empty.
	Prefix string

	// RandomLength is the length of the random part.
	// Defaults to 32 if zero.
	RandomLength int

	// PrefixLength is how many characters to show in the display prefix.
	// Defaults to 12 if zero.
	PrefixLength int
}

func (c APIKeyConfig) prefix() string {
	if c.Prefix == "" {
		return DefaultAPIKeyPrefix
	}
	return c.Prefix
}

func (c APIKeyConfig) randomLength() int {
	if c.RandomLength == 0 {
		return DefaultAPIKeyLength
	}
	return c.RandomLength
}

func (c APIKeyConfig) prefixLength() int {
	if c.PrefixLength == 0 {
		return DefaultAPIKeyPrefixLength
	}
	return c.PrefixLength
}

// APIKeyWithSecret contains the full API key (only available at creation).
type APIKeyWithSecret struct {
	// ID is a unique identifier for the key.
	ID string

	// Secret is the full API key (e.g., "myapp_live_xxxx...").
	// Only available at creation time.
	Secret string

	// Prefix is the display prefix (e.g., "myapp_live_xxxx...").
	Prefix string

	// Hash is the SHA-256 hash of the full key for storage.
	Hash string
}

// GenerateAPIKey creates a new API key with the given configuration.
func GenerateAPIKey(cfg APIKeyConfig) (*APIKeyWithSecret, error) {
	// Generate random part (base62)
	random, err := nanoid.Generate(
		"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
		cfg.randomLength(),
	)
	if err != nil {
		return nil, fmt.Errorf("generate api key: %w", err)
	}

	secret := cfg.prefix() + random

	// Calculate display prefix
	prefixLen := cfg.prefixLength()
	var displayPrefix string
	if len(secret) > prefixLen {
		displayPrefix = secret[:prefixLen] + "..."
	} else {
		displayPrefix = secret
	}

	hash := HashToken(secret)

	id, err := nanoid.New()
	if err != nil {
		return nil, fmt.Errorf("generate api key id: %w", err)
	}

	return &APIKeyWithSecret{
		ID:     "key_" + id,
		Secret: secret,
		Prefix: displayPrefix,
		Hash:   hash,
	}, nil
}

// ValidateAPIKeyFormat checks if a string matches the expected API key format.
func ValidateAPIKeyFormat(key string, cfg APIKeyConfig) bool {
	prefix := cfg.prefix()
	expectedLen := len(prefix) + cfg.randomLength()
	return strings.HasPrefix(key, prefix) && len(key) == expectedLen
}

// ExtractAPIKeyPrefix gets the display prefix from a full key.
func ExtractAPIKeyPrefix(key string, cfg APIKeyConfig) string {
	prefixLen := cfg.prefixLength()
	if len(key) <= prefixLen {
		return key
	}
	return key[:prefixLen] + "..."
}
