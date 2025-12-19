# ADR-018: flowgraph Integration

## Status

Accepted

## Context

devflow builds on flowgraph for graph-based orchestration. We need to decide:

1. How devflow types integrate with flowgraph
2. How to pass devflow services (git, Claude) to nodes
3. How to compose devflow nodes into graphs
4. What devflow provides vs what flowgraph provides

## Decision

### 1. Clear Layer Separation

```
┌─────────────────────────────────────────────────────┐
│                    User Code                         │
│  graph := flowgraph.NewGraph[MyState]()...          │
└────────────────────────┬────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────┐
│                     devflow                          │
│  • GitContext, ClaudeCLI, TranscriptManager         │
│  • Pre-built nodes (GenerateSpecNode, etc.)         │
│  • Artifact management                              │
└────────────────────────┬────────────────────────────┘
                         │
┌────────────────────────▼────────────────────────────┐
│                    flowgraph                         │
│  • Graph[S] definition                              │
│  • Node execution                                   │
│  • Checkpointing                                    │
│  • Error handling                                   │
└─────────────────────────────────────────────────────┘
```

### 2. Services via Context

devflow services are passed through flowgraph's context:

```go
// Setup services
git, _ := devflow.NewGitContext(repoPath)
claude, _ := devflow.NewClaudeCLI(devflow.ClaudeConfig{...})
transcripts, _ := devflow.NewFileTranscriptStore(baseDir)

// Inject into context
ctx := context.Background()
ctx = devflow.WithGitContext(ctx, git)
ctx = devflow.WithClaudeCLI(ctx, claude)
ctx = devflow.WithTranscriptStore(ctx, transcripts)

// Run graph
result, err := graph.Run(ctx, initialState)
```

### 3. Nodes as Functions

devflow nodes match flowgraph's `NodeFunc[S]` signature:

```go
type NodeFunc[S any] func(ctx flowgraph.Context, state S) (S, error)

// devflow node example
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    claude := devflow.ClaudeFromContext(ctx)
    // ...
    return state, nil
}
```

### 4. State Embedding

Users embed devflow types in their state:

```go
type MyWorkflowState struct {
    // User fields
    TicketID string
    Title    string

    // devflow fields
    devflow.GitState         // Worktree, branch info
    devflow.SpecState        // Generated spec
    devflow.ImplementState   // Implementation result
    devflow.ReviewState      // Review result
}
```

### 5. Pre-built Graphs

devflow provides complete graphs for common workflows:

```go
// Pre-built ticket-to-PR workflow
graph := devflow.TicketToPRGraph()

// Or compose your own
graph := flowgraph.NewGraph[DevState]().
    AddNode("generate-spec", devflow.GenerateSpecNode).
    AddNode("implement", devflow.ImplementNode).
    AddNode("review", devflow.ReviewNode).
    // ... edges
```

### 6. Checkpointing Integration

devflow artifacts are saved alongside flowgraph checkpoints:

```
.devflow/runs/run-123/
├── metadata.json          # devflow metadata
├── transcript.json        # devflow transcript
├── artifacts/             # devflow artifacts
└── state-checkpoints/     # flowgraph checkpoints
    ├── generate-spec.json
    └── implement.json
```

## Alternatives Considered

### Alternative 1: Wrap flowgraph

Create a devflow-specific graph type that wraps flowgraph.

**Rejected because:**
- Duplicates functionality
- Harder to compose with raw flowgraph
- More API surface

### Alternative 2: No flowgraph Dependency

Implement own graph execution.

**Rejected because:**
- Reinventing the wheel
- Checkpointing is complex
- flowgraph exists for this

### Alternative 3: Plugin Architecture

Make devflow nodes optional plugins.

**Rejected because:**
- Adds complexity
- All nodes are useful
- Simple import is sufficient

## Consequences

### Positive

- **Clean separation**: Each layer has clear responsibility
- **Composable**: Mix devflow and custom nodes
- **Type-safe**: Generic state types
- **Reusable**: Pre-built graphs for common patterns

### Negative

- **Learning curve**: Users need to understand both layers
- **Context ceremony**: Services must be injected
- **Type constraints**: State must satisfy certain patterns

## Code Example

