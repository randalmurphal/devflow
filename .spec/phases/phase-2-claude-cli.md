# Phase 2: Claude CLI Wrapper

## Overview

Implement Claude CLI wrapper for structured LLM invocation.

**Duration**: Week 2
**Dependencies**: None (parallel with Phase 1)
**Deliverables**: `ClaudeCLI` type with run methods and output parsing

---

## Goals

1. Shell out to `claude` binary with proper flags
2. Support system prompts, context files, timeouts
3. Parse JSON output for metrics
4. Capture file changes made by Claude
5. Handle errors gracefully (timeout, failures)

---

## Components

### ClaudeCLI

Main wrapper type:

```go
type ClaudeCLI struct {
    binaryPath string
    model      string
    timeout    time.Duration
    maxTurns   int
}
```

### RunResult

Output from a Claude run:

```go
type RunResult struct {
    Output     string
    TokensIn   int
    TokensOut  int
    Cost       float64
    SessionID  string
    Duration   time.Duration
    ExitCode   int
    Files      []FileChange
}
```

### RunOption

Functional options for configuration:

```go
type RunOption func(*runConfig)

func WithSystemPrompt(prompt string) RunOption
func WithContext(files ...string) RunOption
func WithWorkDir(dir string) RunOption
func WithMaxTurns(n int) RunOption
func WithTimeout(d time.Duration) RunOption
func WithModel(model string) RunOption
```

---

## Implementation Tasks

### Task 2.1: ClaudeCLI Constructor

```go
func NewClaudeCLI(cfg ClaudeConfig) (*ClaudeCLI, error)

type ClaudeConfig struct {
    BinaryPath string        // Default: "claude"
    Model      string        // Default: "" (use claude default)
    Timeout    time.Duration // Default: 5m
    MaxTurns   int           // Default: 10
}
```

**Acceptance Criteria**:
- [ ] Validates claude binary exists (`exec.LookPath`)
- [ ] Sets sensible defaults
- [ ] Returns error if claude not found

### Task 2.2: Run Method

```go
func (c *ClaudeCLI) Run(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error)
```

**Acceptance Criteria**:
- [ ] Builds command with correct flags
- [ ] Uses `--print --output-format json`
- [ ] Applies all options
- [ ] Respects context cancellation
- [ ] Handles timeout correctly

### Task 2.3: Command Building

Build the claude command:

```bash
claude --print --output-format json \
    [--model MODEL] \
    [--system "SYSTEM_PROMPT"] \
    [--max-turns N] \
    [--allowedTools TOOLS...] \
    -p "PROMPT"
```

**Acceptance Criteria**:
- [ ] All options map to correct flags
- [ ] Handles special characters in prompts
- [ ] Sets working directory correctly

### Task 2.4: Output Parsing

```go
func parseClaudeOutput(stdout, stderr []byte) (*RunResult, error)
```

Parse JSON output:
```json
{
  "result": "...",
  "tokens_in": 1500,
  "tokens_out": 2500,
  "session_id": "abc123",
  "cost": 0.05
}
```

**Acceptance Criteria**:
- [ ] Parses JSON output correctly
- [ ] Falls back to raw output on parse failure
- [ ] Extracts all available metrics

### Task 2.5: Context File Handling

```go
type ContextBuilder struct {
    workDir string
    limits  ContextLimits
    files   []contextFile
}

func (b *ContextBuilder) AddFile(path string) error
func (b *ContextBuilder) AddGlob(pattern string) error
func (b *ContextBuilder) Build() (string, error)
```

**Acceptance Criteria**:
- [ ] Reads files and formats with XML tags
- [ ] Handles glob patterns
- [ ] Enforces size limits
- [ ] Detects and handles binary files

### Task 2.6: File Change Detection

```go
func detectFileChanges(workDir string) ([]FileChange, error)
```

**Acceptance Criteria**:
- [ ] Uses `git status --porcelain`
- [ ] Identifies creates, modifies, deletes
- [ ] Works within worktrees

### Task 2.7: Error Handling

```go
var (
    ErrClaudeTimeout  = errors.New("claude CLI timed out")
    ErrClaudeNotFound = errors.New("claude CLI not found")
    ErrClaudeFailed   = errors.New("claude CLI failed")
)
```

**Acceptance Criteria**:
- [ ] Timeout returns ErrClaudeTimeout
- [ ] Non-zero exit returns ErrClaudeFailed with stderr
- [ ] Missing binary returns ErrClaudeNotFound

### Task 2.8: Session Management

```go
func WithSession(sessionID string) RunOption
```

**Acceptance Criteria**:
- [ ] Passes session ID to claude
- [ ] Returns new session ID in result
- [ ] Supports multi-turn conversations

---

## Testing Strategy

### Unit Tests

| Test | Description |
|------|-------------|
| `TestClaudeCLI_New` | Validates config, checks binary |
| `TestClaudeCLI_BuildCommand` | Command has correct flags |
| `TestParseClaudeOutput` | Parses JSON correctly |
| `TestContextBuilder_Build` | Formats files correctly |
| `TestContextBuilder_Limits` | Enforces size limits |

### Integration Tests

Require Claude CLI installed:

```go
func TestClaudeCLI_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    claude, err := NewClaudeCLI(ClaudeConfig{
        Timeout: 30 * time.Second,
    })
    if err != nil {
        t.Skip("claude not installed")
    }

    result, err := claude.Run(context.Background(),
        "Say hello in exactly 3 words")
    require.NoError(t, err)
    assert.NotEmpty(t, result.Output)
    assert.Greater(t, result.TokensOut, 0)
}
```

### Mock Tests

```go
type MockClaudeCLI struct {
    RunFunc func(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error)
}
```

---

## Error Handling

| Error | Condition | Recovery |
|-------|-----------|----------|
| `ErrClaudeNotFound` | Binary not found | Install claude |
| `ErrClaudeTimeout` | Exceeded timeout | Increase timeout, retry |
| `ErrClaudeFailed` | Non-zero exit | Check stderr, fix issue |
| `ErrContextTooLarge` | Context exceeds limit | Reduce files |

---

## File Structure

```
devflow/
├── claude.go           # ClaudeCLI type and methods
├── claude_options.go   # RunOption implementations
├── claude_parse.go     # Output parsing
├── claude_context.go   # Context file handling
├── claude_test.go      # Tests
└── claude_mock.go      # Mock for testing
```

---

## CLI Flags Reference

| Flag | Purpose |
|------|---------|
| `--print` | Non-interactive mode |
| `--output-format json` | JSON output for parsing |
| `--model` | Model selection |
| `--system` | System prompt |
| `--max-turns` | Conversation turn limit |
| `--allowedTools` | Tool allowlist |
| `--disallowedTools` | Tool denylist |
| `-p` | Prompt text |

---

## Completion Criteria

- [ ] All tasks implemented
- [ ] Unit test coverage > 80%
- [ ] Integration test with real Claude
- [ ] Mock available for testing
- [ ] Error handling complete
- [ ] Documentation complete

---

## References

- ADR-006: Claude CLI Wrapper
- ADR-007: Prompt Management
- ADR-008: Context Files
- ADR-009: Output Parsing
- ADR-010: Session Management
