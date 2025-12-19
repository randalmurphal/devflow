# Feature: Claude CLI Integration

## Overview

Wrapper around the `claude` CLI binary for structured LLM invocation with development context.

## Use Cases

1. **Spec generation**: Generate technical specs from tickets
2. **Code implementation**: Implement features based on specs
3. **Code review**: Review code for issues
4. **Fix suggestions**: Fix issues found in review

## API

### Basic Run

```go
result, err := claude.Run(ctx, "Implement user authentication")
fmt.Println(result.Output)
fmt.Printf("Tokens: %d in, %d out\n", result.TokensIn, result.TokensOut)
```

### With Options

```go
result, err := claude.Run(ctx, "Implement the feature",
    devflow.WithSystemPrompt("You are an expert Go developer"),
    devflow.WithContext("spec.md", "api.go"),
    devflow.WithWorkDir(worktreePath),
    devflow.WithMaxTurns(20),
    devflow.WithTimeout(10 * time.Minute),
)
```

### With Session (Multi-turn)

```go
// First turn
result1, _ := claude.Run(ctx, "Start implementing auth")
sessionID := result1.SessionID

// Continue
result2, _ := claude.Run(ctx, "Now add error handling",
    devflow.WithSession(sessionID),
)
```

## Configuration

```go
type ClaudeConfig struct {
    BinaryPath string        // Path to claude (default: "claude")
    Model      string        // Model to use (default: claude default)
    Timeout    time.Duration // Default timeout (default: 5m)
    MaxTurns   int           // Max turns (default: 10)
}

claude, err := devflow.NewClaudeCLI(devflow.ClaudeConfig{
    Timeout:  10 * time.Minute,
    MaxTurns: 30,
})
```

## Run Options

| Option | Description |
|--------|-------------|
| `WithSystemPrompt(s)` | Set system prompt |
| `WithContext(files...)` | Add context files |
| `WithWorkDir(dir)` | Set working directory |
| `WithMaxTurns(n)` | Limit conversation turns |
| `WithTimeout(d)` | Set timeout |
| `WithModel(m)` | Override model |
| `WithSession(id)` | Resume session |
| `WithAllowedTools(t...)` | Tool allowlist |
| `WithDisallowedTools(t...)` | Tool denylist |

## Result Structure

```go
type RunResult struct {
    Output     string        // Claude's final output
    TokensIn   int           // Input tokens used
    TokensOut  int           // Output tokens generated
    Cost       float64       // Estimated cost
    SessionID  string        // For resuming
    Duration   time.Duration // Wall clock time
    ExitCode   int           // Process exit code
    Files      []FileChange  // Files modified
}
```

## Behavior

### Command Execution

Builds and runs:
```bash
claude --print --output-format json \
    --system "SYSTEM_PROMPT" \
    --max-turns N \
    -p "PROMPT"
```

### Context Files

Files are formatted as:
```xml
<file path="api.go">
package api
// ...
</file>
```

### Error Handling

- Timeout: Returns `ErrClaudeTimeout`
- Non-zero exit: Returns `ErrClaudeFailed` with stderr
- Missing binary: Returns `ErrClaudeNotFound`

## Example

```go
// Setup
claude, err := devflow.NewClaudeCLI(devflow.ClaudeConfig{
    Timeout: 10 * time.Minute,
})
if err != nil {
    log.Fatal(err)
}

// Generate spec
specPrompt := `Generate a technical specification for:
Title: Add user authentication
Description: Implement OAuth2 login with Google and GitHub`

result, err := claude.Run(ctx, specPrompt,
    devflow.WithSystemPrompt("You are a senior architect. Generate detailed, actionable specs."),
)
if err != nil {
    log.Fatal(err)
}

fmt.Println("=== Generated Spec ===")
fmt.Println(result.Output)
fmt.Printf("\nTokens: %d in, %d out (est. $%.4f)\n",
    result.TokensIn, result.TokensOut, result.Cost)
```

## Testing

```go
// Mock for testing
type MockClaudeCLI struct {
    RunFunc func(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error)
}

func (m *MockClaudeCLI) Run(ctx context.Context, prompt string, opts ...RunOption) (*RunResult, error) {
    return m.RunFunc(ctx, prompt, opts...)
}

// Test with mock
func TestWithMockClaude(t *testing.T) {
    mock := &MockClaudeCLI{
        RunFunc: func(_ context.Context, _ string, _ ...RunOption) (*RunResult, error) {
            return &RunResult{
                Output:    "Generated spec...",
                TokensOut: 500,
            }, nil
        },
    }

    // Use mock in test
}
```

## References

- ADR-006: Claude CLI Wrapper
- ADR-007: Prompt Management
- ADR-008: Context Files
- ADR-009: Output Parsing
- ADR-010: Session Management
