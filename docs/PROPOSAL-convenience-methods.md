# Proposal: Convenience Methods & Result Types

**Status**: Implemented (2025-12-25)
**Author**: task-keeper integration analysis
**Date**: 2025-12-25

## Problem Statement

Common git+PR workflows require multiple sequential calls with intermediate error handling:

```go
// Current: ~25 lines for "commit all and push"
ctx, _ := git.NewContext(repoPath)

if err := ctx.StageAll(); err != nil {
    return err
}
if err := ctx.Commit(message); err != nil {
    return err
}

sha, err := ctx.HeadCommit()
if err != nil {
    return err
}

branch, err := ctx.CurrentBranch()
if err != nil {
    return err
}

setUpstream := !ctx.IsBranchPushed(branch)
if err := ctx.Push("origin", branch, setUpstream); err != nil {
    return err
}
```

This verbosity leads to:
1. Copy-paste across projects
2. Inconsistent upstream tracking logic
3. Missing structured results for downstream operations
4. Manual provider detection for PRs

## Proposed Solution

Add convenience methods to existing types that bundle common operations:

```go
// New: 2 lines with structured results
result, err := ctx.CommitAll("Add feature X")
pushResult, err := ctx.PushCurrent()
```

## Design Principles

### 1. Additive Only
- New methods on existing types
- No changes to existing APIs
- Full backward compatibility

### 2. Structured Results
- Return result structs instead of multiple values
- Include all relevant context (SHA, branch, URL)
- Enable method chaining and downstream operations

### 3. Sensible Defaults
- Auto-detect upstream tracking needs
- Use "origin" as default remote
- Smart branch handling

### 4. Escape Hatches Preserved
- Original granular methods remain available
- Result structs expose all data for custom logic

---

## Detailed Design

### Git Context Additions

Location: `git/git.go` (add to existing `Context` type)

#### CommitAll

```go
// CommitResult contains the result of a commit operation.
type CommitResult struct {
    SHA     string // Full commit SHA
    Branch  string // Branch name
    Message string // Commit message
    Author  string // Author name
    Date    time.Time
}

// CommitAll stages all changes and commits with the given message.
// Returns ErrNothingToCommit if there are no changes to commit.
// This is a convenience method combining StageAll + Commit + HeadCommit + CurrentBranch.
func (c *Context) CommitAll(message string) (*CommitResult, error) {
    if err := c.StageAll(); err != nil {
        return nil, fmt.Errorf("stage all: %w", err)
    }

    if err := c.Commit(message); err != nil {
        return nil, err // Already wrapped appropriately
    }

    sha, err := c.HeadCommit()
    if err != nil {
        return nil, fmt.Errorf("get head: %w", err)
    }

    branch, err := c.CurrentBranch()
    if err != nil {
        return nil, fmt.Errorf("get branch: %w", err)
    }

    return &CommitResult{
        SHA:     sha,
        Branch:  branch,
        Message: message,
        Date:    time.Now(),
    }, nil
}
```

#### PushCurrent

```go
// PushResult contains the result of a push operation.
type PushResult struct {
    Remote     string // Remote name (e.g., "origin")
    Branch     string // Branch that was pushed
    SHA        string // Commit SHA that was pushed
    SetUpstream bool  // Whether upstream tracking was set
    URL        string // Remote URL (for reference)
}

// PushCurrent pushes the current branch to origin.
// Automatically sets upstream tracking if the branch hasn't been pushed before.
// This is a convenience method that handles the common case of pushing work.
func (c *Context) PushCurrent() (*PushResult, error) {
    return c.PushCurrentTo("origin")
}

// PushCurrentTo pushes the current branch to the specified remote.
// Automatically sets upstream tracking if needed.
func (c *Context) PushCurrentTo(remote string) (*PushResult, error) {
    branch, err := c.CurrentBranch()
    if err != nil {
        return nil, fmt.Errorf("get current branch: %w", err)
    }

    setUpstream := !c.IsBranchPushed(branch)

    if err := c.Push(remote, branch, setUpstream); err != nil {
        return nil, err
    }

    sha, err := c.HeadCommit()
    if err != nil {
        return nil, fmt.Errorf("get head: %w", err)
    }

    url, _ := c.GetRemoteURL(remote) // Ignore error, URL is optional

    return &PushResult{
        Remote:      remote,
        Branch:      branch,
        SHA:         sha,
        SetUpstream: setUpstream,
        URL:         url,
    }, nil
}
```

#### CheckoutNew

