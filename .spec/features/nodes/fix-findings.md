# Node: fix-findings

## Purpose

Fix issues identified during code review. This node takes review findings and asks Claude to address them, creating a feedback loop until the code passes review.

## Signature

```go
func FixFindingsNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

## Input State Requirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Review` | `*Review` | Yes | Review with findings to fix |
| `Worktree` | `string` | Yes | Working directory path |
| `Spec` | `*Spec` | Optional | Original spec for context |

## Output State Changes

| Field | Type | Description |
|-------|------|-------------|
| `Implementation` | `*Implementation` | Updated with fixes |
| `Files` | `[]FileChange` | Updated file changes |
| `FixTokensIn` | `int` | Input tokens used |
| `FixTokensOut` | `int` | Output tokens generated |

## Prompt Template

Located at: `prompts/fix-findings.txt`

```
You are an expert software engineer. Fix the following issues from code review.

## Review Findings

{{range .Findings}}
### {{.Severity | upper}}: {{.Message}}
- **File**: {{.File}}:{{.Line}}
- **Category**: {{.Category}}
- **Suggested Fix**: {{.Suggestion}}
{{end}}

## Original Specification
{{.Spec.Raw}}

## Current Code

{{range .AffectedFiles}}
### {{.Path}}
```{{.Language}}
{{.Content}}
```
{{end}}

## Instructions

1. Fix each finding in order of severity (errors first)
2. Keep changes minimal and focused
3. Don't introduce new issues
4. Maintain existing code style
5. After fixes, briefly explain what was changed

## Output Format

After making fixes:

### Changes Made
- File: what was fixed

### Notes
Any important context about the fixes.
```

## Implementation

```go
func FixFindingsNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if state.Review == nil {
        return state, fmt.Errorf("fix-findings: review is required")
    }
    if len(state.Review.Findings) == 0 {
        // Nothing to fix
        return state, nil
    }
    if state.Worktree == "" {
        return state, fmt.Errorf("fix-findings: worktree is required")
    }

    claude := devflow.ClaudeCLIFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("fix-findings: ClaudeCLI not in context")
    }

    // Get affected files
    affectedFiles := getAffectedFiles(state.Worktree, state.Review.Findings)

    // Filter to actionable findings
    findings := filterFindings(state.Review.Findings, []string{"error", "warning"})

    // Load prompt
    prompt, err := devflow.LoadPrompt("fix-findings", map[string]any{
        "Findings":      findings,
        "Spec":          state.Spec,
        "AffectedFiles": affectedFiles,
    })
    if err != nil {
        return state, fmt.Errorf("fix-findings: load prompt: %w", err)
    }

    // Get file state before fixes
    beforeState, _ := captureFileState(state.Worktree, affectedFiles)

    // Run fixes
    result, err := claude.Run(ctx, prompt,
        devflow.WithWorkDir(state.Worktree),
        devflow.WithSystemPrompt(fixSystemPrompt),
        devflow.WithMaxTurns(15),
    )
    if err != nil {
        return state, fmt.Errorf("fix-findings: claude: %w", err)
    }

    // Detect what changed
    afterState, _ := captureFileState(state.Worktree, affectedFiles)
    changes := diffFileStates(beforeState, afterState)

    // Update implementation
    if state.Implementation == nil {
        state.Implementation = &Implementation{}
    }
    state.Implementation.Notes += fmt.Sprintf("\n\nFix attempt %d:\n%s",
        state.ReviewAttempts, result.Output)

    // Update files
    state.Files = mergeFileChanges(state.Files, changes)
    state.FixTokensIn += result.TokensIn
    state.FixTokensOut += result.TokensOut

    // Clear review so next iteration generates fresh review
    state.Review = nil

    // Checkpoint
    ctx.Checkpoint("findings-fixed", state)

    return state, nil
}
```

## Flow Integration

Typically part of a review loop:

