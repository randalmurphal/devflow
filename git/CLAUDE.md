# git package

Git repository operations: worktrees, branches, commits, command execution.

## Quick Reference

| Type | Purpose |
|------|---------|
| `Context` | Git repository context with all operations |
| `Option` | Functional option for `NewContext` |
| `CommandRunner` | Interface for executing git commands |
| `MockRunner` | Test double for command execution |
| `BranchNamer` | Generates branch names from tickets/workflows |
| `CommitMessage` | Conventional commit message builder |
| `WorktreeInfo` | Represents an active worktree |

## Key Functions

| Function | Purpose |
|----------|---------|
| `NewContext(path, ...Option)` | Create git context for repository |
| `NewExecRunner()` | Create real command runner |
| `NewMockRunner()` | Create mock for testing |
| `DefaultBranchNamer()` | Create branch namer with defaults |
| `NewCommitMessage(type, subject)` | Create conventional commit |

## Context Methods

**Branch Operations:**
- `CurrentBranch()` - Get current branch name
- `CreateBranch(name)` - Create branch at HEAD
- `DeleteBranch(name, force)` - Delete branch
- `BranchExists(name)` - Check if branch exists
- `Checkout(ref)` - Switch to ref
- `CheckoutNew(name)` - Create and checkout new branch (convenience)
- `CheckoutNewAt(name, ref)` - Create branch at ref (convenience)

**Staging & Commits:**
- `Stage(files...)` - Add files to staging
- `StageAll()` - Stage all changes
- `Commit(message)` - Create commit
- `IsClean()` - Check for uncommitted changes
- `CommitAll(message)` - Stage all + commit, returns `*CommitResult` (convenience)

**Push:**
- `Push(remote, branch, setUpstream)` - Push to remote
- `PushCurrent()` - Push current branch to origin, returns `*PushResult` (convenience)
- `PushCurrentTo(remote)` - Push to specific remote (convenience)
- `CommitAllAndPush(message)` - Full workflow, returns `*CommitAndPushResult` (convenience)

**Worktrees:**
- `CreateWorktree(branch)` - Create isolated worktree
- `CleanupWorktree(path)` - Remove worktree
- `ListWorktrees()` - List all worktrees
- `GetWorktree(branch)` - Find worktree by branch
- `InWorktree(path)` - Get context for worktree

**Remote:**
- `Push(remote, branch, setUpstream)` - Push changes
- `Pull(remote, branch)` - Pull changes
- `Fetch(remote)` - Fetch updates
- `IsBranchPushed(branch)` - Check if on remote
- `GetRemoteURL(remote)` - Get remote URL

## Errors

| Error | When |
|-------|------|
| `ErrNotGitRepo` | Path is not a git repo |
| `ErrWorktreeExists` | Worktree already exists |
| `ErrWorktreeNotFound` | Worktree not found |
| `ErrBranchExists` | Branch already exists |
| `ErrNothingToCommit` | No staged changes |

## Testing Pattern

```go
runner := git.NewMockRunner()
runner.OnCommand("git", "status", "--short").Return("", nil)

ctx, _ := git.NewContext(path, git.WithRunner(runner))
```

## Result Types (Convenience Methods)

| Type | Fields | From Method |
|------|--------|-------------|
| `CommitResult` | SHA, Branch, Message, Date | `CommitAll()` |
| `PushResult` | Remote, Branch, SHA, SetUpstream, URL | `PushCurrent()` |
| `CommitAndPushResult` | Commit, Push | `CommitAllAndPush()` |

## Context Injection

```go
// Add git context to context.Context
ctx := git.ContextWithGit(context.Background(), gitCtx)

// Retrieve later
gitCtx := git.GitFromContext(ctx)
gitCtx := git.MustGitFromContext(ctx)  // panics if missing
```

## File Structure

```
git/
├── git.go             # Context, core operations
├── convenience.go     # CommitAll, PushCurrent, etc.
├── context_helpers.go # ContextWithGit, GitFromContext
├── worktree.go        # Worktree operations
├── branch.go          # BranchNamer
├── commit.go          # CommitMessage
├── runner.go          # CommandRunner, MockRunner, SequentialMockRunner
└── errors.go          # Git-specific errors
```
