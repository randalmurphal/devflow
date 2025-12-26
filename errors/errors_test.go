package errors

import (
	"errors"
	"testing"
)

func TestCLIError(t *testing.T) {
	err := &CLIError{
		Err:        ErrNotAuthenticated,
		Message:    "Test message",
		Suggestion: "Test suggestion",
		Details:    "Test details",
	}

	// Check error message format
	errStr := err.Error()
	if !contains(errStr, "Test message") {
		t.Errorf("expected error to contain 'Test message', got %q", errStr)
	}
	if !contains(errStr, "Test details") {
		t.Errorf("expected error to contain 'Test details', got %q", errStr)
	}
	if !contains(errStr, "Test suggestion") {
		t.Errorf("expected error to contain 'Test suggestion', got %q", errStr)
	}

	// Check unwrap
	if !errors.Is(err, ErrNotAuthenticated) {
		t.Error("expected error to unwrap to ErrNotAuthenticated")
	}
}

func TestCLIError_MinimalFields(t *testing.T) {
	err := &CLIError{
		Err:     ErrConnectionFailed,
		Message: "Connection failed",
	}

	errStr := err.Error()
	if errStr != "Connection failed" {
		t.Errorf("expected 'Connection failed', got %q", errStr)
	}
}

func TestWrapAuthError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantType   error
		wantNil    bool
		wantSubstr string
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:       "token expired",
			err:        errors.New("token has expired"),
			wantType:   ErrSessionExpired,
			wantSubstr: "session has expired",
		},
		{
			name:       "invalid token",
			err:        errors.New("invalid token"),
			wantType:   ErrSessionExpired,
			wantSubstr: "session has expired",
		},
		{
			name:       "unauthenticated",
			err:        errors.New("unauthenticated: missing credentials"),
			wantType:   ErrNotAuthenticated,
			wantSubstr: "not logged in",
		},
		{
			name:       "unauthorized 401",
			err:        errors.New("server returned 401 Unauthorized"),
			wantType:   ErrNotAuthenticated,
			wantSubstr: "not logged in",
		},
		{
			name:       "permission denied",
			err:        errors.New("permission denied for resource"),
			wantType:   ErrPermissionDenied,
			wantSubstr: "don't have permission",
		},
		{
			name:       "forbidden 403",
			err:        errors.New("forbidden: 403"),
			wantType:   ErrPermissionDenied,
			wantSubstr: "don't have permission",
		},
		{
			name:     "other error passthrough",
			err:      errors.New("some other error"),
			wantType: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapAuthError(tt.err)

			if tt.wantNil {
				if wrapped != nil {
					t.Errorf("expected nil, got %v", wrapped)
				}
				return
			}

			if tt.wantType == nil {
				if wrapped != tt.err {
					t.Errorf("expected passthrough, got wrapped error")
				}
				return
			}

			if !errors.Is(wrapped, tt.wantType) {
				t.Errorf("expected error to be %v, got %v", tt.wantType, wrapped)
			}
			if !contains(wrapped.Error(), tt.wantSubstr) {
				t.Errorf("expected error to contain %q, got %q", tt.wantSubstr, wrapped.Error())
			}
		})
	}
}

func TestWrapAuthError_CustomMessenger(t *testing.T) {
	messenger := &testMessenger{
		authMsg:        "Custom auth message",
		authSuggestion: "Custom suggestion",
	}

	err := WrapAuthError(errors.New("unauthenticated"), WithMessenger(messenger))

	if !contains(err.Error(), "Custom auth message") {
		t.Errorf("expected custom message, got %q", err.Error())
	}
	if !contains(err.Error(), "Custom suggestion") {
		t.Errorf("expected custom suggestion, got %q", err.Error())
	}
}

func TestWrapConnectionError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		serverURL  string
		wantType   error
		wantNil    bool
		wantSubstr string
	}{
		{
			name:    "nil error",
			err:     nil,
			wantNil: true,
		},
		{
			name:       "connection refused",
			err:        errors.New("dial tcp: connection refused"),
			serverURL:  "http://localhost:8080",
			wantType:   ErrConnectionFailed,
			wantSubstr: "Cannot connect to server",
		},
		{
			name:       "no such host",
			err:        errors.New("dial tcp: no such host"),
			serverURL:  "http://unknown.host",
			wantType:   ErrConnectionFailed,
			wantSubstr: "Cannot connect to server",
		},
		{
			name:       "certificate error",
			err:        errors.New("x509: certificate signed by unknown authority"),
			serverURL:  "https://secure.example.com",
			wantType:   ErrConnectionFailed,
			wantSubstr: "TLS/certificate error",
		},
		{
			name:       "timeout",
			err:        errors.New("context deadline exceeded"),
			serverURL:  "http://slow.server.com",
			wantType:   ErrConnectionFailed,
			wantSubstr: "timed out",
		},
		{
			name:      "other error passthrough",
			err:       errors.New("some other error"),
			serverURL: "http://localhost:8080",
			wantType:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapConnectionError(tt.err, tt.serverURL)

			if tt.wantNil {
				if wrapped != nil {
					t.Errorf("expected nil, got %v", wrapped)
				}
				return
			}

			if tt.wantType == nil {
				if wrapped != tt.err {
					t.Errorf("expected passthrough, got wrapped error")
				}
				return
			}

			if !errors.Is(wrapped, tt.wantType) {
				t.Errorf("expected error to be %v, got %v", tt.wantType, wrapped)
			}
			if !contains(wrapped.Error(), tt.wantSubstr) {
				t.Errorf("expected error to contain %q, got %q", tt.wantSubstr, wrapped.Error())
			}
		})
	}
}

