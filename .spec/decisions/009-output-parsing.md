# ADR-009: Output Parsing

## Status

Accepted

## Context

Claude's output needs to be parsed for:

1. Token usage metrics
2. File changes made during execution
3. Structured responses (when requested)
4. Error messages and warnings

We need a consistent approach to extracting this information.

## Decision

### 1. JSON Output Mode

Request JSON output from Claude CLI:

```bash
claude --print --output-format json -p "..."
```

JSON output provides structured data:

```json
{
  "result": "The implementation is complete...",
  "tokens_in": 1500,
  "tokens_out": 2300,
  "session_id": "abc123",
  "cost": 0.045,
  "duration_ms": 15000
}
```

### 2. RunResult Structure

```go
type RunResult struct {
    // Core output
    Output     string        // Final text output

    // Token metrics
    TokensIn   int           // Input tokens
    TokensOut  int           // Output tokens
    Cost       float64       // Cost in dollars

    // Session info
    SessionID  string        // For resuming
    Duration   time.Duration // Wall clock time
    ExitCode   int           // Process exit code

    // File changes (if tooling enabled)
    Files      []FileChange  // Created/modified files

    // Transcript (if captured)
    Transcript *Transcript   // Full conversation
}

type FileChange struct {
    Path      string // Relative path
    Operation string // "create", "modify", "delete"
    Content   []byte // New content (for create/modify)
}
```

### 3. File Change Detection

When Claude uses file tools, changes are detected by:

1. **Before/after diff**: Snapshot files before, compare after
2. **Git status**: Use `git status --porcelain` to find changes
3. **Tool output parsing**: Parse tool calls from transcript

Primary approach: Git status (simplest, most reliable)

```go
func (c *ClaudeCLI) detectFileChanges(workDir string) ([]FileChange, error) {
    // Run git status
    cmd := exec.Command("git", "status", "--porcelain")
    cmd.Dir = workDir
    output, err := cmd.Output()
    if err != nil {
        return nil, err
    }

    var changes []FileChange
    for _, line := range strings.Split(string(output), "\n") {
        if len(line) < 3 {
            continue
        }
        status := line[:2]
        path := strings.TrimSpace(line[3:])

        var op string
        switch status[0] {
        case 'A', '?':
            op = "create"
        case 'M', ' ':
            if status[1] == 'M' {
                op = "modify"
            }
        case 'D':
            op = "delete"
        }

        if op != "" {
            changes = append(changes, FileChange{
                Path:      path,
                Operation: op,
            })
        }
    }

    return changes, nil
}
```

### 4. Structured Response Parsing

For prompts that request structured output (JSON, YAML, etc.):

```go
// ParseJSON extracts JSON from Claude's output
func ParseJSON[T any](output string) (T, error) {
    var result T

    // Find JSON block
    start := strings.Index(output, "{")
    if start == -1 {
        start = strings.Index(output, "[")
    }
    if start == -1 {
        return result, fmt.Errorf("no JSON found in output")
    }

    // Find end
    var end int
    if output[start] == '{' {
        end = strings.LastIndex(output, "}") + 1
    } else {
        end = strings.LastIndex(output, "]") + 1
    }

    if err := json.Unmarshal([]byte(output[start:end]), &result); err != nil {
        return result, fmt.Errorf("parse JSON: %w", err)
    }

    return result, nil
}

// Usage
type ReviewResult struct {
    Approved bool     `json:"approved"`
    Findings []string `json:"findings"`
}

review, err := devflow.ParseJSON[ReviewResult](result.Output)
```

### 5. Markdown Code Block Extraction

Extract code from markdown code blocks:

```go
// ExtractCodeBlocks extracts code blocks from markdown
func ExtractCodeBlocks(output string) []CodeBlock {
    var blocks []CodeBlock

    // Regex for ```language\ncode\n```
    re := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)```")
    matches := re.FindAllStringSubmatch(output, -1)

    for _, match := range matches {
        blocks = append(blocks, CodeBlock{
            Language: match[1],
            Content:  match[2],
        })
    }

    return blocks
}

type CodeBlock struct {
    Language string
    Content  string
}
```

## Alternatives Considered

### Alternative 1: Text-Only Output

Parse Claude's text output without JSON mode.

**Rejected because:**
- No reliable token metrics
- Harder to extract structured data
- More fragile parsing

### Alternative 2: Streaming Parse

Parse output as it streams.

**Deferred because:**
- Added complexity
- Not needed for initial implementation
- Can add later if needed

### Alternative 3: Custom Output Format

Ask Claude to output in a custom format.

**Rejected because:**
- Consumes tokens for formatting
- Can fail to follow format
- JSON is standard

## Consequences

### Positive

- **Reliable metrics**: JSON provides accurate token counts
- **Structured data**: Easy to extract specific fields
- **File tracking**: Know what files changed

### Negative

- **Format dependency**: Relies on Claude CLI JSON format
- **Parse failures**: May fail if format changes
- **Extra processing**: Must parse JSON output

### Error Handling

```go
func (c *ClaudeCLI) parseOutput(stdout, stderr []byte) (*RunResult, error) {
    result := &RunResult{}

    // Try JSON parsing
    var jsonOut struct {
        Result    string  `json:"result"`
        TokensIn  int     `json:"tokens_in"`
        TokensOut int     `json:"tokens_out"`
        SessionID string  `json:"session_id"`
        Cost      float64 `json:"cost"`
    }

    if err := json.Unmarshal(stdout, &jsonOut); err != nil {
        // Fallback to raw output
        result.Output = string(stdout)

        // Try to parse tokens from stderr
        result.TokensIn, result.TokensOut = parseTokensFromStderr(string(stderr))
    } else {
        result.Output = jsonOut.Result
        result.TokensIn = jsonOut.TokensIn
        result.TokensOut = jsonOut.TokensOut
        result.SessionID = jsonOut.SessionID
        result.Cost = jsonOut.Cost
    }

    return result, nil
}

