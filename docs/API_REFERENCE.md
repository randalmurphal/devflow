# devflow API Reference

## Git Operations

### NewGitContext

```go
func NewGitContext(repoPath string, opts ...GitOption) (*GitContext, error)
```

Creates a new git context for the repository.

| Parameter | Type | Description |
|-----------|------|-------------|
| `repoPath` | `string` | Path to git repository |
| `opts` | `...GitOption` | Configuration options |

**Options**:
```go
WithWorktreeDir(dir string)      // Custom worktree directory
WithGitHubClient(client *github.Client)
WithGitLabClient(client *gitlab.Client)
```

---

### CreateWorktree

```go
func (g *GitContext) CreateWorktree(branch string) (string, error)
```

Creates a git worktree for isolated work.

| Parameter | Type | Description |
|-----------|------|-------------|
| `branch` | `string` | Branch name (created if doesn't exist) |

**Returns**: `(string, error)` - Worktree path and any error

**Errors**:
- `ErrWorktreeExists` - Worktree already exists

---

### CleanupWorktree

```go
func (g *GitContext) CleanupWorktree(path string) error
```

Removes a worktree and optionally its branch.

---

### Commit

```go
func (g *GitContext) Commit(message string, files ...string) error
```

Stages and commits specified files.

| Parameter | Type | Description |
|-----------|------|-------------|
| `message` | `string` | Commit message |
| `files` | `...string` | Files to commit |

---

### Push

```go
func (g *GitContext) Push(remote, branch string) error
```

Pushes branch to remote.

---

### CreatePR

```go
func (g *GitContext) CreatePR(opts PROptions) (*PullRequest, error)
```

Creates a pull request.

```go
type PROptions struct {
    Title  string   // PR title
    Body   string   // PR body (markdown)
    Base   string   // Base branch
    Labels []string // Labels to apply
    Draft  bool     // Create as draft
}

type PullRequest struct {
    ID     int    // PR number
    URL    string // Web URL
    Branch string // Head branch
    State  string // open, closed, merged
}
```

---

### CurrentBranch

```go
func (g *GitContext) CurrentBranch() (string, error)
```

Returns the current branch name.

---

### Diff

```go
func (g *GitContext) Diff(base, head string) (string, error)
```

Returns diff between two refs.

---

## LLM Client (via flowgraph)

devflow uses flowgraph's `llm.Client` interface. See [flowgraph documentation](https://github.com/rmurphy/flowgraph) for full LLM client API.

### Context Injection

```go
// Inject LLM client into context
func WithLLMClient(ctx context.Context, client llm.Client) context.Context

// Retrieve LLM client from context
func LLMFromContext(ctx context.Context) llm.Client
```

### Quick Reference

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
result.Content            // Response text
result.Usage.InputTokens  // Input tokens
result.Usage.OutputTokens // Output tokens

// Inject into context for workflow nodes
ctx = devflow.WithLLMClient(ctx, client)
```

---

## Transcript Management

### NewTranscriptManager

```go
func NewTranscriptManager(config TranscriptConfig) *TranscriptManager
```

Creates a transcript manager.

```go
type TranscriptConfig struct {
    BaseDir           string // Base directory for runs
    Compress          bool   // Enable compression
    CompressThreshold int64  // Compress above this size
}
```

---

### StartRun

```go
func (m *TranscriptManager) StartRun(runID string, metadata RunMetadata) error
```

Starts recording a new run.

```go
type RunMetadata struct {
    FlowID    string         // Flow identifier
    Input     map[string]any // Flow inputs
    StartedAt time.Time      // Start time
}
```

---

### RecordTurn

```go
func (m *TranscriptManager) RecordTurn(runID string, turn Turn) error
```

Records a conversation turn.

```go
type Turn struct {
    Role      string     // "user", "assistant", "system"
    Content   string     // Message content
    Tokens    int        // Token count
    Timestamp time.Time  // When the turn occurred
    ToolCalls []ToolCall // Tool calls (if any)
}

type ToolCall struct {
    Name   string         // Tool name
    Input  map[string]any // Tool input
    Output string         // Tool output
}
```

---

### EndRun

```go
func (m *TranscriptManager) EndRun(runID string, status RunStatus) error
```

Ends a run and saves the transcript.

```go
type RunStatus string

const (
    RunStatusRunning   RunStatus = "running"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
    RunStatusCanceled  RunStatus = "canceled"
)
```

---

### Load

```go
func (m *TranscriptManager) Load(runID string) (*Transcript, error)
```

Loads a saved transcript.

---

### List

```go
func (m *TranscriptManager) List(filter TranscriptFilter) ([]TranscriptSummary, error)
```

Lists transcripts matching filter.

```go
type TranscriptFilter struct {
    FlowID    string
    Status    RunStatus
    StartedAfter time.Time
    Limit     int
}

type TranscriptSummary struct {
    RunID     string
    FlowID    string
    Status    RunStatus
    StartedAt time.Time
    EndedAt   time.Time
    TurnCount int
    TotalTokens int
}
```

---

### Search

```go
func (m *TranscriptManager) Search(query string) ([]TranscriptSummary, error)
```

Searches transcript content.

---

## Artifact Management

### NewArtifactManager

```go
func NewArtifactManager(config ArtifactConfig) *ArtifactManager
```

Creates an artifact manager.

```go
type ArtifactConfig struct {
    BaseDir       string // Base directory
    CompressAbove int64  // Compress files above this size
    RetentionDays int    // Delete artifacts older than this
}
```

---

### SaveArtifact

```go
func (m *ArtifactManager) SaveArtifact(runID, name string, data []byte) error
```

Saves an artifact.

---

### LoadArtifact

```go
func (m *ArtifactManager) LoadArtifact(runID, name string) ([]byte, error)
```

Loads an artifact.

---

### ListArtifacts

```go
func (m *ArtifactManager) ListArtifacts(runID string) ([]ArtifactInfo, error)
```

Lists artifacts for a run.

```go
type ArtifactInfo struct {
    Name       string
    Size       int64
    Compressed bool
    CreatedAt  time.Time
}
```

---

### Cleanup

```go
func (m *ArtifactManager) Cleanup() (int, error)
```

Removes artifacts older than retention period. Returns count deleted.

---

## Notifications

### NewSlackNotifier

```go
func NewSlackNotifier(webhookURL string, opts ...SlackOption) *SlackNotifier
```

Creates a Slack notifier.

**Options**:
```go
WithSlackChannel(channel string)   // Override default channel
WithSlackUsername(username string) // Bot username
WithSlackIconEmoji(emoji string)   // Bot icon emoji
```

---

### NewWebhookNotifier

```go
func NewWebhookNotifier(url string, headers map[string]string) *WebhookNotifier
```

Creates a generic webhook notifier.

---

### NewMultiNotifier

```go
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier
```

Combines multiple notifiers.

---

### Context Injection

```go
// Inject notifier into context
func WithNotifier(ctx context.Context, n Notifier) context.Context

// Retrieve notifier from context
func NotifierFromContext(ctx context.Context) Notifier

// Notification helpers
func NotifyRunStarted(ctx context.Context, state DevState)
func NotifyRunCompleted(ctx context.Context, state DevState)
func NotifyRunFailed(ctx context.Context, state DevState, err error)
```

---

## Pre-built Workflow Nodes

For use with flowgraph:

```go
// CreateWorktreeNode creates a git worktree
func CreateWorktreeNode(ctx flowgraph.Context, state DevState) (DevState, error)

// GenerateSpecNode generates a spec using Claude
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error)

