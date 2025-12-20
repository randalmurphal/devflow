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
```

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

## File Structure

```
pr/
├── pr.go       # Provider interface, Options, PullRequest
├── builder.go  # PR description builder
├── github.go   # GitHubProvider
├── gitlab.go   # GitLabProvider
└── errors.go   # PR-specific errors
```
