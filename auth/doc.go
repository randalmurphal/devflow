// Package auth provides authentication utilities for CLI applications.
//
// This package includes:
//   - JWT token generation and validation with customizable claims
//   - API key generation with configurable prefixes
//   - Token hashing utilities
//
// # JWT Usage
//
// Configure and use JWT tokens:
//
//	cfg := auth.JWTConfig{
//	    Secret:         []byte("your-32-byte-or-longer-secret-key"),
//	    Issuer:         "my-app",
//	    AccessTokenTTL: 15 * time.Minute,
//	}
//
//	// Generate with default claims
//	token, err := auth.GenerateAccessToken(cfg, "user-123")
//
//	// Validate
//	claims, err := auth.ValidateAccessToken(cfg, token)
//
// # Custom Claims
//
// Extend BaseClaims for application-specific claims:
//
//	type MyClaims struct {
//	    auth.BaseClaims
//	    TenantID string `json:"tid"`
//	    Role     string `json:"role"`
//	}
//
//	// Generate with custom claims
//	token, err := auth.GenerateAccessTokenWithClaims(cfg, func(base auth.BaseClaims) MyClaims {
//	    return MyClaims{
//	        BaseClaims: base,
//	        TenantID:   "tenant-123",
//	        Role:       "admin",
//	    }
//	})
//
//	// Validate with custom claims
//	claims, err := auth.ValidateAccessTokenAs[MyClaims](cfg, token)
//
// # API Keys
//
// Generate API keys with configurable prefixes:
//
//	cfg := auth.APIKeyConfig{
//	    Prefix:       "myapp_live_",
//	    RandomLength: 32,
//	}
//
//	key, err := auth.GenerateAPIKey(cfg)
//	// key.Secret: "myapp_live_aBc123..."
//	// key.Hash: SHA-256 hash for storage
//	// key.Prefix: "myapp_live_aBc1..." (display prefix)
//
// # Token Hashing
//
// Hash tokens for secure storage:
//
//	hash := auth.HashToken(secretToken)
//	// Store hash in database, verify by hashing incoming token
package auth
