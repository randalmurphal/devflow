# devflow

[![Go Reference](https://pkg.go.dev/badge/github.com/yourorg/devflow.svg)](https://pkg.go.dev/github.com/yourorg/devflow)
[![Go Report Card](https://goreportcard.com/badge/github.com/yourorg/devflow)](https://goreportcard.com/report/github.com/yourorg/devflow)

**Development workflow primitives for Go.** Git operations, Claude CLI integration, transcript management, and artifact storage.

## Features

- **Git operations** - Worktrees, commits, branches, PRs
- **Claude integration** - CLI wrapper with context injection
- **Transcripts** - Record and store AI conversations
- **Artifacts** - Manage workflow outputs
- **flowgraph nodes** - Pre-built nodes for workflows

## Installation

```bash
go get github.com/yourorg/devflow
```

## Quick Start

```go
package main

import (
    "context"
    "github.com/yourorg/devflow"
)

func main() {
    // Git operations
    git, _ := devflow.NewGitContext(".")
    worktree, _ := git.CreateWorktree("feature/new-api")
    defer git.CleanupWorktree(worktree)

    // Claude operations
    claude := devflow.NewClaudeCLI(devflow.ClaudeConfig{
        Timeout: 5 * time.Minute,
    })
    result, _ := claude.Run(context.Background(), "Implement the feature",
        devflow.WithWorkDir(worktree),
        devflow.WithContext("main.go"),
    )

    // Commit and PR
    git.Commit("Add feature", "main.go")
    pr, _ := git.CreatePR(devflow.PROptions{
        Title: "Add new feature",
        Base:  "main",
    })
}
```

## Documentation

- [CLAUDE.md](CLAUDE.md) - AI-readable project reference
- [docs/OVERVIEW.md](docs/OVERVIEW.md) - Detailed concepts
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) - Design decisions
- [docs/API_REFERENCE.md](docs/API_REFERENCE.md) - Full API

## Ecosystem

devflow is the middle layer of a three-layer ecosystem:

| Layer | Repo | Purpose |
|-------|------|---------|
| flowgraph | [github.com/yourorg/flowgraph](https://github.com/yourorg/flowgraph) | Graph orchestration engine |
| **devflow** | This repo | Dev workflow primitives |
| task-keeper | [github.com/yourorg/task-keeper](https://github.com/yourorg/task-keeper) | Commercial SaaS product |

## License

MIT License - see [LICENSE](LICENSE) for details.
