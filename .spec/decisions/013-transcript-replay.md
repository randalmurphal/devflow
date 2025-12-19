# ADR-013: Transcript Replay

## Status

Accepted

## Context

Transcript replay is useful for:

1. Debugging failed runs (what did Claude see/say?)
2. Reproducing issues
3. Demonstrating workflows
4. Creating test fixtures

We need to decide how replay works.

## Decision

### 1. Read-Only Replay

Replay is primarily read-only visualization, not re-execution:

```go
// Load and display
transcript, _ := store.Load(runID)
for _, turn := range transcript.Turns {
    fmt.Printf("[%s] %s\n\n", turn.Role, turn.Content)
}
```

Re-execution would require running Claude again with the same prompts, which is expensive and may produce different results.

### 2. Replay Modes

| Mode | Description | Use Case |
|------|-------------|----------|
| `full` | Show all turns | Debugging |
| `summary` | Show metadata + key turns | Quick review |
| `assistant-only` | Show only Claude responses | Output review |
| `diff` | Compare two runs | Regression testing |

### 3. TranscriptViewer Interface

```go
type TranscriptViewer interface {
    // Display modes
    ViewFull(w io.Writer, t *Transcript) error
    ViewSummary(w io.Writer, t *Transcript) error
    ViewTurn(w io.Writer, turn Turn) error

    // Comparison
    Diff(w io.Writer, a, b *Transcript) error

    // Export
    ExportMarkdown(w io.Writer, t *Transcript) error
    ExportHTML(w io.Writer, t *Transcript) error
}
```

### 4. CLI Integration

```bash
# View full transcript
devflow transcript view run-2025-01-15-001

# View summary
devflow transcript view --summary run-2025-01-15-001

# Export to markdown
devflow transcript export --format md run-2025-01-15-001 > transcript.md

# Compare two runs
devflow transcript diff run-001 run-002

# Follow active run
devflow transcript follow run-2025-01-15-001
```

### 5. Format Options

**Terminal output**:
```
═══════════════════════════════════════════════════════════
Run: 2025-01-15-ticket-to-pr-TK421
Flow: ticket-to-pr | Status: completed
Started: 2025-01-15 10:30:00 | Duration: 15m32s
Tokens: 5,200 in / 8,400 out | Cost: $0.12
═══════════════════════════════════════════════════════════

[1] SYSTEM (10:30:00)
───────────────────────────────────────────────────────────
You are an expert software architect...

[2] USER (10:30:01) [1,200 tokens]
───────────────────────────────────────────────────────────
Generate a technical specification for TK-421...

[3] ASSISTANT (10:30:45) [2,500 tokens]
───────────────────────────────────────────────────────────
# Technical Specification: User Authentication

## Overview
...
```

**Markdown export**:
```markdown
# Transcript: 2025-01-15-ticket-to-pr-TK421

## Metadata
- **Flow**: ticket-to-pr
- **Status**: completed
- **Duration**: 15m32s
- **Tokens**: 5,200 in / 8,400 out

## Conversation

### System
You are an expert software architect...

### User
Generate a technical specification for TK-421...

### Assistant
# Technical Specification: User Authentication
...
```

## Alternatives Considered

### Alternative 1: Re-Execution

Actually re-run the prompts through Claude.

**Rejected because:**
- Expensive (token costs)
- Non-deterministic results
- Not useful for debugging

### Alternative 2: Interactive Replay

Step through turns interactively.

**Deferred:**
- Good for TUI
- Not needed for MVP
- Can add later

### Alternative 3: Video Recording

Record screen during run.

**Rejected because:**
- Overkill
- Large files
- Not practical

## Consequences

### Positive

- **Simple**: Just reading and formatting
- **Cheap**: No API calls
- **Complete**: Full conversation available
- **Exportable**: Multiple output formats

### Negative

- **Read-only**: Can't modify and re-run
- **Static**: Can't interact with conversation

## Code Example

