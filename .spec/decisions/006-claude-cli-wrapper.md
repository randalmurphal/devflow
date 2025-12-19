# ADR-006: Claude CLI Wrapper

## Status

Accepted

## Context

devflow needs to invoke Claude for AI-powered development tasks. We need to decide:

1. How to invoke Claude (CLI vs API)
2. How to configure invocations
3. How to capture output and transcripts
4. How to handle timeouts and errors

## Decision

### 1. Shell Out to Claude CLI

Use `exec.Command("claude", ...)` rather than Anthropic API directly.

**Rationale:**
- Claude CLI handles authentication (API keys, OAuth)
- Claude CLI has built-in tooling (file editing, command execution)
- Users already have Claude CLI configured
- Simpler than managing API credentials
- Matches how users interact with Claude

### 2. ClaudeCLI Struct

```go
type ClaudeCLI struct {
    binaryPath string        // Path to claude binary
    model      string        // Default model
    timeout    time.Duration // Default timeout
    maxTurns   int           // Max conversation turns
}

type ClaudeConfig struct {
    BinaryPath string        // Path to claude binary (default: "claude")
    Model      string        // Model (default: uses claude default)
    Timeout    time.Duration // Default timeout (default: 5m)
    MaxTurns   int           // Max turns (default: 10)
}
```

### 3. Run Method with Functional Options

```go
func (c *ClaudeCLI) Run(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error)

// Options
func WithSystemPrompt(prompt string) RunOption
func WithContext(files ...string) RunOption
func WithWorkDir(dir string) RunOption
func WithMaxTurns(n int) RunOption
func WithTimeout(d time.Duration) RunOption
func WithModel(model string) RunOption
func WithAllowedTools(tools ...string) RunOption
func WithDisallowedTools(tools ...string) RunOption
```

### 4. RunResult Structure

```go
type RunResult struct {
    Output     string        // Final output text
    TokensIn   int           // Input tokens consumed
    TokensOut  int           // Output tokens generated
    Transcript *Transcript   // Full conversation (if captured)
    Files      []FileChange  // Files created/modified
    Duration   time.Duration // Execution time
    ExitCode   int           // Process exit code
    SessionID  string        // Claude session ID (for resume)
}
```

### 5. CLI Flags Used

| Flag | Purpose | When Used |
|------|---------|-----------|
| `--print` | Non-interactive mode | Always |
| `--output-format json` | JSON output for parsing | Always |
| `--model` | Model selection | When specified |
| `--system` | System prompt | When specified |
| `--max-turns` | Limit turns | When specified |
| `--allowedTools` | Tool allowlist | When specified |
| `--disallowedTools` | Tool denylist | When specified |

### 6. Error Handling

```go
var (
    ErrClaudeTimeout   = errors.New("claude CLI timed out")
    ErrClaudeFailed    = errors.New("claude CLI failed")
    ErrClaudeNotFound  = errors.New("claude CLI not found")
    ErrContextTooLarge = errors.New("context exceeds limit")
)
```

## Alternatives Considered

### Alternative 1: Anthropic API Directly

Use the Anthropic Go SDK to call Claude API.

**Rejected because:**
- Requires managing API keys
- Loses Claude CLI tooling (file edit, bash)
- Users would need separate auth setup
- More complex implementation

### Alternative 2: HTTP to Claude API

Make raw HTTP calls to Anthropic API.

**Rejected because:**
- Even more complex than SDK
- No advantages over SDK
- Reinventing the wheel

### Alternative 3: Multiple LLM Support

Abstract to support multiple LLMs (OpenAI, etc.).

**Deferred because:**
- YAGNI - we're using Claude
- Can add abstraction layer later if needed
- flowgraph has LLMClient abstraction for this

## Consequences

### Positive

- **Simple**: Shell out is straightforward
- **Authenticated**: Uses user's Claude CLI auth
- **Tooling**: Gets Claude's built-in tools for free
- **Debuggable**: Can run the command manually to debug

### Negative

- **Dependency**: Requires Claude CLI installed
- **Parsing**: Must parse stdout/stderr
- **Less control**: Can't stream tokens easily
- **Platform**: CLI behavior may vary by platform

### Mitigations

1. **Check CLI**: Validate claude is installed at init
2. **JSON output**: Use `--output-format json` for structured parsing
3. **Timeout wrapper**: Wrap with context timeout
4. **Error context**: Include command in error messages

## Code Example

