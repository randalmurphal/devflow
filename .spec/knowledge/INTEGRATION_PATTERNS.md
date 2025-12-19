# Integration Patterns

Patterns for integrating devflow with flowgraph and building complete development workflows.

---

## Using flowgraph.Graph with DevState

### Basic Setup

```go
import (
    "github.com/yourorg/flowgraph"
    "github.com/yourorg/devflow"
)

// DevState is your workflow state
// Embed standard components for common operations
type DevState struct {
    // Identity
    RunID    string `json:"run_id"`
    FlowID   string `json:"flow_id"`

    // Input
    TicketID string         `json:"ticket_id"`
    Ticket   *devflow.Ticket `json:"ticket,omitempty"`

    // Git state
    devflow.GitState

    // Workflow state
    devflow.SpecState
    devflow.ImplementState
    devflow.ReviewState
    devflow.PRState

    // Metrics
    devflow.MetricsState

    // Error (for failed state)
    Error string `json:"error,omitempty"`
}

// Create the graph
graph := flowgraph.NewGraph[DevState]().
    AddNode("create-worktree", devflow.CreateWorktreeNode).
    AddNode("generate-spec", devflow.GenerateSpecNode).
    AddNode("implement", devflow.ImplementNode).
    AddNode("run-tests", devflow.RunTestsNode).
    AddNode("check-lint", devflow.CheckLintNode).
    AddNode("review", devflow.ReviewNode).
    AddNode("fix-findings", devflow.FixFindingsNode).
    AddNode("create-pr", devflow.CreatePRNode).
    AddNode("cleanup", devflow.CleanupNode)
```

### State Composition

Use embedded structs for clean composition:

```go
// GitState holds worktree/branch info
type GitState struct {
    Worktree string `json:"worktree,omitempty"`
    Branch   string `json:"branch,omitempty"`
}

// SpecState holds specification data
type SpecState struct {
    Spec         *Spec `json:"spec,omitempty"`
    SpecTokensIn  int  `json:"spec_tokens_in"`
    SpecTokensOut int  `json:"spec_tokens_out"`
}

// ImplementState holds implementation data
type ImplementState struct {
    Implementation     *Implementation `json:"implementation,omitempty"`
    Files              []FileChange    `json:"files,omitempty"`
    ImplementTokensIn  int             `json:"implement_tokens_in"`
    ImplementTokensOut int             `json:"implement_tokens_out"`
}

// ReviewState holds review data
type ReviewState struct {
    Review          *Review `json:"review,omitempty"`
    ReviewAttempts  int     `json:"review_attempts"`
    ReviewTokensIn  int     `json:"review_tokens_in"`
    ReviewTokensOut int     `json:"review_tokens_out"`
}

// PRState holds PR data
type PRState struct {
    PR *PullRequest `json:"pr,omitempty"`
}

// MetricsState aggregates token usage
type MetricsState struct {
    TotalTokensIn  int           `json:"total_tokens_in"`
    TotalTokensOut int           `json:"total_tokens_out"`
    TotalDuration  time.Duration `json:"total_duration"`
}
```

### State Builders

Use builder pattern for cleaner initialization:

```go
func NewDevState(flowID string) *DevState {
    return &DevState{
        RunID:  generateRunID(flowID),
        FlowID: flowID,
    }
}

func (s *DevState) WithTicket(t *Ticket) *DevState {
    s.TicketID = t.ID
    s.Ticket = t
    return s
}

func (s *DevState) WithBranch(branch string) *DevState {
    s.Branch = branch
    return s
}
```

---

## Passing GitContext and ClaudeCLI Through Context

### Context Injection

Services are passed via context to keep node signatures clean:

```go
// Setup services
git, err := devflow.NewGitContext("/path/to/repo",
    devflow.WithGitHub(os.Getenv("GITHUB_TOKEN")),
)
if err != nil {
    log.Fatal(err)
}

claude, err := devflow.NewClaudeCLI(devflow.ClaudeConfig{
    Timeout:  10 * time.Minute,
    MaxTurns: 20,
})
if err != nil {
    log.Fatal(err)
}

transcripts, err := devflow.NewFileTranscriptStore(".devflow/runs")
if err != nil {
    log.Fatal(err)
}

artifacts, err := devflow.NewArtifactManager(devflow.ArtifactConfig{
    BaseDir: ".devflow/runs",
})
if err != nil {
    log.Fatal(err)
}

// Inject into context
ctx := context.Background()
ctx = devflow.WithGitContext(ctx, git)
ctx = devflow.WithClaudeCLI(ctx, claude)
ctx = devflow.WithTranscriptStore(ctx, transcripts)
ctx = devflow.WithArtifactManager(ctx, artifacts)

// Run workflow
result, err := compiled.Run(ctx, initialState)
```

