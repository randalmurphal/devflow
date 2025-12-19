# ADR-020: Error Handling Strategy

## Status

Accepted

## Context

devflow operations can fail in many ways:

1. Git operations fail (conflicts, permissions)
2. Claude timeouts or errors
3. Parse failures
4. Network issues
5. Validation failures

We need a consistent error handling strategy that:
- Provides actionable error messages
- Enables recovery where possible
- Integrates with flowgraph error handling

## Decision

### 1. Sentinel Errors for Known Failures

Define sentinel errors for expected failure modes:

```go
var (
    // Git errors
    ErrWorktreeExists   = errors.New("worktree already exists")
    ErrGitDirty         = errors.New("uncommitted changes in working directory")
    ErrBranchExists     = errors.New("branch already exists")
    ErrNoPRProvider     = errors.New("no PR provider configured")
    ErrPushFailed       = errors.New("push failed")

    // Claude errors
    ErrClaudeTimeout    = errors.New("claude CLI timed out")
    ErrClaudeNotFound   = errors.New("claude CLI not found")
    ErrContextTooLarge  = errors.New("context exceeds limit")
    ErrMaxTurnsReached  = errors.New("max turns reached")

    // Parse errors
    ErrInvalidJSON      = errors.New("invalid JSON in output")
    ErrNoJSONFound      = errors.New("no JSON found in output")

    // Transcript errors
    ErrRunNotFound      = errors.New("run not found")
    ErrRunNotStarted    = errors.New("run not started")

    // Artifact errors
    ErrArtifactNotFound = errors.New("artifact not found")

    // Validation errors
    ErrMissingRequired  = errors.New("missing required field")
    ErrInvalidState     = errors.New("invalid state")
)
```

### 2. Error Wrapping with Context

Always wrap errors with context:

```go
func CreateWorktreeNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    git := GitFromContext(ctx)
    if git == nil {
        return state, fmt.Errorf("create worktree: %w", ErrGitNotConfigured)
    }

    worktree, err := git.CreateWorktree(state.Branch)
    if err != nil {
        return state, fmt.Errorf("create worktree for %s: %w", state.Branch, err)
    }

    // ...
}
```

### 3. Error Classification

Errors are classified for handling:

| Category | Behavior | Examples |
|----------|----------|----------|
| Retryable | Can retry immediately | Timeout, rate limit |
| Recoverable | Can resume with fix | Git dirty, branch exists |
| Fatal | Must abort | Config missing, auth failed |

```go
type ErrorCategory int

const (
    ErrorRetryable ErrorCategory = iota
    ErrorRecoverable
    ErrorFatal
)

func ClassifyError(err error) ErrorCategory {
    switch {
    case errors.Is(err, ErrClaudeTimeout):
        return ErrorRetryable
    case errors.Is(err, ErrGitDirty):
        return ErrorRecoverable
    case errors.Is(err, ErrClaudeNotFound):
        return ErrorFatal
    default:
        return ErrorFatal
    }
}
```

### 4. Node Error Handling Pattern

Nodes follow a consistent pattern:

```go
func SomeNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // 1. Validate prerequisites
    if err := state.Validate("ticket", "worktree"); err != nil {
        return state, fmt.Errorf("validation: %w", err)
    }

    // 2. Get dependencies from context
    claude := ClaudeFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("ClaudeCLI not configured")
    }

    // 3. Perform operation
    result, err := claude.Run(ctx, prompt)
    if err != nil {
        // 4. Record error in state for debugging
        state.Error = err.Error()
        return state, fmt.Errorf("claude run: %w", err)
    }

    // 5. Parse and validate output
    parsed, err := ParseJSON[ExpectedType](result.Output)
    if err != nil {
        state.Error = fmt.Sprintf("parse failure: %s", result.Output[:200])
        return state, fmt.Errorf("parse output: %w", err)
    }

    // 6. Update state
    state.Whatever = parsed
    return state, nil
}
```

### 5. Retry Behavior

Retryable errors can be retried with backoff:

