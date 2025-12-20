package git

import "errors"

// Git operation errors.
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

// Error wraps a git command error with context.
type Error struct {
	Op     string // Operation that failed (e.g., "commit", "push")
	Cmd    string // Git command that was run
	Output string // Combined stdout/stderr output
	Err    error  // Underlying error
}

func (e *Error) Error() string {
	if e.Output != "" {
		return e.Op + ": " + e.Output
	}
	return e.Op + ": " + e.Err.Error()
}

func (e *Error) Unwrap() error {
	return e.Err
}
