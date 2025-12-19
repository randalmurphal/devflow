# Node: check-lint

## Purpose

Run linting and static analysis to catch code quality issues. This node executes appropriate linters for the project and captures any violations.

## Signature

```go
func CheckLintNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

## Input State Requirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Worktree` | `string` | Yes | Working directory path |

## Output State Changes

| Field | Type | Description |
|-------|------|-------------|
| `LintOutput` | `*LintOutput` | Lint check results |

### LintOutput Structure

```go
type LintOutput struct {
    Passed   bool           // No errors?
    Errors   int            // Error count
    Warnings int            // Warning count
    Issues   []LintIssue    // Individual issues
    Output   string         // Raw output
}

type LintIssue struct {
    Severity string // error, warning, info
    File     string // Relative path
    Line     int    // Line number
    Column   int    // Column number
    Rule     string // Rule ID
    Message  string // Issue description
    Fix      string // Suggested fix (if available)
}
```

## Implementation

```go
func CheckLintNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if state.Worktree == "" {
        return state, fmt.Errorf("check-lint: worktree is required")
    }

    // Detect project type and linters
    linters, err := detectLinters(state.Worktree)
    if err != nil {
        return state, fmt.Errorf("check-lint: detect linters: %w", err)
    }

    var allIssues []LintIssue
    var totalErrors, totalWarnings int
    var rawOutput strings.Builder

    for _, linter := range linters {
        issues, output, err := runLinter(ctx, state.Worktree, linter)
        if err != nil {
            // Linter error is not fatal - capture what we can
            rawOutput.WriteString(fmt.Sprintf("Linter %s error: %v\n", linter.Name, err))
            continue
        }

        rawOutput.WriteString(output)
        allIssues = append(allIssues, issues...)

        for _, issue := range issues {
            if issue.Severity == "error" {
                totalErrors++
            } else {
                totalWarnings++
            }
        }
    }

    state.LintOutput = &LintOutput{
        Passed:   totalErrors == 0,
        Errors:   totalErrors,
        Warnings: totalWarnings,
        Issues:   allIssues,
        Output:   rawOutput.String(),
    }

    // Checkpoint
    ctx.Checkpoint("lint-checked", state)

    // Return error only if errors found AND should stop
    if totalErrors > 0 && shouldStopOnLintError(ctx) {
        return state, fmt.Errorf("check-lint: %d errors found", totalErrors)
    }

    return state, nil
}

type LinterConfig struct {
    Name    string
    Command string
    Args    []string
    Parser  func([]byte) ([]LintIssue, error)
}

func detectLinters(dir string) ([]LinterConfig, error) {
    var linters []LinterConfig

    // Go
    if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
        linters = append(linters, LinterConfig{
            Name:    "golangci-lint",
            Command: "golangci-lint",
            Args:    []string{"run", "--out-format=json"},
            Parser:  parseGolangCILint,
        })
    }

    // Python
    if hasAnyFile(dir, "pyproject.toml", "setup.py", "requirements.txt") {
        linters = append(linters, LinterConfig{
            Name:    "ruff",
            Command: "ruff",
            Args:    []string{"check", "--output-format=json", "."},
            Parser:  parseRuff,
        })
        linters = append(linters, LinterConfig{
            Name:    "pyright",
            Command: "pyright",
            Args:    []string{"--outputjson"},
            Parser:  parsePyright,
        })
    }

    // JavaScript/TypeScript
    if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
        linters = append(linters, LinterConfig{
            Name:    "eslint",
            Command: "npx",
            Args:    []string{"eslint", "--format=json", "."},
            Parser:  parseESLint,
        })
    }

    // Rust
    if _, err := os.Stat(filepath.Join(dir, "Cargo.toml")); err == nil {
        linters = append(linters, LinterConfig{
            Name:    "clippy",
            Command: "cargo",
            Args:    []string{"clippy", "--message-format=json"},
            Parser:  parseClippy,
        })
    }

    return linters, nil
}
```

## Linter Parsers

### golangci-lint

```go
func parseGolangCILint(data []byte) ([]LintIssue, error) {
    var result struct {
        Issues []struct {
            FromLinter string `json:"FromLinter"`
            Text       string `json:"Text"`
            Pos        struct {
                Filename string `json:"Filename"`
                Line     int    `json:"Line"`
                Column   int    `json:"Column"`
            } `json:"Pos"`
            Severity string `json:"Severity"`
        } `json:"Issues"`
    }

    if err := json.Unmarshal(data, &result); err != nil {
        return nil, err
    }

    var issues []LintIssue
    for _, i := range result.Issues {
        severity := i.Severity
        if severity == "" {
            severity = "error"
        }

        issues = append(issues, LintIssue{
            Severity: severity,
            File:     i.Pos.Filename,
            Line:     i.Pos.Line,
            Column:   i.Pos.Column,
            Rule:     i.FromLinter,
            Message:  i.Text,
        })
    }

    return issues, nil
}
```

