# devflow Architecture

## System Design

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                         User Code                                │
│  workflow := devflow.NewWorkflow(...) / flowgraph integration   │
└────────────────────────────────┬────────────────────────────────┘
                                 │
        ┌────────────────────────┼────────────────────────┐
        │                        │                        │
        ▼                        ▼                        ▼
┌───────────────┐    ┌───────────────────┐    ┌──────────────────┐
│   GitContext  │    │    ClaudeCLI      │    │ TranscriptManager│
│               │    │                   │    │                  │
│ • Worktrees   │    │ • Context inject  │    │ • Run lifecycle  │
│ • Commits     │    │ • Transcript      │    │ • Turn recording │
│ • PRs         │    │ • Token tracking  │    │ • Compression    │
└───────┬───────┘    └─────────┬─────────┘    └────────┬─────────┘
        │                      │                       │
        │                      │                       │
        ▼                      ▼                       ▼
┌───────────────┐    ┌───────────────────┐    ┌──────────────────┐
│  go-git / git │    │    claude CLI     │    │  File System     │
│  GitHub API   │    │                   │    │                  │
│  GitLab API   │    │                   │    │                  │
└───────────────┘    └───────────────────┘    └──────────────────┘
```

---

## Type Definitions

### Git Types

```go
// GitContext manages git operations
type GitContext struct {
    repoPath    string
    worktreeDir string
    github      *github.Client
    gitlab      *gitlab.Client
}

// WorktreeInfo represents an active worktree
type WorktreeInfo struct {
    Path   string
    Branch string
    Commit string
}

// PROptions configures pull request creation
type PROptions struct {
    Title  string
    Body   string
    Base   string
    Labels []string
    Draft  bool
}

// PullRequest represents a created PR
type PullRequest struct {
    ID     int
    URL    string
    Branch string
    State  string
}
```

### Claude Types

```go
// ClaudeCLI wraps the claude CLI
type ClaudeCLI struct {
    binaryPath string
    model      string
    timeout    time.Duration
    maxTurns   int
}

// ClaudeConfig configures the Claude CLI wrapper
type ClaudeConfig struct {
    BinaryPath string
    Model      string
    Timeout    time.Duration
    MaxTurns   int
}

// RunOption configures a Claude run
type RunOption func(*runConfig)

// RunResult contains the outcome of a Claude run
type RunResult struct {
    Output     string
    TokensIn   int
    TokensOut  int
    Transcript *Transcript
    Files      []FileChange
    Duration   time.Duration
    ExitCode   int
}

// FileChange represents a file created/modified
type FileChange struct {
    Path      string
    Operation string // "create", "modify", "delete"
    Content   []byte
}
```

### Transcript Types

```go
// TranscriptManager handles transcript storage
type TranscriptManager struct {
    baseDir  string
    compress bool
}

// RunMetadata describes a run
type RunMetadata struct {
    FlowID     string
    Input      map[string]any
    StartedAt  time.Time
    EndedAt    time.Time
    Status     RunStatus
}

// Turn represents a conversation turn
type Turn struct {
    Role      string    // "user", "assistant", "system"
    Content   string
    Tokens    int
    Timestamp time.Time
    ToolCalls []ToolCall
}

// Transcript is the full run record
type Transcript struct {
    RunID    string
    Metadata RunMetadata
    Turns    []Turn
}

// RunStatus indicates run outcome
type RunStatus string

const (
    RunStatusRunning   RunStatus = "running"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
    RunStatusCanceled  RunStatus = "canceled"
)
```

### Artifact Types

```go
// ArtifactManager handles run artifacts
type ArtifactManager struct {
    baseDir       string
    compressAbove int64
    retentionDays int
}

// ArtifactInfo describes a stored artifact
type ArtifactInfo struct {
    Name       string
    Size       int64
    Compressed bool
    CreatedAt  time.Time
}
```

---

## Git Operations

### Worktree Management

```go
func (g *GitContext) CreateWorktree(branch string) (string, error) {
    // 1. Sanitize branch name for filesystem
    safeName := sanitizeBranchName(branch)

    // 2. Determine worktree path
    worktreePath := filepath.Join(g.worktreeDir, safeName)

    // 3. Check if already exists
    if exists(worktreePath) {
        return "", ErrWorktreeExists
    }

    // 4. Create worktree via git command
    cmd := exec.Command("git", "worktree", "add", "-b", branch, worktreePath)
    cmd.Dir = g.repoPath
    if err := cmd.Run(); err != nil {
        // Try without -b if branch exists
        cmd = exec.Command("git", "worktree", "add", worktreePath, branch)
        cmd.Dir = g.repoPath
        if err := cmd.Run(); err != nil {
            return "", fmt.Errorf("create worktree: %w", err)
        }
    }

    return worktreePath, nil
}