```go
// CheckoutNew creates and checks out a new branch at the current HEAD.
// This is a convenience method combining CreateBranch + Checkout.
func (c *Context) CheckoutNew(name string) error {
    if err := c.CreateBranch(name); err != nil {
        return err
    }
    return c.Checkout(name)
}

// CheckoutNewAt creates and checks out a new branch at the specified ref.
func (c *Context) CheckoutNewAt(name, ref string) error {
    // First checkout the ref
    if err := c.Checkout(ref); err != nil {
        return fmt.Errorf("checkout %q: %w", ref, err)
    }
    // Then create the branch
    if err := c.CreateBranch(name); err != nil {
        return fmt.Errorf("create branch %q: %w", name, err)
    }
    return nil
}
```

#### CommitAndPush

```go
// CommitAndPushResult contains the result of a commit and push operation.
type CommitAndPushResult struct {
    Commit *CommitResult
    Push   *PushResult
}

// CommitAllAndPush stages all changes, commits, and pushes to origin.
// This is the most common workflow: save work and push it.
func (c *Context) CommitAllAndPush(message string) (*CommitAndPushResult, error) {
    commit, err := c.CommitAll(message)
    if err != nil {
        return nil, err
    }

    push, err := c.PushCurrent()
    if err != nil {
        return &CommitAndPushResult{Commit: commit}, err
    }

    return &CommitAndPushResult{
        Commit: commit,
        Push:   push,
    }, nil
}
```

---

### PR Provider Detection

Location: `pr/detect.go` (new file)

```go
package pr

import (
    "context"
    "fmt"
    "os"
    "strings"
)

// DetectProvider determines the git provider from a remote URL.
// Returns "github" or "gitlab".
func DetectProvider(remoteURL string) (string, error) {
    url := strings.ToLower(remoteURL)

    switch {
    case strings.Contains(url, "github.com"):
        return "github", nil
    case strings.Contains(url, "gitlab.com"):
        return "gitlab", nil
    case strings.Contains(url, "gitlab"):
        // Self-hosted GitLab instances often contain "gitlab" in URL
        return "gitlab", nil
    default:
        return "", fmt.Errorf("%w: cannot detect provider from %q", ErrUnknownProvider, remoteURL)
    }
}

// ProviderFromEnv creates a provider based on remote URL and environment.
// Automatically detects GitHub vs GitLab and uses appropriate token env var.
//
// Environment variables checked:
//   - GITHUB_TOKEN for GitHub
//   - GITLAB_TOKEN for GitLab
//   - GIT_TOKEN as fallback for either
func ProviderFromEnv(remoteURL string) (Provider, error) {
    platform, err := DetectProvider(remoteURL)
    if err != nil {
        return nil, err
    }

    switch platform {
    case "github":
        token := os.Getenv("GITHUB_TOKEN")
        if token == "" {
            token = os.Getenv("GIT_TOKEN")
        }
        if token == "" {
            return nil, fmt.Errorf("GITHUB_TOKEN or GIT_TOKEN required for GitHub")
        }
        return NewGitHubProviderFromURL(token, remoteURL)

    case "gitlab":
        token := os.Getenv("GITLAB_TOKEN")
        if token == "" {
            token = os.Getenv("GIT_TOKEN")
        }
        if token == "" {
            return nil, fmt.Errorf("GITLAB_TOKEN or GIT_TOKEN required for GitLab")
        }
        return NewGitLabProviderFromURL(token, remoteURL)

    default:
        return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, platform)
    }
}

// ProviderFromEnvWithToken creates a provider with an explicit token.
// Use this when you have the token from configuration rather than environment.
func ProviderFromEnvWithToken(remoteURL, token string) (Provider, error) {
    platform, err := DetectProvider(remoteURL)
    if err != nil {
        return nil, err
    }

    switch platform {
    case "github":
        return NewGitHubProviderFromURL(token, remoteURL)
    case "gitlab":
        return NewGitLabProviderFromURL(token, remoteURL)
    default:
        return nil, fmt.Errorf("%w: %s", ErrUnknownProvider, platform)
    }
}
```

---

### Context Injection Helpers

Location: `git/context_helpers.go` (new file)

```go
package git

import "context"

// contextKey is a private type for context keys to avoid collisions.
type contextKey struct{ name string }

var gitContextKey = &contextKey{"git-context"}

// ContextWithGit adds a git Context to a context.Context.
// Use GitFromContext to retrieve it.
func ContextWithGit(ctx context.Context, gc *Context) context.Context {
    return context.WithValue(ctx, gitContextKey, gc)
}

// GitFromContext retrieves a git Context from a context.Context.
// Returns nil if no git Context is present.
func GitFromContext(ctx context.Context) *Context {
    if gc, ok := ctx.Value(gitContextKey).(*Context); ok {
        return gc
    }
    return nil
}

// MustGitFromContext retrieves a git Context or panics.
// Use in code where git context is required and missing is a programming error.
func MustGitFromContext(ctx context.Context) *Context {
    gc := GitFromContext(ctx)
    if gc == nil {
        panic("git.Context not found in context")
    }
    return gc
}
```

