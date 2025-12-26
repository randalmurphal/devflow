package auth

import "errors"

// Authentication errors.
var (
	// ErrInvalidToken indicates the token is malformed or has an invalid signature.
	ErrInvalidToken = errors.New("invalid token")

	// ErrTokenExpired indicates the token has expired.
	ErrTokenExpired = errors.New("token expired")

	// ErrSecretTooShort indicates the JWT secret is too short.
	ErrSecretTooShort = errors.New("JWT secret must be at least 32 bytes")

	// ErrInvalidAPIKey indicates the API key format is invalid.
	ErrInvalidAPIKey = errors.New("invalid API key format")
)
