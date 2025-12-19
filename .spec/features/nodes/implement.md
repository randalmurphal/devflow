# Node: implement

## Purpose

Implement code based on a technical specification. This node runs Claude CLI in a worktree to generate the actual code changes defined in the spec.

## Signature

```go
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

## Input State Requirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Spec` | `*Spec` | Yes | Technical specification |
| `Worktree` | `string` | Yes | Working directory path |

## Output State Changes

| Field | Type | Description |
|-------|------|-------------|
| `Implementation` | `*Implementation` | Implementation details |
| `Files` | `[]FileChange` | Files created/modified |
| `ImplementTokensIn` | `int` | Input tokens used |
| `ImplementTokensOut` | `int` | Output tokens generated |

### Implementation Structure

```go
type Implementation struct {
    Summary     string       // What was done
    FilesAdded  []string     // New files created
    FilesChanged []string    // Existing files modified
    Decisions   []string     // Key decisions made
    Notes       string       // Implementation notes
}

type FileChange struct {
    Path    string  // Relative path
    Action  string  // add, modify, delete
    Diff    string  // Unified diff
}
```

## Prompt Template

Located at: `prompts/implement.txt`

```
You are an expert software engineer. Implement the following specification.

## Specification
{{.Spec.Raw}}

## Working Directory
{{.Worktree}}

## Instructions

1. Read the specification carefully
2. Implement each file listed in the spec
3. Follow the project's existing code style
4. Add appropriate error handling
5. Include inline comments for complex logic
6. Do NOT write tests (separate node handles tests)

## Project Context
{{range .ContextFiles}}
File: {{.Path}}
```
{{.Content}}
```
{{end}}

## Constraints
- Only modify files listed in the specification
- Use existing patterns from the codebase
- Keep changes minimal and focused
- If you can't complete something, explain why

## Output
After implementation, provide a summary:
- What files were created/modified
- Key implementation decisions
- Any issues or notes
```

## Implementation

```go
func ImplementNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if state.Spec == nil {
        return state, fmt.Errorf("implement: spec is required")
    }
    if state.Worktree == "" {
        return state, fmt.Errorf("implement: worktree is required")
    }

    claude := devflow.ClaudeCLIFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("implement: ClaudeCLI not in context")
    }

    // Gather context files
    contextFiles, err := gatherContextFiles(state.Worktree, state.Spec.Files)
    if err != nil {
        return state, fmt.Errorf("implement: gather context: %w", err)
    }

    // Load prompt
    prompt, err := devflow.LoadPrompt("implement", map[string]any{
        "Spec":         state.Spec,
        "Worktree":     state.Worktree,
        "ContextFiles": contextFiles,
    })
    if err != nil {
        return state, fmt.Errorf("implement: load prompt: %w", err)
    }

    // Get state before implementation
    beforeFiles, err := getTrackedFiles(state.Worktree)
    if err != nil {
        return state, fmt.Errorf("implement: get tracked files: %w", err)
    }

    // Run Claude with file access
    result, err := claude.Run(ctx, prompt,
        devflow.WithWorkDir(state.Worktree),
        devflow.WithSystemPrompt(implementSystemPrompt),
        devflow.WithMaxTurns(20), // Allow multiple tool uses
    )
    if err != nil {
        return state, fmt.Errorf("implement: claude: %w", err)
    }

    // Detect file changes
    changes, err := detectChanges(state.Worktree, beforeFiles)
    if err != nil {
        return state, fmt.Errorf("implement: detect changes: %w", err)
    }

    // Parse implementation summary
    impl, err := parseImplementation(result.Output)
    if err != nil {
        // Non-fatal: use basic summary
        impl = &Implementation{Summary: result.Output}
    }

    // Update state
    state.Implementation = impl
    state.Files = changes
    state.ImplementTokensIn = result.TokensIn
    state.ImplementTokensOut = result.TokensOut

    // Checkpoint
    ctx.Checkpoint("implemented", state)

    return state, nil
}
```

## Error Cases

| Error | Cause | Handling |
|-------|-------|----------|
| `spec is required` | No spec in state | Fail, run generate-spec first |
| `worktree is required` | No worktree path | Fail, run create-worktree first |
| `claude: timeout` | Implementation took too long | Retry with longer timeout |
| `detect changes: git error` | Git operation failed | Fail, check worktree state |

### Partial Implementation

If Claude partially completes, the node:
1. Captures what was done
2. Records in state
3. Returns error with context
4. Allows retry from checkpoint

```go
if len(changes) > 0 && err != nil {
    state.Files = changes
    state.Error = err.Error()
    ctx.Checkpoint("partial-implementation", state)
    return state, fmt.Errorf("implement: partial completion: %w", err)
}
```

## Test Cases

### Unit Tests

```go
func TestImplementNode(t *testing.T) {
    tests := []struct {
        name      string
        state     DevState
        wantFiles int
        wantErr   bool
    }{
        {
            name: "implements from spec",
            state: DevState{
                Spec: &Spec{
                    Files: []FileSpec{
                        {Path: "api.go", Action: "create"},
                    },
                },
                Worktree: "/tmp/test-worktree",
            },
            wantFiles: 1,
        },
        {
            name: "fails without spec",
            state: DevState{
                Worktree: "/tmp/test",
            },
            wantErr: true,
        },
        {
            name: "fails without worktree",
            state: DevState{
                Spec: &Spec{},
            },
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockClaude := &MockClaudeCLI{
                FilesToCreate: tt.wantFiles,
            }
            ctx := devflow.WithClaudeCLI(context.Background(), mockClaude)

            result, err := ImplementNode(
                flowgraph.WrapContext(ctx),
                tt.state,
            )

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Len(t, result.Files, tt.wantFiles)
        })
    }
}
```

### Integration Tests

```go
func TestImplementNode_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup worktree
    git := devflow.NewGitContext(testRepoPath)
    worktree, err := git.CreateWorktree("test-implement")
    require.NoError(t, err)
    defer git.CleanupWorktree(worktree)

    claude := devflow.NewClaudeCLI(devflow.ClaudeConfig{
        Timeout: 5 * time.Minute,
    })

    ctx := context.Background()
    ctx = devflow.WithClaudeCLI(ctx, claude)
    ctx = devflow.WithGitContext(ctx, git)

    state := DevState{
        Spec: &Spec{
            Overview: "Add health check endpoint",
            Files: []FileSpec{
                {Path: "health.go", Action: "create", Description: "Health endpoint handler"},
            },
            Raw: "Create /health endpoint returning 200 OK",
        },
        Worktree: worktree,
    }

    result, err := ImplementNode(flowgraph.WrapContext(ctx), state)
    require.NoError(t, err)

    // Verify files created
    assert.NotEmpty(t, result.Files)
    assert.FileExists(t, filepath.Join(worktree, "health.go"))

    // Verify tokens counted
    assert.Greater(t, result.ImplementTokensIn, 0)
}
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `PromptFile` | `implement.txt` | Prompt template file |
| `MaxTurns` | `20` | Max Claude turns |
| `Timeout` | `10m` | Implementation timeout |

## Artifacts Saved

| Artifact | Path | Description |
|----------|------|-------------|
| Implementation diff | `implementation.diff` | Unified diff of all changes |
| File list | `files-changed.json` | List of changed files |

## References

- ADR-006: Claude CLI Wrapper
- ADR-019: State Design
- Feature: Claude CLI Integration
- Phase 5: Workflow Nodes