// ImplementNode implements code using Claude
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error)

// ReviewNode reviews implementation using Claude
func ReviewNode(ctx flowgraph.Context, state DevState) (DevState, error)

// FixFindingsNode fixes review findings using Claude
func FixFindingsNode(ctx flowgraph.Context, state DevState) (DevState, error)

// CreatePRNode creates a pull request
func CreatePRNode(ctx flowgraph.Context, state DevState) (DevState, error)

// CleanupNode cleans up worktree
func CleanupNode(ctx flowgraph.Context, state DevState) (DevState, error)

// RunTestsNode runs tests
func RunTestsNode(ctx flowgraph.Context, state DevState) (DevState, error)

// CheckLintNode runs linters
func CheckLintNode(ctx flowgraph.Context, state DevState) (DevState, error)

// NotifyNode sends notifications
func NotifyNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

---

## Error Types

```go
var (
    // Git
    ErrWorktreeExists = errors.New("worktree already exists")
    ErrGitDirty       = errors.New("uncommitted changes")
    ErrNoPRProvider   = errors.New("no PR provider configured")
    ErrBranchExists   = errors.New("branch already exists")

    // Transcript
    ErrTranscriptNotFound = errors.New("transcript not found")
    ErrRunNotStarted      = errors.New("run not started")
    ErrRunAlreadyEnded    = errors.New("run already ended")

    // Artifact
    ErrArtifactNotFound = errors.New("artifact not found")

    // Context
    ErrContextTooLarge = errors.New("context exceeds size limits")

    // General
    ErrTimeout = errors.New("operation timed out")
)
```

