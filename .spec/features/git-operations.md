# Feature: Git Operations

## Overview

Core git operations for development workflows: branching, staging, committing, and pushing.

## Use Cases

1. **Automated commits**: AI-generated code committed automatically
2. **Branch management**: Create feature branches from tickets
3. **Push to remote**: Enable PR creation
4. **Diff generation**: Capture changes for review

## API

### Branch Operations

```go
// Get current branch
branch, err := git.CurrentBranch()

// Create new branch
err := git.CreateBranch("feature/my-feature")

// Checkout existing branch
err := git.Checkout("main")
```

### Staging and Committing

```go
// Stage files
err := git.Stage("file1.go", "file2.go")

// Commit with message
err := git.Commit("feat: add new feature")

// Stage and commit in one step
err := git.CommitFiles("feat: add new feature", "file1.go", "file2.go")
```

### Push

```go
// Push branch
err := git.Push("origin", "feature/my-feature")

// Push with upstream tracking
err := git.PushWithUpstream("origin", "feature/my-feature")
```

### Diff

```go
// Diff between branches
diff, err := git.Diff("main", "feature/my-feature")

// Diff of staged changes
diff, err := git.DiffStaged()
```

## Implementation

All operations shell out to `git` binary:

```go
func (g *GitContext) runGit(args ...string) (string, error) {
    cmd := exec.Command("git", args...)
    cmd.Dir = g.repoPath

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("%w: %s", err, stderr.String())
    }

    return strings.TrimSpace(stdout.String()), nil
}
```

## Behavior

### Commit

1. Stage specified files with `git add`
2. Create commit with `git commit -m`
3. Return error if nothing to commit

### Push

1. Check branch exists locally
2. Run `git push -u origin {branch}`
3. Return error on rejection

## Error Handling

| Error | Cause | Resolution |
|-------|-------|------------|
| `ErrGitDirty` | Uncommitted changes | Commit or stash |
| `ErrBranchExists` | Branch already exists | Use different name |
| `ErrPushFailed` | Push rejected | Pull and retry |
| `ErrNothingToCommit` | No staged changes | Stage files first |

## Configuration

Operations work in worktree context automatically when `git.repoPath` is a worktree.

## Example

```go
git, _ := devflow.NewGitContext("/path/to/repo")

// Create feature branch
worktree, _ := git.CreateWorktree("feature/add-auth")

// Work in worktree (or configure git to use worktree path)
// ... make changes ...

// Commit changes
err := git.CommitFiles(
    "feat(auth): add OAuth2 authentication",
    "auth/handler.go",
    "auth/handler_test.go",
)
if err != nil {
    log.Fatal(err)
}

// Push
err = git.Push("origin", "feature/add-auth")
if err != nil {
    log.Fatal(err)
}
```

## Testing

```go
func TestGitOperations(t *testing.T) {
    // Create temp git repo
    // Create branch
    // Make changes
    // Stage and commit
    // Verify commit exists
}
```

## References

- ADR-002: Git Operations Interface
- ADR-004: Commit Formatting
