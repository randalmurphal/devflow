# Node: review-code

## Purpose

Review implementation for issues before proceeding. This node acts as quality gate, checking code against the spec and identifying problems that need fixing.

## Signature

```go
func ReviewNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

## Input State Requirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Spec` | `*Spec` | Yes | Original specification |
| `Worktree` | `string` | Yes | Working directory with changes |
| `Implementation` | `*Implementation` | Optional | Implementation summary |

## Output State Changes

| Field | Type | Description |
|-------|------|-------------|
| `Review` | `*Review` | Review results |
| `ReviewAttempts` | `int` | Incremented counter |
| `ReviewTokensIn` | `int` | Input tokens used |
| `ReviewTokensOut` | `int` | Output tokens generated |

### Review Structure

```go
type Review struct {
    Approved    bool            // Ready to merge?
    Summary     string          // Overall assessment
    Findings    []ReviewFinding // Issues found
    Suggestions []string        // Non-blocking improvements
}

type ReviewFinding struct {
    Severity    string  // error, warning, info
    Category    string  // bug, security, performance, style
    File        string  // Relative path
    Line        int     // Line number (0 if general)
    Message     string  // Description
    Suggestion  string  // How to fix
}
```

## Prompt Template

Located at: `prompts/review-code.txt`

```
You are a senior code reviewer. Review the implementation against the specification.

## Original Specification
{{.Spec.Raw}}

## Files Changed
{{range .Files}}
### {{.Path}} ({{.Action}})
```diff
{{.Diff}}
```
{{end}}

## Review Criteria

Check for:
1. **Correctness**: Does the implementation match the spec?
2. **Bugs**: Logic errors, edge cases, error handling
3. **Security**: Input validation, injection risks, secrets
4. **Performance**: Obvious inefficiencies
5. **Style**: Consistency with codebase, readability

## Output Format

### Approved: [yes/no]

### Summary
Overall assessment in 2-3 sentences.

### Findings
For each issue:
- Severity: error | warning | info
- Category: bug | security | performance | style
- File: path/to/file.go
- Line: 42
- Issue: Description of the problem
- Fix: How to resolve it

### Suggestions
Non-blocking improvements for future consideration.
```

## Implementation

```go
func ReviewNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if state.Spec == nil {
        return state, fmt.Errorf("review: spec is required")
    }
    if state.Worktree == "" {
        return state, fmt.Errorf("review: worktree is required")
    }

    claude := devflow.ClaudeCLIFromContext(ctx)
    if claude == nil {
        return state, fmt.Errorf("review: ClaudeCLI not in context")
    }

    // Get diff for review
    git := devflow.GitContextFromContext(ctx)
    diff, err := git.Diff(state.Worktree, "HEAD")
    if err != nil {
        return state, fmt.Errorf("review: get diff: %w", err)
    }

    // Build file change list
    files := parseFilesFromDiff(diff)

    // Load prompt
    prompt, err := devflow.LoadPrompt("review-code", map[string]any{
        "Spec":  state.Spec,
        "Files": files,
    })
    if err != nil {
        return state, fmt.Errorf("review: load prompt: %w", err)
    }

    // Run review
    result, err := claude.Run(ctx, prompt,
        devflow.WithSystemPrompt(reviewSystemPrompt),
        devflow.WithWorkDir(state.Worktree),
    )
    if err != nil {
        return state, fmt.Errorf("review: claude: %w", err)
    }

    // Parse review
    review, err := parseReview(result.Output)
    if err != nil {
        return state, fmt.Errorf("review: parse: %w", err)
    }

    // Update state
    state.Review = review
    state.ReviewAttempts++
    state.ReviewTokensIn = result.TokensIn
    state.ReviewTokensOut = result.TokensOut

    // Checkpoint
    ctx.Checkpoint("reviewed", state)

    return state, nil
}
```

## Router Function

After review, route based on approval:

