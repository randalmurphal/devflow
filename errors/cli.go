package errors

import (
	"fmt"
	"strings"
)

// CLIError wraps an error with user-friendly context and suggestions.
type CLIError struct {
	// Err is the underlying error
	Err error

	// Message is a user-friendly description of what went wrong
	Message string

	// Suggestion is an actionable hint for the user
	Suggestion string

	// Details provides additional context (optional)
	Details string
}

func (e *CLIError) Error() string {
	var sb strings.Builder
	sb.WriteString(e.Message)

	if e.Details != "" {
		sb.WriteString("\n")
		sb.WriteString(e.Details)
	}

	if e.Suggestion != "" {
		sb.WriteString("\n\n")
		sb.WriteString(e.Suggestion)
	}

	return sb.String()
}

func (e *CLIError) Unwrap() error {
	return e.Err
}

// ErrorMessenger provides customizable error messages.
// Implement this interface to customize suggestions for your CLI.
type ErrorMessenger interface {
	// AuthErrorMessage returns the message and suggestion for unauthenticated errors.
	AuthErrorMessage() (message, suggestion string)

	// SessionExpiredMessage returns the message and suggestion for expired sessions.
	SessionExpiredMessage() (message, suggestion string)

	// PermissionDeniedMessage returns the message and suggestion for permission errors.
	PermissionDeniedMessage() (message, suggestion string)

	// ConnectionErrorMessage returns the message and suggestion for connection errors.
	// The serverURL parameter is the URL that failed to connect.
	ConnectionErrorMessage(serverURL string) (message, suggestion string)

	// TLSErrorMessage returns the message and suggestion for TLS/certificate errors.
	TLSErrorMessage(serverURL string) (message, suggestion string)

	// TimeoutErrorMessage returns the message and suggestion for timeout errors.
	TimeoutErrorMessage(serverURL string) (message, suggestion string)

	// NotInGitRepoMessage returns the message and suggestion for git repo errors.
	NotInGitRepoMessage() (message, suggestion string)

	// NoProjectLinkedMessage returns the message and suggestion for project errors.
	NoProjectLinkedMessage() (message, suggestion string)

	// ProjectNotFoundMessage returns the message and suggestion for missing projects.
	ProjectNotFoundMessage() (message, suggestion string)
}

// DefaultMessenger provides default error messages.
type DefaultMessenger struct{}

func (m DefaultMessenger) AuthErrorMessage() (string, string) {
	return "You are not logged in.", "Please authenticate before continuing."
}

func (m DefaultMessenger) SessionExpiredMessage() (string, string) {
	return "Your session has expired.", "Please log in again."
}

func (m DefaultMessenger) PermissionDeniedMessage() (string, string) {
	return "You don't have permission to perform this action.",
		"Contact your administrator for access."
}

func (m DefaultMessenger) ConnectionErrorMessage(serverURL string) (string, string) {
	return fmt.Sprintf("Cannot connect to server at %s", serverURL),
		"Check that:\n  - The server is running\n  - The URL is correct\n  - Your network connection is working"
}

func (m DefaultMessenger) TLSErrorMessage(serverURL string) (string, string) {
	return fmt.Sprintf("TLS/certificate error connecting to %s", serverURL),
		"Check that the server certificate is valid."
}

func (m DefaultMessenger) TimeoutErrorMessage(serverURL string) (string, string) {
	return fmt.Sprintf("Connection to %s timed out", serverURL),
		"The server may be overloaded or unreachable.\nTry again in a moment."
}

func (m DefaultMessenger) NotInGitRepoMessage() (string, string) {
	return "This command must be run from within a git repository.",
		"Run this command from a git repository or specify the project explicitly."
}

func (m DefaultMessenger) NoProjectLinkedMessage() (string, string) {
	return "No project is linked to this repository.",
		"Initialize project linking or specify a project ID."
}

func (m DefaultMessenger) ProjectNotFoundMessage() (string, string) {
	return "Project not found.",
		"Check the project ID or initialize with the correct project."
}

// WrapConfig configures error wrapping behavior.
type WrapConfig struct {
	Messenger ErrorMessenger
}

// Option configures WrapConfig.
type Option func(*WrapConfig)

// WithMessenger sets a custom error messenger.
func WithMessenger(m ErrorMessenger) Option {
	return func(c *WrapConfig) {
		c.Messenger = m
	}
}

