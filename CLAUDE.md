# devflow

**Go library for AI-powered development workflows.** Git operations, LLM integration via flowgraph, transcript management, artifact storage, and notifications.

## Implementation Status

| Phase | Status | Description |
|-------|--------|-------------|
| 1 - Git Primitives | ✅ Complete | GitContext, worktrees, branches, PRs |
| 2 - Claude CLI | ✅ Complete | Uses flowgraph's llm.Client |
| 3 - Transcripts | ✅ Complete | Recording, search, view, export |
| 4 - Artifacts | ✅ Complete | Save, load, lifecycle, types |
| 5 - Workflow Nodes | ✅ Complete | 9 nodes, state, context injection |
| 6 - Polish | ✅ Complete | flowgraph integration, notifications, examples |

**Tests**: All passing with race detection (`go test -race ./...`)
**Coverage**: 83.1%

---

## Vision

Dev workflow primitives for AI agents. Builds on flowgraph to provide development-specific functionality. Part of a three-layer ecosystem:

| Layer | Purpose | Repo |
|-------|---------|------|
| flowgraph | Graph orchestration engine + LLM abstraction | Open source |
| **devflow** | Dev workflow primitives (this repo) | Open source |
| task-keeper | Commercial SaaS product | Commercial |

**Depends on**: flowgraph (for graph orchestration and LLM client)

---

## Core Components

| Component | Description | Key Type |
|-----------|-------------|----------|
| **GitContext** | Git operations (worktrees, commits, branches) | `GitContext` |
| **LLM Client** | Uses flowgraph's `llm.Client` interface | `llm.Client` |
| **TranscriptManager** | Recording and storing conversation transcripts | `TranscriptManager` |
| **ArtifactManager** | Storing run artifacts (files, outputs) | `ArtifactManager` |
| **Notifier** | Workflow event notifications (Slack, webhook) | `Notifier` |
| **TaskType** | Task-based model selection | `TaskType`, `NewTaskSelector` |
| **http.Client** | HTTP client for integrations | `http.Client` |
| **testutil** | Test utilities (git repos, contexts, fixtures) | Various helpers |

---

## Quick Reference

### Git Operations

```go
git := devflow.NewGitContext(repoPath)

// Create worktree for isolated work
worktree, err := git.CreateWorktree("feature/my-branch")
defer git.CleanupWorktree(worktree)

// Commit changes
err = git.Commit("Add feature", "file1.go", "file2.go")

// Create PR (GitHub)
pr, err := git.CreatePR(devflow.PROptions{
    Title: "Add feature",
    Body:  "Description",
    Base:  "main",
})
```

### LLM Client (via flowgraph)

```go
import "github.com/rmurphy/flowgraph/pkg/flowgraph/llm"

// Create LLM client
client := llm.NewClaudeCLI(
    llm.WithModel("claude-sonnet-4-20250514"),
    llm.WithWorkdir(repoPath),
)

// Run completion
result, err := client.Complete(ctx, llm.CompletionRequest{
    SystemPrompt: "You are an expert Go developer",
    Messages:     []llm.Message{{Role: llm.RoleUser, Content: "Implement the feature"}},
})

// Access results
fmt.Println(result.Content)
fmt.Println(result.Usage.InputTokens, result.Usage.OutputTokens)
```

### Context Injection

```go
// Inject services into context for workflow nodes
ctx := context.Background()
ctx = devflow.WithGitContext(ctx, git)
ctx = devflow.WithLLMClient(ctx, client)         // uses flowgraph llm.Client
ctx = devflow.WithTranscriptManager(ctx, transcripts)
ctx = devflow.WithArtifactManager(ctx, artifacts)
ctx = devflow.WithNotifier(ctx, notifier)        // notifications
ctx = devflow.WithCommandRunner(ctx, runner)     // for testing with MockRunner

// Or use DevServices for convenience
services := &devflow.DevServices{
    Git:      git,
    LLM:      client,  // flowgraph llm.Client
    Notifier: devflow.NewSlackNotifier(webhookURL),
    Runner:   devflow.NewMockRunner(),  // optional: for testing
}
ctx = services.InjectAll(ctx)
```

### Notifications

```go
// Create notifiers
slackNotifier := devflow.NewSlackNotifier(webhookURL,
    devflow.WithSlackChannel("#dev-alerts"),
    devflow.WithSlackUsername("devflow-bot"),
)

webhookNotifier := devflow.NewWebhookNotifier(url, headers)

// Combine multiple notifiers
multiNotifier := devflow.NewMultiNotifier(slackNotifier, webhookNotifier)

// Inject into context
ctx = devflow.WithNotifier(ctx, multiNotifier)

// Notify events
devflow.NotifyRunStarted(ctx, state)
devflow.NotifyRunCompleted(ctx, state)
devflow.NotifyRunFailed(ctx, state, err)

// Or use NotifyNode in workflows
result, err := devflow.NotifyNode(ctx, state)
```

### Transcripts

