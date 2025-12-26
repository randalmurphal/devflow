// Package errors provides CLI error patterns with user-friendly messaging.
//
// Core types:
//   - CLIError: Wraps errors with message, suggestion, and details
//   - ErrorMessenger: Interface for customizing error messages
//
// Sentinel errors for common scenarios:
//   - ErrNotAuthenticated: User needs to log in
//   - ErrSessionExpired: Auth token has expired
//   - ErrNotInGitRepo: Command requires a git repository
//   - ErrNoProjectLinked: No project is configured
//   - ErrConnectionFailed: Server is unreachable
//   - ErrPermissionDenied: Insufficient permissions
//
// Example usage:
//
//	// Wrap an auth error with default messages
//	if err := doAuthThing(); err != nil {
//	    return errors.WrapAuthError(err)
//	}
//
//	// Wrap with custom messages
//	type MyMessenger struct{}
//	func (m MyMessenger) AuthErrorMessage() (string, string) {
//	    return "Please log in.", "Run 'myapp login' to authenticate."
//	}
//
//	wrapped := errors.WrapAuthError(err, errors.WithMessenger(MyMessenger{}))
//
//	// Check error types
//	if errors.IsAuthError(err) {
//	    // Handle auth-related error
//	}
package errors
