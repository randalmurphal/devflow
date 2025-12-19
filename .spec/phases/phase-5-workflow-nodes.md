# Phase 5: Dev Workflow Nodes

## Overview

Implement pre-built nodes for common development workflow operations.

**Duration**: Week 5
**Dependencies**: Phases 1-4, flowgraph
**Deliverables**: Pre-built nodes and composed workflows

---

## Goals

1. Create nodes that match flowgraph's `NodeFunc[S]` signature
2. Implement common dev workflow operations
3. Provide composed graphs for complete workflows
4. Support context injection for devflow services

---

## Components

### Context Keys

```go
const (
    gitContextKey        contextKey = "devflow.git"
    claudeContextKey     contextKey = "devflow.claude"
    transcriptContextKey contextKey = "devflow.transcripts"
    artifactContextKey   contextKey = "devflow.artifacts"
    promptLoaderKey      contextKey = "devflow.prompts"
)
```

### DevState

Standard state type:

```go
type DevState struct {
    RunID      string
    FlowID     string
    TicketID   string
    Ticket     *Ticket

    GitState
    SpecState
    ImplementState
    ReviewState
    PRState
    MetricsState

    Error string
}
```

---

## Nodes to Implement

### Node 1: CreateWorktreeNode

Creates isolated git worktree.

```go
func CreateWorktreeNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.TicketID` or `state.Branch` set

**Output Changes**:
- `state.Worktree` = worktree path
- `state.Branch` = branch name

**Errors**:
- GitContext not in context
- Worktree creation failed

### Node 2: GenerateSpecNode

Generates technical specification from ticket.

```go
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.Ticket` set

**Output Changes**:
- `state.Spec` = generated specification
- `state.SpecTokensIn/Out` = token usage

**Prompt**: `prompts/generate-spec.txt`

### Node 3: ImplementNode

Implements code based on specification.

```go
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.Spec` set
- `state.Worktree` set

**Output Changes**:
- `state.Implementation` = Claude output
- `state.Files` = created/modified files
- `state.ImplementTokensIn/Out` = token usage

**Prompt**: `prompts/implement.txt`

### Node 4: ReviewNode

Reviews implementation for issues.

```go
func ReviewNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.Spec` set
- `state.Implementation` or files in worktree

**Output Changes**:
- `state.Review` = ReviewResult with findings
- `state.ReviewAttempts` incremented
- `state.ReviewTokensIn/Out` = token usage

**Prompt**: `prompts/review-code.txt`

### Node 5: FixFindingsNode

Fixes issues found in review.

```go
func FixFindingsNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.Review` with findings
- `state.Worktree` set

**Output Changes**:
- `state.Implementation` updated
- `state.Files` updated

**Prompt**: `prompts/fix-findings.txt`

### Node 6: RunTestsNode

Runs test suite.

```go
func RunTestsNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.Worktree` set

**Output Changes**:
- `state.TestOutput` = test results

**Implementation**: Runs `go test` or detected test command

### Node 7: CheckLintNode

Runs linting and type checks.

```go
func CheckLintNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.Worktree` set

**Output Changes**:
- `state.LintOutput` = lint results

**Implementation**: Runs configured lint command

### Node 8: CreatePRNode

Creates pull request.

```go
func CreatePRNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.Branch` set and pushed
- GitContext with PR provider

**Output Changes**:
- `state.PR` = created PullRequest

### Node 9: CleanupNode

Cleans up worktree.