```go
mgr := devflow.NewTranscriptManager(devflow.TranscriptConfig{
    BaseDir: ".devflow/runs",
})

err := mgr.StartRun("run-123", devflow.RunMetadata{
    FlowID: "ticket-to-pr",
    Input:  map[string]any{"ticket": "TK-421"},
})

err = mgr.RecordTurn("run-123", devflow.Turn{
    Role:    "assistant",
    Content: "I'll implement this feature...",
    Tokens:  1500,
})

err = mgr.EndRun("run-123", devflow.RunStatusCompleted)
```

### Artifacts

```go
artifacts := devflow.NewArtifactManager(devflow.ArtifactConfig{
    BaseDir:       ".devflow/runs",
    CompressAbove: 1024, // Compress files > 1KB
})

// Save artifact
err := artifacts.SaveArtifact("run-123", "output.json", data)

// Load artifact
data, err := artifacts.LoadArtifact("run-123", "output.json")
```

---

## Project Structure

```
devflow/
├── git.go                  # GitContext - worktrees, branches, commits
├── branch.go               # BranchNamer - naming conventions
├── commit.go               # CommitMessage - conventional commits
├── pr.go                   # PRProvider interface, PRBuilder
├── github.go               # GitHub PR provider
├── gitlab.go               # GitLab MR provider
├── prompt.go               # PromptLoader - template loading
├── context.go              # Service context injection helpers
├── context_builder.go      # ContextBuilder, FileSelector, MIME detection
├── notification.go         # Notifier interface + implementations
├── transcript.go           # Transcript types
├── transcript_store.go     # FileTranscriptStore - storage
├── transcript_search.go    # TranscriptSearcher - grep-based search
├── transcript_view.go      # TranscriptViewer - display/export
├── artifact.go             # ArtifactManager - save/load
├── artifact_types.go       # ReviewResult, TestOutput, LintOutput
├── artifact_lifecycle.go   # LifecycleManager - cleanup/archive
├── state.go                # DevState, state components, Ticket
├── runner.go               # CommandRunner interface + implementations
├── errors.go               # Error types
├── task.go                 # TaskType + model selection
│
├── nodes.go                # NodeFunc type, NodeConfig, wrappers
├── node_worktree.go        # CreateWorktreeNode, CleanupNode
├── node_spec.go            # GenerateSpecNode
├── node_implement.go       # ImplementNode
├── node_review.go          # ReviewNode, FixFindingsNode, ReviewRouter
├── node_testing.go         # RunTestsNode
├── node_lint.go            # CheckLintNode
├── node_pr.go              # CreatePRNode
│
├── http/                   # HTTP client utilities for integrations
│   ├── client.go           # HTTPClient with retry
│   ├── pagination.go       # PageIterator for paginated APIs
│   └── errors.go           # API error types
│
├── testutil/               # Test utilities
│   ├── git.go              # SetupTestRepo, branches, commits
│   ├── context.go          # TestContext helpers
│   └── fixtures.go         # LoadFixture, TempFile
│
├── *_test.go               # Unit tests for each file
└── prompts/                # Default prompt templates
    ├── generate-spec.txt
    ├── implement.txt
    └── review-code.txt
```

---

## Integration with flowgraph

devflow now uses flowgraph's LLM abstraction layer. Nodes use `llm.Client` interface:

```go
import (
    "github.com/rmurphy/flowgraph"
    "github.com/rmurphy/flowgraph/pkg/flowgraph/llm"
    "github.com/rmurphy/devflow"
)

// Create LLM client from flowgraph
client := llm.NewClaudeCLI(
    llm.WithModel("claude-sonnet-4-20250514"),
    llm.WithWorkdir(repoPath),
    llm.WithDangerouslySkipPermissions(), // For automation
)

// Build workflow graph
graph := flowgraph.NewGraph[devflow.DevState]().
    AddNode("create-worktree", devflow.CreateWorktreeNode).
    AddNode("generate-spec", devflow.GenerateSpecNode).
    AddNode("implement", devflow.ImplementNode).
    AddNode("review", devflow.ReviewNode).
    AddNode("create-pr", devflow.CreatePRNode).
    AddNode("notify", devflow.NotifyNode).
    AddEdge("create-worktree", "generate-spec").
    AddEdge("generate-spec", "implement").
    AddEdge("implement", "review").
    AddEdge("review", "create-pr").
    AddEdge("create-pr", "notify").
    AddEdge("notify", flowgraph.END).
    SetEntry("create-worktree")

// Inject services
ctx := context.Background()
ctx = devflow.WithGitContext(ctx, git)
ctx = devflow.WithLLMClient(ctx, client)
ctx = devflow.WithNotifier(ctx, slackNotifier)

// Execute
state := devflow.NewDevState("ticket-to-pr")
result, err := graph.Execute(ctx, state)
```

---

## Directory Conventions

```
.devflow/
├── runs/
│   └── 2025-01-15-ticket-to-pr-TK421/
│       ├── transcript.json      # Conversation log
│       ├── metadata.json        # Run metadata
│       ├── artifacts/           # Generated files
│       │   ├── spec.md
│       │   └── output.json
│       └── state-checkpoints/   # flowgraph checkpoints
│           ├── generate-spec.json
│           └── implement.json
├── prompts/                     # Prompt templates
│   ├── spec-generation.txt
│   └── implementation.txt
└── config.json                  # devflow configuration
```

