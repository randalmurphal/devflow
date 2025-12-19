# Node: create-pr

## Purpose

Create a pull request from the implemented and reviewed changes. This is typically the final step in a development workflow, pushing changes and creating a PR/MR.

## Signature

```go
func CreatePRNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

## Input State Requirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Worktree` | `string` | Yes | Working directory with changes |
| `Branch` | `string` | Yes | Branch name to push |
| `Ticket` | `*Ticket` | Optional | Source ticket for PR body |
| `Spec` | `*Spec` | Optional | Spec for PR description |
| `Implementation` | `*Implementation` | Optional | Summary for PR body |

## Output State Changes

| Field | Type | Description |
|-------|------|-------------|
| `PR` | `*PullRequest` | Created PR details |

### PullRequest Structure

```go
type PullRequest struct {
    ID        string    // Provider-specific ID
    Number    int       // PR number
    URL       string    // Web URL
    Title     string    // PR title
    Body      string    // PR description
    State     string    // open, merged, closed
    CreatedAt time.Time
}
```

## Prompt Template

Located at: `prompts/create-pr.txt` (for generating PR body)

```
Generate a pull request description for the following changes.

## Ticket
{{if .Ticket}}
ID: {{.Ticket.ID}}
Title: {{.Ticket.Title}}
{{end}}

## Specification Summary
{{if .Spec}}
{{.Spec.Overview}}
{{end}}

## Implementation Summary
{{if .Implementation}}
{{.Implementation.Summary}}

Files changed:
{{range .Implementation.FilesAdded}}- Added: {{.}}
{{end}}
{{range .Implementation.FilesChanged}}- Modified: {{.}}
{{end}}
{{end}}

## Output Format

# [Title - matches ticket title or brief summary]

## Summary
2-3 bullet points describing the changes.

## Changes
- List key changes

## Testing
How this was tested.

## Ticket
{{if .Ticket}}Closes #{{.Ticket.ID}}{{end}}
```

## Implementation

```go
func CreatePRNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if state.Worktree == "" {
        return state, fmt.Errorf("create-pr: worktree is required")
    }
    if state.Branch == "" {
        return state, fmt.Errorf("create-pr: branch is required")
    }

    git := devflow.GitContextFromContext(ctx)
    if git == nil {
        return state, fmt.Errorf("create-pr: GitContext not in context")
    }

    // Commit any uncommitted changes
    hasChanges, err := git.HasUncommittedChanges(state.Worktree)
    if err != nil {
        return state, fmt.Errorf("create-pr: check changes: %w", err)
    }

    if hasChanges {
        msg := generateCommitMessage(state)
        err = git.CommitAll(state.Worktree, msg)
        if err != nil {
            return state, fmt.Errorf("create-pr: commit: %w", err)
        }
    }

    // Push branch
    err = git.Push(state.Worktree, "origin", state.Branch)
    if err != nil {
        return state, fmt.Errorf("create-pr: push: %w", err)
    }

    // Generate PR body
    prBody, err := generatePRBody(ctx, state)
    if err != nil {
        // Non-fatal: use basic body
        prBody = fmt.Sprintf("Implementation for %s", state.Branch)
    }

    // Determine title
    title := state.Branch
    if state.Ticket != nil {
        title = fmt.Sprintf("[%s] %s", state.Ticket.ID, state.Ticket.Title)
    }

    // Create PR
    pr, err := git.CreatePR(devflow.PROptions{
        Title:  title,
        Body:   prBody,
        Head:   state.Branch,
        Base:   "main", // Configurable
        Draft:  false,
    })
    if err != nil {
        return state, fmt.Errorf("create-pr: create: %w", err)
    }

    state.PR = pr

    // Checkpoint
    ctx.Checkpoint("pr-created", state)

    return state, nil
}

func generateCommitMessage(state DevState) string {
    var parts []string

    if state.Ticket != nil {
        parts = append(parts, fmt.Sprintf("[%s]", state.Ticket.ID))
    }

    if state.Spec != nil && state.Spec.Overview != "" {
        // First line of overview
        overview := strings.Split(state.Spec.Overview, "\n")[0]
        parts = append(parts, overview)
    } else if state.Ticket != nil {
        parts = append(parts, state.Ticket.Title)
    } else {
        parts = append(parts, "Implementation")
    }

    return strings.Join(parts, " ")
}
```

## Provider Support

### GitHub

```go
type GitHubPRProvider struct {
    client *github.Client
    owner  string
    repo   string
}

func (p *GitHubPRProvider) CreatePR(opts PROptions) (*PullRequest, error) {
    pr, _, err := p.client.PullRequests.Create(ctx, p.owner, p.repo, &github.NewPullRequest{
        Title: github.String(opts.Title),
        Body:  github.String(opts.Body),
        Head:  github.String(opts.Head),
        Base:  github.String(opts.Base),
        Draft: github.Bool(opts.Draft),
    })
    if err != nil {
        return nil, err
    }

    return &PullRequest{
        ID:     strconv.Itoa(pr.GetID()),
        Number: pr.GetNumber(),
        URL:    pr.GetHTMLURL(),
        Title:  pr.GetTitle(),
        Body:   pr.GetBody(),
        State:  pr.GetState(),
    }, nil
}
```

