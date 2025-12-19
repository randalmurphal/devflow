# Node: generate-spec

## Purpose

Generate a technical specification from a ticket/issue description. This is typically the first AI-powered step in a development workflow, transforming user requirements into actionable technical plans.

## Signature

```go
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

## Input State Requirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Ticket` | `*Ticket` | Yes | Ticket with ID, title, description |
| `Worktree` | `string` | Optional | Working directory for context |

### Ticket Structure

```go
type Ticket struct {
    ID          string            // TK-421
    Title       string            // Feature title
    Description string            // Full description
    Labels      []string          // Priority, type, etc.
    Metadata    map[string]any    // Custom fields
}
```

## Output State Changes

| Field | Type | Description |
|-------|------|-------------|
| `Spec` | `*Spec` | Generated specification |
| `SpecTokensIn` | `int` | Input tokens used |
| `SpecTokensOut` | `int` | Output tokens generated |

### Spec Structure

```go
type Spec struct {
    Overview      string      // High-level summary
    Requirements  []string    // Functional requirements
    Technical     string      // Technical approach
    Files         []FileSpec  // Files to create/modify
    TestPlan      string      // Testing approach
    Risks         []string    // Potential issues
    Raw           string      // Full markdown output
}

type FileSpec struct {
    Path        string   // Relative path
    Action      string   // create, modify, delete
    Description string   // What changes
}
```

## Prompt Template

Located at: `prompts/generate-spec.txt`

```
You are an expert software architect. Generate a technical specification for the following ticket.

## Ticket
ID: {{.Ticket.ID}}
Title: {{.Ticket.Title}}
Description:
{{.Ticket.Description}}

{{if .Worktree}}
## Project Context
Working in: {{.Worktree}}

Key files in project:
{{range .ContextFiles}}
- {{.}}
{{end}}
{{end}}

## Output Format

Provide a specification with these sections:

### Overview
Brief summary of what needs to be done.

### Requirements
- List each functional requirement
- Be specific and testable

### Technical Approach
How to implement this. Include:
- Architecture decisions
- Key algorithms or patterns
- Integration points

### Files to Change
For each file:
- Path: relative/path/to/file.go
- Action: create | modify | delete
- Changes: What to add/change

### Test Plan
How to verify this works:
- Unit tests needed
- Integration tests
- Manual testing steps

### Risks
Potential issues to watch for.
```

## Implementation

```go
func GenerateSpecNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if state.Ticket == nil {
        return state, fmt.Errorf("generate-spec: ticket is required")
    }

    // Get Claude CLI from context
    claude := devflow.ClaudeCLIFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("generate-spec: ClaudeCLI not in context")
    }

    // Load prompt template
    prompt, err := devflow.LoadPrompt("generate-spec", map[string]any{
        "Ticket":       state.Ticket,
        "Worktree":     state.Worktree,
        "ContextFiles": getProjectFiles(state.Worktree),
    })
    if err != nil {
        return state, fmt.Errorf("generate-spec: load prompt: %w", err)
    }

    // Options for Claude CLI
    opts := []devflow.RunOption{
        devflow.WithSystemPrompt(specSystemPrompt),
    }
    if state.Worktree != "" {
        opts = append(opts, devflow.WithWorkDir(state.Worktree))
    }

    // Run Claude
    result, err := claude.Run(ctx, prompt, opts...)
    if err != nil {
        return state, fmt.Errorf("generate-spec: claude: %w", err)
    }

    // Parse output
    spec, err := parseSpec(result.Output)
    if err != nil {
        return state, fmt.Errorf("generate-spec: parse: %w", err)
    }

    // Update state
    state.Spec = spec
    state.SpecTokensIn = result.TokensIn
    state.SpecTokensOut = result.TokensOut

    // Checkpoint
    ctx.Checkpoint("spec-generated", state)

    return state, nil
}
```

## Error Cases

| Error | Cause | Handling |
|-------|-------|----------|
| `ticket is required` | No ticket in state | Fail fast, programming error |
| `ClaudeCLI not in context` | Missing context injection | Fail, setup error |
| `load prompt` | Template not found | Fail, check prompt files |
| `claude: timeout` | Claude took too long | Retry with longer timeout |
| `claude: rate limit` | API rate limited | Wait and retry |
| `parse: invalid format` | Claude output malformed | Retry with clarification |

### Retry Strategy

```go
specWithRetry := devflow.WithRetry(GenerateSpecNode, devflow.RetryConfig{
    MaxAttempts: 3,
    Backoff:     time.Second,
    RetryOn: []error{
        devflow.ErrTimeout,
        devflow.ErrRateLimit,
    },
})
```

## Test Cases

### Unit Tests

```go
func TestGenerateSpecNode(t *testing.T) {
    tests := []struct {
        name        string
        state       DevState
        claudeOut   string
        wantSpec    bool
        wantErr     bool
    }{
        {
            name: "generates spec from ticket",
            state: DevState{
                Ticket: &Ticket{
                    ID:          "TK-421",
                    Title:       "Add user auth",
                    Description: "Implement OAuth2 login",
                },
            },
            claudeOut: "### Overview\nAdd OAuth2...",
            wantSpec:  true,
        },
        {
            name:    "fails without ticket",
            state:   DevState{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockClaude := &MockClaudeCLI{
                Output: tt.claudeOut,
            }
            ctx := devflow.WithClaudeCLI(context.Background(), mockClaude)

            result, err := GenerateSpecNode(
                flowgraph.WrapContext(ctx),
                tt.state,
            )

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            if tt.wantSpec {
                assert.NotNil(t, result.Spec)
                assert.NotEmpty(t, result.Spec.Overview)
            }
        })
    }
}
```

### Integration Tests

```go
func TestGenerateSpecNode_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    claude := devflow.NewClaudeCLI(devflow.ClaudeConfig{
        Timeout: 2 * time.Minute,
    })

    ctx := devflow.WithClaudeCLI(context.Background(), claude)

    state := DevState{
        Ticket: &Ticket{
            ID:          "TEST-1",
            Title:       "Add health endpoint",
            Description: "Add /health endpoint that returns 200 OK",
        },
    }

    result, err := GenerateSpecNode(flowgraph.WrapContext(ctx), state)
    require.NoError(t, err)

    // Verify spec structure
    assert.NotNil(t, result.Spec)
    assert.NotEmpty(t, result.Spec.Overview)
    assert.NotEmpty(t, result.Spec.Requirements)
    assert.NotEmpty(t, result.Spec.Files)

    // Verify tokens were counted
    assert.Greater(t, result.SpecTokensIn, 0)
    assert.Greater(t, result.SpecTokensOut, 0)
}
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `PromptFile` | `generate-spec.txt` | Prompt template file |
| `MaxTokens` | `4000` | Max output tokens |
| `Temperature` | `0.3` | Lower for consistency |

## Artifacts Saved

| Artifact | Path | Description |
|----------|------|-------------|
| Specification | `spec.md` | Full specification markdown |
| Parsed spec | `spec.json` | Structured spec data |

## References

- ADR-007: Prompt Management
- ADR-019: State Design
- Feature: Prompt Loading
- Phase 5: Workflow Nodes
