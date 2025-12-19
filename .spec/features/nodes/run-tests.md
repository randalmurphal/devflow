# Node: run-tests

## Purpose

Run the project's test suite to validate implementation. This node executes tests and captures results for decision-making in the workflow.

## Signature

```go
func RunTestsNode(ctx flowgraph.Context, state DevState) (DevState, error)
```

## Input State Requirements

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `Worktree` | `string` | Yes | Working directory path |

## Output State Changes

| Field | Type | Description |
|-------|------|-------------|
| `TestOutput` | `*TestOutput` | Test execution results |

### TestOutput Structure

```go
type TestOutput struct {
    Passed    bool          // All tests passed?
    Total     int           // Total tests run
    Failed    int           // Failed count
    Skipped   int           // Skipped count
    Duration  time.Duration // Total execution time
    Output    string        // Raw output
    Failures  []TestFailure // Failed test details
}

type TestFailure struct {
    Name     string // Test name
    Package  string // Package path
    Output   string // Failure output
    Duration time.Duration
}
```

## Implementation

```go
func RunTestsNode(ctx flowgraph.Context, state DevState) (DevState, error) {
    if state.Worktree == "" {
        return state, fmt.Errorf("run-tests: worktree is required")
    }

    // Detect test framework
    framework, err := detectTestFramework(state.Worktree)
    if err != nil {
        return state, fmt.Errorf("run-tests: detect framework: %w", err)
    }

    // Build command
    var cmd *exec.Cmd
    switch framework {
    case "go":
        cmd = exec.Command("go", "test", "-v", "-json", "./...")
    case "pytest":
        cmd = exec.Command("pytest", "-v", "--tb=short", "--json-report")
    case "jest":
        cmd = exec.Command("npm", "test", "--", "--json")
    case "cargo":
        cmd = exec.Command("cargo", "test", "--", "--format=json")
    default:
        return state, fmt.Errorf("run-tests: unknown framework: %s", framework)
    }

    cmd.Dir = state.Worktree

    // Capture output
    var stdout, stderr bytes.Buffer
    cmd.Stdout = &stdout
    cmd.Stderr = &stderr

    start := time.Now()
    err = cmd.Run()
    duration := time.Since(start)

    // Parse output based on framework
    output, parseErr := parseTestOutput(framework, stdout.Bytes(), stderr.Bytes())
    if parseErr != nil {
        // Use raw output if parsing fails
        output = &TestOutput{
            Passed:   err == nil,
            Duration: duration,
            Output:   stdout.String() + stderr.String(),
        }
    }

    output.Duration = duration
    output.Passed = err == nil

    state.TestOutput = output

    // Checkpoint
    ctx.Checkpoint("tests-run", state)

    // Return error only if tests failed AND we should stop
    if !output.Passed && shouldStopOnTestFailure(ctx) {
        return state, fmt.Errorf("run-tests: %d tests failed", output.Failed)
    }

    return state, nil
}

func detectTestFramework(dir string) (string, error) {
    checks := []struct {
        file      string
        framework string
    }{
        {"go.mod", "go"},
        {"go.sum", "go"},
        {"pytest.ini", "pytest"},
        {"pyproject.toml", "pytest"},
        {"setup.py", "pytest"},
        {"package.json", "jest"},
        {"Cargo.toml", "cargo"},
    }

    for _, check := range checks {
        if _, err := os.Stat(filepath.Join(dir, check.file)); err == nil {
            return check.framework, nil
        }
    }

    return "", fmt.Errorf("no known test framework detected")
}
```

## Go Test Output Parsing

```go
func parseGoTestOutput(data []byte) (*TestOutput, error) {
    output := &TestOutput{}

    scanner := bufio.NewScanner(bytes.NewReader(data))
    for scanner.Scan() {
        var event struct {
            Action  string  `json:"Action"`
            Package string  `json:"Package"`
            Test    string  `json:"Test"`
            Output  string  `json:"Output"`
            Elapsed float64 `json:"Elapsed"`
        }

        if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
            continue
        }

        switch event.Action {
        case "pass":
            if event.Test != "" {
                output.Total++
            }
        case "fail":
            if event.Test != "" {
                output.Total++
                output.Failed++
                output.Failures = append(output.Failures, TestFailure{
                    Name:    event.Test,
                    Package: event.Package,
                })
            }
        case "skip":
            if event.Test != "" {
                output.Total++
                output.Skipped++
            }
        }
    }

    output.Passed = output.Failed == 0
    return output, nil
}
```

