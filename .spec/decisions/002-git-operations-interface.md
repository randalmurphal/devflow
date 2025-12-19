# ADR-002: Git Operations Interface

## Status

Accepted

## Context

devflow needs to perform git operations: worktree management, branching, committing, pushing, and PR creation. We need to decide:

1. **Implementation approach**: Use go-git library or shell out to git binary?
2. **Interface design**: What abstraction level?
3. **Error handling**: How to surface git errors?
4. **Provider support**: How to handle GitHub vs GitLab differences?

## Decision

### 1. Shell Out to Git Binary

Use `exec.Command("git", ...)` rather than go-git library.

**Rationale:**
- Simpler to maintain (no library version management)
- Behaves exactly like user's git (same config, same edge cases)
- Easy to debug (just run the command)
- Handles complex scenarios that go-git might not

```go
// This:
cmd := exec.Command("git", "commit", "-m", message)
cmd.Dir = worktreePath
output, err := cmd.CombinedOutput()

// Not this:
repo, _ := git.PlainOpen(path)
worktree, _ := repo.Worktree()
commit, _ := worktree.Commit(message, &git.CommitOptions{...})
```

### 2. GitContext as Central Type

Single `GitContext` struct holds repository context and provides all operations:

```go
type GitContext struct {
    repoPath    string           // Path to main repository
    worktreeDir string           // Where worktrees are created
    github      *GitHubProvider  // GitHub API (if configured)
    gitlab      *GitLabProvider  // GitLab API (if configured)
}
```

### 3. Functional Options for Configuration

```go
git, err := NewGitContext("/path/to/repo",
    WithWorktreeDir(".worktrees"),
    WithGitHub(githubClient),
    WithGitLab(gitlabClient),
)
```

### 4. Explicit Error Types

```go
var (
    ErrWorktreeExists   = errors.New("worktree already exists")
    ErrBranchExists     = errors.New("branch already exists")
    ErrGitDirty         = errors.New("working directory has uncommitted changes")
    ErrNoPRProvider     = errors.New("no PR provider configured")
    ErrPushFailed       = errors.New("push failed")
    ErrMergeConflict    = errors.New("merge conflict")
)
```

### 5. PR Provider Abstraction

PRs/MRs use same interface regardless of provider:

```go
type PRProvider interface {
    CreatePR(ctx context.Context, opts PROptions) (*PullRequest, error)
    GetPR(ctx context.Context, id int) (*PullRequest, error)
    UpdatePR(ctx context.Context, id int, opts PRUpdateOptions) (*PullRequest, error)
    MergePR(ctx context.Context, id int, opts MergeOptions) error
    ListPRs(ctx context.Context, filter PRFilter) ([]*PullRequest, error)
}
```

GitContext delegates to configured provider:

```go
func (g *GitContext) CreatePR(ctx context.Context, opts PROptions) (*PullRequest, error) {
    if g.github != nil {
        return g.github.CreatePR(ctx, opts)
    }
    if g.gitlab != nil {
        return g.gitlab.CreatePR(ctx, opts)
    }
    return nil, ErrNoPRProvider
}
```

## Alternatives Considered

### Alternative 1: go-git Library

