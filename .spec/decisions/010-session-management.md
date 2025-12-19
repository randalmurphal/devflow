# ADR-010: Session Management

## Status

Accepted

## Context

Some devflow operations require multi-turn conversations with Claude:

1. Interactive implementation with feedback
2. Review cycles with fix requests
3. Exploratory conversations

We need to decide how to manage conversation state across turns.

## Decision

### 1. Stateless by Default

Most devflow operations are single-turn:

```go
// Single turn - no session needed
result, err := claude.Run(ctx, "Implement the user endpoint")
```

### 2. Session ID for Resume

When resuming is needed, use Claude's session ID:

```go
// First turn
result1, err := claude.Run(ctx, "Start implementing auth")
sessionID := result1.SessionID

// Resume later
result2, err := claude.Run(ctx, "Continue with OAuth",
    devflow.WithSession(sessionID),
)
```

### 3. devflow Does NOT Manage State

Claude CLI manages conversation state. devflow just:
- Captures session IDs
- Passes session IDs back when resuming
- Records transcripts separately (ADR-011)

### 4. Multi-Turn Patterns

For iterative workflows, use loops:

```go
// Review loop
for {
    // Get review
    review, _ := claude.Run(ctx, reviewPrompt, devflow.WithSession(sessionID))
    reviewResult := parseReview(review.Output)

    if reviewResult.Approved {
        break
    }

    // Fix findings
    _, _ = claude.Run(ctx, fixPrompt(reviewResult.Findings), devflow.WithSession(sessionID))
}
```

### 5. Session Limits

| Limit | Value | Behavior |
|-------|-------|----------|
| Max turns per session | 50 | Error, start new session |
| Session timeout | 24 hours | Error, start new session |
| Max session size | 200K tokens | Error, summarize and continue |

These are Claude CLI limits, not devflow-imposed.

## Alternatives Considered

### Alternative 1: devflow State Management

Maintain conversation state in devflow.

**Rejected because:**
- Duplicates Claude CLI functionality
- Complex to maintain
- State synchronization issues

### Alternative 2: Database Sessions

Store sessions in database.

**Rejected because:**
- Adds database dependency
- Overkill for typical use
- Claude CLI already persists

### Alternative 3: File-Based Sessions

Store session state in JSON files.

**Rejected because:**
- Duplicates Claude CLI state
- Synchronization risk
- Unnecessary complexity

## Consequences

### Positive

- **Simple**: Delegates to Claude CLI
- **No state management**: devflow stays stateless
- **Resume capability**: Can continue conversations

### Negative

- **Claude CLI dependency**: Relies on CLI session handling
- **Limited control**: Can't manipulate conversation history
- **Opaque state**: Don't know what's in session

### Session Workflow

```
┌──────────────────────────────────────────────────────────────┐
│                      devflow Session Flow                     │
├──────────────────────────────────────────────────────────────┤
│                                                               │
│  ┌─────────┐     ┌─────────────┐     ┌─────────────┐        │
│  │ Turn 1  │────▶│ Claude CLI  │────▶│ Session ID  │        │
│  │ (start) │     │ (creates)   │     │ (returned)  │        │
│  └─────────┘     └─────────────┘     └──────┬──────┘        │
│                                              │               │
│                                              ▼               │
│  ┌─────────┐     ┌─────────────┐     ┌─────────────┐        │
│  │ Turn 2  │────▶│ Claude CLI  │────▶│ Continue    │        │
│  │ (resume)│     │ (loads)     │     │ conversation│        │
│  └─────────┘     └─────────────┘     └──────┬──────┘        │
│       │                                      │               │
│       └──────────────────────────────────────┘               │
│                   (repeat as needed)                         │
│                                                               │
└──────────────────────────────────────────────────────────────┘
```

## Code Example

