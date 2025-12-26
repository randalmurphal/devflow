package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	nanoid "github.com/matoous/go-nanoid/v2"
)

// Default token lifetimes.
const (
	DefaultAccessTokenTTL  = 15 * time.Minute
	DefaultRefreshTokenTTL = 7 * 24 * time.Hour
)

// JWTConfig holds configuration for JWT generation and validation.
type JWTConfig struct {
	// Secret is the HMAC signing key (must be at least 32 bytes).
	Secret []byte

	// Issuer is the token issuer (e.g., "my-app").
	Issuer string

	// AccessTokenTTL is the lifetime of access tokens.
	// Defaults to DefaultAccessTokenTTL (15 minutes) if zero.
	AccessTokenTTL time.Duration

	// RefreshTokenTTL is the lifetime of refresh tokens.
	// Defaults to DefaultRefreshTokenTTL (7 days) if zero.
	RefreshTokenTTL time.Duration
}

func (c JWTConfig) accessTTL() time.Duration {
	if c.AccessTokenTTL == 0 {
		return DefaultAccessTokenTTL
	}
	return c.AccessTokenTTL
}

func (c JWTConfig) refreshTTL() time.Duration {
	if c.RefreshTokenTTL == 0 {
		return DefaultRefreshTokenTTL
	}
	return c.RefreshTokenTTL
}

// BaseClaims represents the standard JWT claims.
// Embed this in custom claims types for application-specific data.
type BaseClaims struct {
	jwt.RegisteredClaims
}

// TokenPair contains an access token and refresh token.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds until access token expires
}

// GenerateAccessToken creates a new JWT access token with the given subject.
func GenerateAccessToken(cfg JWTConfig, subject string) (string, error) {
	return GenerateAccessTokenWithClaims(cfg, func(base BaseClaims) BaseClaims {
		base.Subject = subject
		return base
	})
}

// GenerateAccessTokenWithClaims creates a JWT with custom claims.
// The builder function receives a BaseClaims with standard fields pre-populated.
func GenerateAccessTokenWithClaims[T jwt.Claims](cfg JWTConfig, builder func(BaseClaims) T) (string, error) {
	if len(cfg.Secret) < 32 {
		return "", ErrSecretTooShort
	}

	tokenID, err := nanoid.New()
	if err != nil {
		return "", fmt.Errorf("generate token ID: %w", err)
	}

	now := time.Now()
	base := BaseClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    cfg.Issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.accessTTL())),
			ID:        tokenID,
		},
	}

	claims := builder(base)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(cfg.Secret)
}

// ValidateAccessToken parses and validates a JWT, returning BaseClaims.
func ValidateAccessToken(cfg JWTConfig, tokenString string) (*BaseClaims, error) {
	claims := &BaseClaims{}
	if err := validateAccessTokenInto(cfg, tokenString, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

// ValidateAccessTokenAs parses and validates a JWT into the provided claims pointer.
// Pass a pointer to your custom claims type.
//
// Example:
//
//	claims := &MyClaims{}
//	if err := auth.ValidateAccessTokenAs(cfg, token, claims); err != nil {
//	    return err
//	}
//	fmt.Println(claims.TenantID)
func ValidateAccessTokenAs(cfg JWTConfig, tokenString string, claims jwt.Claims) error {
	return validateAccessTokenInto(cfg, tokenString, claims)
}

func validateAccessTokenInto(cfg JWTConfig, tokenString string, claims jwt.Claims) error {
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return cfg.Secret, nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return ErrTokenExpired
		}
		return ErrInvalidToken
	}

	if !token.Valid {
		return ErrInvalidToken
	}

	// Verify issuer if configured
	if cfg.Issuer != "" {
		issuer, err := token.Claims.GetIssuer()
		if err != nil || issuer != cfg.Issuer {
			return ErrInvalidToken
		}
	}

	return nil
}

// GenerateRefreshToken creates a new opaque refresh token.
// Returns the token (to give to client) and its hash (for storage).
func GenerateRefreshToken() (token, hash string, err error) {
	token, err = nanoid.Generate(
		"0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz",
		64,
	)
	if err != nil {
		return "", "", fmt.Errorf("generate refresh token: %w", err)
	}

	hash = HashToken(token)
	return token, hash, nil
}

// GenerateTokenPair creates both access and refresh tokens.
func GenerateTokenPair(cfg JWTConfig, subject string) (*TokenPair, string, error) {
	return GenerateTokenPairWithClaims(cfg, func(base BaseClaims) BaseClaims {
		base.Subject = subject
		return base
	})
}

// GenerateTokenPairWithClaims creates both tokens with custom claims.
func GenerateTokenPairWithClaims[T jwt.Claims](cfg JWTConfig, builder func(BaseClaims) T) (*TokenPair, string, error) {
	accessToken, err := GenerateAccessTokenWithClaims(cfg, builder)
	if err != nil {
		return nil, "", err
	}

	refreshToken, refreshHash, err := GenerateRefreshToken()
	if err != nil {
		return nil, "", err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(cfg.accessTTL().Seconds()),
	}, refreshHash, nil
}