Use [go-git](https://github.com/go-git/go-git) for native Go implementation.

**Rejected because:**
- Another dependency to maintain
- Different behavior from user's git
- Complex API for our simple needs
- Missing some features (worktrees partially supported)

### Alternative 2: Separate Types Per Operation

Have `WorktreeManager`, `CommitHelper`, `PRCreator` etc.

**Rejected because:**
- More types to manage
- Operations often interrelated
- Single context simplifies testing

### Alternative 3: Repository-Agnostic Interface

Abstract away git entirely behind a generic VCS interface.

**Rejected because:**
- We only need git
- Abstraction adds complexity without value
- Git-specific features (worktrees) hard to abstract

## Consequences

### Positive

- **Simple implementation**: Shell commands are straightforward
- **Familiar behavior**: Works like git users expect
- **Easy debugging**: Print the command to debug
- **Provider flexibility**: GitHub/GitLab through same interface

### Negative

- **Git dependency**: Requires git binary installed
- **Parse complexity**: Must parse git output
- **Error handling**: Git error messages can be cryptic
- **Platform differences**: Minor git behavior differences across platforms

### Mitigations

1. **Check git version**: Validate git is installed at init
2. **Structured output**: Use `--porcelain` flags where available
3. **Error wrapping**: Wrap git errors with context
4. **Integration tests**: Test on Linux and macOS

## Code Example

```go
package devflow

import (
    "bytes"
    "context"
    "errors"
    "fmt"
    "os/exec"
    "strings"
)

// GitContext manages git operations for a repository
type GitContext struct {
    repoPath    string
    worktreeDir string
    github      PRProvider
    gitlab      PRProvider
}

// GitOption configures GitContext
type GitOption func(*GitContext)

// NewGitContext creates a new git context for the repository
func NewGitContext(repoPath string, opts ...GitOption) (*GitContext, error) {
    // Verify it's a git repository
    cmd := exec.Command("git", "rev-parse", "--git-dir")
    cmd.Dir = repoPath
    if err := cmd.Run(); err != nil {
        return nil, fmt.Errorf("not a git repository: %s", repoPath)
    }

    g := &GitContext{
        repoPath:    repoPath,
        worktreeDir: ".worktrees",
    }

    for _, opt := range opts {
        opt(g)
    }

    return g, nil
}

// CurrentBranch returns the current branch name
func (g *GitContext) CurrentBranch() (string, error) {
    return g.runGit("rev-parse", "--abbrev-ref", "HEAD")
}

// Commit stages and commits the specified files
func (g *GitContext) Commit(message string, files ...string) error {
    // Stage files
    args := append([]string{"add"}, files...)
    if _, err := g.runGit(args...); err != nil {
        return fmt.Errorf("stage files: %w", err)
    }

    // Commit
    if _, err := g.runGit("commit", "-m", message); err != nil {
        return fmt.Errorf("commit: %w", err)
    }

    return nil
}

// Push pushes the branch to remote
func (g *GitContext) Push(remote, branch string) error {
    _, err := g.runGit("push", "-u", remote, branch)
    if err != nil {
        return fmt.Errorf("push: %w", err)
    }
    return nil
}

// Diff returns the diff between two refs
func (g *GitContext) Diff(base, head string) (string, error) {
    return g.runGit("diff", base+"..."+head)
}

// runGit executes a git command and returns stdout
func (g *GitContext) runGit(args ...string) (string, error) {
    cmd := exec.Command("git", args...)
    cmd.Dir = g.repoPath

    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    if err := cmd.Run(); err != nil {
        return "", fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
    }

    return strings.TrimSpace(stdout.String()), nil
}

// Option constructors
func WithWorktreeDir(dir string) GitOption {
    return func(g *GitContext) {
        g.worktreeDir = dir
    }
}

func WithGitHub(provider PRProvider) GitOption {
    return func(g *GitContext) {
        g.github = provider
    }
}

func WithGitLab(provider PRProvider) GitOption {
    return func(g *GitContext) {
        g.gitlab = provider
    }
}
```

## Testing Strategy

```go
func TestGitContext_Commit(t *testing.T) {
    // Create temp directory with git repo
    dir := t.TempDir()
    exec.Command("git", "init", dir).Run()
    exec.Command("git", "-C", dir, "config", "user.email", "test@test.com").Run()
    exec.Command("git", "-C", dir, "config", "user.name", "Test").Run()

    // Create file
    os.WriteFile(filepath.Join(dir, "test.txt"), []byte("content"), 0644)

    // Test
    git, err := NewGitContext(dir)
    require.NoError(t, err)

    err = git.Commit("Test commit", "test.txt")
    require.NoError(t, err)

    // Verify
    log, _ := git.runGit("log", "--oneline", "-1")
    assert.Contains(t, log, "Test commit")
}
```

## References

- ADR-001: Worktree Strategy
- GitHub API: https://docs.github.com/en/rest/pulls
- GitLab API: https://docs.gitlab.com/ee/api/merge_requests.html