```go
graph := flowgraph.NewGraph[DevState]().
    AddNode("implement", devflow.ImplementNode).
    AddNode("review", devflow.ReviewNode).
    AddNode("fix-findings", devflow.FixFindingsNode).
    AddNode("create-pr", devflow.CreatePRNode).
    AddEdge("implement", "review").
    AddConditionalEdge("review", devflow.ReviewRouter).
    AddEdge("fix-findings", "review").  // Loop back
    AddEdge("create-pr", flowgraph.END)
```

## Error Cases

| Error | Cause | Handling |
|-------|-------|----------|
| `review is required` | No review in state | Fail, programming error |
| `no findings` | Empty findings list | Return unchanged (no-op) |
| `worktree is required` | No worktree path | Fail, setup error |
| `claude: timeout` | Fixes took too long | Retry with subset of findings |

### Partial Fixes

If not all findings are fixed:
1. Record what was fixed
2. Leave remaining findings for next iteration
3. Max attempts guard prevents infinite loop

```go
// In ReviewRouter
if state.ReviewAttempts >= 3 {
    // Escalate to human review
    return "human-review", nil
}
```

## Test Cases

### Unit Tests

```go
func TestFixFindingsNode(t *testing.T) {
    tests := []struct {
        name      string
        findings  []ReviewFinding
        wantFixed bool
    }{
        {
            name: "fixes error findings",
            findings: []ReviewFinding{
                {
                    Severity:   "error",
                    File:       "api.go",
                    Line:       42,
                    Message:    "Missing error check",
                    Suggestion: "Add if err != nil",
                },
            },
            wantFixed: true,
        },
        {
            name:      "no-op for empty findings",
            findings:  []ReviewFinding{},
            wantFixed: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockClaude := &MockClaudeCLI{}
            ctx := devflow.WithClaudeCLI(context.Background(), mockClaude)

            state := DevState{
                Review: &Review{Findings: tt.findings},
                Worktree: t.TempDir(),
            }

            result, err := FixFindingsNode(flowgraph.WrapContext(ctx), state)
            require.NoError(t, err)

            if tt.wantFixed {
                assert.True(t, mockClaude.Called)
                assert.Nil(t, result.Review) // Cleared for re-review
            }
        })
    }
}
```

### Integration Tests

```go
func TestFixFindingsNode_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup worktree with fixable issue
    git := devflow.NewGitContext(testRepoPath)
    worktree, _ := git.CreateWorktree("test-fix")
    defer git.CleanupWorktree(worktree)

    // Write code with issue
    code := `package main

func divide(a, b int) int {
    return a / b
}`
    os.WriteFile(filepath.Join(worktree, "math.go"), []byte(code), 0644)

    claude := devflow.NewClaudeCLI(devflow.ClaudeConfig{
        Timeout: 3 * time.Minute,
    })
    ctx := devflow.WithClaudeCLI(context.Background(), claude)

    state := DevState{
        Review: &Review{
            Findings: []ReviewFinding{
                {
                    Severity:   "error",
                    Category:   "bug",
                    File:       "math.go",
                    Line:       4,
                    Message:    "Division by zero not handled",
                    Suggestion: "Check if b is zero and return error",
                },
            },
        },
        Worktree: worktree,
    }

    result, err := FixFindingsNode(flowgraph.WrapContext(ctx), state)
    require.NoError(t, err)

    // Verify file was modified
    content, _ := os.ReadFile(filepath.Join(worktree, "math.go"))
    assert.Contains(t, string(content), "b == 0") // Zero check added
}
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `PromptFile` | `fix-findings.txt` | Prompt template file |
| `MaxFindings` | `10` | Max findings per fix attempt |
| `PrioritizeErrors` | `true` | Fix errors before warnings |

## Artifacts Saved

| Artifact | Path | Description |
|----------|------|-------------|
| Fix summary | `fix-attempt-{n}.md` | What was fixed |
| Updated diff | `implementation.diff` | Updated changes |

## References

- ADR-020: Error Handling
- Node: review-code
- Feature: Dev Workflow Nodes
- Phase 5: Workflow Nodes
