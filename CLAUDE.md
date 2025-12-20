# devflow

**Go library for AI-powered development workflows.** Git operations, LLM integration via flowgraph, transcript management, artifact storage, and notifications.

## Package Structure

```
devflow/
├── git/           # Git operations, worktrees, branches, commits
├── pr/            # Pull request providers (GitHub, GitLab)
├── artifact/      # Workflow artifact storage, lifecycle
├── transcript/    # Conversation recording, search, export
├── notify/        # Notification services (Slack, webhook)
├── workflow/      # State, workflow nodes
├── prompt/        # Prompt file loading
├── task/          # Task primitives
├── http/          # HTTP client utilities
├── context/       # Service dependency injection
├── testutil/      # Test utilities
└── integrationtest/ # Integration tests
```

See individual package CLAUDE.md files for details.

---

## Quick Start

```go
import (
    "github.com/randalmurphal/devflow/git"
    "github.com/randalmurphal/devflow/workflow"
    "github.com/randalmurphal/devflow/context"
    "github.com/randalmurphal/devflow/notify"
    "github.com/randalmurphal/flowgraph"
)

// Create services
gitCtx, _ := git.NewContext("/path/to/repo")
notifier := notify.NewSlack(webhookURL)

// Build workflow graph
graph := flowgraph.NewGraph[workflow.State]().
    AddNode("worktree", workflow.CreateWorktreeNode).
    AddNode("spec", workflow.GenerateSpecNode).
    AddNode("implement", workflow.ImplementNode).
    AddNode("review", workflow.ReviewNode).
    AddNode("pr", workflow.CreatePRNode).
    AddEdge("worktree", "spec").
    AddEdge("spec", "implement").
    AddEdge("implement", "review").
    AddEdge("review", "pr").
    AddEdge("pr", flowgraph.END).
    SetEntry("worktree")

// Inject services
services := &context.Services{
    Git:      gitCtx,
    LLM:      llmClient,
    Notifier: notifier,
}
ctx := services.InjectAll(ctx)

// Execute
state := workflow.NewState("ticket-to-pr")
result, _ := graph.Execute(ctx, state)
```

---

## Package Quick Reference

| Package | Key Types | Purpose |
|---------|-----------|---------|
| `git` | `Context`, `MockRunner`, `BranchNamer` | Git repository operations |
| `pr` | `Provider`, `Options`, `PullRequest` | GitHub/GitLab PR creation |
| `transcript` | `Manager`, `FileStore`, `Searcher` | Conversation recording |
| `artifact` | `Manager`, `ReviewResult`, `TestOutput` | Artifact storage |
| `workflow` | `State`, `NodeFunc`, workflow nodes | Workflow execution |
| `notify` | `Notifier`, `SlackNotifier` | Event notifications |
| `context` | `Services`, `WithGit`, `WithLLM` | Dependency injection |
| `prompt` | `Loader` | Template loading |
| `task` | `Type`, `Selector` | Model selection |

---

## Common Import Patterns

```go
// Git operations
import "github.com/randalmurphal/devflow/git"
gitCtx, _ := git.NewContext(path)

// Workflow with flowgraph
import "github.com/randalmurphal/devflow/workflow"
import "github.com/randalmurphal/flowgraph"
graph := flowgraph.NewGraph[workflow.State]()

// Context injection (alias to avoid conflict with stdlib)
import devcontext "github.com/randalmurphal/devflow/context"
ctx = devcontext.WithGit(ctx, gitCtx)

// Transcripts
import "github.com/randalmurphal/devflow/transcript"
store, _ := transcript.NewFileStore(transcript.StoreConfig{BaseDir: dir})

// Artifacts
import "github.com/randalmurphal/devflow/artifact"
mgr := artifact.NewManager(artifact.Config{BaseDir: dir})

// Notifications
import "github.com/randalmurphal/devflow/notify"
notifier := notify.NewSlack(webhookURL)
```

---

## Testing

```bash
go test -race ./...                    # Unit tests
go test -race -tags=integration ./...  # Integration tests
go build ./...                         # Verify compilation
```

---

## Depends On

- **flowgraph**: Graph orchestration + LLM abstraction (`github.com/randalmurphal/flowgraph`)
- **go-github**: GitHub API client
- **go-gitlab**: GitLab API client

---

## Related Documentation

| File | Purpose |
|------|---------|
| `git/CLAUDE.md` | Git package details |
| `workflow/CLAUDE.md` | Workflow nodes and state |
| `transcript/CLAUDE.md` | Transcript management |
| `artifact/CLAUDE.md` | Artifact storage |
| `docs/ARCHITECTURE.md` | Full architecture |
| `.spec/` | Specification documents |
