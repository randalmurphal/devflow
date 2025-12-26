package errors

import "errors"

// Common CLI errors with actionable guidance.
var (
	// ErrNotAuthenticated indicates the user needs to log in.
	ErrNotAuthenticated = errors.New("not authenticated")

	// ErrNotInGitRepo indicates the command requires a git repository.
	ErrNotInGitRepo = errors.New("not in a git repository")

	// ErrNoProjectLinked indicates no project is configured.
	ErrNoProjectLinked = errors.New("no project linked")

	// ErrConnectionFailed indicates the server is unreachable.
	ErrConnectionFailed = errors.New("connection failed")

	// ErrPermissionDenied indicates insufficient permissions.
	ErrPermissionDenied = errors.New("permission denied")

	// ErrSessionExpired indicates the auth token has expired.
	ErrSessionExpired = errors.New("session expired")
)
