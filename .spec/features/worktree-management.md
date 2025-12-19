# Feature: Worktree Management

## Overview

Git worktrees enable parallel development by creating multiple working directories from a single repository.

## Use Cases

1. **Parallel agent work**: Multiple AI agents working on different features
2. **Isolated experiments**: Try approaches without affecting main checkout
3. **Review while developing**: Keep main checkout clean while reviewing PRs

## API

### Create Worktree

```go
worktree, err := git.CreateWorktree("feature/my-branch")
// worktree = "/path/to/repo/.worktrees/feature-my-branch"
```

### Cleanup Worktree

```go
err := git.CleanupWorktree(worktree)
```

### List Worktrees

```go
worktrees, err := git.ListWorktrees()
for _, wt := range worktrees {
    fmt.Printf("%s -> %s\n", wt.Branch, wt.Path)
}
```

## Directory Structure

```
repo/
├── .git/
├── .worktrees/
│   ├── feature-auth/
│   ├── feature-api/
│   └── bugfix-login/
└── src/
```

## Behavior

### Creation

1. Sanitize branch name for filesystem
2. Create directory at `.worktrees/{safe-name}`
3. Run `git worktree add -b {branch} {path}`
4. If branch exists, run without `-b`

### Cleanup

1. Run `git worktree remove {path}`
2. If uncommitted changes, force with `--force`
3. Optionally delete branch

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| `ErrWorktreeExists` | Worktree already exists | Use existing or cleanup first |
| `ErrBranchExists` | Branch exists, worktree doesn't | Git worktree add without -b |

## Configuration

```go
type GitContext struct {
    worktreeDir string // Default: ".worktrees"
}

// Option
func WithWorktreeDir(dir string) GitOption
```

## Example

```go
git, _ := devflow.NewGitContext("/path/to/repo")

// Create worktree for feature
worktree, err := git.CreateWorktree("feature/TK-421-add-auth")
if err != nil {
    log.Fatal(err)
}
defer git.CleanupWorktree(worktree)

// Work in worktree
os.Chdir(worktree)
// ... make changes ...

// Commit and push
git.Stage("auth.go", "auth_test.go")
git.Commit("feat(auth): add OAuth support")
git.Push("origin", "feature/TK-421-add-auth")
```

## Testing

```go
func TestCreateWorktree(t *testing.T) {
    // Create temp git repo
    // Create worktree
    // Verify directory exists
    // Verify branch created
    // Cleanup
    // Verify removed
}
```

## References

- [Git Worktrees](https://git-scm.com/docs/git-worktree)
- ADR-001: Worktree Strategy
