package devflow

import (
	"errors"
	"testing"
)

func TestNewExecRunner(t *testing.T) {
	runner := NewExecRunner()
	if runner == nil {
		t.Error("NewExecRunner should return non-nil runner")
	}
}

func TestExecRunner_Run_Success(t *testing.T) {
	runner := NewExecRunner()

	// Run a simple command
	output, err := runner.Run("", "echo", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if output != "hello" {
		t.Errorf("output = %q, want %q", output, "hello")
	}
}

func TestExecRunner_Run_Error(t *testing.T) {
	runner := NewExecRunner()

	// Run a command that will fail
	_, err := runner.Run("", "ls", "/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent path")
	}

	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) {
		t.Errorf("error should be CommandError, got %T", err)
	}
}

func TestCommandError_Error(t *testing.T) {
	t.Run("with output", func(t *testing.T) {
		err := &CommandError{
			Command: "git",
			Args:    []string{"status"},
			Output:  "fatal: not a git repository",
			Err:     errors.New("exit status 128"),
		}

		got := err.Error()
		want := "fatal: not a git repository"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("without output", func(t *testing.T) {
		underlying := errors.New("exit status 1")
		err := &CommandError{
			Command: "git",
			Args:    []string{"push"},
			Err:     underlying,
		}

		got := err.Error()
		want := "exit status 1"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})

	t.Run("no output or error", func(t *testing.T) {
		err := &CommandError{
			Command: "test",
		}

		got := err.Error()
		want := "command failed"
		if got != want {
			t.Errorf("Error() = %q, want %q", got, want)
		}
	})
}

func TestCommandError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	err := &CommandError{
		Command: "git",
		Args:    []string{"commit"},
		Err:     underlying,
	}

	unwrapped := err.Unwrap()
	if unwrapped != underlying {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, underlying)
	}

	// Test that errors.Is works
	if !errors.Is(err, underlying) {
		t.Error("errors.Is should return true for underlying error")
	}
}

func TestNewMockRunner(t *testing.T) {
	runner := NewMockRunner()
	if runner == nil {
		t.Error("NewMockRunner should return non-nil runner")
	}
	if runner.Responses == nil {
		t.Error("Responses map should be initialized")
	}
}

func TestMockRunner_Run(t *testing.T) {
	t.Run("exact match", func(t *testing.T) {
		runner := NewMockRunner()
		runner.OnCommand("git", "status", "--short").Return("M file.go", nil)

		output, err := runner.Run("/repo", "git", "status", "--short")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if output != "M file.go" {
			t.Errorf("output = %q, want %q", output, "M file.go")
		}
	})

	t.Run("command only match", func(t *testing.T) {
		runner := NewMockRunner()
		runner.Responses["git"] = MockResponse{Stdout: "git response", Err: nil}

		output, err := runner.Run("/repo", "git", "log")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if output != "git response" {
			t.Errorf("output = %q, want %q", output, "git response")
		}
	})

	t.Run("wildcard match", func(t *testing.T) {
		runner := NewMockRunner()
		runner.OnAnyCommand().Return("wildcard", nil)

		output, err := runner.Run("/repo", "any", "command")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if output != "wildcard" {
			t.Errorf("output = %q, want %q", output, "wildcard")
		}
	})

	t.Run("default response", func(t *testing.T) {
		runner := NewMockRunner()
		runner.DefaultResponse = MockResponse{Stdout: "default", Err: nil}

		output, err := runner.Run("/repo", "cmd")
		if err != nil {
			t.Fatalf("Run: %v", err)
		}
		if output != "default" {
			t.Errorf("output = %q, want %q", output, "default")
		}
	})

	t.Run("with error", func(t *testing.T) {
		runner := NewMockRunner()
		expectedErr := errors.New("mock error")
		runner.OnCommand("fail").Return("", expectedErr)

		_, err := runner.Run("/repo", "fail")
		if err != expectedErr {
			t.Errorf("error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMockRunner_Calls(t *testing.T) {
	runner := NewMockRunner()
	runner.OnAnyCommand().Return("", nil)

	runner.Run("/repo", "git", "status")
	runner.Run("/other", "git", "log")

	if len(runner.Calls) != 2 {
		t.Errorf("Calls = %d, want 2", len(runner.Calls))
	}

	if runner.Calls[0].Command != "git" {
		t.Errorf("first call command = %q, want %q", runner.Calls[0].Command, "git")
	}
	if runner.Calls[0].WorkDir != "/repo" {
		t.Errorf("first call workdir = %q, want %q", runner.Calls[0].WorkDir, "/repo")
	}
}

func TestMockRunner_WasCalled_Detailed(t *testing.T) {
	runner := NewMockRunner()
	runner.OnAnyCommand().Return("", nil)

	runner.Run("/repo", "git", "status")

	if !runner.WasCalled("git") {
		t.Error("WasCalled should return true for git")
	}
	if !runner.WasCalled("git", "status") {
		t.Error("WasCalled should return true for git status")
	}
	if runner.WasCalled("git", "log") {
		t.Error("WasCalled should return false for git log")
	}
	if runner.WasCalled("npm") {
		t.Error("WasCalled should return false for npm")
	}
}

func TestMockRunner_CallCount_Detailed(t *testing.T) {
	runner := NewMockRunner()
	runner.OnAnyCommand().Return("", nil)

	runner.Run("/repo", "git", "status")
	runner.Run("/repo", "git", "add", ".")
	runner.Run("/repo", "npm", "install")

	if count := runner.CallCount("git"); count != 2 {
		t.Errorf("git call count = %d, want 2", count)
	}
	if count := runner.CallCount("npm"); count != 1 {
		t.Errorf("npm call count = %d, want 1", count)
	}
	if count := runner.CallCount("yarn"); count != 0 {
		t.Errorf("yarn call count = %d, want 0", count)
	}
}

func TestArgsMatch(t *testing.T) {
	tests := []struct {
		name     string
		actual   []string
		expected []string
		want     bool
	}{
		{"equal", []string{"a", "b"}, []string{"a", "b"}, true},
		{"different length", []string{"a"}, []string{"a", "b"}, false},
		{"different values", []string{"a", "c"}, []string{"a", "b"}, false},
		{"empty", []string{}, []string{}, true},
		{"nil", nil, nil, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := argsMatch(tt.actual, tt.expected)
			if got != tt.want {
				t.Errorf("argsMatch(%v, %v) = %v, want %v", tt.actual, tt.expected, got, tt.want)
			}
		})
	}
}
