// Package http provides shared HTTP client patterns for integration clients.
package http

import (
	"errors"
	"fmt"
	"time"
)

// Standard sentinel errors for integration clients.
var (
	// ErrNotFound indicates the requested resource does not exist.
	ErrNotFound = errors.New("resource not found")

	// ErrUnauthorized indicates invalid or missing authentication.
	ErrUnauthorized = errors.New("authentication failed")

	// ErrForbidden indicates the user lacks permission for the operation.
	ErrForbidden = errors.New("permission denied")

	// ErrRateLimited indicates the API rate limit was exceeded.
	ErrRateLimited = errors.New("rate limit exceeded")

	// ErrBadRequest indicates the request was malformed.
	ErrBadRequest = errors.New("bad request")

	// ErrServerError indicates a server-side error occurred.
	ErrServerError = errors.New("server error")
)

// APIError represents an error from an external API.
type APIError struct {
	// Service is the name of the integration (e.g., "jira", "gitlab").
	Service string

	// StatusCode is the HTTP status code returned.
	StatusCode int

	// Message is the error message from the API.
	Message string

	// Endpoint is the API endpoint that was called.
	Endpoint string

	// RequestID is the request ID for debugging (if available).
	RequestID string
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.RequestID != "" {
		return fmt.Sprintf("%s API error (%d) at %s [%s]: %s",
			e.Service, e.StatusCode, e.Endpoint, e.RequestID, e.Message)
	}
	return fmt.Sprintf("%s API error (%d) at %s: %s",
		e.Service, e.StatusCode, e.Endpoint, e.Message)
}

// Unwrap returns the underlying sentinel error based on status code.
func (e *APIError) Unwrap() error {
	switch e.StatusCode {
	case 400:
		return ErrBadRequest
	case 401:
		return ErrUnauthorized
	case 403:
		return ErrForbidden
	case 404:
		return ErrNotFound
	case 429:
		return ErrRateLimited
	default:
		if e.StatusCode >= 500 {
			return ErrServerError
		}
		return nil
	}
}

// AuthError represents an authentication failure.
type AuthError struct {
	// Service is the integration that failed authentication.
	Service string

	// Reason explains why authentication failed.
	Reason string
}

// Error implements the error interface.
func (e *AuthError) Error() string {
	return fmt.Sprintf("%s authentication failed: %s", e.Service, e.Reason)
}

// Unwrap returns ErrUnauthorized.
func (e *AuthError) Unwrap() error {
	return ErrUnauthorized
}

// RateLimitError represents a rate limit being exceeded.
type RateLimitError struct {
	// Service is the integration that rate limited.
	Service string

	// RetryAfter is how long to wait before retrying.
	RetryAfter time.Duration

	// Limit is the rate limit that was exceeded (if known).
	Limit int

	// Remaining is how many requests remain (usually 0).
	Remaining int
}

// Error implements the error interface.
func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("%s rate limit exceeded, retry after %s", e.Service, e.RetryAfter)
	}
	return fmt.Sprintf("%s rate limit exceeded", e.Service)
}

// Unwrap returns ErrRateLimited.
func (e *RateLimitError) Unwrap() error {
	return ErrRateLimited
}

// ValidationError represents validation failures for request data.
type ValidationError struct {
	// Service is the integration that rejected the request.
	Service string

	// Field is the field that failed validation.
	Field string

	// Message explains the validation failure.
	Message string
}

// Error implements the error interface.
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("%s validation error on %s: %s", e.Service, e.Field, e.Message)
	}
	return fmt.Sprintf("%s validation error: %s", e.Service, e.Message)
}

// Unwrap returns ErrBadRequest.
func (e *ValidationError) Unwrap() error {
	return ErrBadRequest
}

// IsNotFound reports whether the error indicates a resource was not found.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsUnauthorized reports whether the error indicates authentication failed.
func IsUnauthorized(err error) bool {
	return errors.Is(err, ErrUnauthorized)
}

// IsForbidden reports whether the error indicates permission was denied.
func IsForbidden(err error) bool {
	return errors.Is(err, ErrForbidden)
}

// IsRateLimited reports whether the error indicates rate limiting.
func IsRateLimited(err error) bool {
	return errors.Is(err, ErrRateLimited)
}

// IsRetryable reports whether the error is transient and should be retried.
func IsRetryable(err error) bool {
	if errors.Is(err, ErrRateLimited) || errors.Is(err, ErrServerError) {
		return true
	}

	var apiErr *APIError
	if errors.As(err, &apiErr) {
		// 5xx errors are retryable
		return apiErr.StatusCode >= 500 && apiErr.StatusCode < 600
	}

	return false
}