func (g *GitContext) CleanupWorktree(path string) error {
    // 1. Remove worktree via git
    cmd := exec.Command("git", "worktree", "remove", path)
    cmd.Dir = g.repoPath
    if err := cmd.Run(); err != nil {
        // Force remove if needed
        cmd = exec.Command("git", "worktree", "remove", "--force", path)
        cmd.Dir = g.repoPath
        return cmd.Run()
    }
    return nil
}
```

### PR Creation

```go
func (g *GitContext) CreatePR(opts PROptions) (*PullRequest, error) {
    if g.github != nil {
        return g.createGitHubPR(opts)
    }
    if g.gitlab != nil {
        return g.createGitLabMR(opts)
    }
    return nil, ErrNoPRProvider
}

func (g *GitContext) createGitHubPR(opts PROptions) (*PullRequest, error) {
    // Get current branch
    branch, err := g.CurrentBranch()
    if err != nil {
        return nil, err
    }

    // Create PR via API
    pr, _, err := g.github.PullRequests.Create(ctx, owner, repo, &github.NewPullRequest{
        Title: &opts.Title,
        Body:  &opts.Body,
        Head:  &branch,
        Base:  &opts.Base,
        Draft: &opts.Draft,
    })
    if err != nil {
        return nil, fmt.Errorf("create GitHub PR: %w", err)
    }

    return &PullRequest{
        ID:     pr.GetNumber(),
        URL:    pr.GetHTMLURL(),
        Branch: branch,
        State:  pr.GetState(),
    }, nil
}
```

---

## Claude CLI Integration

### Running Claude

```go
func (c *ClaudeCLI) Run(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error) {
    cfg := &runConfig{
        timeout:  c.timeout,
        maxTurns: c.maxTurns,
    }
    for _, opt := range opts {
        opt(cfg)
    }

    // Build command
    args := []string{"--print"}
    if cfg.systemPrompt != "" {
        args = append(args, "--system", cfg.systemPrompt)
    }
    for _, file := range cfg.contextFiles {
        args = append(args, "--file", file)
    }
    args = append(args, "-p", prompt)

    // Create command with timeout
    ctx, cancel := context.WithTimeout(ctx, cfg.timeout)
    defer cancel()

    cmd := exec.CommandContext(ctx, c.binaryPath, args...)
    if cfg.workDir != "" {
        cmd.Dir = cfg.workDir
    }

    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    start := time.Now()
    err := cmd.Run()
    duration := time.Since(start)

    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            return nil, ErrClaudeTimeout
        }
        return nil, fmt.Errorf("claude failed: %w\nstderr: %s", err, stderr.String())
    }

    // Parse result
    result := &RunResult{
        Output:   stdout.String(),
        Duration: duration,
        ExitCode: cmd.ProcessState.ExitCode(),
    }

    // Parse token usage from stderr if available
    result.TokensIn, result.TokensOut = parseTokenUsage(stderr.String())

    return result, nil
}
```

### Run Options

```go
func WithSystemPrompt(prompt string) RunOption {
    return func(cfg *runConfig) {
        cfg.systemPrompt = prompt
    }
}

func WithContext(files ...string) RunOption {
    return func(cfg *runConfig) {
        cfg.contextFiles = append(cfg.contextFiles, files...)
    }
}

func WithWorkDir(dir string) RunOption {
    return func(cfg *runConfig) {
        cfg.workDir = dir
    }
}

func WithMaxTurns(n int) RunOption {
    return func(cfg *runConfig) {
        cfg.maxTurns = n
    }
}

func WithTimeout(d time.Duration) RunOption {
    return func(cfg *runConfig) {
        cfg.timeout = d
    }
}
```

---

## Transcript Storage

### Directory Structure

```
.devflow/runs/
└── 2025-01-15-ticket-to-pr-TK421/
    ├── metadata.json
    ├── transcript.json.gz     # Compressed if large
    ├── artifacts/
    │   ├── spec.md
    │   ├── implementation.diff
    │   └── review.json
    └── state-checkpoints/
        ├── parse-ticket.json
        ├── generate-spec.json
        └── implement.json
