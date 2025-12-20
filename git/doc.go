// Package git provides Git operations including repository management,
// worktree creation, branch operations, commits, and command execution.
//
// Core types:
//   - Context: Git repository context with worktree and branch operations
//   - CommandRunner: Interface for executing git commands (with mock for testing)
//   - BranchNamer: Generates branch names from tickets/descriptions
//   - CommitMessage: Conventional commit message builder
//
// Example usage:
//
//	ctx := git.NewContext("/path/to/repo")
//	worktree, err := ctx.CreateWorktree("feature/my-branch")
//	defer ctx.CleanupWorktree(worktree)
//
//	err = ctx.Commit("Add feature", "file1.go", "file2.go")
package git
