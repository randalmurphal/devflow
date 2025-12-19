package devflow

import (
	"errors"
	"testing"
)

func TestGitError_Error_WithOutput(t *testing.T) {
	err := &GitError{
		Op:     "push",
		Cmd:    "git push origin main",
		Output: "remote: Permission denied",
		Err:    ErrPushFailed,
	}

	errStr := err.Error()
	if errStr != "push: remote: Permission denied" {
		t.Errorf("Error() = %q, want %q", errStr, "push: remote: Permission denied")
	}
}

func TestGitError_Error_WithoutOutput(t *testing.T) {
	err := &GitError{
		Op:  "commit",
		Cmd: "git commit -m test",
		Err: ErrNothingToCommit,
	}

	errStr := err.Error()
	if errStr != "commit: nothing to commit" {
		t.Errorf("Error() = %q, want %q", errStr, "commit: nothing to commit")
	}
}

func TestGitError_Unwrap(t *testing.T) {
	origErr := ErrPushFailed
	err := &GitError{
		Op:  "push",
		Err: origErr,
	}

	unwrapped := err.Unwrap()
	if unwrapped != origErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, origErr)
	}
}

func TestGitError_Is(t *testing.T) {
	err := &GitError{
		Op:  "push",
		Err: ErrPushFailed,
	}

	if !errors.Is(err, ErrPushFailed) {
		t.Error("errors.Is should return true for wrapped error")
	}

	if errors.Is(err, ErrMergeConflict) {
		t.Error("errors.Is should return false for different error")
	}
}

func TestGitErrors_Defined(t *testing.T) {
	// Verify all git errors are defined and have unique messages
	gitErrors := []error{
		ErrNotGitRepo,
		ErrWorktreeExists,
		ErrWorktreeNotFound,
		ErrBranchExists,
		ErrBranchNotFound,
		ErrGitDirty,
		ErrNothingToCommit,
		ErrPushFailed,
		ErrMergeConflict,
	}

	seen := make(map[string]bool)
	for _, err := range gitErrors {
		if err == nil {
			t.Error("Git error should not be nil")
			continue
		}
		msg := err.Error()
		if msg == "" {
			t.Error("Git error message should not be empty")
		}
		if seen[msg] {
			t.Errorf("Duplicate error message: %q", msg)
		}
		seen[msg] = true
	}
}

func TestPRErrors_Defined(t *testing.T) {
	// Verify all PR errors are defined and have unique messages
	prErrors := []error{
		ErrNoPRProvider,
		ErrUnknownProvider,
		ErrPRExists,
		ErrPRNotFound,
		ErrPRClosed,
		ErrPRMerged,
		ErrBranchNotPushed,
		ErrNoChanges,
	}

	seen := make(map[string]bool)
	for _, err := range prErrors {
		if err == nil {
			t.Error("PR error should not be nil")
			continue
		}
		msg := err.Error()
		if msg == "" {
			t.Error("PR error message should not be empty")
		}
		if seen[msg] {
			t.Errorf("Duplicate error message: %q", msg)
		}
		seen[msg] = true
	}
}