---

## Error Handling

| Error | When | Handling |
|-------|------|----------|
| `ErrWorktreeExists` | Worktree already exists | Clean up or use existing |
| `ErrGitDirty` | Uncommitted changes | Stash or abort |
| `ErrTimeout` | Operation timed out | Retry with longer timeout |
| `ErrTranscriptNotFound` | Run ID doesn't exist | Check run ID |
| `ErrContextTooLarge` | Context exceeds limits | Reduce file count or size |

---

## Testing

```bash
go test -race ./...                    # Unit tests
go test -race -tags=integration ./...  # With real git/Claude
go test -race -coverprofile=coverage.out ./...  # With coverage
```

---

## CI/CD

GitHub Actions workflow (`.github/workflows/ci.yml`):
- **Test**: Runs on Go 1.23 and 1.24 with race detection
- **Lint**: golangci-lint with project config (`.golangci.yml`)
- **Build**: Verifies compilation including examples
- **Coverage**: Uploads to Codecov

Dependabot configured for Go modules and GitHub Actions updates.

---

## Dependencies

```bash
# Core (now uses flowgraph's llm package)
go get github.com/rmurphy/flowgraph

# GitHub/GitLab
go get github.com/google/go-github/v57
go get github.com/xanzy/go-gitlab
```

---

## References

| Doc | Purpose |
|-----|---------|
| `docs/OVERVIEW.md` | Detailed vision and concepts |
| `docs/ARCHITECTURE.md` | Component design, data flow |
| `docs/API_REFERENCE.md` | Complete public API |
| `README.md` | User-facing documentation |
| `CHANGELOG.md` | Version history |

---

## Related Repos

- **flowgraph**: Foundation layer (graph orchestration + LLM abstraction)
- **task-keeper**: Product layer (commercial SaaS)

---

## Specification Documents

Complete specifications are in `.spec/`. **Read these before implementing.**

```
.spec/
├── PLANNING.md              # Overall roadmap and design philosophy
├── DECISIONS.md             # ADR index with decision summaries
├── SESSION_PROMPT.md        # Current session handoff/instructions
├── decisions/               # 20 Architecture Decision Records
│   ├── 001-020              # Git, Claude CLI, Transcripts, Artifacts, Integration
├── phases/                  # 6 implementation phases (6 weeks)
│   ├── phase-1-git-primitives.md
│   ├── phase-2-claude-cli.md
│   ├── phase-3-transcripts.md
│   ├── phase-4-artifacts.md
│   ├── phase-5-workflow-nodes.md
│   └── phase-6-polish.md
├── features/                # 8 feature specifications
│   ├── worktree-management.md
│   ├── git-operations.md
│   ├── claude-cli.md
│   ├── prompt-loading.md
│   ├── transcript-recording.md
│   ├── transcript-replay.md
│   ├── artifact-storage.md
│   ├── dev-workflow-nodes.md
│   └── nodes/               # 7 node specifications
│       ├── generate-spec.md
│       ├── implement.md
│       ├── review-code.md
│       ├── fix-findings.md
│       ├── create-pr.md
│       ├── run-tests.md
│       └── check-lint.md
├── knowledge/
│   └── INTEGRATION_PATTERNS.md  # flowgraph integration patterns
└── tracking/
    ├── PROGRESS.md          # Implementation progress
    └── CHANGELOG.md         # Change history
```

### Implementation Order

| Phase | Focus | Status |
|-------|-------|--------|
| 1 | Git Primitives | ✅ Complete |
| 2 | Claude CLI | ✅ Complete (migrated to flowgraph) |
| 3 | Transcripts | ✅ Complete |
| 4 | Artifacts | ✅ Complete |
| 5 | Workflow Nodes | ✅ Complete |
| 6 | Polish | ✅ Complete |

### Key Design Decisions

- **Shell out to git** (ADR-001): Don't use go-git for worktrees, shell out to git binary
- **Use flowgraph llm.Client** (ADR-006): Use flowgraph's LLM abstraction, not a devflow-specific wrapper
- **JSON files for storage** (ADR-012): Simple file-based storage, no database
- **grep for search** (ADR-014): Use grep for transcript search, not a search engine
- **Context injection** (ADR-018): Pass services via context.Context, not state

### Phase 6 Status

All Phase 6 tasks completed:
- ✅ flowgraph dependency added
- ✅ LLM context injection (`WithLLMClient`, `LLMFromContext`)
- ✅ All nodes updated to use `llm.Client`
- ✅ Notification system implemented (Slack, Webhook, Log, Multi)
- ✅ Tests updated and passing (83.1% coverage)
- ✅ Example applications (`examples/basic/main.go`)
- ✅ Documentation polish
- ✅ Code cleanup (removed deprecated ClaudeCLI, split large files)
- ✅ CI/CD with GitHub Actions (test, lint, build)
- ✅ Dependabot configuration