```go
type RetryConfig struct {
    MaxAttempts int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

func WithRetry(node flowgraph.NodeFunc[DevState], cfg RetryConfig) flowgraph.NodeFunc[DevState] {
    return func(ctx flowgraph.Context, state DevState) (DevState, error) {
        var lastErr error
        delay := cfg.InitialDelay

        for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
            result, err := node(ctx, state)
            if err == nil {
                return result, nil
            }

            lastErr = err
            if ClassifyError(err) != ErrorRetryable {
                return result, err
            }

            ctx.Logger().Warn("retrying",
                "attempt", attempt,
                "error", err,
                "delay", delay)

            select {
            case <-ctx.Done():
                return result, ctx.Err()
            case <-time.After(delay):
            }

            delay = time.Duration(float64(delay) * cfg.Multiplier)
            if delay > cfg.MaxDelay {
                delay = cfg.MaxDelay
            }
        }

        return state, fmt.Errorf("max retries exceeded: %w", lastErr)
    }
}
```

### 6. Error Recording

Errors are recorded for debugging:

```go
// In state
type DevState struct {
    // ... other fields

    // Error tracking
    Error      string   `json:"error,omitempty"`
    ErrorTrace []string `json:"errorTrace,omitempty"`
}

// Record error with trace
func (s *DevState) RecordError(err error) {
    s.Error = err.Error()
    s.ErrorTrace = append(s.ErrorTrace, fmt.Sprintf("[%s] %s",
        time.Now().Format(time.RFC3339), err.Error()))
}
```

## Alternatives Considered

### Alternative 1: Panic/Recover

Use panics for unexpected errors.

**Rejected because:**
- Go idiom is explicit error returns
- flowgraph expects error returns
- Panics are for truly exceptional cases

### Alternative 2: Error Codes

Use numeric error codes.

**Rejected because:**
- Sentinel errors more Go-idiomatic
- Strings more debuggable
- errors.Is/As work with sentinels

### Alternative 3: Result Types

Use Result[T] type instead of (T, error).

**Rejected because:**
- Not Go standard
- flowgraph uses (T, error)
- Would require custom types

## Consequences

### Positive

- **Consistent**: Same pattern across all nodes
- **Debuggable**: Rich error context
- **Recoverable**: Classification enables smart handling
- **Traceable**: Error history in state

### Negative

- **Verbose**: Lots of error wrapping
- **Memory**: Error trace grows
- **Complexity**: Retry logic adds code

### Error Message Guidelines

Good error messages include:

1. **What** failed
2. **Why** (immediate cause)
3. **Context** (inputs, state)
4. **Recovery** (what user can do)

```go
// Good
fmt.Errorf("create worktree for branch %s: %w (try deleting existing worktree)", branch, err)

// Bad
fmt.Errorf("failed")
```

## Code Example