```go
package devflow

import (
    "fmt"
    "io"
    "strings"
    "time"
)

// TranscriptViewer displays transcripts
type TranscriptViewer struct {
    colorEnabled bool
}

// NewTranscriptViewer creates a viewer
func NewTranscriptViewer(colorEnabled bool) *TranscriptViewer {
    return &TranscriptViewer{colorEnabled: colorEnabled}
}

// ViewFull displays the complete transcript
func (v *TranscriptViewer) ViewFull(w io.Writer, t *Transcript) error {
    // Header
    v.writeHeader(w, t)

    // Turns
    for _, turn := range t.Turns {
        v.writeTurn(w, turn)
    }

    return nil
}

// ViewSummary displays a brief summary
func (v *TranscriptViewer) ViewSummary(w io.Writer, t *Transcript) error {
    v.writeHeader(w, t)

    fmt.Fprintf(w, "\nTurn Summary:\n")
    for _, turn := range t.Turns {
        preview := turn.Content
        if len(preview) > 100 {
            preview = preview[:100] + "..."
        }
        preview = strings.ReplaceAll(preview, "\n", " ")
        fmt.Fprintf(w, "  [%d] %s: %s\n", turn.ID, turn.Role, preview)
    }

    return nil
}

func (v *TranscriptViewer) writeHeader(w io.Writer, t *Transcript) {
    sep := strings.Repeat("═", 60)

    fmt.Fprintln(w, sep)
    fmt.Fprintf(w, "Run: %s\n", t.RunID)
    fmt.Fprintf(w, "Flow: %s | Status: %s\n", t.Metadata.FlowID, t.Metadata.Status)

    duration := t.Metadata.EndedAt.Sub(t.Metadata.StartedAt)
    fmt.Fprintf(w, "Started: %s | Duration: %s\n",
        t.Metadata.StartedAt.Format("2006-01-02 15:04:05"),
        duration.Round(time.Second))

    fmt.Fprintf(w, "Tokens: %d in / %d out | Cost: $%.2f\n",
        t.Metadata.TotalTokensIn,
        t.Metadata.TotalTokensOut,
        t.Metadata.TotalCost)

    fmt.Fprintln(w, sep)
}

func (v *TranscriptViewer) writeTurn(w io.Writer, turn Turn) {
    fmt.Fprintln(w)

    // Turn header
    header := fmt.Sprintf("[%d] %s (%s)",
        turn.ID,
        strings.ToUpper(turn.Role),
        turn.Timestamp.Format("15:04:05"))

    if turn.TokensIn > 0 {
        header += fmt.Sprintf(" [%d tokens in]", turn.TokensIn)
    }
    if turn.TokensOut > 0 {
        header += fmt.Sprintf(" [%d tokens out]", turn.TokensOut)
    }

    fmt.Fprintln(w, header)
    fmt.Fprintln(w, strings.Repeat("─", 60))

    // Content
    fmt.Fprintln(w, turn.Content)

    // Tool calls
    for _, tc := range turn.ToolCalls {
        fmt.Fprintf(w, "\n  📎 Tool: %s\n", tc.Name)
        fmt.Fprintf(w, "     Input: %v\n", tc.Input)
        if tc.Output != "" {
            output := tc.Output
            if len(output) > 200 {
                output = output[:200] + "..."
            }
            fmt.Fprintf(w, "     Output: %s\n", output)
        }
    }
}

// ExportMarkdown exports to markdown format
func (v *TranscriptViewer) ExportMarkdown(w io.Writer, t *Transcript) error {
    fmt.Fprintf(w, "# Transcript: %s\n\n", t.RunID)

    // Metadata
    fmt.Fprintf(w, "## Metadata\n\n")
    fmt.Fprintf(w, "- **Flow**: %s\n", t.Metadata.FlowID)
    fmt.Fprintf(w, "- **Status**: %s\n", t.Metadata.Status)
    fmt.Fprintf(w, "- **Started**: %s\n", t.Metadata.StartedAt.Format(time.RFC3339))
    fmt.Fprintf(w, "- **Ended**: %s\n", t.Metadata.EndedAt.Format(time.RFC3339))
    fmt.Fprintf(w, "- **Tokens**: %d in / %d out\n",
        t.Metadata.TotalTokensIn, t.Metadata.TotalTokensOut)
    fmt.Fprintf(w, "- **Cost**: $%.2f\n\n", t.Metadata.TotalCost)

    // Conversation
    fmt.Fprintf(w, "## Conversation\n\n")

    for _, turn := range t.Turns {
        fmt.Fprintf(w, "### %s\n\n", strings.Title(turn.Role))
        fmt.Fprintf(w, "%s\n\n", turn.Content)

        for _, tc := range turn.ToolCalls {
            fmt.Fprintf(w, "**Tool Call**: `%s`\n\n", tc.Name)
            fmt.Fprintf(w, "```json\n%s\n```\n\n", formatJSON(tc.Input))
        }
    }

    return nil
}

// Diff compares two transcripts
func (v *TranscriptViewer) Diff(w io.Writer, a, b *Transcript) error {
    fmt.Fprintf(w, "Comparing:\n")
    fmt.Fprintf(w, "  A: %s (%s)\n", a.RunID, a.Metadata.Status)
    fmt.Fprintf(w, "  B: %s (%s)\n", b.RunID, b.Metadata.Status)
    fmt.Fprintln(w)

    // Compare turn counts
    fmt.Fprintf(w, "Turns: %d vs %d\n", len(a.Turns), len(b.Turns))

    // Compare tokens
    fmt.Fprintf(w, "Tokens In: %d vs %d\n",
        a.Metadata.TotalTokensIn, b.Metadata.TotalTokensIn)
    fmt.Fprintf(w, "Tokens Out: %d vs %d\n",
        a.Metadata.TotalTokensOut, b.Metadata.TotalTokensOut)

    // Compare assistant outputs (simplified diff)
    fmt.Fprintln(w, "\nAssistant Output Comparison:")
    maxTurns := len(a.Turns)
    if len(b.Turns) > maxTurns {
        maxTurns = len(b.Turns)
    }

    for i := 0; i < maxTurns; i++ {
        var turnA, turnB *Turn
        if i < len(a.Turns) {
            turnA = &a.Turns[i]
        }
        if i < len(b.Turns) {
            turnB = &b.Turns[i]
        }

        if turnA != nil && turnB != nil && turnA.Role == "assistant" && turnB.Role == "assistant" {
            if turnA.Content == turnB.Content {
                fmt.Fprintf(w, "  Turn %d: identical\n", i+1)
            } else {
                fmt.Fprintf(w, "  Turn %d: different (%d vs %d chars)\n",
                    i+1, len(turnA.Content), len(turnB.Content))
            }
        }
    }

    return nil
}
```

