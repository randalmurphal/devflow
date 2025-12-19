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

## Claude CLI

### NewClaudeCLI

```go
func NewClaudeCLI(config ClaudeConfig) *ClaudeCLI
```

Creates a Claude CLI wrapper.

```go
type ClaudeConfig struct {
    BinaryPath string        // Path to claude binary (default: "claude")
    Model      string        // Model to use
    Timeout    time.Duration // Default timeout
    MaxTurns   int           // Max conversation turns
}
```

---

### Run

```go
func (c *ClaudeCLI) Run(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error)
```

Runs Claude with the given prompt.

| Parameter | Type | Description |
|-----------|------|-------------|
| `ctx` | `context.Context` | Context for cancellation |
| `prompt` | `string` | User prompt |
| `opts` | `...RunOption` | Configuration options |

**Options**:
```go
WithSystemPrompt(prompt string)   // Set system prompt
WithContext(files ...string)      // Add context files
WithWorkDir(dir string)           // Working directory
WithMaxTurns(n int)               // Max turns
WithTimeout(d time.Duration)      // Timeout
```

**Returns**:
```go
type RunResult struct {
    Output     string        // Claude's response
    TokensIn   int           // Input tokens
    TokensOut  int           // Output tokens
    Transcript *Transcript   // Full conversation
    Files      []FileChange  // Files created/modified
    Duration   time.Duration // Execution time
    ExitCode   int           // Process exit code
}
```

---

### RunWithFiles

```go
func (c *ClaudeCLI) RunWithFiles(
    ctx context.Context,
    prompt string,
    files []string,
    opts ...RunOption,
) (*RunResult, error)
```

Convenience method to run with context files.

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

    // Claude
    ErrClaudeTimeout = errors.New("claude CLI timed out")
    ErrClaudeFailed  = errors.New("claude CLI failed")

    // Transcript
    ErrTranscriptNotFound = errors.New("transcript not found")
    ErrRunNotStarted      = errors.New("run not started")
    ErrRunAlreadyEnded    = errors.New("run already ended")

    // Artifact
    ErrArtifactNotFound = errors.New("artifact not found")
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

    "github.com/yourorg/devflow"
)

func main() {
    // Setup
    git, _ := devflow.NewGitContext(".")
    claude := devflow.NewClaudeCLI(devflow.ClaudeConfig{
        Timeout: 5 * time.Minute,
    })
    transcripts := devflow.NewTranscriptManager(devflow.TranscriptConfig{
        BaseDir:  ".devflow",
        Compress: true,
    })
    artifacts := devflow.NewArtifactManager(devflow.ArtifactConfig{
        BaseDir: ".devflow",
    })

    ctx := context.Background()
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

    // Run Claude
    result, err := claude.Run(ctx, "Implement a hello world endpoint",
        devflow.WithWorkDir(worktree),
        devflow.WithSystemPrompt("You are an expert Go developer"),
    )
    if err != nil {
        transcripts.EndRun(runID, devflow.RunStatusFailed)
        log.Fatal(err)
    }

    // Record transcript
    transcripts.RecordTurn(runID, devflow.Turn{
        Role:    "assistant",
        Content: result.Output,
        Tokens:  result.TokensOut,
    })

    // Save artifact
    artifacts.SaveArtifact(runID, "output.txt", []byte(result.Output))

    // Commit and PR
    git.Commit("Add hello world endpoint", "main.go")
    git.Push("origin", "feature/new-feature")
    pr, _ := git.CreatePR(devflow.PROptions{
        Title: "Add hello world endpoint",
        Body:  result.Output,
        Base:  "main",
    })

    // End transcript
    transcripts.EndRun(runID, devflow.RunStatusCompleted)

    log.Printf("Created PR: %s", pr.URL)
}
```