Location: `pr/context_helpers.go` (new file)

```go
package pr

import "context"

type contextKey struct{ name string }

var prProviderKey = &contextKey{"pr-provider"}

// ContextWithProvider adds a PR Provider to a context.Context.
func ContextWithProvider(ctx context.Context, p Provider) context.Context {
    return context.WithValue(ctx, prProviderKey, p)
}

// ProviderFromContext retrieves a PR Provider from a context.Context.
// Returns nil if no Provider is present.
func ProviderFromContext(ctx context.Context) Provider {
    if p, ok := ctx.Value(prProviderKey).(Provider); ok {
        return p
    }
    return nil
}
```

---

### PR Result Enhancements

Location: `pr/pr.go` (additions to existing file)

```go
// CreatePRResult extends PullRequest with creation-specific metadata.
type CreatePRResult struct {
    *PullRequest

    // Created indicates this is a newly created PR (vs existing).
    Created bool

    // ExistingID is set if a PR already existed for this branch.
    ExistingID int
}

// CreateOrGetPR creates a PR or returns the existing one if it already exists.
// This handles the common case where you want to ensure a PR exists.
func (p *GitHubProvider) CreateOrGetPR(ctx context.Context, opts Options) (*CreatePRResult, error) {
    pr, err := p.CreatePR(ctx, opts)
    if err == nil {
        return &CreatePRResult{PullRequest: pr, Created: true}, nil
    }

    if errors.Is(err, ErrExists) {
        // Find existing PR
        prs, listErr := p.ListPRs(ctx, Filter{
            Head:  opts.Head,
            Base:  opts.Base,
            State: StateOpen,
            Limit: 1,
        })
        if listErr != nil {
            return nil, fmt.Errorf("find existing PR: %w", listErr)
        }
        if len(prs) > 0 {
            return &CreatePRResult{
                PullRequest: prs[0],
                Created:     false,
                ExistingID:  prs[0].ID,
            }, nil
        }
    }

    return nil, err
}
```

---

## Usage Examples

### Common Workflow: Feature Branch

```go
package main

import (
    "log"

    "github.com/randalmurphal/devflow/git"
    "github.com/randalmurphal/devflow/pr"
)

func main() {
    ctx, _ := git.NewContext(".")

    // Create feature branch
    if err := ctx.CheckoutNew("feature/add-auth"); err != nil {
        log.Fatal(err)
    }

    // ... make changes ...

    // Commit and push in one call
    result, err := ctx.CommitAllAndPush("feat(auth): add OAuth2 support")
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Pushed %s to %s/%s",
        result.Commit.SHA[:8],
        result.Push.Remote,
        result.Push.Branch)

    // Create PR with auto-detected provider
    remoteURL, _ := ctx.GetRemoteURL("origin")
    provider, err := pr.ProviderFromEnv(remoteURL)
    if err != nil {
        log.Fatal(err)
    }

    prResult, err := provider.CreatePR(ctx, pr.Options{
        Title: "[TK-123] Add OAuth2 authentication",
        Body:  "Implements OAuth2 flow for SSO",
        Head:  result.Push.Branch,
        Base:  "main",
    })
    if err != nil {
        log.Fatal(err)
    }

    log.Printf("Created PR: %s", prResult.HTMLURL)
}
```

### With Context Injection

```go
package main

import (
    "context"

    "github.com/randalmurphal/devflow/git"
    "github.com/randalmurphal/devflow/pr"
)

func setupContext() context.Context {
    ctx := context.Background()

    gitCtx, _ := git.NewContext(".")
    ctx = git.ContextWithGit(ctx, gitCtx)

    remoteURL, _ := gitCtx.GetRemoteURL("origin")
    provider, _ := pr.ProviderFromEnv(remoteURL)
    ctx = pr.ContextWithProvider(ctx, provider)

    return ctx
}

func commitAndPush(ctx context.Context, message string) error {
    gitCtx := git.GitFromContext(ctx)
    _, err := gitCtx.CommitAllAndPush(message)
    return err
}

func createPR(ctx context.Context, title, body string) (*pr.PullRequest, error) {
    provider := pr.ProviderFromContext(ctx)
    gitCtx := git.GitFromContext(ctx)

    branch, _ := gitCtx.CurrentBranch()

    return provider.CreatePR(ctx, pr.Options{
        Title: title,
        Body:  body,
        Head:  branch,
        Base:  "main",
    })
}
```

### Testing with Mocks

