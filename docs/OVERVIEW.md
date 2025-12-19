# devflow Overview

## Purpose

devflow is a Go library providing development workflow primitives for AI-powered automation. It bridges the gap between generic graph orchestration (flowgraph) and specific development tasks:

- **Git operations** - Worktrees, branches, commits, PRs
- **LLM integration** - Uses flowgraph's `llm.Client` interface
- **Transcript management** - Recording and storing AI conversations
- **Artifact storage** - Managing files generated during workflows

---

## Why devflow?

### The Problem

AI-powered development workflows need:
1. **Isolated workspaces** - Work on features without affecting main branch
2. **LLM integration** - Invoke Claude/GPT with project context
3. **Audit trail** - Record what the AI did and why
4. **Artifact management** - Store generated specs, code, outputs

These are common patterns that shouldn't be reimplemented for each project.

### The Solution

devflow provides reusable primitives:
- **GitContext** - Worktree management, commit, push, PR creation
- **LLM Client** - Uses flowgraph's `llm.Client` for Claude integration
- **TranscriptManager** - Structured storage of AI conversations
- **ArtifactManager** - Run artifact storage with compression
- **Notifier** - Workflow event notifications (Slack, webhook)

---

## Core Components

### GitContext

Manages git operations for development workflows.

**Key capabilities**:
- **Worktrees** - Create isolated working directories per branch
- **Commits** - Stage and commit specific files
- **PRs** - Create pull requests (GitHub, GitLab)
- **Branches** - Create, checkout, push

**Why worktrees?** Multiple agents can work on different branches simultaneously without checkout conflicts.

```go
git := devflow.NewGitContext("/path/to/repo")

// Create worktree
worktree, err := git.CreateWorktree("feature/new-api")
// worktree = "/path/to/repo/.worktrees/feature-new-api"

// Work in worktree...

// Commit
err = git.Commit("Add API endpoint", "api.go", "api_test.go")

// Push
err = git.Push("origin", "feature/new-api")

// Create PR
pr, err := git.CreatePR(devflow.PROptions{
    Title: "Add new API endpoint",
    Body:  "## Summary\n...",
    Base:  "main",
})

// Cleanup
err = git.CleanupWorktree(worktree)
```

### LLM Client (via flowgraph)

devflow uses flowgraph's `llm.Client` interface for LLM integration.

**Key capabilities**:
- **Model selection** - Use different Claude models
- **Working directory** - Run in specific directory
- **Context injection** - Via devflow context helpers
- **Token tracking** - Monitor usage
- **Session management** - Multi-turn conversations

```go
import "github.com/rmurphy/flowgraph/pkg/flowgraph/llm"

// Create LLM client
client := llm.NewClaudeCLI(
    llm.WithModel("claude-sonnet-4-20250514"),
    llm.WithWorkdir(worktree),
)

// Run completion
result, err := client.Complete(ctx, llm.CompletionRequest{
    SystemPrompt: systemPrompt,
    Messages:     []llm.Message{{Role: llm.RoleUser, Content: "Implement the login endpoint"}},
})

// Result
result.Content           // Final response text
result.Usage.InputTokens // Input tokens consumed
result.Usage.OutputTokens // Output tokens generated

// Inject into context for workflow nodes
ctx = devflow.WithLLMClient(ctx, client)
```

### TranscriptManager

Records and stores AI conversation transcripts.

**Key capabilities**:
- **Run lifecycle** - Start, record turns, end
- **Metadata** - Store run context (flow ID, inputs)
- **Compression** - Gzip for large transcripts
- **Search** - Find runs by criteria
- **Export** - Export to various formats

```go
mgr := devflow.NewTranscriptManager(devflow.TranscriptConfig{
    BaseDir:    ".devflow/runs",
    Compress:   true,
    MaxSizeMB:  100,
})

// Start run
err := mgr.StartRun("run-2025-01-15-001", devflow.RunMetadata{
    FlowID:    "ticket-to-pr",
    Input:     map[string]any{"ticket": "TK-421"},
    StartedAt: time.Now(),
})

// Record turns
err = mgr.RecordTurn("run-2025-01-15-001", devflow.Turn{
    Role:      "user",
    Content:   "Implement feature X",
    Tokens:    50,
    Timestamp: time.Now(),
})

err = mgr.RecordTurn("run-2025-01-15-001", devflow.Turn{
    Role:      "assistant",
    Content:   "I'll implement feature X by...",
    Tokens:    1500,
    Timestamp: time.Now(),
})

// End run
err = mgr.EndRun("run-2025-01-15-001", devflow.RunStatusCompleted)

// Later: load transcript
transcript, err := mgr.Load("run-2025-01-15-001")
```

### ArtifactManager

Stores files generated during workflow runs.

**Key capabilities**:
- **Organized storage** - By run ID
- **Compression** - Auto-compress large files
- **Retention** - Cleanup old artifacts
- **Listing** - Find artifacts for a run

