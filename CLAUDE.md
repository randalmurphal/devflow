# devflow

**Go library for AI-powered development workflows.** Git operations, LLM integration via flowgraph, transcript management, artifact storage, and notifications.

## Package Structure

```
devflow/
├── git/           # Git operations, worktrees, branches, commits
├── git/parallel/  # Parallel worktree orchestration for fork/join workflows
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
| `git/parallel` | `Manager`, `MergeResult`, `ConflictFile` | Parallel worktree orchestration |
| `pr` | `Provider`, `Options`, `PullRequest` | GitHub/GitLab PR creation |
| `transcript` | `Manager`, `FileStore`, `Searcher` | Conversation recording |
| `artifact` | `Manager`, `ReviewResult`, `TestOutput` | Artifact storage |
| `workflow` | `State`, `NodeFunc`, workflow nodes | Workflow execution |
| `notify` | `Notifier`, `SlackNotifier` | Event notifications |
| `context` | `Services`, `WithGit`, `WithLLM` | Dependency injection |
| `prompt` | `Loader` | Template loading |
| `task` | `Type`, `Selector` | Model selection |

---

## Parallel Worktree Orchestration

The `git/parallel` package manages multiple git worktrees for parallel branch execution:

```go
import "github.com/randalmurphal/devflow/git/parallel"

// Create manager from base repository
mgr, err := parallel.NewManager(parallel.Config{
    BaseRepoPath: "/path/to/repo",
    WorktreeDir:  "/tmp/worktrees",  // Where worktrees are created
})

// Create isolated worktree for each parallel branch
worktreePath, err := mgr.CreateBranchWorktree("branch-1", "feature-branch-1")
worktreePath2, err := mgr.CreateBranchWorktree("branch-2", "feature-branch-2")

// Get git context for a branch's worktree
gitCtx, err := mgr.GitContextForBranch("branch-1")

// Merge all branches back to base branch
results, err := mgr.MergeBranches(ctx, parallel.MergeConfig{
    CommitMessage: "Merge parallel branches",
    NoFastForward: true,
    SquashMerge:   false,
})

// Handle conflicts
for _, result := range results {
    if !result.Success && len(result.Conflicts) > 0 {
        for _, conflict := range result.Conflicts {
            // conflict.Path, conflict.Markers, conflict.OursContent, conflict.TheirsContent
        }
    }
}

// Resolve a conflict and continue merge
err = mgr.ResolveConflict(conflictPath, resolvedContent)
err = mgr.ContinueMerge("Merge with resolved conflicts")

// Cleanup all worktrees when done
err = mgr.CleanupAll()
```

**Key types:**
- `Manager` - Orchestrates multiple worktrees for parallel execution
- `MergeResult` - Outcome of merging a branch (success, conflicts, commit SHA)
- `ConflictFile` - Details about a merge conflict (path, markers, both versions)
- `MergeConfig` - Options for merge (commit message, no-ff, squash)

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
