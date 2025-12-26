package auth

import (
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestGenerateAccessToken(t *testing.T) {
	cfg := JWTConfig{
		Secret: []byte("this-is-a-test-secret-key-32-bytes!"),
		Issuer: "test-app",
	}

	t.Run("basic generation", func(t *testing.T) {
		token, err := GenerateAccessToken(cfg, "user-123")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}
		if token == "" {
			t.Fatal("GenerateAccessToken() returned empty token")
		}
	})

	t.Run("validate generated token", func(t *testing.T) {
		token, err := GenerateAccessToken(cfg, "user-123")
		if err != nil {
			t.Fatalf("GenerateAccessToken() error = %v", err)
		}

		claims, err := ValidateAccessToken(cfg, token)
		if err != nil {
			t.Fatalf("ValidateAccessToken() error = %v", err)
		}
		if claims.Subject != "user-123" {
			t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
		}
		if claims.Issuer != "test-app" {
			t.Errorf("Issuer = %q, want %q", claims.Issuer, "test-app")
		}
	})

	t.Run("secret too short", func(t *testing.T) {
		shortCfg := JWTConfig{Secret: []byte("short")}
		_, err := GenerateAccessToken(shortCfg, "user-123")
		if !errors.Is(err, ErrSecretTooShort) {
			t.Errorf("error = %v, want ErrSecretTooShort", err)
		}
	})
}

// CustomClaims for testing generic claims support.
type CustomClaims struct {
	BaseClaims
	TenantID string `json:"tid"`
	Role     string `json:"role"`
}

func TestGenerateAccessTokenWithClaims(t *testing.T) {
	cfg := JWTConfig{
		Secret: []byte("this-is-a-test-secret-key-32-bytes!"),
		Issuer: "test-app",
	}

	t.Run("custom claims", func(t *testing.T) {
		token, err := GenerateAccessTokenWithClaims(cfg, func(base BaseClaims) CustomClaims {
			base.Subject = "user-123"
			return CustomClaims{
				BaseClaims: base,
				TenantID:   "tenant-456",
				Role:       "admin",
			}
		})
		if err != nil {
			t.Fatalf("GenerateAccessTokenWithClaims() error = %v", err)
		}

		claims := &CustomClaims{}
		if err := ValidateAccessTokenAs(cfg, token, claims); err != nil {
			t.Fatalf("ValidateAccessTokenAs() error = %v", err)
		}
		if claims.Subject != "user-123" {
			t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
		}
		if claims.TenantID != "tenant-456" {
			t.Errorf("TenantID = %q, want %q", claims.TenantID, "tenant-456")
		}
		if claims.Role != "admin" {
			t.Errorf("Role = %q, want %q", claims.Role, "admin")
		}
	})
}