```go
func ReviewRouter(ctx flowgraph.Context, state DevState) (string, error) {
    if state.Review == nil {
        return "", fmt.Errorf("no review in state")
    }

    if state.Review.Approved {
        return "create-pr", nil
    }

    // Check for max attempts
    if state.ReviewAttempts >= 3 {
        return "max-attempts-exceeded", nil
    }

    // Has blocking issues?
    hasErrors := false
    for _, f := range state.Review.Findings {
        if f.Severity == "error" {
            hasErrors = true
            break
        }
    }

    if hasErrors {
        return "fix-findings", nil
    }

    // Warnings only - proceed
    return "create-pr", nil
}
```

## Error Cases

| Error | Cause | Handling |
|-------|-------|----------|
| `spec is required` | Missing spec | Fail, programming error |
| `get diff: no changes` | Nothing to review | Return empty review (approved) |
| `parse: invalid format` | Malformed review output | Retry with clearer prompt |

## Test Cases

### Unit Tests

```go
func TestReviewNode(t *testing.T) {
    tests := []struct {
        name         string
        claudeOutput string
        wantApproved bool
        wantFindings int
    }{
        {
            name:         "approves clean implementation",
            claudeOutput: "### Approved: yes\n### Summary\nLooks good!",
            wantApproved: true,
            wantFindings: 0,
        },
        {
            name: "rejects with findings",
            claudeOutput: `### Approved: no
### Summary
Found issues.
### Findings
- Severity: error
- Category: bug
- File: api.go
- Line: 42
- Issue: Missing error handling
- Fix: Add error check`,
            wantApproved: false,
            wantFindings: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            mockClaude := &MockClaudeCLI{Output: tt.claudeOutput}
            mockGit := &MockGitContext{Diff: "diff --git a/file.go..."}

            ctx := context.Background()
            ctx = devflow.WithClaudeCLI(ctx, mockClaude)
            ctx = devflow.WithGitContext(ctx, mockGit)

            state := DevState{
                Spec:     &Spec{Raw: "Test spec"},
                Worktree: "/tmp/test",
            }

            result, err := ReviewNode(flowgraph.WrapContext(ctx), state)
            require.NoError(t, err)

            assert.Equal(t, tt.wantApproved, result.Review.Approved)
            assert.Len(t, result.Review.Findings, tt.wantFindings)
        })
    }
}
```

### Integration Tests

```go
func TestReviewNode_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    // Setup with actual changes
    git := devflow.NewGitContext(testRepoPath)
    worktree, _ := git.CreateWorktree("test-review")
    defer git.CleanupWorktree(worktree)

    // Create a file with obvious issues
    badCode := `func divide(a, b int) int {
    return a / b  // No zero check
}`
    os.WriteFile(filepath.Join(worktree, "math.go"), []byte(badCode), 0644)

    claude := devflow.NewClaudeCLI(devflow.ClaudeConfig{
        Timeout: 2 * time.Minute,
    })

    ctx := context.Background()
    ctx = devflow.WithClaudeCLI(ctx, claude)
    ctx = devflow.WithGitContext(ctx, git)

    state := DevState{
        Spec: &Spec{
            Raw: "Add divide function with proper error handling",
        },
        Worktree: worktree,
    }

    result, err := ReviewNode(flowgraph.WrapContext(ctx), state)
    require.NoError(t, err)

    // Should find the division by zero risk
    assert.False(t, result.Review.Approved)
    assert.NotEmpty(t, result.Review.Findings)
}
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `PromptFile` | `review-code.txt` | Prompt template file |
| `MaxAttempts` | `3` | Max review cycles |
| `IgnoreSeverity` | `[]` | Severities to ignore |

## Artifacts Saved

| Artifact | Path | Description |
|----------|------|-------------|
| Review results | `review.json` | Structured review data |
| Review markdown | `review.md` | Human-readable review |

## References

- ADR-020: Error Handling
- Feature: Dev Workflow Nodes
- Phase 5: Workflow Nodes