func TestWrapProjectError(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		wantSubstr string
	}{
		{
			name:       "not found",
			err:        errors.New("project not found"),
			wantSubstr: "Project not found",
		},
		{
			name:       "404 error",
			err:        errors.New("server returned 404"),
			wantSubstr: "Project not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			wrapped := WrapProjectError(tt.err)
			if !contains(wrapped.Error(), tt.wantSubstr) {
				t.Errorf("expected error to contain %q, got %q", tt.wantSubstr, wrapped.Error())
			}
		})
	}
}

func TestNewErrors(t *testing.T) {
	t.Run("NewNotInGitRepoError", func(t *testing.T) {
		err := NewNotInGitRepoError()
		if !errors.Is(err, ErrNotInGitRepo) {
			t.Error("expected error to be ErrNotInGitRepo")
		}
		if !contains(err.Error(), "git repository") {
			t.Errorf("expected error to contain 'git repository', got %q", err.Error())
		}
	})

	t.Run("NewNoProjectLinkedError", func(t *testing.T) {
		err := NewNoProjectLinkedError()
		if !errors.Is(err, ErrNoProjectLinked) {
			t.Error("expected error to be ErrNoProjectLinked")
		}
		if !contains(err.Error(), "No project") {
			t.Errorf("expected error to contain 'No project', got %q", err.Error())
		}
	})

	t.Run("NewNotAuthenticatedError", func(t *testing.T) {
		err := NewNotAuthenticatedError()
		if !errors.Is(err, ErrNotAuthenticated) {
			t.Error("expected error to be ErrNotAuthenticated")
		}
		if !contains(err.Error(), "not logged in") {
			t.Errorf("expected error to contain 'not logged in', got %q", err.Error())
		}
	})
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "not authenticated",
			err:  ErrNotAuthenticated,
			want: true,
		},
		{
			name: "session expired",
			err:  ErrSessionExpired,
			want: true,
		},
		{
			name: "wrapped auth error",
			err:  &CLIError{Err: ErrNotAuthenticated, Message: "test"},
			want: true,
		},
		{
			name: "unauthenticated string",
			err:  errors.New("unauthenticated"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.want {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsConnectionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "connection failed",
			err:  ErrConnectionFailed,
			want: true,
		},
		{
			name: "connection refused string",
			err:  errors.New("connection refused"),
			want: true,
		},
		{
			name: "no such host",
			err:  errors.New("dial tcp: no such host"),
			want: true,
		},
		{
			name: "network unreachable",
			err:  errors.New("network is unreachable"),
			want: true,
		},
		{
			name: "certificate error",
			err:  errors.New("x509: certificate signed by unknown authority"),
			want: true,
		},
		{
			name: "TLS error",
			err:  errors.New("tls: handshake failure"),
			want: true,
		},
		{
			name: "timeout",
			err:  errors.New("connection timeout"),
			want: true,
		},
		{
			name: "deadline exceeded",
			err:  errors.New("context deadline exceeded"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConnectionError(tt.err); got != tt.want {
				t.Errorf("IsConnectionError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsProjectError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "no project linked sentinel",
			err:  ErrNoProjectLinked,
			want: true,
		},
		{
			name: "wrapped project error",
			err:  &CLIError{Err: ErrNoProjectLinked, Message: "test"},
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsProjectError(tt.err); got != tt.want {
				t.Errorf("IsProjectError() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "permission denied sentinel",
			err:  ErrPermissionDenied,
			want: true,
		},
		{
			name: "permission denied string",
			err:  errors.New("permission denied"),
			want: true,
		},
		{
			name: "forbidden",
			err:  errors.New("forbidden"),
			want: true,
		},
		{
			name: "403",
			err:  errors.New("server returned 403"),
			want: true,
		},
		{
			name: "other error",
			err:  errors.New("some other error"),
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPermissionError(tt.err); got != tt.want {
				t.Errorf("IsPermissionError() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper functions

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// testMessenger is a mock messenger for testing custom messages.
type testMessenger struct {
	authMsg        string
	authSuggestion string
}

func (m *testMessenger) AuthErrorMessage() (string, string) {
	return m.authMsg, m.authSuggestion
}

func (m *testMessenger) SessionExpiredMessage() (string, string) {
	return "Session expired", "Log in again"
}

func (m *testMessenger) PermissionDeniedMessage() (string, string) {
	return "Permission denied", "Contact admin"
}

func (m *testMessenger) ConnectionErrorMessage(serverURL string) (string, string) {
	return "Connection failed to " + serverURL, "Check server"
}

func (m *testMessenger) TLSErrorMessage(serverURL string) (string, string) {
	return "TLS error to " + serverURL, "Check certificate"
}

func (m *testMessenger) TimeoutErrorMessage(serverURL string) (string, string) {
	return "Timeout to " + serverURL, "Try again"
}

func (m *testMessenger) NotInGitRepoMessage() (string, string) {
	return "Not in git repo", "Go to a repo"
}

func (m *testMessenger) NoProjectLinkedMessage() (string, string) {
	return "No project linked", "Link a project"
}

func (m *testMessenger) ProjectNotFoundMessage() (string, string) {
	return "Project not found", "Check ID"
}