### Usage

```go
// Load transcript
transcript, err := store.Load("2025-01-15-ticket-to-pr-TK421")
if err != nil {
    return err
}

// View in terminal
viewer := devflow.NewTranscriptViewer(true)
viewer.ViewFull(os.Stdout, transcript)

// Export to markdown
f, _ := os.Create("transcript.md")
defer f.Close()
viewer.ExportMarkdown(f, transcript)

// Compare two runs
t1, _ := store.Load("run-001")
t2, _ := store.Load("run-002")
viewer.Diff(os.Stdout, t1, t2)
```

### CLI Implementation

```go
// cmd/devflow/transcript.go

var transcriptCmd = &cobra.Command{
    Use:   "transcript",
    Short: "Manage transcripts",
}

var viewCmd = &cobra.Command{
    Use:   "view [run-id]",
    Short: "View a transcript",
    Args:  cobra.ExactArgs(1),
    Run: func(cmd *cobra.Command, args []string) {
        store, _ := devflow.NewFileTranscriptStore(".devflow")
        transcript, err := store.Load(args[0])
        if err != nil {
            log.Fatal(err)
        }

        viewer := devflow.NewTranscriptViewer(true)

        summary, _ := cmd.Flags().GetBool("summary")
        if summary {
            viewer.ViewSummary(os.Stdout, transcript)
        } else {
            viewer.ViewFull(os.Stdout, transcript)
        }
    },
}

func init() {
    viewCmd.Flags().Bool("summary", false, "Show summary only")
    transcriptCmd.AddCommand(viewCmd)
}
```

## References

- ADR-011: Transcript Format
- ADR-012: Transcript Storage
- ADR-014: Transcript Search