### Context Keys

```go
type contextKey string

const (
    gitContextKey        contextKey = "devflow.git"
    claudeCLIKey         contextKey = "devflow.claude"
    transcriptStoreKey   contextKey = "devflow.transcripts"
    artifactManagerKey   contextKey = "devflow.artifacts"
)

func WithGitContext(ctx context.Context, g *GitContext) context.Context {
    return context.WithValue(ctx, gitContextKey, g)
}

func GitContextFromContext(ctx context.Context) *GitContext {
    if v := ctx.Value(gitContextKey); v != nil {
        return v.(*GitContext)
    }
    return nil
}
```

### Using in Nodes

```go
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // Extract services from context
    claude := devflow.ClaudeCLIFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("ClaudeCLI not found in context")
    }

    transcripts := devflow.TranscriptStoreFromContext(ctx)
    if transcripts != nil {
        // Record this interaction
        transcripts.RecordTurn(state.RunID, devflow.Turn{
            Role:    "system",
            Content: "Generating specification...",
        })
    }

    // Use claude...
    result, err := claude.Run(ctx, prompt)

    // ...
}
```

---

## Checkpointing Dev Workflows

### Enable Checkpointing

```go
// Create checkpoint store
checkpoints, err := devflow.NewSQLiteCheckpointStore(".devflow/checkpoints.db")
if err != nil {
    log.Fatal(err)
}

// Compile with checkpointing
compiled, err := graph.Compile(
    flowgraph.WithCheckpointStore(checkpoints),
    flowgraph.WithCheckpointFrequency(flowgraph.AfterEachNode),
)
```

### Checkpoint in Nodes

```go
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // ... implementation ...

    // Explicit checkpoint after expensive operation
    ctx.Checkpoint("post-implementation", state)

    return state, nil
}
```

### Checkpoint Naming

Use consistent checkpoint names:

| Checkpoint | After |
|------------|-------|
| `worktree-created` | CreateWorktreeNode |
| `spec-generated` | GenerateSpecNode |
| `implemented` | ImplementNode |
| `tests-run` | RunTestsNode |
| `lint-checked` | CheckLintNode |
| `reviewed` | ReviewNode |
| `findings-fixed` | FixFindingsNode |
| `pr-created` | CreatePRNode |
| `cleaned-up` | CleanupNode |

---

## Resume After Crash

### Resuming a Run

```go
// Check for incomplete runs
runs, err := checkpoints.ListIncomplete()
if err != nil {
    log.Fatal(err)
}

if len(runs) > 0 {
    fmt.Printf("Found %d incomplete runs:\n", len(runs))
    for _, run := range runs {
        fmt.Printf("  %s: stopped at %s (%s)\n",
            run.RunID, run.LastNode, run.StoppedAt)
    }

    // Resume specific run
    runID := runs[0].RunID

    result, err := compiled.Resume(ctx, runID)
    if err != nil {
        log.Fatalf("Resume failed: %v", err)
    }

    fmt.Printf("Resumed and completed: %+v\n", result)
}
```

### Resume Logic

```go
func (g *CompiledGraph[S]) Resume(ctx context.Context, runID string) (S, error) {
    // Load last checkpoint
    checkpoint, err := g.checkpoints.LoadLatest(runID)
    if err != nil {
        return *new(S), fmt.Errorf("load checkpoint: %w", err)
    }

    // Deserialize state
    var state S
    if err := json.Unmarshal(checkpoint.State, &state); err != nil {
        return state, fmt.Errorf("deserialize state: %w", err)
    }

    // Find next node
    nextNode, err := g.findNextNode(checkpoint.NodeID)
    if err != nil {
        return state, fmt.Errorf("find next node: %w", err)
    }

    // Continue execution from next node
    return g.runFrom(ctx, state, nextNode)
}
```

### Handling Partial State

```go
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // Check if we have partial progress
    if state.Implementation != nil && len(state.Files) > 0 {
        // We crashed mid-implementation, continue from partial
        log.Printf("Resuming from partial implementation (%d files done)",
            len(state.Files))
    }

    // ... continue implementation ...
}
```

---

## Composing Pre-built and Custom Nodes

### Mixing Node Types

