# ADR-019: State Design Patterns

## Status

Accepted

## Context

flowgraph uses typed state that flows through the graph. devflow needs state patterns for:

1. Common dev workflow data (ticket, spec, code)
2. Git context (worktree, branch)
3. Execution metrics (tokens, cost, duration)
4. Optional fields (not all workflows need all fields)

## Decision

### 1. Embeddable State Components

devflow provides embeddable state structs:

```go
// Users embed in their state
type MyState struct {
    // Their fields
    TicketID string

    // Embed devflow components
    devflow.GitState
    devflow.SpecState
    devflow.ImplementState
    devflow.ReviewState
    devflow.PRState
}
```

### 2. State Component Definitions

```go
// GitState tracks git workspace
type GitState struct {
    Worktree   string `json:"worktree,omitempty"`
    Branch     string `json:"branch,omitempty"`
    BaseBranch string `json:"baseBranch,omitempty"`
}

// SpecState tracks specification generation
type SpecState struct {
    Spec          string    `json:"spec,omitempty"`
    SpecTokensIn  int       `json:"specTokensIn,omitempty"`
    SpecTokensOut int       `json:"specTokensOut,omitempty"`
    SpecGeneratedAt time.Time `json:"specGeneratedAt,omitempty"`
}

// ImplementState tracks implementation
type ImplementState struct {
    Implementation    string       `json:"implementation,omitempty"`
    Files             []FileChange `json:"files,omitempty"`
    ImplementTokensIn  int         `json:"implementTokensIn,omitempty"`
    ImplementTokensOut int         `json:"implementTokensOut,omitempty"`
}

// ReviewState tracks code review
type ReviewState struct {
    Review          *ReviewResult `json:"review,omitempty"`
    ReviewAttempts  int           `json:"reviewAttempts,omitempty"`
    ReviewTokensIn  int           `json:"reviewTokensIn,omitempty"`
    ReviewTokensOut int           `json:"reviewTokensOut,omitempty"`
}

// PRState tracks pull request
type PRState struct {
    PR        *PullRequest `json:"pr,omitempty"`
    PRCreated time.Time    `json:"prCreated,omitempty"`
}

// MetricsState tracks execution metrics
type MetricsState struct {
    TotalTokensIn  int           `json:"totalTokensIn"`
    TotalTokensOut int           `json:"totalTokensOut"`
    TotalCost      float64       `json:"totalCost"`
    TotalDuration  time.Duration `json:"totalDuration"`
}
```

### 3. DevState Convenience Type

A complete state type for common workflows:

```go
// DevState is a full-featured state for dev workflows
type DevState struct {
    // Identification
    RunID    string `json:"runId"`
    FlowID   string `json:"flowId"`
    TicketID string `json:"ticketId,omitempty"`

    // Input data
    Ticket *Ticket `json:"ticket,omitempty"`

    // Embedded components
    GitState
    SpecState
    ImplementState
    ReviewState
    PRState
    MetricsState

    // Error tracking
    Error string `json:"error,omitempty"`
}
```

### 4. State Initialization

Helper functions to initialize state:

```go
// NewDevState creates a new state with run ID
func NewDevState(flowID string) DevState {
    return DevState{
        RunID:  generateRunID(flowID),
        FlowID: flowID,
    }
}

// WithTicket adds ticket information
func (s DevState) WithTicket(ticket *Ticket) DevState {
    s.TicketID = ticket.ID
    s.Ticket = ticket
    return s
}
```

### 5. State Validation

Nodes validate required state fields:

```go
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // Validate prerequisites
    if state.Ticket == nil {
        return state, fmt.Errorf("Ticket required for spec generation")
    }

    // ... generate spec
}
```

### 6. Immutable Updates

State is updated by returning new state:

```go
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    result, err := runImplementation(ctx, state)
    if err != nil {
        return state, err
    }

    // Update state immutably
    state.Implementation = result.Output
    state.Files = result.Files
    state.ImplementTokensIn = result.TokensIn
    state.ImplementTokensOut = result.TokensOut

    // Update totals
    state.TotalTokensIn += result.TokensIn
    state.TotalTokensOut += result.TokensOut

    return state, nil
}
```

## Alternatives Considered

### Alternative 1: Interface-Based State

Define state as interface with getters/setters.

**Rejected because:**
- Verbose
- Loses compile-time type checking
- Go structs work fine

### Alternative 2: Single Monolithic State

One big state struct with all fields.

**Rejected because:**
- Not composable
- Unused fields waste memory
- Harder to understand

### Alternative 3: Map-Based State

Use `map[string]any` for flexibility.

**Rejected because:**
- No type safety
- Runtime errors instead of compile errors
- Harder to document

## Consequences

### Positive

- **Type-safe**: Compile-time checks
- **Composable**: Embed what you need
- **Documented**: Clear struct definitions
- **Serializable**: JSON-friendly for checkpointing

### Negative

- **Boilerplate**: Some repeated patterns
- **Embedding quirks**: Go embedding has gotchas
- **Required fields**: Must remember to set them

### Best Practices

1. **Initialize early**: Set RunID and FlowID at start
2. **Validate prerequisites**: Check required fields in nodes
3. **Update totals**: Keep MetricsState current
4. **Document requirements**: Comment what each node needs

## Code Example

