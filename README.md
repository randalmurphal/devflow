# devflow

[![CI](https://github.com/randalmurphal/devflow/actions/workflows/ci.yml/badge.svg)](https://github.com/randalmurphal/devflow/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/randalmurphal/devflow.svg)](https://pkg.go.dev/github.com/randalmurphal/devflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/randalmurphal/devflow)](https://goreportcard.com/report/github.com/randalmurphal/devflow)
[![Coverage](https://img.shields.io/badge/coverage-83%25-brightgreen)](https://github.com/randalmurphal/devflow)

**Development workflow primitives for AI-powered automation.** Git operations, LLM integration via flowgraph, transcript management, artifact storage, and notifications.

## Features

- **Git Operations** - Worktrees, commits, branches, PRs (GitHub & GitLab)
- **LLM Integration** - Uses [flowgraph](https://github.com/randalmurphal/flowgraph)'s `llm.Client` interface
- **LLM Utilities** - Token counting, text truncation, prompt templates, response parsing
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
    "github.com/randalmurphal/devflow/git"
)

func main() {
    // Create git context for repository operations
    gitCtx, _ := git.NewContext(".")

    // Create an isolated worktree for feature work
    worktree, _ := gitCtx.CreateWorktree("feature/new-api")
    defer gitCtx.CleanupWorktree(worktree)

    // Check status and branch
    branch, _ := gitCtx.CurrentBranch()
    clean, _ := gitCtx.IsClean()
    fmt.Printf("Branch: %s, Clean: %v\n", branch, clean)

    // Commit changes
    _ = gitCtx.Commit("Add new feature")
}
```

### Workflow with LLM Integration

```go
package main

import (
    "context"
    "fmt"

    devcontext "github.com/randalmurphal/devflow/context"
    "github.com/randalmurphal/devflow/git"
    "github.com/randalmurphal/devflow/notify"
    "github.com/randalmurphal/devflow/workflow"
    "github.com/randalmurphal/flowgraph/pkg/flowgraph"
    "github.com/randalmurphal/flowgraph/pkg/flowgraph/llm"
)

func main() {
    ctx := context.Background()

    // Create services
    gitCtx, _ := git.NewContext(".")
    client := llm.NewClaudeCLI(
        llm.WithModel("claude-sonnet-4-20250514"),
        llm.WithWorkdir("."),
    )
    notifier := notify.NewSlack("https://hooks.slack.com/...")

    // Inject services into context
    ctx = devcontext.WithGit(ctx, gitCtx)
    ctx = devcontext.WithLLM(ctx, client)
    ctx = notify.WithNotifier(ctx, notifier)

    // Build workflow graph using flowgraph
    graph := flowgraph.NewGraph[workflow.State]().
        AddNode("create-worktree", workflow.CreateWorktreeNode).
        AddNode("generate-spec", workflow.GenerateSpecNode).
        AddNode("implement", workflow.ImplementNode).
        AddNode("review", workflow.ReviewNode).
        AddNode("create-pr", workflow.CreatePRNode).
        AddEdge("create-worktree", "generate-spec").
        AddEdge("generate-spec", "implement").
        AddEdge("implement", "review").
        AddEdge("review", "create-pr").
        AddEdge("create-pr", flowgraph.END).
        SetEntry("create-worktree")

    // Execute workflow
    state := workflow.NewState("ticket-to-pr")
    result, _ := graph.Execute(ctx, state)
    fmt.Printf("PR created: %s\n", result.PR.URL)
}
```

### Transcript Management

```go
import "github.com/randalmurphal/devflow/transcript"

store, _ := transcript.NewFileStore(transcript.StoreConfig{
    BaseDir: ".devflow/runs",
})

// Start a run
runID := "run-123"
_ = store.StartRun(runID, transcript.RunMetadata{
    FlowID: "code-review",
    Input:  map[string]any{"pr": 456},
})

// Record conversation turns
_ = store.RecordTurn(runID, transcript.Turn{
    Role:    "user",
    Content: "Review this pull request",
})
_ = store.RecordTurn(runID, transcript.Turn{
    Role:    "assistant",
    Content: "I'll analyze the changes...",
})

// End run
_ = store.EndRun(runID, transcript.RunStatusCompleted)

// Search transcripts
searcher := transcript.NewSearcher(".devflow/runs")
results, _ := searcher.Search("error handling")
```

### Artifact Storage

```go
import "github.com/randalmurphal/devflow/artifact"

artifacts := artifact.NewManager(artifact.Config{
    BaseDir:       ".devflow/runs",
    CompressAbove: 1024, // Compress files > 1KB
})

// Save artifacts
_ = artifacts.SaveArtifact("run-123", "spec.md", []byte("# Specification\n..."))
_ = artifacts.SaveArtifact("run-123", "output.json", jsonData)

// Load artifacts
data, _ := artifacts.LoadArtifact("run-123", "spec.md")

// Lifecycle management
lifecycle := artifact.NewLifecycleManager(artifacts, artifact.LifecycleConfig{
    RetentionDays:  30,
    ArchiveAfter:   7,
})
_ = lifecycle.Cleanup() // Archive/delete old runs
```

### Notifications

```go
import "github.com/randalmurphal/devflow/notify"

// Slack notifications
slack := notify.NewSlack(webhookURL,
    notify.WithChannel("#dev-alerts"),
    notify.WithUsername("devflow-bot"),
)

// Webhook notifications
webhook := notify.NewWebhook(url, headers)

// Combine multiple notifiers
multi := notify.NewMulti(slack, webhook)

// Inject and use
ctx = notify.WithNotifier(ctx, multi)

// Send notification
_ = multi.Notify(ctx, notify.Event{
    Type:    notify.EventRunCompleted,
    Message: "Workflow completed",
})
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

## Package Structure

```
github.com/randalmurphal/devflow/
├── artifact/      # Artifact storage and lifecycle
├── context/       # Context injection helpers
├── git/           # Git operations (worktrees, commits, branches)
├── http/          # HTTP client with connection pooling
├── llm/           # LLM utilities
│   ├── parser/    # Response parsing (JSON, YAML, code blocks)
│   ├── template/  # Prompt template engine (Handlebars-style)
│   ├── tokens/    # Token counting and budget management
│   └── truncate/  # Text truncation strategies
├── notify/        # Notifications (Slack, webhooks)
├── pr/            # Pull request operations (GitHub, GitLab)
├── prompt/        # Prompt loading
├── task/          # Task primitives
├── testutil/      # Test utilities
├── transcript/    # Conversation transcripts
└── workflow/      # Pre-built workflow nodes
```

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