### ruff

```go
func parseRuff(data []byte) ([]LintIssue, error) {
    var results []struct {
        Code     string `json:"code"`
        Message  string `json:"message"`
        Fix      *struct {
            Message string `json:"message"`
        } `json:"fix"`
        Location struct {
            File   string `json:"file"`
            Row    int    `json:"row"`
            Column int    `json:"column"`
        } `json:"location"`
    }

    if err := json.Unmarshal(data, &results); err != nil {
        return nil, err
    }

    var issues []LintIssue
    for _, r := range results {
        issue := LintIssue{
            Severity: "error",
            File:     r.Location.File,
            Line:     r.Location.Row,
            Column:   r.Location.Column,
            Rule:     r.Code,
            Message:  r.Message,
        }
        if r.Fix != nil {
            issue.Fix = r.Fix.Message
        }
        issues = append(issues, issue)
    }

    return issues, nil
}
```

## Router Function

```go
func LintRouter(ctx flowgraph.Context, state DevState) (string, error) {
    if state.LintOutput == nil {
        return "", fmt.Errorf("no lint output")
    }

    if state.LintOutput.Passed {
        return "review", nil
    }

    // Has errors - try to auto-fix
    if hasAutoFixable(state.LintOutput.Issues) {
        return "auto-fix-lint", nil
    }

    // Manual fix needed
    return "fix-lint", nil
}
```

## Auto-Fix Support

```go
func AutoFixLintNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    // Run auto-fixers
    fixers := map[string][]string{
        "go":     {"gofmt", "-w", "."},
        "python": {"ruff", "check", "--fix", "."},
        "js":     {"npx", "eslint", "--fix", "."},
    }

    lang := detectLanguage(state.Worktree)
    if cmd, ok := fixers[lang]; ok {
        exec.Command(cmd[0], cmd[1:]...).Run()
    }

    // Re-run lint check
    return CheckLintNode(ctx, state)
}
```

## Error Cases

| Error | Cause | Handling |
|-------|-------|----------|
| `worktree is required` | No worktree path | Fail, programming error |
| `no linters found` | Unknown project type | Warning, skip lint |
| `linter not installed` | Tool missing | Warning, skip that linter |
| `lint errors found` | Code quality issues | Continue to fix or review |

## Test Cases

### Unit Tests

```go
func TestCheckLintNode(t *testing.T) {
    tests := []struct {
        name        string
        files       map[string]string
        wantPassed  bool
        wantErrors  int
    }{
        {
            name: "clean go code",
            files: map[string]string{
                "go.mod": "module test\n\ngo 1.21",
                "main.go": `package main

func main() {
    println("hello")
}
`,
            },
            wantPassed: true,
            wantErrors: 0,
        },
        {
            name: "go code with issues",
            files: map[string]string{
                "go.mod": "module test\n\ngo 1.21",
                "main.go": `package main

func main() {
    x := 1  // unused variable
}
`,
            },
            wantPassed: false,
            wantErrors: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir()
            for name, content := range tt.files {
                os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
            }

            state := DevState{Worktree: dir}
            result, _ := CheckLintNode(flowgraph.WrapContext(context.Background()), state)

            assert.Equal(t, tt.wantPassed, result.LintOutput.Passed)
            assert.Equal(t, tt.wantErrors, result.LintOutput.Errors)
        })
    }
}
```

### Integration Tests

```go
func TestCheckLintNode_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    git := devflow.NewGitContext(testRepoPath)
    worktree, _ := git.CreateWorktree("test-lint")
    defer git.CleanupWorktree(worktree)

    // Write clean Go code
    goMod := `module test

go 1.21
`
    main := `package main

import "fmt"

func main() {
    fmt.Println("hello")
}
`
    os.WriteFile(filepath.Join(worktree, "go.mod"), []byte(goMod), 0644)
    os.WriteFile(filepath.Join(worktree, "main.go"), []byte(main), 0644)

    state := DevState{Worktree: worktree}
    result, err := CheckLintNode(flowgraph.WrapContext(context.Background()), state)
    require.NoError(t, err)

    assert.True(t, result.LintOutput.Passed)
    assert.Equal(t, 0, result.LintOutput.Errors)
}
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `Timeout` | `2m` | Max lint execution time |
| `StopOnError` | `false` | Stop workflow on lint errors |
| `IgnoreWarnings` | `false` | Don't count warnings |
| `AutoFix` | `false` | Run auto-fix first |

## Artifacts Saved

| Artifact | Path | Description |
|----------|------|-------------|
| Lint results | `lint-output.json` | Structured results |
| Raw output | `lint-output.txt` | Raw linter output |

## References

- ADR-020: Error Handling
- Feature: Dev Workflow Nodes
- Phase 5: Workflow Nodes