```go
package devflow

import (
    "bytes"
    "context"
    "encoding/json"
    "errors"
    "fmt"
    "os/exec"
    "time"
)

// ClaudeCLI wraps the claude CLI binary
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

// NewClaudeCLI creates a new Claude CLI wrapper
func NewClaudeCLI(cfg ClaudeConfig) (*ClaudeCLI, error) {
    binaryPath := cfg.BinaryPath
    if binaryPath == "" {
        binaryPath = "claude"
    }

    // Verify claude is installed
    if _, err := exec.LookPath(binaryPath); err != nil {
        return nil, ErrClaudeNotFound
    }

    timeout := cfg.Timeout
    if timeout == 0 {
        timeout = 5 * time.Minute
    }

    maxTurns := cfg.MaxTurns
    if maxTurns == 0 {
        maxTurns = 10
    }

    return &ClaudeCLI{
        binaryPath: binaryPath,
        model:      cfg.Model,
        timeout:    timeout,
        maxTurns:   maxTurns,
    }, nil
}

// runConfig holds configuration for a single run
type runConfig struct {
    systemPrompt   string
    contextFiles   []string
    workDir        string
    maxTurns       int
    timeout        time.Duration
    model          string
    allowedTools   []string
    disallowedTools []string
}

// RunOption configures a run
type RunOption func(*runConfig)

// WithSystemPrompt sets the system prompt
func WithSystemPrompt(prompt string) RunOption {
    return func(cfg *runConfig) {
        cfg.systemPrompt = prompt
    }
}

// WithContext adds context files
func WithContext(files ...string) RunOption {
    return func(cfg *runConfig) {
        cfg.contextFiles = append(cfg.contextFiles, files...)
    }
}

// WithWorkDir sets the working directory
func WithWorkDir(dir string) RunOption {
    return func(cfg *runConfig) {
        cfg.workDir = dir
    }
}

// WithMaxTurns limits conversation turns
func WithMaxTurns(n int) RunOption {
    return func(cfg *runConfig) {
        cfg.maxTurns = n
    }
}

// WithTimeout sets the timeout
func WithTimeout(d time.Duration) RunOption {
    return func(cfg *runConfig) {
        cfg.timeout = d
    }
}

// Run executes Claude with the given prompt
func (c *ClaudeCLI) Run(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error) {
    // Apply defaults and options
    cfg := &runConfig{
        timeout:  c.timeout,
        maxTurns: c.maxTurns,
        model:    c.model,
    }
    for _, opt := range opts {
        opt(cfg)
    }

    // Build command arguments
    args := []string{"--print", "--output-format", "json"}

    if cfg.model != "" {
        args = append(args, "--model", cfg.model)
    }
    if cfg.systemPrompt != "" {
        args = append(args, "--system", cfg.systemPrompt)
    }
    if cfg.maxTurns > 0 {
        args = append(args, "--max-turns", fmt.Sprintf("%d", cfg.maxTurns))
    }
    for _, tool := range cfg.allowedTools {
        args = append(args, "--allowedTools", tool)
    }
    for _, tool := range cfg.disallowedTools {
        args = append(args, "--disallowedTools", tool)
    }

    // Add prompt
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

    // Handle errors
    if err != nil {
        if errors.Is(ctx.Err(), context.DeadlineExceeded) {
            return nil, fmt.Errorf("%w: after %v", ErrClaudeTimeout, cfg.timeout)
        }
        return nil, fmt.Errorf("%w: %s: %s", ErrClaudeFailed, err, stderr.String())
    }

    // Parse JSON output
    result, err := parseClaudeOutput(stdout.Bytes())
    if err != nil {
        // Fallback to raw output
        result = &RunResult{
            Output: stdout.String(),
        }
    }

    result.Duration = duration
    if cmd.ProcessState != nil {
        result.ExitCode = cmd.ProcessState.ExitCode()
    }

    return result, nil
}

// parseClaudeOutput parses JSON output from claude CLI
func parseClaudeOutput(data []byte) (*RunResult, error) {
    var output struct {
        Result      string `json:"result"`
        TokensIn    int    `json:"tokens_in"`
        TokensOut   int    `json:"tokens_out"`
        SessionID   string `json:"session_id"`
    }

    if err := json.Unmarshal(data, &output); err != nil {
        return nil, err
    }

    return &RunResult{
        Output:    output.Result,
        TokensIn:  output.TokensIn,
        TokensOut: output.TokensOut,
        SessionID: output.SessionID,
    }, nil
}
```

### Usage

```go
// Create wrapper
claude, err := devflow.NewClaudeCLI(devflow.ClaudeConfig{
    Timeout:  10 * time.Minute,
    MaxTurns: 20,
})

// Simple run
result, err := claude.Run(ctx, "Explain this code",
    devflow.WithContext("main.go"),
)

// Complex run
result, err := claude.Run(ctx, "Implement the feature described in the spec",
    devflow.WithSystemPrompt("You are an expert Go developer. Write clean, idiomatic Go code."),
    devflow.WithContext("spec.md", "api.go", "types.go"),
    devflow.WithWorkDir(worktreePath),
    devflow.WithMaxTurns(30),
    devflow.WithTimeout(15*time.Minute),
)

fmt.Printf("Output: %s\n", result.Output)
fmt.Printf("Tokens: %d in, %d out\n", result.TokensIn, result.TokensOut)
```

## References

- [Claude CLI Documentation](https://docs.anthropic.com/en/docs/claude-cli)
- ADR-007: Prompt Management (how prompts are loaded)
- ADR-008: Context Files (how files are passed)
