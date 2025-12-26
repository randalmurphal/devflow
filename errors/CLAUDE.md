# errors package

CLI error patterns with user-friendly messaging and customizable suggestions.

## Quick Reference

| Type | Purpose |
|------|---------|
| `CLIError` | Wraps error with message, suggestion, details |
| `ErrorMessenger` | Interface for customizing error messages |
| `DefaultMessenger` | Default implementation of ErrorMessenger |
| `WrapConfig` | Configuration for error wrapping |

## Sentinel Errors

| Error | When |
|-------|------|
| `ErrNotAuthenticated` | User needs to log in |
| `ErrSessionExpired` | Auth token has expired |
| `ErrNotInGitRepo` | Command requires git repository |
| `ErrNoProjectLinked` | No project is configured |
| `ErrConnectionFailed` | Server is unreachable |
| `ErrPermissionDenied` | Insufficient permissions |

## Wrap Functions

| Function | Purpose |
|----------|---------|
| `WrapAuthError(err, opts...)` | Wrap auth-related errors |
| `WrapConnectionError(err, url, opts...)` | Wrap connection errors |
| `WrapProjectError(err, opts...)` | Wrap project errors |
| `NewNotInGitRepoError(opts...)` | Create git repo error |
| `NewNoProjectLinkedError(opts...)` | Create no-project error |
| `NewNotAuthenticatedError(opts...)` | Create auth error |

## Predicates

| Function | Checks |
|----------|--------|
| `IsAuthError(err)` | Auth or session errors |
| `IsConnectionError(err)` | Connection failures |
| `IsProjectError(err)` | Project-related errors |
| `IsPermissionError(err)` | Permission denied |

## Custom Messages

```go
// Implement ErrorMessenger for custom messages
type MyMessenger struct{}

func (m MyMessenger) AuthErrorMessage() (string, string) {
    return "Please log in.", "Run 'myapp login' to authenticate."
}
// ... implement other methods

// Use with wrapping functions
err := errors.WrapAuthError(originalErr, errors.WithMessenger(MyMessenger{}))
```

## File Structure

```
errors/
├── doc.go           # Package documentation
├── errors.go        # Sentinel errors
├── cli.go           # CLIError, wrapping functions, ErrorMessenger
├── predicates.go    # IsAuthError, IsConnectionError, etc.
└── errors_test.go   # Tests
```