```

### Transcript Format

```json
{
  "runId": "2025-01-15-ticket-to-pr-TK421",
  "metadata": {
    "flowId": "ticket-to-pr",
    "input": {"ticket": "TK-421"},
    "startedAt": "2025-01-15T10:30:00Z",
    "endedAt": "2025-01-15T10:45:32Z",
    "status": "completed"
  },
  "turns": [
    {
      "role": "system",
      "content": "You are an expert software engineer...",
      "tokens": 150,
      "timestamp": "2025-01-15T10:30:00Z"
    },
    {
      "role": "user",
      "content": "Implement the feature described in TK-421...",
      "tokens": 500,
      "timestamp": "2025-01-15T10:30:01Z"
    },
    {
      "role": "assistant",
      "content": "I'll implement this feature by...",
      "tokens": 2500,
      "timestamp": "2025-01-15T10:30:45Z",
      "toolCalls": [
        {
          "name": "write_file",
          "input": {"path": "api.go", "content": "..."}
        }
      ]
    }
  ]
}
```

### Storage Implementation

```go
func (m *TranscriptManager) Save(runID string) error {
    // Get run directory
    runDir := filepath.Join(m.baseDir, "runs", runID)

    // Marshal transcript
    data, err := json.MarshalIndent(m.transcripts[runID], "", "  ")
    if err != nil {
        return fmt.Errorf("marshal transcript: %w", err)
    }

    // Compress if large
    path := filepath.Join(runDir, "transcript.json")
    if len(data) > m.compressThreshold {
        path += ".gz"
        var buf bytes.Buffer
        gz := gzip.NewWriter(&buf)
        gz.Write(data)
        gz.Close()
        data = buf.Bytes()
    }

    return os.WriteFile(path, data, 0644)
}

func (m *TranscriptManager) Load(runID string) (*Transcript, error) {
    runDir := filepath.Join(m.baseDir, "runs", runID)

    // Try compressed first
    path := filepath.Join(runDir, "transcript.json.gz")
    data, err := os.ReadFile(path)
    if err == nil {
        // Decompress
        gz, _ := gzip.NewReader(bytes.NewReader(data))
        data, _ = io.ReadAll(gz)
    } else {
        // Try uncompressed
        path = filepath.Join(runDir, "transcript.json")
        data, err = os.ReadFile(path)
        if err != nil {
            return nil, ErrTranscriptNotFound
        }
    }

    var transcript Transcript
    if err := json.Unmarshal(data, &transcript); err != nil {
        return nil, fmt.Errorf("unmarshal transcript: %w", err)
    }

    return &transcript, nil
}
```

---

## flowgraph Integration

### Pre-built Nodes

```go
// CreateWorktreeNode creates a git worktree for isolated work
func CreateWorktreeNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    git := ctx.Value(gitContextKey).(*GitContext)

    branch := fmt.Sprintf("feature/%s", state.TicketID)
    worktree, err := git.CreateWorktree(branch)
    if err != nil {
        return state, fmt.Errorf("create worktree: %w", err)
    }

    state.Worktree = worktree
    return state, nil
}

// ImplementNode runs Claude to implement code
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    claude := ctx.Value(claudeKey).(*ClaudeCLI)

    result, err := claude.Run(ctx, state.Spec.ImplementationPrompt,
        WithWorkDir(state.Worktree),
        WithSystemPrompt(implementationSystemPrompt),
    )
    if err != nil {
        return state, err
    }

    state.Implementation = &Implementation{
        Output:    result.Output,
        Files:     result.Files,
        TokensIn:  result.TokensIn,
        TokensOut: result.TokensOut,
    }

    return state, nil
}

// CreatePRNode creates a pull request
func CreatePRNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    git := ctx.Value(gitContextKey).(*GitContext)

    pr, err := git.CreatePR(PROptions{
        Title: fmt.Sprintf("[%s] %s", state.TicketID, state.Ticket.Summary),
        Body:  formatPRBody(state),
        Base:  "main",
    })
    if err != nil {
        return state, err
    }

    state.PR = pr
    return state, nil
}
```

---

## Error Handling

### Error Types

```go
var (
    // Git errors
    ErrWorktreeExists = errors.New("worktree already exists")
    ErrGitDirty       = errors.New("working directory has uncommitted changes")
    ErrNoPRProvider   = errors.New("no PR provider configured")
    ErrBranchExists   = errors.New("branch already exists")

    // Claude errors
    ErrClaudeTimeout  = errors.New("claude CLI timed out")
    ErrClaudeFailed   = errors.New("claude CLI failed")

    // Transcript errors
    ErrTranscriptNotFound = errors.New("transcript not found")
    ErrRunNotStarted      = errors.New("run not started")

    // Artifact errors
    ErrArtifactNotFound = errors.New("artifact not found")
)
```

---

## Performance Considerations

### Worktree Storage

Worktrees share git objects with main repo, but:
- Each worktree has its own working files
- Large repos = significant disk usage per worktree
- Cleanup worktrees after use

### Transcript Size

Long-running workflows generate large transcripts:
- Enable compression (`Compress: true`)
- Consider streaming writes for very long runs
- Implement retention policies

### Concurrent Worktrees

Multiple worktrees enable parallel work, but:
- Each worktree is a separate checkout
- Shared git operations may contend
- Consider per-worktree locking