// parseTokensFromStderr extracts token info from stderr
func parseTokensFromStderr(stderr string) (int, int) {
    // Look for patterns like "Input tokens: 1500" or similar
    // This is a fallback when JSON parsing fails
    var in, out int

    if m := regexp.MustCompile(`input.?tokens?\D*(\d+)`).FindStringSubmatch(strings.ToLower(stderr)); len(m) > 1 {
        in, _ = strconv.Atoi(m[1])
    }
    if m := regexp.MustCompile(`output.?tokens?\D*(\d+)`).FindStringSubmatch(strings.ToLower(stderr)); len(m) > 1 {
        out, _ = strconv.Atoi(m[1])
    }

    return in, out
}
```

## Code Example

```go
package devflow

import (
    "encoding/json"
    "fmt"
    "regexp"
    "strings"
)

// RunResult contains parsed output from Claude
type RunResult struct {
    Output     string
    TokensIn   int
    TokensOut  int
    Cost       float64
    SessionID  string
    Duration   time.Duration
    ExitCode   int
    Files      []FileChange
    Transcript *Transcript
}

// ParseJSON extracts and parses JSON from output
func ParseJSON[T any](output string) (T, error) {
    var result T

    // Find JSON boundaries
    jsonStr := extractJSON(output)
    if jsonStr == "" {
        return result, fmt.Errorf("no JSON found in output")
    }

    if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
        return result, fmt.Errorf("parse JSON: %w", err)
    }

    return result, nil
}

// extractJSON finds JSON in text
func extractJSON(text string) string {
    // Find object
    if start := strings.Index(text, "{"); start != -1 {
        depth := 0
        for i := start; i < len(text); i++ {
            switch text[i] {
            case '{':
                depth++
            case '}':
                depth--
                if depth == 0 {
                    return text[start : i+1]
                }
            }
        }
    }

    // Find array
    if start := strings.Index(text, "["); start != -1 {
        depth := 0
        for i := start; i < len(text); i++ {
            switch text[i] {
            case '[':
                depth++
            case ']':
                depth--
                if depth == 0 {
                    return text[start : i+1]
                }
            }
        }
    }

    return ""
}

// ExtractCodeBlocks extracts fenced code blocks
func ExtractCodeBlocks(output string) []CodeBlock {
    var blocks []CodeBlock

    re := regexp.MustCompile("(?s)```(\\w*)\\n(.*?)\\n```")
    matches := re.FindAllStringSubmatch(output, -1)

    for _, match := range matches {
        blocks = append(blocks, CodeBlock{
            Language: match[1],
            Content:  strings.TrimSpace(match[2]),
        })
    }

    return blocks
}

// CodeBlock represents a fenced code block
type CodeBlock struct {
    Language string
    Content  string
}

// ExtractSection extracts a markdown section by heading
func ExtractSection(output, heading string) string {
    // Find heading
    pattern := fmt.Sprintf(`(?m)^#{1,6}\s*%s\s*$`, regexp.QuoteMeta(heading))
    re := regexp.MustCompile(pattern)

    loc := re.FindStringIndex(output)
    if loc == nil {
        return ""
    }

    start := loc[1]

    // Find next heading of same or higher level
    nextHeading := regexp.MustCompile(`(?m)^#{1,6}\s+`)
    remaining := output[start:]

    if nextLoc := nextHeading.FindStringIndex(remaining); nextLoc != nil {
        return strings.TrimSpace(remaining[:nextLoc[0]])
    }

    return strings.TrimSpace(remaining)
}
```

### Usage Examples

```go
// Parse review result
type ReviewResult struct {
    Approved bool     `json:"approved"`
    Findings []Finding `json:"findings"`
}

type Finding struct {
    File     string `json:"file"`
    Line     int    `json:"line"`
    Severity string `json:"severity"`
    Message  string `json:"message"`
}

result, err := claude.Run(ctx, reviewPrompt)
if err != nil {
    return err
}

review, err := devflow.ParseJSON[ReviewResult](result.Output)
if err != nil {
    return fmt.Errorf("parse review: %w", err)
}

if !review.Approved {
    for _, f := range review.Findings {
        fmt.Printf("%s:%d [%s] %s\n", f.File, f.Line, f.Severity, f.Message)
    }
}

// Extract code from implementation
result, err := claude.Run(ctx, implementPrompt)
blocks := devflow.ExtractCodeBlocks(result.Output)

for _, block := range blocks {
    if block.Language == "go" {
        fmt.Println("Generated Go code:")
        fmt.Println(block.Content)
    }
}

// Extract specific section
result, err := claude.Run(ctx, specPrompt)
testPlan := devflow.ExtractSection(result.Output, "Test Plan")
```

## References

- [Claude CLI Output Formats](https://docs.anthropic.com/en/docs/claude-cli)
- ADR-006: Claude CLI Wrapper
- ADR-010: Session Management
