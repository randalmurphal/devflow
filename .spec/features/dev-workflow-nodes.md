# Feature: Dev Workflow Nodes

## Overview

Pre-built nodes for common development workflow operations, compatible with flowgraph.

## Use Cases

1. **Ticket-to-PR automation**: Complete workflow from ticket to pull request
2. **Code review automation**: Automated code review with findings
3. **Custom workflows**: Mix pre-built and custom nodes

## Node Signature

All nodes match flowgraph's `NodeFunc[S]`:

```go
type NodeFunc[S any] func(ctx flowgraph.Context, state S) (S, error)
```

## Available Nodes

### CreateWorktreeNode

Creates isolated git worktree for the workflow.

```go
func CreateWorktreeNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.TicketID` or `state.Branch`
**Updates**: `state.Worktree`, `state.Branch`

### GenerateSpecNode

Generates technical specification from ticket.

```go
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.Ticket`
**Updates**: `state.Spec`, `state.SpecTokensIn/Out`

### ImplementNode

Implements code based on specification.

```go
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.Spec`, `state.Worktree`
**Updates**: `state.Implementation`, `state.Files`, `state.ImplementTokensIn/Out`

### ReviewNode

Reviews implementation for issues.

```go
func ReviewNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.Spec`, `state.Worktree`
**Updates**: `state.Review`, `state.ReviewAttempts`, `state.ReviewTokensIn/Out`

### FixFindingsNode

Fixes issues found in review.

```go
func FixFindingsNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.Review.Findings`, `state.Worktree`
**Updates**: `state.Implementation`, `state.Files`

### CreatePRNode

Creates pull request from changes.

```go
func CreatePRNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.Branch` (pushed), PR provider configured
**Updates**: `state.PR`

### CleanupNode

Cleans up worktree after workflow.

```go
func CleanupNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.Worktree`
**Updates**: `state.Worktree` (cleared)

### RunTestsNode

Runs test suite.

```go
func RunTestsNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.Worktree`
**Updates**: `state.TestOutput`

### CheckLintNode

Runs linting and type checks.

```go
func CheckLintNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Requires**: `state.Worktree`
**Updates**: `state.LintOutput`

## DevState

Standard state type embedding all components:

```go
type DevState struct {
    RunID    string
    FlowID   string
    TicketID string
    Ticket   *Ticket

    GitState        // Worktree, Branch
    SpecState       // Spec, tokens
    ImplementState  // Implementation, Files, tokens
    ReviewState     // Review, attempts, tokens
    PRState         // PR
    MetricsState    // Total tokens, cost

    Error string
}
```

## Context Injection

Services are injected via context:

```go
ctx := context.Background()
ctx = devflow.WithGitContext(ctx, git)
ctx = devflow.WithClaudeCLI(ctx, claude)
ctx = devflow.WithTranscriptStore(ctx, store)

result, err := graph.Run(ctx, initialState)
```

## Pre-built Graphs

### TicketToPRGraph

```go
graph := devflow.TicketToPRGraph()
compiled, _ := graph.Compile()

result, err := compiled.Run(ctx, devflow.NewDevState("ticket-to-pr").
    WithTicket(ticket))
```

Flow:
```
create-worktree → generate-spec → implement → run-tests → check-lint → review
                                                                         ↓
                                  fix-findings ← (if not approved) ←──────┘
                                       ↓
                                   run-tests (loop back)

                                  create-pr ← (if approved)
                                       ↓
                                   cleanup → END
```

## Example

```go
// Setup services
git, _ := devflow.NewGitContext("/path/to/repo",
    devflow.WithGitHub(os.Getenv("GITHUB_TOKEN")),
)
claude, _ := devflow.NewClaudeCLI(devflow.ClaudeConfig{
    Timeout: 10 * time.Minute,
})
store, _ := devflow.NewFileTranscriptStore(".devflow")

// Setup context
ctx := context.Background()
ctx = devflow.WithGitContext(ctx, git)
ctx = devflow.WithClaudeCLI(ctx, claude)
ctx = devflow.WithTranscriptStore(ctx, store)

// Use pre-built graph
graph := devflow.TicketToPRGraph()
compiled, _ := graph.Compile()

// Run workflow
initial := devflow.NewDevState("ticket-to-pr").WithTicket(&devflow.Ticket{
    ID:          "TK-421",
    Title:       "Add user authentication",
    Description: "Implement OAuth2 login with Google and GitHub providers",
})

result, err := compiled.Run(ctx, initial)
if err != nil {
    log.Fatalf("Workflow failed: %v", err)
}

fmt.Printf("Created PR: %s\n", result.PR.URL)
fmt.Printf("Total tokens: %d in, %d out\n", result.TotalTokensIn, result.TotalTokensOut)
```

## Custom Workflow

Mix pre-built and custom nodes:

```go
graph := flowgraph.NewGraph[devflow.DevState]().
    AddNode("setup", mySetupNode).
    AddNode("generate-spec", devflow.GenerateSpecNode).
    AddNode("implement", devflow.ImplementNode).
    AddNode("my-validation", myValidationNode).
    AddNode("review", devflow.ReviewNode).
    AddNode("create-pr", devflow.CreatePRNode).
    // ... edges
```

## Node Wrappers

Enhance nodes with additional behavior:

```go
// Retry on timeout
implementWithRetry := devflow.WithRetry(
    devflow.ImplementNode,
    devflow.RetryConfig{MaxAttempts: 3},
)

// Record to transcript
implementWithTranscript := devflow.WithTranscript(devflow.ImplementNode)
```

## Testing

```go
func TestTicketToPRWorkflow(t *testing.T) {
    // Setup mocks
    mockGit := &MockGitContext{...}
    mockClaude := &MockClaudeCLI{
        RunFunc: func(...) (*RunResult, error) {
            return &RunResult{Output: "Generated..."}, nil
        },
    }

    ctx := context.Background()
    ctx = devflow.WithGitContext(ctx, mockGit)
    ctx = devflow.WithClaudeCLI(ctx, mockClaude)

    graph := devflow.TicketToPRGraph()
    compiled, _ := graph.Compile()

    result, err := compiled.Run(ctx, devflow.NewDevState("test").
        WithTicket(&devflow.Ticket{ID: "TEST-1", Title: "Test"}))

    require.NoError(t, err)
    assert.NotNil(t, result.PR)
}
```

## References

- ADR-018: flowgraph Integration
- ADR-019: State Design
- ADR-020: Error Handling
- Phase 5: Workflow Nodes
