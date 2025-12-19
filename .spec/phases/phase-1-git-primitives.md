# Phase 1: Git Primitives

## Overview

Implement core git operations for development workflows.

**Duration**: Week 1
**Dependencies**: None
**Deliverables**: `GitContext` type with worktree and basic git operations

---

## Goals

1. Create isolated workspaces via git worktrees
2. Perform common git operations (checkout, branch, commit, push)
3. Abstract PR creation for GitHub/GitLab
4. Shell out to `git` binary (not go-git)

---

## Components

### GitContext

Central type for all git operations:

```go
type GitContext struct {
    repoPath    string
    worktreeDir string
    github      PRProvider
    gitlab      PRProvider
}
```

### WorktreeInfo

Represents an active worktree:

```go
type WorktreeInfo struct {
    Path   string // Filesystem path
    Branch string // Branch name
    Commit string // HEAD commit
}
```

### PRProvider Interface

Abstract PR creation:

```go
type PRProvider interface {
    CreatePR(ctx context.Context, opts PROptions) (*PullRequest, error)
    GetPR(ctx context.Context, id int) (*PullRequest, error)
    UpdatePR(ctx context.Context, id int, opts PRUpdateOptions) (*PullRequest, error)
    MergePR(ctx context.Context, id int, opts MergeOptions) error
}
```

---

## Implementation Tasks

### Task 1.1: GitContext Constructor

```go
func NewGitContext(repoPath string, opts ...GitOption) (*GitContext, error)

// Options:
func WithWorktreeDir(dir string) GitOption
func WithGitHub(token string) GitOption
func WithGitLab(token string, baseURL string) GitOption
```

**Acceptance Criteria**:
- [ ] Validates path is a git repository
- [ ] Sets default worktree directory (`.worktrees/`)
- [ ] Configures PR providers if tokens provided
- [ ] Returns error for non-git directory

### Task 1.2: Worktree Operations

```go
func (g *GitContext) CreateWorktree(branch string) (string, error)
func (g *GitContext) CleanupWorktree(path string) error
func (g *GitContext) ListWorktrees() ([]WorktreeInfo, error)
func (g *GitContext) GetWorktree(branch string) (*WorktreeInfo, error)
```

**Acceptance Criteria**:
- [ ] Creates worktree at `.worktrees/{sanitized-branch}`
- [ ] Creates branch if it doesn't exist
- [ ] Returns error for existing worktree (ErrWorktreeExists)
- [ ] Cleanup removes worktree registration
- [ ] List returns all active worktrees

### Task 1.3: Basic Git Operations

```go
func (g *GitContext) CurrentBranch() (string, error)
func (g *GitContext) Checkout(ref string) error
func (g *GitContext) CreateBranch(name string) error
func (g *GitContext) Stage(files ...string) error
func (g *GitContext) Commit(message string) error
func (g *GitContext) Push(remote, branch string) error
func (g *GitContext) Diff(base, head string) (string, error)
```

**Acceptance Criteria**:
- [ ] All operations work in worktree context
- [ ] Push includes `-u` for new branches
- [ ] Commit returns error if nothing staged
- [ ] Diff produces unified diff format

### Task 1.4: Branch Naming

```go
type BranchNamer struct {
    TypePrefix   string
    IncludeTitle bool
    MaxLength    int
}

func (n *BranchNamer) ForTicket(ticketID, title string) string
func (n *BranchNamer) ForWorkflow(workflowID, identifier string) string
```

**Acceptance Criteria**:
- [ ] Sanitizes names for filesystem (no `/`, spaces)
- [ ] Respects max length
- [ ] Includes timestamp for workflow branches

### Task 1.5: Commit Formatting

```go
type CommitMessage struct {
    Type        CommitType
    Scope       string
    Subject     string
    Body        string
    TicketRefs  []string
    GeneratedBy string
}

func (c *CommitMessage) String() string
```

**Acceptance Criteria**:
- [ ] Follows conventional commit format
- [ ] Includes `Generated-By: devflow` footer
- [ ] Wraps body at 72 characters

### Task 1.6: GitHub PR Provider

```go
type GitHubProvider struct {
    client *github.Client
    owner  string
    repo   string
}

func NewGitHubProvider(token, owner, repo string) (*GitHubProvider, error)
```

**Acceptance Criteria**:
- [ ] Creates PRs with title, body, labels
- [ ] Supports draft PRs
- [ ] Returns PR URL and number

### Task 1.7: GitLab MR Provider

```go
type GitLabProvider struct {
    client    *gitlab.Client
    projectID string
}

func NewGitLabProvider(token, baseURL, projectID string) (*GitLabProvider, error)
```

**Acceptance Criteria**:
- [ ] Creates MRs with title, body, labels
- [ ] Supports draft MRs
- [ ] Works with self-hosted GitLab

---

## Testing Strategy

### Unit Tests

| Test | Description |
|------|-------------|
| `TestGitContext_New` | Validates repo, rejects non-git |
| `TestGitContext_CreateWorktree` | Creates worktree, handles existing |
| `TestGitContext_CleanupWorktree` | Removes worktree cleanly |
| `TestBranchNamer_ForTicket` | Generates correct names |
| `TestCommitMessage_String` | Formats correctly |

### Integration Tests

Require real git repository:

```go
func TestGitContext_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Create temp repo
    dir := t.TempDir()
    exec.Command("git", "init", dir).Run()
    // ... configure
}
```

| Test | Description |
|------|-------------|
| `TestGitContext_WorktreeLifecycle` | Create, work, cleanup |
| `TestGitContext_CommitAndPush` | Full commit workflow |
| `TestGitContext_BranchOperations` | Create, checkout, list |

### Mock PR Provider Tests

```go
type MockPRProvider struct {
    CreatePRFunc func(ctx context.Context, opts PROptions) (*PullRequest, error)
}

func (m *MockPRProvider) CreatePR(ctx context.Context, opts PROptions) (*PullRequest, error) {
    return m.CreatePRFunc(ctx, opts)
}
```

---

## Error Handling

| Error | Condition | User Action |
|-------|-----------|-------------|
| `ErrNotGitRepo` | Path not a git repository | Check path |
| `ErrWorktreeExists` | Worktree already exists | Use existing or cleanup |
| `ErrBranchExists` | Branch already exists | Choose different name |
| `ErrGitDirty` | Uncommitted changes | Commit or stash |
| `ErrPushFailed` | Push rejected | Pull and retry |

---

## File Structure

```
devflow/
├── git.go              # GitContext, options
├── git_worktree.go     # Worktree operations
├── git_branch.go       # Branch naming
├── git_commit.go       # Commit formatting
├── github.go           # GitHub provider
├── gitlab.go           # GitLab provider
├── errors.go           # Error definitions
└── git_test.go         # Tests
```

---

## Dependencies

```go
// go.mod
require (
    github.com/google/go-github/v57 v57.0.0
    github.com/xanzy/go-gitlab v0.95.0
)
```

---

## Completion Criteria

- [ ] All tasks implemented
- [ ] Unit test coverage > 80%
- [ ] Integration tests pass
- [ ] No lint warnings
- [ ] Documentation complete

---

## References

- ADR-001: Worktree Strategy
- ADR-002: Git Operations Interface
- ADR-003: Branch Naming
- ADR-004: Commit Formatting
- ADR-005: PR Creation