```go
package devflow

import (
    "errors"
    "fmt"
    "time"
)

// Sentinel errors
var (
    // Git
    ErrWorktreeExists = errors.New("worktree already exists")
    ErrGitDirty       = errors.New("uncommitted changes")
    ErrBranchExists   = errors.New("branch already exists")
    ErrNoPRProvider   = errors.New("no PR provider configured")

    // Claude
    ErrClaudeTimeout  = errors.New("claude timed out")
    ErrClaudeNotFound = errors.New("claude CLI not found")
    ErrMaxTurns       = errors.New("max turns reached")

    // Parse
    ErrInvalidJSON = errors.New("invalid JSON")
    ErrNoJSON      = errors.New("no JSON found")

    // State
    ErrMissingField = errors.New("missing required field")
)

// ErrorCategory classifies errors
type ErrorCategory int

const (
    ErrorRetryable ErrorCategory = iota
    ErrorRecoverable
    ErrorFatal
)

// ClassifyError determines error category
func ClassifyError(err error) ErrorCategory {
    switch {
    case errors.Is(err, ErrClaudeTimeout):
        return ErrorRetryable
    case errors.Is(err, ErrMaxTurns):
        return ErrorRetryable
    case errors.Is(err, ErrWorktreeExists):
        return ErrorRecoverable
    case errors.Is(err, ErrGitDirty):
        return ErrorRecoverable
    case errors.Is(err, ErrBranchExists):
        return ErrorRecoverable
    default:
        return ErrorFatal
    }
}

// IsRetryable checks if error can be retried
func IsRetryable(err error) bool {
    return ClassifyError(err) == ErrorRetryable
}

// ValidationError provides detailed validation failure
type ValidationError struct {
    Field   string
    Message string
}

func (e ValidationError) Error() string {
    return fmt.Sprintf("validation failed: %s: %s", e.Field, e.Message)
}

// WrapWithContext wraps an error with operation context
func WrapWithContext(op string, err error) error {
    if err == nil {
        return nil
    }
    return fmt.Errorf("%s: %w", op, err)
}

// RetryConfig configures retry behavior
type RetryConfig struct {
    MaxAttempts  int
    InitialDelay time.Duration
    MaxDelay     time.Duration
    Multiplier   float64
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
    return RetryConfig{
        MaxAttempts:  3,
        InitialDelay: 1 * time.Second,
        MaxDelay:     30 * time.Second,
        Multiplier:   2.0,
    }
}

// WithRetry wraps a node with retry logic
func WithRetry(
    node func(ctx flowgraph.Context, state DevState) (DevState, error),
    cfg RetryConfig,
) func(ctx flowgraph.Context, state DevState) (DevState, error) {
    return func(ctx flowgraph.Context, state DevState) (DevState, error) {
        var lastErr error
        delay := cfg.InitialDelay

        for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
            result, err := node(ctx, state)
            if err == nil {
                return result, nil
            }

            lastErr = err

            // Only retry retryable errors
            if !IsRetryable(err) {
                return result, err
            }

            // Log retry
            ctx.Logger().Warn("retrying after error",
                "attempt", attempt,
                "max_attempts", cfg.MaxAttempts,
                "error", err,
                "next_delay", delay,
            )

            // Wait before retry
            select {
            case <-ctx.Done():
                return result, ctx.Err()
            case <-time.After(delay):
            }

            // Increase delay
            delay = time.Duration(float64(delay) * cfg.Multiplier)
            if delay > cfg.MaxDelay {
                delay = cfg.MaxDelay
            }
        }

        return state, fmt.Errorf("max retries (%d) exceeded: %w", cfg.MaxAttempts, lastErr)
    }
}

// Example node with proper error handling
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // Validate prerequisites
    if state.Spec == "" {
        return state, ValidationError{Field: "Spec", Message: "specification required"}
    }
    if state.Worktree == "" {
        return state, ValidationError{Field: "Worktree", Message: "worktree required"}
    }

    // Get dependencies
    claude := ClaudeFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("implement: claude CLI not configured")
    }

    // Execute with context
    result, err := claude.Run(ctx,
        formatImplementPrompt(state.Spec),
        WithWorkDir(state.Worktree),
        WithMaxTurns(30),
    )
    if err != nil {
        state.RecordError(err)
        return state, WrapWithContext("implement", err)
    }

    // Update state
    state.Implementation = result.Output
    state.Files = result.Files
    state.ImplementTokensIn = result.TokensIn
    state.ImplementTokensOut = result.TokensOut
    state.AddTokens(result.TokensIn, result.TokensOut)

    return state, nil
}

// Using retry
var ImplementNodeWithRetry = WithRetry(ImplementNode, DefaultRetryConfig())
```

### Usage in Graph

```go
graph := flowgraph.NewGraph[DevState]().
    AddNode("implement", devflow.WithRetry(
        devflow.ImplementNode,
        devflow.RetryConfig{
            MaxAttempts:  5,
            InitialDelay: 2 * time.Second,
            MaxDelay:     60 * time.Second,
            Multiplier:   2.0,
        },
    )).
    // ...
```

## References

- flowgraph error handling documentation
- ADR-018: flowgraph Integration
- ADR-019: State Design
- [Go Error Handling](https://go.dev/blog/go1.13-errors)