---

## Complete Example

```go
package main

import (
    "context"
    "log"
    "time"

    "github.com/randalmurphal/devflow"
    "github.com/rmurphy/flowgraph/pkg/flowgraph/llm"
)

func main() {
    ctx := context.Background()

    // Setup services
    git, _ := devflow.NewGitContext(".")
    client := llm.NewClaudeCLI(
        llm.WithModel("claude-sonnet-4-20250514"),
        llm.WithWorkdir("."),
    )
    transcripts, _ := devflow.NewTranscriptManager(devflow.TranscriptConfig{
        BaseDir: ".devflow",
    })
    artifacts := devflow.NewArtifactManager(devflow.ArtifactConfig{
        BaseDir: ".devflow",
    })
    notifier := devflow.NewSlackNotifier("https://hooks.slack.com/...")

    // Inject into context
    ctx = devflow.WithGitContext(ctx, git)
    ctx = devflow.WithLLMClient(ctx, client)
    ctx = devflow.WithTranscriptManager(ctx, transcripts)
    ctx = devflow.WithArtifactManager(ctx, artifacts)
    ctx = devflow.WithNotifier(ctx, notifier)

    runID := "run-" + time.Now().Format("2006-01-02-150405")

    // Start transcript
    transcripts.StartRun(runID, devflow.RunMetadata{
        FlowID: "manual-run",
    })

    // Create worktree
    worktree, err := git.CreateWorktree("feature/new-feature")
    if err != nil {
        log.Fatal(err)
    }
    defer git.CleanupWorktree(worktree)

    // Run LLM completion
    result, err := client.Complete(ctx, llm.CompletionRequest{
        SystemPrompt: "You are an expert Go developer",
        Messages: []llm.Message{
            {Role: llm.RoleUser, Content: "Implement a hello world endpoint"},
        },
    })
    if err != nil {
        transcripts.EndRun(runID, devflow.RunStatusFailed)
        log.Fatal(err)
    }

    // Record transcript
    transcripts.RecordTurn(runID, devflow.Turn{
        Role:      "assistant",
        Content:   result.Content,
        TokensOut: result.Usage.OutputTokens,
    })

    // Save artifact
    artifacts.SaveArtifact(runID, "output.txt", []byte(result.Content))

    // Commit and PR
    git.Commit("Add hello world endpoint", "main.go")
    git.Push("origin", "feature/new-feature")
    pr, _ := git.CreatePR(devflow.PROptions{
        Title: "Add hello world endpoint",
        Body:  result.Content,
        Base:  "main",
    })

    // End transcript
    transcripts.EndRun(runID, devflow.RunStatusCompleted)

    log.Printf("Created PR: %s", pr.URL)
}
```