```go
package devflow

import (
    "fmt"
    "time"
)

// State components

type GitState struct {
    Worktree   string `json:"worktree,omitempty"`
    Branch     string `json:"branch,omitempty"`
    BaseBranch string `json:"baseBranch,omitempty"`
}

type SpecState struct {
    Spec            string    `json:"spec,omitempty"`
    SpecTokensIn    int       `json:"specTokensIn,omitempty"`
    SpecTokensOut   int       `json:"specTokensOut,omitempty"`
    SpecGeneratedAt time.Time `json:"specGeneratedAt,omitempty"`
}

type ImplementState struct {
    Implementation     string       `json:"implementation,omitempty"`
    Files              []FileChange `json:"files,omitempty"`
    ImplementTokensIn  int          `json:"implementTokensIn,omitempty"`
    ImplementTokensOut int          `json:"implementTokensOut,omitempty"`
}

type ReviewState struct {
    Review          *ReviewResult `json:"review,omitempty"`
    ReviewAttempts  int           `json:"reviewAttempts,omitempty"`
    ReviewTokensIn  int           `json:"reviewTokensIn,omitempty"`
    ReviewTokensOut int           `json:"reviewTokensOut,omitempty"`
}

type PRState struct {
    PR        *PullRequest `json:"pr,omitempty"`
    PRCreated time.Time    `json:"prCreated,omitempty"`
}

type MetricsState struct {
    TotalTokensIn  int           `json:"totalTokensIn"`
    TotalTokensOut int           `json:"totalTokensOut"`
    TotalCost      float64       `json:"totalCost"`
    StartTime      time.Time     `json:"startTime"`
    TotalDuration  time.Duration `json:"totalDuration"`
}

// Ticket represents input ticket data
type Ticket struct {
    ID          string `json:"id"`
    Title       string `json:"title"`
    Description string `json:"description"`
    Priority    string `json:"priority,omitempty"`
    Labels      []string `json:"labels,omitempty"`
}

// DevState is the complete dev workflow state
type DevState struct {
    // Identification
    RunID    string `json:"runId"`
    FlowID   string `json:"flowId"`
    TicketID string `json:"ticketId,omitempty"`

    // Input
    Ticket *Ticket `json:"ticket,omitempty"`

    // Components
    GitState
    SpecState
    ImplementState
    ReviewState
    PRState
    MetricsState

    // Error tracking
    Error string `json:"error,omitempty"`
}

// NewDevState creates a new dev workflow state
func NewDevState(flowID string) DevState {
    runID := fmt.Sprintf("%s-%s", time.Now().Format("2006-01-02"), flowID)

    return DevState{
        RunID:  runID,
        FlowID: flowID,
        MetricsState: MetricsState{
            StartTime: time.Now(),
        },
    }
}

// WithTicket adds ticket to state
func (s DevState) WithTicket(ticket *Ticket) DevState {
    s.TicketID = ticket.ID
    s.Ticket = ticket
    return s
}

// AddTokens updates token metrics
func (s *DevState) AddTokens(in, out int) {
    s.TotalTokensIn += in
    s.TotalTokensOut += out
    // Rough cost estimate ($3/1M in, $15/1M out for Opus)
    s.TotalCost += (float64(in) * 0.000003) + (float64(out) * 0.000015)
}

// FinalizeDuration sets total duration
func (s *DevState) FinalizeDuration() {
    s.TotalDuration = time.Since(s.StartTime)
}

// Validate checks if state has required fields for a node
func (s DevState) Validate(requirements ...string) error {
    for _, req := range requirements {
        switch req {
        case "ticket":
            if s.Ticket == nil {
                return fmt.Errorf("Ticket required")
            }
        case "worktree":
            if s.Worktree == "" {
                return fmt.Errorf("Worktree required")
            }
        case "spec":
            if s.Spec == "" {
                return fmt.Errorf("Spec required")
            }
        case "implementation":
            if s.Implementation == "" {
                return fmt.Errorf("Implementation required")
            }
        case "review":
            if s.Review == nil {
                return fmt.Errorf("Review required")
            }
        }
    }
    return nil
}

// Example node using state validation
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if err := state.Validate("ticket"); err != nil {
        return state, err
    }

    claude := ClaudeFromContext(ctx)
    result, err := claude.Run(ctx, formatSpecPrompt(state.Ticket))
    if err != nil {
        state.Error = err.Error()
        return state, err
    }

    state.Spec = result.Output
    state.SpecTokensIn = result.TokensIn
    state.SpecTokensOut = result.TokensOut
    state.SpecGeneratedAt = time.Now()
    state.AddTokens(result.TokensIn, result.TokensOut)

    return state, nil
}
```

### Custom State Example

```go
// Custom state for a specific workflow
type CodeReviewState struct {
    // My fields
    PRNumber int
    PRURL    string
    Diff     string

    // Embed what I need from devflow
    devflow.ReviewState
    devflow.MetricsState
}

// Custom node using devflow services
func ReviewPRNode(ctx flowgraph.Context, state CodeReviewState) (CodeReviewState, error) {
    claude := devflow.ClaudeFromContext(ctx)

    result, err := claude.Run(ctx,
        fmt.Sprintf("Review this diff:\n%s", state.Diff),
        devflow.WithSystemPrompt(reviewPrompt),
    )
    if err != nil {
        return state, err
    }

    review, _ := devflow.ParseJSON[devflow.ReviewResult](result.Output)

    state.Review = &review
    state.ReviewTokensIn = result.TokensIn
    state.ReviewTokensOut = result.TokensOut
    state.TotalTokensIn += result.TokensIn
    state.TotalTokensOut += result.TokensOut

    return state, nil
}
```

## References

- ADR-018: flowgraph Integration
- ADR-020: Error Handling
- flowgraph state documentation
