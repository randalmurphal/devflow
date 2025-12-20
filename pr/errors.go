package pr

import "errors"

// PR provider errors
var (
	// ErrNoProvider indicates no PR provider is configured.
	ErrNoProvider = errors.New("no PR provider configured")

	// ErrUnknownProvider indicates the git remote uses an unknown provider.
	ErrUnknownProvider = errors.New("unknown git provider")

	// ErrExists indicates a PR already exists for the branch.
	ErrExists = errors.New("pull request already exists for this branch")

	// ErrNotFound indicates the PR does not exist.
	ErrNotFound = errors.New("pull request not found")

	// ErrClosed indicates the PR is closed.
	ErrClosed = errors.New("pull request is closed")

	// ErrMerged indicates the PR is already merged.
	ErrMerged = errors.New("pull request is already merged")

	// ErrBranchNotPushed indicates the branch has not been pushed to remote.
	ErrBranchNotPushed = errors.New("branch not pushed to remote")

	// ErrNoChanges indicates there are no changes between branches.
	ErrNoChanges = errors.New("no changes between branches")

	// ErrMergeConflict indicates a merge conflict occurred.
	ErrMergeConflict = errors.New("merge conflict")
)