### GitLab

```go
type GitLabMRProvider struct {
    client *gitlab.Client
    project string
}

func (p *GitLabMRProvider) CreatePR(opts PROptions) (*PullRequest, error) {
    mr, _, err := p.client.MergeRequests.CreateMergeRequest(p.project, &gitlab.CreateMergeRequestOptions{
        Title:        gitlab.String(opts.Title),
        Description:  gitlab.String(opts.Body),
        SourceBranch: gitlab.String(opts.Head),
        TargetBranch: gitlab.String(opts.Base),
    })
    if err != nil {
        return nil, err
    }

    return &PullRequest{
        ID:     strconv.Itoa(mr.ID),
        Number: mr.IID,
        URL:    mr.WebURL,
        Title:  mr.Title,
        Body:   mr.Description,
        State:  mr.State,
    }, nil
}
```

## Error Cases

| Error | Cause | Handling |
|-------|-------|----------|
| `worktree is required` | No worktree path | Fail, programming error |
| `branch is required` | No branch name | Fail, run create-worktree first |
| `commit: nothing to commit` | No changes staged | Skip commit, proceed |
| `push: rejected` | Remote conflicts | Pull/rebase then retry |
| `create: already exists` | PR exists for branch | Return existing PR |

### Handle Existing PR

```go
pr, err := git.CreatePR(opts)
if errors.Is(err, devflow.ErrPRExists) {
    // Get existing PR
    pr, err = git.GetPRForBranch(state.Branch)
    if err != nil {
        return state, fmt.Errorf("create-pr: get existing: %w", err)
    }
}
```

## Test Cases

### Unit Tests

```go
func TestCreatePRNode(t *testing.T) {
    tests := []struct {
        name    string
        state   DevState
        mockPR  *PullRequest
        wantErr bool
    }{
        {
            name: "creates PR with ticket",
            state: DevState{
                Worktree: "/tmp/test",
                Branch:   "feature/TK-421",
                Ticket:   &Ticket{ID: "TK-421", Title: "Add feature"},
            },
            mockPR: &PullRequest{
                Number: 42,
                URL:    "https://github.com/org/repo/pull/42",
            },
        },
        {
            name: "fails without worktree",
            state: DevState{
                Branch: "feature/test",
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockGit := &MockGitContext{
                PR: tt.mockPR,
            }
            ctx := devflow.WithGitContext(context.Background(), mockGit)

            result, err := CreatePRNode(flowgraph.WrapContext(ctx), tt.state)

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.NotNil(t, result.PR)
            assert.Equal(t, tt.mockPR.URL, result.PR.URL)
        })
    }
}
```

### Integration Tests

```go
func TestCreatePRNode_Integration(t *testing.T) {
    if testing.Short() || os.Getenv("GITHUB_TOKEN") == "" {
        t.Skip("skipping integration test")
    }

    git := devflow.NewGitContext(testRepoPath,
        devflow.WithGitHub(os.Getenv("GITHUB_TOKEN")),
    )
    worktree, _ := git.CreateWorktree("test-pr-" + randomSuffix())
    defer git.CleanupWorktree(worktree)

    // Make a change
    os.WriteFile(
        filepath.Join(worktree, "test.txt"),
        []byte("test content"),
        0644,
    )

    ctx := devflow.WithGitContext(context.Background(), git)

    state := DevState{
        Worktree: worktree,
        Branch:   "test-pr-" + randomSuffix(),
        Ticket: &Ticket{
            ID:    "TEST-1",
            Title: "Integration test PR",
        },
    }

    result, err := CreatePRNode(flowgraph.WrapContext(ctx), state)
    require.NoError(t, err)

    assert.NotNil(t, result.PR)
    assert.NotEmpty(t, result.PR.URL)
    assert.Equal(t, "open", result.PR.State)

    // Cleanup: close PR
    t.Cleanup(func() {
        git.ClosePR(result.PR.Number)
    })
}
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `BaseBranch` | `main` | Target branch for PR |
| `Draft` | `false` | Create as draft PR |
| `AutoMerge` | `false` | Enable auto-merge |
| `Labels` | `[]` | Labels to apply |
| `Reviewers` | `[]` | Reviewers to request |

## Artifacts Saved

| Artifact | Path | Description |
|----------|------|-------------|
| PR details | `pr.json` | PR metadata |

## References

- ADR-005: PR Creation
- Feature: Git Operations
- Phase 5: Workflow Nodes