```go
func CleanupNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

**Input Requirements**:
- `state.Worktree` set

**Output Changes**:
- `state.Worktree` = "" (cleared)

---

## Composed Graphs

### TicketToPRGraph

Complete ticket-to-PR workflow:

```go
func TicketToPRGraph() *flowgraph.Graph[DevState] {
    return flowgraph.NewGraph[DevState]().
        AddNode("create-worktree", CreateWorktreeNode).
        AddNode("generate-spec", GenerateSpecNode).
        AddNode("implement", ImplementNode).
        AddNode("run-tests", RunTestsNode).
        AddNode("check-lint", CheckLintNode).
        AddNode("review", ReviewNode).
        AddNode("fix-findings", FixFindingsNode).
        AddNode("create-pr", CreatePRNode).
        AddNode("cleanup", CleanupNode).
        // Edges
        AddEdge("create-worktree", "generate-spec").
        AddEdge("generate-spec", "implement").
        AddEdge("implement", "run-tests").
        AddEdge("run-tests", "check-lint").
        AddEdge("check-lint", "review").
        AddConditionalEdge("review", reviewRouter).
        AddEdge("fix-findings", "run-tests").
        AddEdge("create-pr", "cleanup").
        AddEdge("cleanup", flowgraph.END).
        SetEntry("create-worktree")
}

func reviewRouter(s DevState) string {
    if s.Review != nil && s.Review.Approved {
        return "create-pr"
    }
    if s.ReviewAttempts >= 3 {
        return "create-pr" // Give up, create as draft
    }
    return "fix-findings"
}
```

### CodeReviewGraph

PR review workflow:

```go
func CodeReviewGraph() *flowgraph.Graph[ReviewState]
```

### RefactoringGraph

Refactoring workflow:

```go
func RefactoringGraph() *flowgraph.Graph[RefactorState]
```

---

## Implementation Tasks

### Task 5.1: Context Helpers

```go
func WithGitContext(ctx context.Context, git *GitContext) context.Context
func GitFromContext(ctx context.Context) *GitContext
// ... for all services
```

### Task 5.2: State Types

```go
type DevState struct { ... }
func NewDevState(flowID string) DevState
func (s DevState) Validate(requirements ...string) error
```

### Task 5.3: Individual Nodes

Implement each node with:
- Prerequisite validation
- Service extraction from context
- Error handling with context
- State updates

### Task 5.4: Composed Graphs

Create pre-built graphs with:
- Conditional edges for review loops
- Error handling edges
- Cleanup always runs

### Task 5.5: Node Wrappers

```go
// Add retry capability
func WithRetry(node NodeFunc, cfg RetryConfig) NodeFunc

// Add transcript recording
func WithTranscript(node NodeFunc) NodeFunc

// Add timing metrics
func WithTiming(node NodeFunc) NodeFunc
```

---

## Testing Strategy

### Unit Tests

| Test | Description |
|------|-------------|
| `TestCreateWorktreeNode` | Creates worktree |
| `TestGenerateSpecNode` | Calls Claude, updates state |
| `TestReviewRouter` | Routes correctly |
| `TestDevState_Validate` | Validates requirements |

### Integration Tests

```go
func TestTicketToPRGraph_Integration(t *testing.T) {
    // Setup mock services
    git := &MockGitContext{}
    claude := &MockClaudeCLI{
        RunFunc: func(...) (*RunResult, error) {
            return &RunResult{Output: "Generated..."}, nil
        },
    }

    ctx := context.Background()
    ctx = WithGitContext(ctx, git)
    ctx = WithClaudeCLI(ctx, claude)

    graph := TicketToPRGraph()
    compiled, _ := graph.Compile()

    initial := NewDevState("test").WithTicket(&Ticket{
        ID: "TK-421",
        Title: "Add auth",
    })

    result, err := compiled.Run(ctx, initial)
    require.NoError(t, err)
    assert.NotNil(t, result.PR)
}
```

---

## File Structure

```
devflow/
├── context.go          # Context helpers
├── state.go            # DevState and components
├── nodes.go            # All node implementations
├── graphs.go           # Pre-built graphs
├── nodes_test.go       # Node tests
└── graphs_test.go      # Graph tests
```

---

## Completion Criteria

- [ ] All nodes implemented
- [ ] All graphs composed
- [ ] Wrappers working (retry, transcript)
- [ ] Unit test coverage > 80%
- [ ] Integration tests with mocks
- [ ] Documentation complete

---

## References

- ADR-018: flowgraph Integration
- ADR-019: State Design
- ADR-020: Error Handling
