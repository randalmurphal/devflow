// Package devflow provides dev workflow primitives for AI-powered development automation.
package devflow

import "errors"

// Git operation errors
var (
	// ErrNotGitRepo indicates the path is not a git repository.
	ErrNotGitRepo = errors.New("not a git repository")

	// ErrWorktreeExists indicates a worktree already exists for the branch.
	ErrWorktreeExists = errors.New("worktree already exists for this branch")

	// ErrWorktreeNotFound indicates the worktree does not exist.
	ErrWorktreeNotFound = errors.New("worktree not found")

	// ErrBranchExists indicates the branch already exists.
	ErrBranchExists = errors.New("branch already exists")

	// ErrBranchNotFound indicates the branch does not exist.
	ErrBranchNotFound = errors.New("branch not found")

	// ErrGitDirty indicates the working directory has uncommitted changes.
	ErrGitDirty = errors.New("working directory has uncommitted changes")

	// ErrNothingToCommit indicates there are no staged changes to commit.
	ErrNothingToCommit = errors.New("nothing to commit")

	// ErrPushFailed indicates a push operation failed.
	ErrPushFailed = errors.New("push failed")

	// ErrMergeConflict indicates a merge conflict occurred.
	ErrMergeConflict = errors.New("merge conflict")
)

// PR provider errors
var (
	// ErrNoPRProvider indicates no PR provider is configured.
	ErrNoPRProvider = errors.New("no PR provider configured")

	// ErrUnknownProvider indicates the git remote uses an unknown provider.
	ErrUnknownProvider = errors.New("unknown git provider")

	// ErrPRExists indicates a PR already exists for the branch.
	ErrPRExists = errors.New("pull request already exists for this branch")

	// ErrPRNotFound indicates the PR does not exist.
	ErrPRNotFound = errors.New("pull request not found")

	// ErrPRClosed indicates the PR is closed.
	ErrPRClosed = errors.New("pull request is closed")

	// ErrPRMerged indicates the PR is already merged.
	ErrPRMerged = errors.New("pull request is already merged")

	// ErrBranchNotPushed indicates the branch has not been pushed to remote.
	ErrBranchNotPushed = errors.New("branch not pushed to remote")

	// ErrNoChanges indicates there are no changes between branches.
	ErrNoChanges = errors.New("no changes between branches")
)

// GitError wraps a git command error with context.
type GitError struct {
	Op     string // Operation that failed (e.g., "commit", "push")
	Cmd    string // Git command that was run
	Output string // Combined stdout/stderr output
	Err    error  // Underlying error
}

func (e *GitError) Error() string {
	if e.Output != "" {
		return e.Op + ": " + e.Output
	}
	return e.Op + ": " + e.Err.Error()
}

func (e *GitError) Unwrap() error {
	return e.Err
}