```go
package devflow

// Session represents a Claude conversation session
type Session struct {
    ID        string    // Claude session ID
    StartedAt time.Time
    TurnCount int
}

// WithSession resumes an existing session
func WithSession(sessionID string) RunOption {
    return func(cfg *runConfig) {
        cfg.sessionID = sessionID
    }
}

// MultiTurnRunner manages multi-turn conversations
type MultiTurnRunner struct {
    claude    *ClaudeCLI
    sessionID string
    turns     int
    maxTurns  int
}

// NewMultiTurnRunner creates a runner for multi-turn conversations
func NewMultiTurnRunner(claude *ClaudeCLI, maxTurns int) *MultiTurnRunner {
    if maxTurns == 0 {
        maxTurns = 50
    }
    return &MultiTurnRunner{
        claude:   claude,
        maxTurns: maxTurns,
    }
}

// Run executes a turn, managing session automatically
func (r *MultiTurnRunner) Run(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error) {
    if r.turns >= r.maxTurns {
        return nil, fmt.Errorf("max turns reached: %d", r.maxTurns)
    }

    // Add session if we have one
    if r.sessionID != "" {
        opts = append(opts, WithSession(r.sessionID))
    }

    result, err := r.claude.Run(ctx, prompt, opts...)
    if err != nil {
        return nil, err
    }

    // Capture session ID for next turn
    r.sessionID = result.SessionID
    r.turns++

    return result, nil
}

// Reset starts a fresh session
func (r *MultiTurnRunner) Reset() {
    r.sessionID = ""
    r.turns = 0
}

// TurnCount returns number of turns in current session
func (r *MultiTurnRunner) TurnCount() int {
    return r.turns
}

// SessionID returns current session ID
func (r *MultiTurnRunner) SessionID() string {
    return r.sessionID
}
```

### Usage: Simple Resume

```go
// Start conversation
result1, err := claude.Run(ctx, "Explain the authentication flow")
if err != nil {
    return err
}
sessionID := result1.SessionID

// ... later ...

// Continue conversation
result2, err := claude.Run(ctx, "Now explain the authorization part",
    devflow.WithSession(sessionID),
)
```

### Usage: Multi-Turn Runner

```go
runner := devflow.NewMultiTurnRunner(claude, 20)

// First turn
result, err := runner.Run(ctx, "Start implementing the auth endpoint")
if err != nil {
    return err
}

// Review loop
for {
    // Run tests
    testOutput := runTests()

    if testOutput.AllPassed {
        break
    }

    // Ask for fixes
    result, err = runner.Run(ctx, fmt.Sprintf("Tests failed:\n%s\n\nPlease fix.", testOutput.Failures))
    if err != nil {
        return err
    }

    if runner.TurnCount() > 10 {
        return fmt.Errorf("too many fix attempts")
    }
}

fmt.Printf("Completed in %d turns\n", runner.TurnCount())
```

### Usage: Review Cycle

```go
func ReviewCycle(ctx context.Context, claude *ClaudeCLI, workDir string) error {
    runner := devflow.NewMultiTurnRunner(claude, 30)

    // Initial implementation
    _, err := runner.Run(ctx, implementPrompt,
        devflow.WithWorkDir(workDir),
    )
    if err != nil {
        return err
    }

    // Review loop
    maxReviews := 3
    for i := 0; i < maxReviews; i++ {
        // Review
        reviewResult, err := runner.Run(ctx, reviewPrompt)
        if err != nil {
            return err
        }

        review, err := devflow.ParseJSON[ReviewResult](reviewResult.Output)
        if err != nil {
            return err
        }

        if review.Approved {
            return nil // Success!
        }

        // Fix findings
        fixPrompt := fmt.Sprintf("Please fix these issues:\n%s",
            formatFindings(review.Findings))

        _, err = runner.Run(ctx, fixPrompt)
        if err != nil {
            return err
        }
    }

    return fmt.Errorf("review not approved after %d attempts", maxReviews)
}
```

### Integration with flowgraph

For graph-based workflows, session state lives in the graph state:

```go
type DevState struct {
    TicketID   string
    Spec       *Spec
    SessionID  string  // Track session across nodes
    TurnCount  int
}

func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    claude := ctx.Value(claudeKey).(*ClaudeCLI)

    var opts []devflow.RunOption
    if state.SessionID != "" {
        opts = append(opts, devflow.WithSession(state.SessionID))
    }

    result, err := claude.Run(ctx, state.Spec.ImplementPrompt, opts...)
    if err != nil {
        return state, err
    }

    state.SessionID = result.SessionID
    state.TurnCount++
    state.Implementation = result.Output

    return state, nil
}
```

## References

- [Claude CLI Sessions](https://docs.anthropic.com/en/docs/claude-cli)
- ADR-006: Claude CLI Wrapper
- ADR-011: Transcript Format