```go
// Custom validation node
func ValidateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if state.Spec == nil {
        return state, fmt.Errorf("no spec to validate")
    }

    // Custom validation logic
    issues := validateSpec(state.Spec)
    if len(issues) > 0 {
        state.SpecValidation = &SpecValidation{
            Valid:  false,
            Issues: issues,
        }
        return state, fmt.Errorf("spec validation failed: %d issues", len(issues))
    }

    state.SpecValidation = &SpecValidation{Valid: true}
    return state, nil
}

// Compose with pre-built nodes
graph := flowgraph.NewGraph[DevState]().
    AddNode("create-worktree", devflow.CreateWorktreeNode).
    AddNode("generate-spec", devflow.GenerateSpecNode).
    AddNode("validate-spec", ValidateSpecNode).          // Custom
    AddNode("implement", devflow.ImplementNode).
    AddNode("custom-tests", runCustomTests).              // Custom
    AddNode("review", devflow.ReviewNode).
    AddNode("create-pr", devflow.CreatePRNode).
    AddEdge("create-worktree", "generate-spec").
    AddEdge("generate-spec", "validate-spec").
    AddEdge("validate-spec", "implement").
    // ...
```

### Node Wrappers

Enhance pre-built nodes with additional behavior:

```go
// Add retry logic
func WithRetry[S any](
    node flowgraph.NodeFunc[S],
    config RetryConfig,
) flowgraph.NodeFunc[S] {
    return func(ctx flowgraph.Context, state S) (S, error) {
        var lastErr error
        for attempt := 0; attempt < config.MaxAttempts; attempt++ {
            result, err := node(ctx, state)
            if err == nil {
                return result, nil
            }
            lastErr = err

            if !isRetryable(err, config.RetryOn) {
                return result, err
            }

            time.Sleep(config.Backoff * time.Duration(attempt+1))
        }
        return state, fmt.Errorf("max retries exceeded: %w", lastErr)
    }
}

// Usage
graph.AddNode("implement", WithRetry(devflow.ImplementNode, RetryConfig{
    MaxAttempts: 3,
    Backoff:     time.Second,
    RetryOn:     []error{devflow.ErrTimeout, devflow.ErrRateLimit},
}))
```

### Transcript Recording Wrapper

```go
func WithTranscript[S any](node flowgraph.NodeFunc[S]) flowgraph.NodeFunc[S] {
    return func(ctx flowgraph.Context, state S) (S, error) {
        store := devflow.TranscriptStoreFromContext(ctx)

        // Get run ID from state (assumes state has RunID field)
        runID := getRunID(state)

        if store != nil {
            store.RecordTurn(runID, devflow.Turn{
                Role:      "system",
                Content:   fmt.Sprintf("Starting node execution"),
                Timestamp: time.Now(),
            })
        }

        result, err := node(ctx, state)

        if store != nil {
            status := "completed"
            if err != nil {
                status = fmt.Sprintf("failed: %v", err)
            }
            store.RecordTurn(runID, devflow.Turn{
                Role:      "system",
                Content:   fmt.Sprintf("Node %s", status),
                Timestamp: time.Now(),
            })
        }

        return result, err
    }
}
```

---

## Complete Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    "github.com/yourorg/flowgraph"
    "github.com/yourorg/devflow"
)