func getMessenger(opts []Option) ErrorMessenger {
	cfg := &WrapConfig{
		Messenger: DefaultMessenger{},
	}
	for _, opt := range opts {
		opt(cfg)
	}
	return cfg.Messenger
}

// WrapAuthError wraps authentication-related errors with helpful guidance.
func WrapAuthError(err error, opts ...Option) error {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())
	messenger := getMessenger(opts)

	// Check for token expiration
	if strings.Contains(errStr, "token") && (strings.Contains(errStr, "expired") || strings.Contains(errStr, "invalid")) {
		msg, suggestion := messenger.SessionExpiredMessage()
		return &CLIError{
			Err:        ErrSessionExpired,
			Message:    msg,
			Suggestion: suggestion,
		}
	}

	// Check for unauthenticated
	if strings.Contains(errStr, "unauthenticated") || strings.Contains(errStr, "unauthorized") ||
		strings.Contains(errStr, "401") {
		msg, suggestion := messenger.AuthErrorMessage()
		return &CLIError{
			Err:        ErrNotAuthenticated,
			Message:    msg,
			Suggestion: suggestion,
		}
	}

	// Check for permission denied
	if strings.Contains(errStr, "permission denied") || strings.Contains(errStr, "forbidden") ||
		strings.Contains(errStr, "403") {
		msg, suggestion := messenger.PermissionDeniedMessage()
		return &CLIError{
			Err:        ErrPermissionDenied,
			Message:    msg,
			Suggestion: suggestion,
		}
	}

	return err
}

// WrapConnectionError wraps connection-related errors with helpful guidance.
func WrapConnectionError(err error, serverURL string, opts ...Option) error {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())
	messenger := getMessenger(opts)

	// Check for connection refused
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "no such host") ||
		strings.Contains(errStr, "network is unreachable") ||
		strings.Contains(errStr, "dial tcp") {
		msg, suggestion := messenger.ConnectionErrorMessage(serverURL)
		return &CLIError{
			Err:        ErrConnectionFailed,
			Message:    msg,
			Suggestion: suggestion,
		}
	}

	// Check for TLS/certificate errors
	if strings.Contains(errStr, "certificate") || strings.Contains(errStr, "tls") ||
		strings.Contains(errStr, "x509") {
		msg, suggestion := messenger.TLSErrorMessage(serverURL)
		return &CLIError{
			Err:        ErrConnectionFailed,
			Message:    msg,
			Details:    err.Error(),
			Suggestion: suggestion,
		}
	}

	// Check for timeout
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded") {
		msg, suggestion := messenger.TimeoutErrorMessage(serverURL)
		return &CLIError{
			Err:        ErrConnectionFailed,
			Message:    msg,
			Suggestion: suggestion,
		}
	}

	return err
}

// WrapProjectError wraps project-related errors with helpful guidance.
func WrapProjectError(err error, opts ...Option) error {
	if err == nil {
		return nil
	}

	errStr := strings.ToLower(err.Error())
	messenger := getMessenger(opts)

	// Check for not found
	if strings.Contains(errStr, "not found") || strings.Contains(errStr, "404") {
		msg, suggestion := messenger.ProjectNotFoundMessage()
		return &CLIError{
			Err:        err,
			Message:    msg,
			Suggestion: suggestion,
		}
	}

	return err
}

// NewNotInGitRepoError creates an error for commands that require a git repository.
func NewNotInGitRepoError(opts ...Option) error {
	messenger := getMessenger(opts)
	msg, suggestion := messenger.NotInGitRepoMessage()
	return &CLIError{
		Err:        ErrNotInGitRepo,
		Message:    msg,
		Suggestion: suggestion,
	}
}

// NewNoProjectLinkedError creates an error when no project is configured.
func NewNoProjectLinkedError(opts ...Option) error {
	messenger := getMessenger(opts)
	msg, suggestion := messenger.NoProjectLinkedMessage()
	return &CLIError{
		Err:        ErrNoProjectLinked,
		Message:    msg,
		Suggestion: suggestion,
	}
}

// NewNotAuthenticatedError creates an error for unauthenticated users.
func NewNotAuthenticatedError(opts ...Option) error {
	messenger := getMessenger(opts)
	msg, suggestion := messenger.AuthErrorMessage()
	return &CLIError{
		Err:        ErrNotAuthenticated,
		Message:    msg,
		Suggestion: suggestion,
	}
}