```go
package devflow

import (
    "context"

    "github.com/yourorg/flowgraph"
)

// Context keys for devflow services
type contextKey string

const (
    gitContextKey        contextKey = "devflow.git"
    claudeContextKey     contextKey = "devflow.claude"
    transcriptContextKey contextKey = "devflow.transcripts"
    artifactContextKey   contextKey = "devflow.artifacts"
)

// Context injection functions
func WithGitContext(ctx context.Context, git *GitContext) context.Context {
    return context.WithValue(ctx, gitContextKey, git)
}

func GitFromContext(ctx context.Context) *GitContext {
    if v := ctx.Value(gitContextKey); v != nil {
        return v.(*GitContext)
    }
    return nil
}

func WithClaudeCLI(ctx context.Context, claude *ClaudeCLI) context.Context {
    return context.WithValue(ctx, claudeContextKey, claude)
}

func ClaudeFromContext(ctx context.Context) *ClaudeCLI {
    if v := ctx.Value(claudeContextKey); v != nil {
        return v.(*ClaudeCLI)
    }
    return nil
}

func WithTranscriptStore(ctx context.Context, store TranscriptManager) context.Context {
    return context.WithValue(ctx, transcriptContextKey, store)
}

func TranscriptsFromContext(ctx context.Context) TranscriptManager {
    if v := ctx.Value(transcriptContextKey); v != nil {
        return v.(TranscriptManager)
    }
    return nil
}

// Embeddable state types
type GitState struct {
    Worktree string `json:"worktree,omitempty"`
    Branch   string `json:"branch,omitempty"`
}

type SpecState struct {
    Spec       string `json:"spec,omitempty"`
    SpecTokens int    `json:"specTokens,omitempty"`
}

type ImplementState struct {
    Implementation string       `json:"implementation,omitempty"`
    Files          []FileChange `json:"files,omitempty"`
    ImplementTokens int         `json:"implementTokens,omitempty"`
}

type ReviewState struct {
    Review        *ReviewResult `json:"review,omitempty"`
    ReviewTokens  int           `json:"reviewTokens,omitempty"`
}

// Pre-built nodes

// CreateWorktreeNode creates a git worktree
func CreateWorktreeNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    git := GitFromContext(ctx)
    if git == nil {
        return state, fmt.Errorf("GitContext not in context")
    }

    branch := fmt.Sprintf("feature/%s", state.TicketID)
    worktree, err := git.CreateWorktree(branch)
    if err != nil {
        return state, fmt.Errorf("create worktree: %w", err)
    }

    state.Worktree = worktree
    state.Branch = branch
    return state, nil
}

// GenerateSpecNode generates a specification
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    claude := ClaudeFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("ClaudeCLI not in context")
    }

    result, err := claude.Run(ctx,
        formatSpecPrompt(state.Ticket),
        WithSystemPrompt(specSystemPrompt),
        WithWorkDir(state.Worktree),
    )
    if err != nil {
        return state, fmt.Errorf("generate spec: %w", err)
    }

    state.Spec = result.Output
    state.SpecTokens = result.TokensIn + result.TokensOut
    return state, nil
}

// ImplementNode implements code from spec
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    claude := ClaudeFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("ClaudeCLI not in context")
    }

    result, err := claude.Run(ctx,
        formatImplementPrompt(state.Spec),
        WithSystemPrompt(implementSystemPrompt),
        WithWorkDir(state.Worktree),
        WithMaxTurns(30),
    )
    if err != nil {
        return state, fmt.Errorf("implement: %w", err)
    }

    state.Implementation = result.Output
    state.Files = result.Files
    state.ImplementTokens = result.TokensIn + result.TokensOut
    return state, nil
}

// ReviewNode reviews implementation
func ReviewNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    claude := ClaudeFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("ClaudeCLI not in context")
    }

    result, err := claude.Run(ctx,
        formatReviewPrompt(state.Spec, state.Implementation),
        WithSystemPrompt(reviewSystemPrompt),
        WithWorkDir(state.Worktree),
    )
    if err != nil {
        return state, fmt.Errorf("review: %w", err)
    }

    review, err := ParseJSON[ReviewResult](result.Output)
    if err != nil {
        return state, fmt.Errorf("parse review: %w", err)
    }

    state.Review = &review
    state.ReviewTokens = result.TokensIn + result.TokensOut
    return state, nil
}

// Pre-built graph
func TicketToPRGraph() *flowgraph.Graph[DevState] {
    return flowgraph.NewGraph[DevState]().
        AddNode("create-worktree", CreateWorktreeNode).
        AddNode("generate-spec", GenerateSpecNode).
        AddNode("implement", ImplementNode).
        AddNode("review", ReviewNode).
        AddNode("fix-findings", FixFindingsNode).
        AddNode("create-pr", CreatePRNode).
        AddNode("cleanup", CleanupNode).
        AddEdge("create-worktree", "generate-spec").
        AddEdge("generate-spec", "implement").
        AddEdge("implement", "review").
        AddConditionalEdge("review", func(s DevState) string {
            if s.Review != nil && s.Review.Approved {
                return "create-pr"
            }
            return "fix-findings"
        }).
        AddEdge("fix-findings", "review").
        AddEdge("create-pr", "cleanup").
        AddEdge("cleanup", flowgraph.END).
        SetEntry("create-worktree")
}
```

### Usage

```go
// Setup services
git, _ := devflow.NewGitContext("/path/to/repo")
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

// Or build custom graph
graph := flowgraph.NewGraph[devflow.DevState]().
    AddNode("generate-spec", devflow.GenerateSpecNode).
    AddNode("implement", devflow.ImplementNode).
    AddNode("my-custom-node", myCustomNode).
    // ...

// Run
initial := devflow.DevState{
    TicketID: "TK-421",
    Ticket:   ticket,
}
result, err := compiled.Run(ctx, initial)
if err != nil {
    log.Fatal(err)
}

fmt.Printf("Created PR: %s\n", result.PR.URL)
```

## References

- flowgraph documentation
- ADR-019: State Design
- ADR-020: Error Handling
