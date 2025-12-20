# devflow

[![CI](https://github.com/randalmurphal/devflow/actions/workflows/ci.yml/badge.svg)](https://github.com/randalmurphal/devflow/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/randalmurphal/devflow.svg)](https://pkg.go.dev/github.com/randalmurphal/devflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/randalmurphal/devflow)](https://goreportcard.com/report/github.com/randalmurphal/devflow)
[![Coverage](https://img.shields.io/badge/coverage-83%25-brightgreen)](https://github.com/randalmurphal/devflow)

**Development workflow primitives for AI-powered automation.** Git operations, LLM integration via flowgraph, transcript management, artifact storage, and notifications.

## Features

- **Git Operations** - Worktrees, commits, branches, PRs (GitHub & GitLab)
- **LLM Integration** - Uses [flowgraph](https://github.com/randalmurphal/flowgraph)'s `llm.Client` interface
- **Transcripts** - Record, search, and export AI conversations
- **Artifacts** - Store and manage workflow outputs with lifecycle management
- **Notifications** - Slack, webhook, and logging integrations
- **Workflow Nodes** - Pre-built nodes for common dev workflows

## Installation

```bash
go get github.com/randalmurphal/devflow
```

**Note:** devflow depends on [flowgraph](https://github.com/randalmurphal/flowgraph) for graph orchestration and LLM abstraction.

## Quick Start

### Git Operations

```go
package main

import (
    "fmt"
    "github.com/randalmurphal/devflow"
)

func main() {
    // Create git context for repository operations
    git, _ := devflow.NewGitContext(".")

    // Create an isolated worktree for feature work
    worktree, _ := git.CreateWorktree("feature/new-api")
    defer git.CleanupWorktree(worktree)

    // Check status and branch
    branch, _ := git.CurrentBranch()
    clean, _ := git.IsClean()
    fmt.Printf("Branch: %s, Clean: %v\n", branch, clean)

    // Commit changes
    _ = git.Commit("Add new feature", "main.go", "utils.go")

    // Create PR (with GitHub provider configured)
    pr, _ := git.CreatePR(devflow.PROptions{
        Title: "Add new feature",
        Body:  "Implements the new API endpoint",
        Base:  "main",
    })
    fmt.Printf("PR URL: %s\n", pr.URL)
}
```

### Workflow with LLM Integration

```go
package main

import (
    "context"
    "fmt"

    "github.com/randalmurphal/devflow"
    "github.com/randalmurphal/flowgraph"
    "github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"
)

func main() {
    ctx := context.Background()

    // Create services
    git, _ := devflow.NewGitContext(".")
    client := llm.NewClaudeCLI(
        llm.WithModel("claude-sonnet-4-20250514"),
        llm.WithWorkdir("."),
    )
    notifier := devflow.NewSlackNotifier("https://hooks.slack.com/...")

    // Inject services into context
    ctx = devflow.WithGitContext(ctx, git)
    ctx = devflow.WithLLMClient(ctx, client)
    ctx = devflow.WithNotifier(ctx, notifier)

    // Build workflow graph using flowgraph
    graph := flowgraph.NewGraph[devflow.DevState]().
        AddNode("create-worktree", devflow.CreateWorktreeNode).
        AddNode("generate-spec", devflow.GenerateSpecNode).
        AddNode("implement", devflow.ImplementNode).
        AddNode("review", devflow.ReviewNode).
        AddNode("create-pr", devflow.CreatePRNode).
        AddEdge("create-worktree", "generate-spec").
        AddEdge("generate-spec", "implement").
        AddEdge("implement", "review").
        AddEdge("review", "create-pr").
        AddEdge("create-pr", flowgraph.END).
        SetEntry("create-worktree")

    // Execute workflow
    state := devflow.NewDevState("ticket-to-pr")
    result, _ := graph.Execute(ctx, state)
    fmt.Printf("PR created: %s\n", result.PR.URL)
}
```

### Transcript Management

```go
store, _ := devflow.NewFileTranscriptStore(".devflow/runs")

// Start a run
runID := "run-123"
_ = store.StartRun(runID, devflow.RunMetadata{
    FlowID: "code-review",
    Input:  map[string]any{"pr": 456},
})

// Record conversation turns
_ = store.RecordTurn(runID, devflow.Turn{
    Role:    "user",
    Content: "Review this pull request",
})
_ = store.RecordTurn(runID, devflow.Turn{
    Role:    "assistant",
    Content: "I'll analyze the changes...",
})

// End run
_ = store.EndRun(runID, devflow.RunStatusCompleted)

// Search transcripts
searcher := devflow.NewTranscriptSearcher(".devflow/runs")
results, _ := searcher.Search("error handling")
```

### Artifact Storage

```go
artifacts := devflow.NewArtifactManager(devflow.ArtifactConfig{
    BaseDir:       ".devflow/runs",
    CompressAbove: 1024, // Compress files > 1KB
})

// Save artifacts
_ = artifacts.SaveArtifact("run-123", "spec.md", []byte("# Specification\n..."))
_ = artifacts.SaveArtifact("run-123", "output.json", jsonData)

// Load artifacts
data, _ := artifacts.LoadArtifact("run-123", "spec.md")

// Lifecycle management
lifecycle := devflow.NewLifecycleManager(devflow.LifecycleConfig{
    BaseDir:        ".devflow/runs",
    RetentionDays:  30,
    ArchiveEnabled: true,
})
_ = lifecycle.Cleanup() // Archive/delete old runs
```

### Notifications

```go
// Slack notifications
slack := devflow.NewSlackNotifier(webhookURL,
    devflow.WithSlackChannel("#dev-alerts"),
    devflow.WithSlackUsername("devflow-bot"),
)

// Webhook notifications
webhook := devflow.NewWebhookNotifier(url, headers)

// Combine multiple notifiers
multi := devflow.NewMultiNotifier(slack, webhook)

// Inject and use
ctx = devflow.WithNotifier(ctx, multi)
devflow.NotifyRunStarted(ctx, state)
devflow.NotifyRunCompleted(ctx, state)
```

## Workflow Nodes

devflow provides pre-built nodes for common development workflows:

| Node | Purpose |
|------|---------|
| `CreateWorktreeNode` | Create isolated git worktree |
| `GenerateSpecNode` | Generate specification from ticket |
| `ImplementNode` | Implement code from spec |
| `ReviewNode` | Review code changes |
| `FixFindingsNode` | Fix review findings |
| `RunTestsNode` | Execute test suite |
| `CheckLintNode` | Run linters |
| `CreatePRNode` | Create pull request |
| `CleanupNode` | Clean up worktree |
| `NotifyNode` | Send notifications |

## Documentation

- [CLAUDE.md](CLAUDE.md) - AI-readable project reference
- [docs/OVERVIEW.md](docs/OVERVIEW.md) - Detailed concepts and vision
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - Component design and data flow
- [docs/API_REFERENCE.md](docs/API_REFERENCE.md) - Complete public API

## Ecosystem

devflow is the middle layer of a three-layer ecosystem:

| Layer | Repo | Purpose |
|-------|------|---------|
| flowgraph | [github.com/randalmurphal/flowgraph](https://github.com/randalmurphal/flowgraph) | Graph orchestration engine + LLM abstraction |
| **devflow** | This repo | Dev workflow primitives |
| task-keeper | Commercial | SaaS product built on devflow |

## Development

```bash
# Run tests
go test -race ./...

# Run tests with coverage
go test -race -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run linter
golangci-lint run
```

## License

MIT License - see [LICENSE](LICENSE) for details.