## Router Function

Route based on test results:

```go
func TestRouter(ctx flowgraph.Context, state DevState) (string, error) {
    if state.TestOutput == nil {
        return "", fmt.Errorf("no test output")
    }

    if state.TestOutput.Passed {
        return "check-lint", nil
    }

    // Tests failed - go to fix
    return "fix-tests", nil
}
```

## Error Cases

| Error | Cause | Handling |
|-------|-------|----------|
| `worktree is required` | No worktree path | Fail, programming error |
| `detect framework` | No known test files | Warning, skip tests |
| `tests failed` | Test failures | Continue to fix or review |
| `command not found` | Test runner not installed | Fail with clear message |
| `timeout` | Tests took too long | Kill and return partial |

### Timeout Handling

```go
ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
defer cancel()

cmd := exec.CommandContext(ctx, "go", "test", "./...")
err := cmd.Run()

if ctx.Err() == context.DeadlineExceeded {
    return state, fmt.Errorf("run-tests: timeout after 5 minutes")
}
```

## Test Cases

### Unit Tests

```go
func TestRunTestsNode(t *testing.T) {
    tests := []struct {
        name       string
        files      map[string]string
        wantPassed bool
        wantErr    bool
    }{
        {
            name: "passing go tests",
            files: map[string]string{
                "go.mod":    "module test\n\ngo 1.21",
                "main.go":   "package main",
                "main_test.go": `package main
import "testing"
func TestMain(t *testing.T) {}`,
            },
            wantPassed: true,
        },
        {
            name: "failing go tests",
            files: map[string]string{
                "go.mod": "module test\n\ngo 1.21",
                "main_test.go": `package main
import "testing"
func TestFail(t *testing.T) { t.Fail() }`,
            },
            wantPassed: false,
        },
        {
            name:    "no test framework",
            files:   map[string]string{},
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            dir := t.TempDir()
            for name, content := range tt.files {
                os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
            }

            state := DevState{Worktree: dir}
            result, err := RunTestsNode(flowgraph.WrapContext(context.Background()), state)

            if tt.wantErr {
                require.Error(t, err)
                return
            }

            // Note: actual test execution might error on failure
            if tt.wantPassed {
                require.NoError(t, err)
                assert.True(t, result.TestOutput.Passed)
            }
        })
    }
}
```

### Integration Tests

```go
func TestRunTestsNode_Integration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    git := devflow.NewGitContext(testRepoPath)
    worktree, _ := git.CreateWorktree("test-runner")
    defer git.CleanupWorktree(worktree)

    // Write a simple test
    testCode := `package main

import "testing"

func TestAddition(t *testing.T) {
    if 1+1 != 2 {
        t.Fail()
    }
}
`
    os.WriteFile(filepath.Join(worktree, "go.mod"), []byte("module test\n\ngo 1.21"), 0644)
    os.WriteFile(filepath.Join(worktree, "main_test.go"), []byte(testCode), 0644)

    state := DevState{Worktree: worktree}
    result, err := RunTestsNode(flowgraph.WrapContext(context.Background()), state)
    require.NoError(t, err)

    assert.True(t, result.TestOutput.Passed)
    assert.Equal(t, 1, result.TestOutput.Total)
    assert.Equal(t, 0, result.TestOutput.Failed)
}
```

## Configuration

| Option | Default | Description |
|--------|---------|-------------|
| `Timeout` | `5m` | Max test execution time |
| `Verbose` | `true` | Verbose output |
| `ContinueOnFail` | `true` | Continue workflow on failure |
| `Coverage` | `false` | Collect coverage data |

## Artifacts Saved

| Artifact | Path | Description |
|----------|------|-------------|
| Test results | `test-output.json` | Structured results |
| Raw output | `test-output.txt` | Raw test output |
| Coverage | `coverage.out` | Coverage data (if enabled) |

## References

- ADR-020: Error Handling
- Feature: Dev Workflow Nodes
- Phase 5: Workflow Nodes
