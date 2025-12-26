# pr package

Pull request operations for GitHub and GitLab.

## Quick Reference

| Type | Purpose |
|------|---------|
| `Provider` | Interface for PR/MR operations |
| `Options` | Configuration for creating PRs |
| `PullRequest` | Created PR with URL, number |
| `Builder` | Fluent builder for PR descriptions |
| `GitHubProvider` | GitHub implementation |
| `GitLabProvider` | GitLab implementation |
| `MockProvider` | Mock for testing |

## Provider Interface

```go
type Provider interface {
    CreatePR(ctx context.Context, opts Options) (*PullRequest, error)
    GetPR(ctx context.Context, number int) (*PullRequest, error)
    UpdatePR(ctx context.Context, number int, opts Options) error
    MergePR(ctx context.Context, number int, method string) error
    ClosePR(ctx context.Context, number int) error
}
```

## Creating Providers

```go
// GitHub
github, err := pr.NewGitHubProvider(token, "owner", "repo")

// GitLab
gitlab, err := pr.NewGitLabProvider(token, projectID)

// Auto-detect from remote URL (uses environment tokens)
remoteURL, _ := gitCtx.GetRemoteURL("origin")
provider, err := pr.ProviderFromEnv(remoteURL)

// With explicit token
provider, err := pr.ProviderFromEnvWithToken(remoteURL, token)
```

**Environment variables for auto-detection:**
- `GITHUB_TOKEN` - For GitHub repos
- `GITLAB_TOKEN` - For GitLab repos
- `GIT_TOKEN` - Fallback for either

## Creating Pull Requests

```go
pull, err := provider.CreatePR(ctx, pr.Options{
    Title:  "Add user authentication",
    Body:   "Implements OAuth2 flow",
    Base:   "main",
    Head:   "feature/auth",
    Draft:  false,
    Labels: []string{"enhancement"},
})

fmt.Println(pull.URL)    // https://github.com/...
fmt.Println(pull.Number) // 42
```

## PR Builder

```go
body := pr.NewBuilder().
    WithSummary("Implements authentication").
    WithChanges([]string{"Added OAuth2 flow", "Added tests"}).
    WithTestPlan("Run auth tests").
    WithTicketRef("TK-123").
    Build()
```

## Context Injection

```go
// Add provider to context.Context
ctx := pr.ContextWithProvider(context.Background(), provider)

// Retrieve later
provider := pr.ProviderFromContext(ctx)
provider := pr.MustProviderFromContext(ctx)  // panics if missing
```

## Testing with Mock

```go
mock := &pr.MockProvider{
    CreatePRFunc: func(ctx context.Context, opts pr.Options) (*pr.PullRequest, error) {
        return &pr.PullRequest{ID: 42, URL: "https://example.com/pr/42"}, nil
    },
}
ctx := pr.ContextWithProvider(context.Background(), mock)
```

## File Structure

```
pr/
├── pr.go              # Provider interface, Options, PullRequest
├── detect.go          # ProviderFromEnv, ProviderFromEnvWithToken
├── context_helpers.go # ContextWithProvider, ProviderFromContext
├── builder.go         # PR description builder
├── github.go          # GitHubProvider
├── gitlab.go          # GitLabProvider
├── mock.go            # MockProvider for testing
└── errors.go          # PR-specific errors
```