func TestValidateAccessToken(t *testing.T) {
	cfg := JWTConfig{
		Secret: []byte("this-is-a-test-secret-key-32-bytes!"),
		Issuer: "test-app",
	}

	t.Run("invalid token", func(t *testing.T) {
		_, err := ValidateAccessToken(cfg, "invalid-token")
		if !errors.Is(err, ErrInvalidToken) {
			t.Errorf("error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("wrong secret", func(t *testing.T) {
		token, _ := GenerateAccessToken(cfg, "user-123")

		wrongCfg := JWTConfig{
			Secret: []byte("wrong-secret-key-that-is-32-bytes!"),
			Issuer: "test-app",
		}
		_, err := ValidateAccessToken(wrongCfg, token)
		if !errors.Is(err, ErrInvalidToken) {
			t.Errorf("error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("wrong issuer", func(t *testing.T) {
		token, _ := GenerateAccessToken(cfg, "user-123")

		wrongCfg := JWTConfig{
			Secret: cfg.Secret,
			Issuer: "different-app",
		}
		_, err := ValidateAccessToken(wrongCfg, token)
		if !errors.Is(err, ErrInvalidToken) {
			t.Errorf("error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("empty issuer accepts any", func(t *testing.T) {
		token, _ := GenerateAccessToken(cfg, "user-123")

		noIssuerCfg := JWTConfig{
			Secret: cfg.Secret,
			Issuer: "",
		}
		claims, err := ValidateAccessToken(noIssuerCfg, token)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		if claims == nil {
			t.Error("expected claims, got nil")
		}
	})

	t.Run("expired token", func(t *testing.T) {
		pastTime := time.Now().Add(-2 * time.Hour)
		claims := BaseClaims{
			RegisteredClaims: jwt.RegisteredClaims{
				Subject:   "user-123",
				Issuer:    "test-app",
				ExpiresAt: jwt.NewNumericDate(pastTime.Add(1 * time.Hour)),
				IssuedAt:  jwt.NewNumericDate(pastTime),
				ID:        "test-id",
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, _ := token.SignedString(cfg.Secret)

		_, err := ValidateAccessToken(cfg, tokenString)
		if !errors.Is(err, ErrTokenExpired) {
			t.Errorf("error = %v, want ErrTokenExpired", err)
		}
		// Should NOT be ErrInvalidToken
		if errors.Is(err, ErrInvalidToken) {
			t.Error("expired token should return ErrTokenExpired, not ErrInvalidToken")
		}
	})

	t.Run("empty token", func(t *testing.T) {
		_, err := ValidateAccessToken(cfg, "")
		if !errors.Is(err, ErrInvalidToken) {
			t.Errorf("error = %v, want ErrInvalidToken", err)
		}
	})

	t.Run("malformed token", func(t *testing.T) {
		_, err := ValidateAccessToken(cfg, "not.a.valid.jwt")
		if !errors.Is(err, ErrInvalidToken) {
			t.Errorf("error = %v, want ErrInvalidToken", err)
		}
	})
}

func TestGenerateRefreshToken(t *testing.T) {
	token, hash, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}
	if token == "" {
		t.Error("token is empty")
	}
	if hash == "" {
		t.Error("hash is empty")
	}
	if token == hash {
		t.Error("token and hash should be different")
	}

	// Verify hash is consistent
	rehash := HashToken(token)
	if rehash != hash {
		t.Errorf("HashToken() = %q, want %q", rehash, hash)
	}
}

func TestGenerateRefreshToken_Uniqueness(t *testing.T) {
	tokens := make(map[string]bool)
	hashes := make(map[string]bool)

	for i := 0; i < 10; i++ {
		token, hash, err := GenerateRefreshToken()
		if err != nil {
			t.Fatalf("GenerateRefreshToken() error = %v", err)
		}
		if tokens[token] {
			t.Error("duplicate token generated")
		}
		tokens[token] = true
		if hashes[hash] {
			t.Error("duplicate hash generated")
		}
		hashes[hash] = true
	}
}

func TestGenerateTokenPair(t *testing.T) {
	cfg := JWTConfig{
		Secret:         []byte("this-is-a-test-secret-key-32-bytes!"),
		Issuer:         "test-app",
		AccessTokenTTL: 15 * time.Minute,
	}

	pair, refreshHash, err := GenerateTokenPair(cfg, "user-123")
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if pair.RefreshToken == "" {
		t.Error("RefreshToken is empty")
	}
	if pair.ExpiresIn != int64((15 * time.Minute).Seconds()) {
		t.Errorf("ExpiresIn = %d, want %d", pair.ExpiresIn, int64((15*time.Minute).Seconds()))
	}
	if refreshHash == "" {
		t.Error("refreshHash is empty")
	}

	// Verify access token is valid
	claims, err := ValidateAccessToken(cfg, pair.AccessToken)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("Subject = %q, want %q", claims.Subject, "user-123")
	}

	// Verify refresh token hash
	if HashToken(pair.RefreshToken) != refreshHash {
		t.Error("refresh token hash mismatch")
	}
}

func TestGenerateTokenPair_InvalidSecret(t *testing.T) {
	shortCfg := JWTConfig{
		Secret: []byte("short"),
		Issuer: "test",
	}

	pair, hash, err := GenerateTokenPair(shortCfg, "user-123")
	if err == nil {
		t.Error("expected error for short secret")
	}
	if pair != nil {
		t.Error("expected nil pair on error")
	}
	if hash != "" {
		t.Error("expected empty hash on error")
	}
}

func TestGenerateTokenPairWithClaims(t *testing.T) {
	cfg := JWTConfig{
		Secret:         []byte("this-is-a-test-secret-key-32-bytes!"),
		Issuer:         "test-app",
		AccessTokenTTL: 15 * time.Minute,
	}

	pair, refreshHash, err := GenerateTokenPairWithClaims(cfg, func(base BaseClaims) CustomClaims {
		base.Subject = "user-123"
		return CustomClaims{
			BaseClaims: base,
			TenantID:   "tenant-456",
			Role:       "user",
		}
	})
	if err != nil {
		t.Fatalf("GenerateTokenPairWithClaims() error = %v", err)
	}
	if pair.AccessToken == "" {
		t.Error("AccessToken is empty")
	}
	if refreshHash == "" {
		t.Error("refreshHash is empty")
	}

	// Verify custom claims
	claims := &CustomClaims{}
	if err := ValidateAccessTokenAs(cfg, pair.AccessToken, claims); err != nil {
		t.Fatalf("ValidateAccessTokenAs() error = %v", err)
	}
	if claims.TenantID != "tenant-456" {
		t.Errorf("TenantID = %q, want %q", claims.TenantID, "tenant-456")
	}
}

func TestJWTConfig_Defaults(t *testing.T) {
	cfg := JWTConfig{
		Secret: []byte("this-is-a-test-secret-key-32-bytes!"),
	}

	// Verify defaults are applied
	if cfg.accessTTL() != DefaultAccessTokenTTL {
		t.Errorf("accessTTL() = %v, want %v", cfg.accessTTL(), DefaultAccessTokenTTL)
	}
	if cfg.refreshTTL() != DefaultRefreshTokenTTL {
		t.Errorf("refreshTTL() = %v, want %v", cfg.refreshTTL(), DefaultRefreshTokenTTL)
	}

	// Custom TTL
	cfg.AccessTokenTTL = 30 * time.Minute
	if cfg.accessTTL() != 30*time.Minute {
		t.Errorf("accessTTL() = %v, want 30m", cfg.accessTTL())
	}
}