func main() {
    // Initialize services
    git, _ := devflow.NewGitContext(".",
        devflow.WithGitHub(os.Getenv("GITHUB_TOKEN")),
    )
    claude, _ := devflow.NewClaudeCLI(devflow.ClaudeConfig{
        Timeout:  10 * time.Minute,
        MaxTurns: 20,
    })
    transcripts, _ := devflow.NewFileTranscriptStore(".devflow/runs")
    checkpoints, _ := devflow.NewSQLiteCheckpointStore(".devflow/checkpoints.db")

    // Build graph
    graph := flowgraph.NewGraph[devflow.DevState]().
        AddNode("create-worktree", devflow.CreateWorktreeNode).
        AddNode("generate-spec", devflow.WithTranscript(devflow.GenerateSpecNode)).
        AddNode("implement", devflow.WithRetry(devflow.ImplementNode, devflow.RetryConfig{
            MaxAttempts: 3,
        })).
        AddNode("run-tests", devflow.RunTestsNode).
        AddNode("check-lint", devflow.CheckLintNode).
        AddNode("review", devflow.ReviewNode).
        AddNode("fix-findings", devflow.FixFindingsNode).
        AddNode("create-pr", devflow.CreatePRNode).
        AddNode("cleanup", devflow.CleanupNode).
        AddEdge("create-worktree", "generate-spec").
        AddEdge("generate-spec", "implement").
        AddEdge("implement", "run-tests").
        AddEdge("run-tests", "check-lint").
        AddEdge("check-lint", "review").
        AddConditionalEdge("review", devflow.ReviewRouter).
        AddEdge("fix-findings", "run-tests").
        AddEdge("create-pr", "cleanup").
        AddEdge("cleanup", flowgraph.END).
        SetEntry("create-worktree")

    // Compile with checkpointing
    compiled, err := graph.Compile(
        flowgraph.WithCheckpointStore(checkpoints),
    )
    if err != nil {
        log.Fatalf("Compile failed: %v", err)
    }

    // Setup context with services
    ctx := context.Background()
    ctx = devflow.WithGitContext(ctx, git)
    ctx = devflow.WithClaudeCLI(ctx, claude)
    ctx = devflow.WithTranscriptStore(ctx, transcripts)

    // Check for incomplete runs
    incomplete, _ := checkpoints.ListIncomplete()
    if len(incomplete) > 0 {
        runID := incomplete[0].RunID
        fmt.Printf("Resuming run: %s\n", runID)

        result, err := compiled.Resume(ctx, runID)
        if err != nil {
            log.Fatalf("Resume failed: %v", err)
        }
        fmt.Printf("Completed: PR %s\n", result.PR.URL)
        return
    }

    // Start new run
    initial := devflow.NewDevState("ticket-to-pr").
        WithTicket(&devflow.Ticket{
            ID:          "TK-421",
            Title:       "Add user authentication",
            Description: "Implement OAuth2 login with Google and GitHub",
        })

    // Start transcript
    transcripts.StartRun(initial.RunID, devflow.RunMetadata{
        FlowID:    initial.FlowID,
        TicketID:  initial.TicketID,
        StartedAt: time.Now(),
    })

    // Run workflow
    result, err := compiled.Run(ctx, *initial)

    // End transcript
    status := devflow.RunStatusCompleted
    if err != nil {
        status = devflow.RunStatusFailed
    }
    transcripts.EndRun(initial.RunID, status)

    if err != nil {
        log.Fatalf("Workflow failed: %v", err)
    }

    fmt.Printf("Created PR: %s\n", result.PR.URL)
    fmt.Printf("Total tokens: %d in, %d out\n",
        result.TotalTokensIn, result.TotalTokensOut)
}
```

---

## Best Practices

### 1. Always Inject Services via Context

Don't pass services through state - it makes serialization harder:

```go
// Bad
type DevState struct {
    Claude *ClaudeCLI  // Can't serialize
}

// Good
ctx = devflow.WithClaudeCLI(ctx, claude)
```

### 2. Keep State Serializable

State must be JSON-serializable for checkpointing:

```go
// Bad
type DevState struct {
    Callback func()  // Can't serialize
}

// Good
type DevState struct {
    CallbackName string  // Reference by name
}
```

### 3. Use Explicit Checkpoints for Expensive Operations

```go
func ExpensiveNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // Before expensive operation
    result, err := expensiveOperation()
    if err != nil {
        return state, err
    }

    state.Result = result

    // Checkpoint immediately after success
    ctx.Checkpoint("expensive-done", state)

    return state, nil
}
```

### 4. Handle Partial Progress in Nodes

```go
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // Resuming from partial?
    startFrom := 0
    if len(state.Files) > 0 {
        startFrom = len(state.Files)
        log.Printf("Resuming from file %d", startFrom)
    }

    for i := startFrom; i < len(state.Spec.Files); i++ {
        // Implement each file...

        // Checkpoint after each file for long implementations
        if i > 0 && i%5 == 0 {
            ctx.Checkpoint(fmt.Sprintf("implemented-%d", i), state)
        }
    }

    return state, nil
}
```

### 5. Use Conditional Edges for Branching

```go
func ReviewRouter(ctx flowgraph.Context, state DevState) (string, error) {
    if state.Review.Approved {
        return "create-pr", nil
    }

    if state.ReviewAttempts >= 3 {
        return "escalate", nil  // Human review
    }

    return "fix-findings", nil
}

graph.AddConditionalEdge("review", ReviewRouter)
```

---

## References

- ADR-018: flowgraph Integration
- ADR-019: State Design
- ADR-020: Error Handling
- Phase 5: Workflow Nodes
- Feature: Dev Workflow Nodes