```go
func TestCommitAll(t *testing.T) {
    runner := git.NewMockRunner()
    runner.OnCommand("git", "add", "-A").Return("", nil)
    runner.OnCommand("git", "commit", "-m", "test").Return("", nil)
    runner.OnCommand("git", "rev-parse", "HEAD").Return("abc123", nil)
    runner.OnCommand("git", "rev-parse", "--abbrev-ref", "HEAD").Return("main", nil)

    ctx, _ := git.NewContext(t.TempDir(), git.WithRunner(runner))

    result, err := ctx.CommitAll("test")
    if err != nil {
        t.Fatal(err)
    }

    if result.SHA != "abc123" {
        t.Errorf("SHA = %q, want %q", result.SHA, "abc123")
    }
    if result.Branch != "main" {
        t.Errorf("Branch = %q, want %q", result.Branch, "main")
    }
}
```

---

## Result Type Summary

| Method | Returns | Fields |
|--------|---------|--------|
| `CommitAll(msg)` | `*CommitResult` | SHA, Branch, Message, Author, Date |
| `PushCurrent()` | `*PushResult` | Remote, Branch, SHA, SetUpstream, URL |
| `PushCurrentTo(remote)` | `*PushResult` | Same as above |
| `CommitAllAndPush(msg)` | `*CommitAndPushResult` | Commit, Push |
| `CreateOrGetPR(opts)` | `*CreatePRResult` | *PullRequest, Created, ExistingID |

---

## Implementation Plan

### Phase 1: Core Convenience Methods (DONE)
- [x] `CommitResult` type (git/convenience.go)
- [x] `CommitAll()` method (git/convenience.go)
- [x] `PushResult` type (git/convenience.go)
- [x] `PushCurrent()` and `PushCurrentTo()` (git/convenience.go)
- [x] `CommitAllAndPush()` (git/convenience.go)

### Phase 2: Branch Helpers (DONE)
- [x] `CheckoutNew()` (git/convenience.go)
- [x] `CheckoutNewAt()` (git/convenience.go)

### Phase 3: PR Provider Detection (DONE)
- [x] `DetectProvider()` (pr/detect.go - already existed)
- [x] `ProviderFromEnv()` (pr/detect.go)
- [x] `ProviderFromEnvWithToken()` (pr/detect.go)
- [ ] `CreateOrGetPR()` (not implemented - optional enhancement)

### Phase 4: Context Injection (DONE)
- [x] `git.ContextWithGit()` / `GitFromContext()` (git/context_helpers.go)
- [x] `pr.ContextWithProvider()` / `ProviderFromContext()` (pr/context_helpers.go)

### Phase 5: Documentation (DONE)
- [x] Update CLAUDE.md
- [x] Add tests (git/convenience_test.go)
- [ ] Update README with new methods (optional)

---

## Migration Path

### For Existing Users

No changes required. All additions are new methods.

### For task-keeper

Replace `internal/devflow/` wrapper:

```go
// Old: internal wrapper
import tkdevflow "github.com/randalmurphal/task-keeper/internal/devflow"
client := tkdevflow.NewGitClient(path)
result, _ := client.CommitAll(msg)

// New: direct use
import "github.com/randalmurphal/devflow/git"
ctx, _ := git.NewContext(path)
result, _ := ctx.CommitAll(msg)
```

---

## Alternatives Considered

### 1. Separate High-Level Client

```go
client := git.NewHighLevelClient(path)
client.CommitAll(msg)
```

**Rejected because:**
- Creates parallel API surface
- Confusing which to use
- Methods naturally belong on Context

### 2. Return Multiple Values

```go
sha, branch, err := ctx.CommitAll(msg)
```

**Rejected because:**
- Hard to extend (adding fields breaks signature)
- Can't pass result as single value
- No room for optional metadata

### 3. Fluent/Builder Pattern

```go
ctx.Stage().All().Commit(msg).Push().Execute()
```

**Rejected because:**
- Overcomplicated for simple operations
- Harder to handle errors
- Doesn't match existing API style

---

## Open Questions

1. **Return partial results on error?**
   - `CommitAllAndPush` returns `*CommitAndPushResult` even if push fails
   - Allows caller to see what succeeded
   - Current proposal: yes, return partial results

2. **Author extraction for CommitResult?**
   - Requires additional git command
   - Maybe make it lazy/optional?

3. **Should `CreateOrGetPR` be on Provider interface?**
   - Currently only on GitHubProvider
   - Would need GitLab implementation too

---

## References

- task-keeper integration: `internal/devflow/git.go`, `pr.go`
- Current devflow API: `git/git.go`, `pr/pr.go`
- Builder patterns: `pr/pr.go` (existing Builder)