```go
artifacts := devflow.NewArtifactManager(devflow.ArtifactConfig{
    BaseDir:       ".devflow/runs",
    CompressAbove: 1024,     // Compress > 1KB
    RetentionDays: 30,       // Keep for 30 days
})

// Save artifact
data := []byte(`{"spec": "..."}`)
err := artifacts.SaveArtifact("run-123", "spec.json", data)

// Load artifact
loaded, err := artifacts.LoadArtifact("run-123", "spec.json")

// List artifacts
infos, err := artifacts.ListArtifacts("run-123")
for _, info := range infos {
    fmt.Printf("%s: %d bytes\n", info.Name, info.Size)
}

// Cleanup old artifacts
deleted, err := artifacts.Cleanup()
```

### Notifier

Sends workflow event notifications.

**Key capabilities**:
- **Slack** - Webhook-based Slack notifications
- **Webhooks** - Generic HTTP webhooks
- **Multiple targets** - Combine multiple notifiers
- **Context injection** - Use in workflow nodes

```go
// Create notifiers
slack := devflow.NewSlackNotifier(webhookURL,
    devflow.WithSlackChannel("#dev-alerts"),
    devflow.WithSlackUsername("devflow-bot"),
)

webhook := devflow.NewWebhookNotifier(url, headers)

// Combine multiple notifiers
multi := devflow.NewMultiNotifier(slack, webhook)

// Inject into context
ctx = devflow.WithNotifier(ctx, multi)

// Send notifications
devflow.NotifyRunStarted(ctx, state)
devflow.NotifyRunCompleted(ctx, state)
devflow.NotifyRunFailed(ctx, state, err)
```

---

## Workflow Integration

devflow provides pre-built nodes for flowgraph graphs:

```go
type TicketState struct {
    TicketID       string
    Ticket         *jira.Issue
    Spec           *Spec
    Worktree       string
    Implementation *Implementation
    Review         *ReviewResult
    PR             *PullRequest
}

graph := flowgraph.NewGraph[TicketState]().
    // Setup
    AddNode("create-worktree", devflow.CreateWorktreeNode).

    // Spec generation
    AddNode("parse-ticket", parseTicketNode).
    AddNode("generate-spec", devflow.GenerateSpecNode).

    // Implementation
    AddNode("implement", devflow.ImplementNode).
    AddNode("review", devflow.ReviewNode).
    AddNode("fix-findings", devflow.FixFindingsNode).

    // Completion
    AddNode("create-pr", devflow.CreatePRNode).
    AddNode("cleanup", devflow.CleanupNode).

    // Edges
    AddEdge("create-worktree", "parse-ticket").
    AddEdge("parse-ticket", "generate-spec").
    AddEdge("generate-spec", "implement").
    AddEdge("implement", "review").
    AddConditionalEdge("review", func(s TicketState) string {
        if s.Review.Approved {
            return "create-pr"
        }
        return "fix-findings"
    }).
    AddEdge("fix-findings", "review").
    AddEdge("create-pr", "cleanup").
    AddEdge("cleanup", flowgraph.END).
    SetEntry("create-worktree")
```

---

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DEVFLOW_BASE_DIR` | Base directory for runs | `.devflow` |
| `GITHUB_TOKEN` | GitHub API token | - |
| `GITLAB_TOKEN` | GitLab API token | - |
| `SLACK_WEBHOOK_URL` | Slack notification webhook | - |

### Directory Structure

```
.devflow/
├── runs/                    # Transcript and artifact storage
│   └── run-2025-01-15-001/
│       ├── transcript.json
│       ├── metadata.json
│       └── artifacts/
├── prompts/                 # Custom prompt templates
│   └── my-prompt.txt
└── config.json              # Optional configuration
```

---

## Use Cases

### 1. Ticket-to-PR Automation

```
Ticket → Parse → Spec → Implement → Review → PR
                            ↓           ↑
                          Fix ─────────┘
```

### 2. Code Review Automation

```
PR → Fetch Diff → Review → Generate Comments → Post
```

### 3. Documentation Generation

```
Code → Analyze → Generate Docs → Review → Commit
```

### 4. Refactoring Assistance

```
Code → Identify Issues → Plan → Refactor → Test → PR
```

---

## Relationship to Other Layers

### flowgraph (Foundation)

devflow uses flowgraph for:
- Graph-based workflow definition
- State management and checkpointing
- Execution engine
- LLM client interface (`llm.Client`)

devflow adds:
- Dev-specific operations (git, PRs)
- Transcript/artifact storage
- Pre-built workflow nodes
- Notification system

### task-keeper (Product)

task-keeper uses devflow for:
- Executing workflows
- Git operations
- Claude integration

task-keeper adds:
- Task management
- Visual flow builder
- Prompt studio
- User management
- Web/TUI/CLI interfaces
