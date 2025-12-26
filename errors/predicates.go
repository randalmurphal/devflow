package errors

import (
	"errors"
	"strings"
)

// IsAuthError checks if an error is authentication-related.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrNotAuthenticated) || errors.Is(err, ErrSessionExpired) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "unauthenticated") ||
		strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "401")
}

// IsConnectionError checks if an error is connection-related.
// This includes TLS errors, timeouts, and network connectivity issues.
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrConnectionFailed) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	// Network connectivity
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "dial tcp") {
		return true
	}
	// TLS/certificate errors (consistent with WrapConnectionError)
	if strings.Contains(errStr, "certificate") ||
		strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "x509") {
		return true
	}
	// Timeout errors (consistent with WrapConnectionError)
	if strings.Contains(errStr, "timeout") ||
		strings.Contains(errStr, "deadline exceeded") {
		return true
	}
	return false
}

// IsProjectError checks if an error is project-related.
func IsProjectError(err error) bool {
	if err == nil {
		return false
	}

	return errors.Is(err, ErrNoProjectLinked)
}

// IsPermissionError checks if an error is permission-related.
func IsPermissionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrPermissionDenied) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "permission denied") ||
		strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "403")
}
