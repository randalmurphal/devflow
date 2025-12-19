# ADR-001: Worktree Strategy

## Status

Accepted

## Context

devflow needs to support parallel AI-powered development work. Multiple agents or workflows may need to work on different features simultaneously. Traditional git branch operations require checking out one branch at a time, which creates contention.

Git worktrees allow multiple working directories, each checked out to a different branch, sharing the same repository. This enables true parallel work without checkout conflicts.

**Key questions:**
1. Where should worktrees be created?
2. How should they be named?
3. When should they be cleaned up?
4. How do we handle worktree conflicts?

## Decision

### 1. Worktree Location

Worktrees are created in a `.worktrees/` directory at the repository root:

```
repo/
├── .git/
├── .worktrees/
│   ├── feature-tk-421/       # Worktree for feature/tk-421 branch
│   ├── feature-tk-422/       # Worktree for feature/tk-422 branch
│   └── bugfix-auth-fix/      # Worktree for bugfix/auth-fix branch
└── src/
```

**Rationale:**
- Keeps worktrees close to repo (easy to find)
- Single location for cleanup
- `.worktrees/` should be in `.gitignore`

### 2. Worktree Naming

Branch names are sanitized for filesystem safety:

| Branch | Worktree Directory |
|--------|-------------------|
| `feature/tk-421` | `feature-tk-421` |
| `bugfix/auth-fix` | `bugfix-auth-fix` |
| `user/bob/experiment` | `user-bob-experiment` |

Sanitization rules:
- Replace `/` with `-`
- Replace spaces with `-`
- Remove characters invalid for filesystems
- Lowercase

### 3. Worktree Lifecycle

```go
// Create worktree
worktree, err := git.CreateWorktree("feature/tk-421")
// worktree = "/path/to/repo/.worktrees/feature-tk-421"

// Work in worktree...

// Cleanup when done
defer git.CleanupWorktree(worktree)
```

Cleanup removes:
1. The worktree directory
2. The worktree registration in `.git/worktrees/`
3. Optionally, the branch (if merged or specified)

### 4. Conflict Handling

If a worktree already exists for a branch:

```go
worktree, err := git.CreateWorktree("feature/tk-421")
if errors.Is(err, ErrWorktreeExists) {
    // Option 1: Use existing
    worktree, err = git.GetWorktree("feature/tk-421")

    // Option 2: Clean up and recreate
    git.CleanupWorktree(existingPath)
    worktree, err = git.CreateWorktree("feature/tk-421")
}
```

## Alternatives Considered

### Alternative 1: Temporary Directories

Create worktrees in `/tmp` or OS temp directory.

**Rejected because:**
- Harder to find/debug
- May be cleaned up by OS
- Path unpredictable

### Alternative 2: No Worktrees (Clone Per Branch)

Clone the repository for each parallel operation.

**Rejected because:**
- Much slower (full clone)
- Wastes disk space
- Doesn't share git objects

### Alternative 3: Single Directory with Checkout

Use single working directory, checkout branches as needed.

**Rejected because:**
- Creates contention
- Can't parallelize
- State conflicts

## Consequences

### Positive

- **True parallelism**: Multiple workflows can run simultaneously
- **Clean isolation**: Each worktree has independent state
- **Efficient**: Worktrees share git objects (disk space)
- **Debuggable**: Easy to inspect worktree state

### Negative

- **Disk usage**: Each worktree duplicates working files
- **Cleanup required**: Must clean up worktrees or disk fills up
- **Git knowledge**: Team needs to understand worktrees
- **Path handling**: Code must handle working in different directories

### Gotchas

1. **Worktrees can't share branches**: Can't create two worktrees for same branch
2. **Stale worktrees**: If process crashes, worktrees may be left behind
3. **Concurrent git operations**: Some git operations may contend (push, fetch)

## Code Example

```go
package devflow

// CreateWorktree creates an isolated worktree for the branch
func (g *GitContext) CreateWorktree(branch string) (string, error) {
    // Sanitize branch name for filesystem
    safeName := sanitizeBranchName(branch)
    worktreePath := filepath.Join(g.repoPath, ".worktrees", safeName)

    // Check if already exists
    if _, err := os.Stat(worktreePath); err == nil {
        return "", ErrWorktreeExists
    }

    // Ensure .worktrees directory exists
    if err := os.MkdirAll(filepath.Dir(worktreePath), 0755); err != nil {
        return "", fmt.Errorf("create worktrees dir: %w", err)
    }

    // Create worktree with new branch
    cmd := exec.Command("git", "worktree", "add", "-b", branch, worktreePath, "HEAD")
    cmd.Dir = g.repoPath

    if output, err := cmd.CombinedOutput(); err != nil {
        // Branch may already exist, try without -b
        cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
        cmd.Dir = g.repoPath
        if output, err := cmd.CombinedOutput(); err != nil {
            return "", fmt.Errorf("create worktree: %s: %w", output, err)
        }
    }

    return worktreePath, nil
}

// CleanupWorktree removes a worktree and optionally its branch
func (g *GitContext) CleanupWorktree(worktreePath string) error {
    cmd := exec.Command("git", "worktree", "remove", worktreePath)
    cmd.Dir = g.repoPath

    if output, err := cmd.CombinedOutput(); err != nil {
        // Force remove if needed (uncommitted changes, etc.)
        cmd = exec.Command("git", "worktree", "remove", "--force", worktreePath)
        cmd.Dir = g.repoPath
        if output, err := cmd.CombinedOutput(); err != nil {
            return fmt.Errorf("cleanup worktree: %s: %w", output, err)
        }
    }

    return nil
}

func sanitizeBranchName(branch string) string {
    // Replace / with -
    safe := strings.ReplaceAll(branch, "/", "-")
    // Lowercase
    safe = strings.ToLower(safe)
    // Remove invalid characters
    safe = regexp.MustCompile(`[^a-z0-9-]`).ReplaceAllString(safe, "")
    return safe
}
```

## References

- [Git Worktrees Documentation](https://git-scm.com/docs/git-worktree)
- flowgraph checkpointing (state per worktree)
