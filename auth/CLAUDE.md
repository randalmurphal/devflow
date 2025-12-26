# auth package

JWT and API key authentication utilities for CLI applications.

## Quick Reference

| Type | Purpose |
|------|---------|
| `JWTConfig` | Configuration for JWT generation (secret, issuer, TTL) |
| `BaseClaims` | Standard JWT claims, embed for custom claims |
| `TokenPair` | Access token + refresh token bundle |
| `APIKeyConfig` | Configuration for API key generation (prefix, length) |
| `APIKeyWithSecret` | Generated API key with ID, secret, prefix, hash |

## JWT Functions

| Function | Purpose |
|----------|---------|
| `GenerateAccessToken(cfg, subject)` | Create JWT with subject |
| `GenerateAccessTokenWithClaims[T](cfg, builder)` | Create JWT with custom claims |
| `ValidateAccessToken(cfg, token)` | Validate and parse JWT |
| `ValidateAccessTokenAs(cfg, token, claims)` | Validate into custom claims pointer |
| `GenerateRefreshToken()` | Create opaque refresh token + hash |
| `GenerateTokenPair(cfg, subject)` | Create access + refresh tokens |
| `GenerateTokenPairWithClaims[T](cfg, builder)` | Create pair with custom claims |

## API Key Functions

| Function | Purpose |
|----------|---------|
| `GenerateAPIKey(cfg)` | Create new API key with ID, secret, hash |
| `ValidateAPIKeyFormat(key, cfg)` | Check if key matches expected format |
| `ExtractAPIKeyPrefix(key, cfg)` | Get display prefix from full key |
| `HashToken(token)` | SHA-256 hash for secure storage |

## Errors

| Error | When |
|-------|------|
| `ErrInvalidToken` | Token malformed or bad signature |
| `ErrTokenExpired` | Token has expired |
| `ErrSecretTooShort` | JWT secret < 32 bytes |
| `ErrInvalidAPIKey` | API key format invalid |

## Custom Claims Pattern

```go
// Define custom claims embedding BaseClaims
type MyClaims struct {
    auth.BaseClaims
    TenantID string `json:"tid"`
    Role     string `json:"role"`
}

// Generate with custom claims
token, err := auth.GenerateAccessTokenWithClaims(cfg, func(base auth.BaseClaims) MyClaims {
    base.Subject = userID
    return MyClaims{
        BaseClaims: base,
        TenantID:   tenantID,
        Role:       "admin",
    }
})

// Validate with custom claims
claims := &MyClaims{}
if err := auth.ValidateAccessTokenAs(cfg, token, claims); err != nil {
    // Handle ErrInvalidToken or ErrTokenExpired
}
fmt.Println(claims.TenantID, claims.Role)
```

## Application-Specific Wrappers

Applications should create thin wrappers with their own defaults:

```go
package auth

import devauth "github.com/randalmurphal/devflow/auth"

const (
    APIKeyPrefix = "myapp_live_"
    JWTIssuer    = "my-application"
)

var jwtCfg = devauth.JWTConfig{
    Secret:         []byte(os.Getenv("JWT_SECRET")),
    Issuer:         JWTIssuer,
    AccessTokenTTL: 15 * time.Minute,
}

var apiKeyCfg = devauth.APIKeyConfig{
    Prefix:       APIKeyPrefix,
    RandomLength: 32,
}

// Re-export errors
var (
    ErrInvalidToken = devauth.ErrInvalidToken
    ErrTokenExpired = devauth.ErrTokenExpired
)

func GenerateAPIKey() (*devauth.APIKeyWithSecret, error) {
    return devauth.GenerateAPIKey(apiKeyCfg)
}
```

## File Structure

```
auth/
├── doc.go           # Package documentation
├── errors.go        # Sentinel errors
├── hash.go          # HashToken utility
├── jwt.go           # JWT generation/validation
├── apikey.go        # API key generation
├── jwt_test.go      # JWT tests
├── apikey_test.go   # API key tests
└── hash_test.go     # Hash tests
```
