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
func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, ErrConnectionFailed) {
		return true
	}

	errStr := strings.ToLower(err.Error())
	return strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "dial tcp")
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
