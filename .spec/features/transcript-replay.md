# Feature: Transcript Replay

## Overview

View and analyze recorded transcripts for debugging and auditing.

## Use Cases

1. **Debug failed runs**: See exactly what happened
2. **Review AI outputs**: Check quality of generated content
3. **Compare runs**: See differences between runs
4. **Export for sharing**: Generate markdown reports

## API

### Load Transcript

```go
store, _ := devflow.NewFileTranscriptStore(".devflow")
transcript, err := store.Load("2025-01-15-ticket-to-pr-TK421")
```

### View Full

```go
viewer := devflow.NewTranscriptViewer(true) // colorEnabled
viewer.ViewFull(os.Stdout, transcript)
```

### View Summary

```go
viewer.ViewSummary(os.Stdout, transcript)
```

### Export Markdown

```go
f, _ := os.Create("transcript.md")
defer f.Close()
viewer.ExportMarkdown(f, transcript)
```

### Compare Runs

```go
t1, _ := store.Load("run-001")
t2, _ := store.Load("run-002")
viewer.Diff(os.Stdout, t1, t2)
```

## View Formats

### Full View (Terminal)

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

### Summary View

```
═══════════════════════════════════════════════════════════
Run: 2025-01-15-ticket-to-pr-TK421
Flow: ticket-to-pr | Status: completed
Tokens: 5,200 in / 8,400 out | Cost: $0.12
═══════════════════════════════════════════════════════════

Turn Summary:
  [1] system: You are an expert software architect...
  [2] user: Generate a technical specification for TK-421...
  [3] assistant: # Technical Specification: User Authentica...
```

### Markdown Export

```markdown
# Transcript: 2025-01-15-ticket-to-pr-TK421

## Metadata
- **Flow**: ticket-to-pr
- **Status**: completed
- **Started**: 2025-01-15T10:30:00Z
- **Ended**: 2025-01-15T10:45:32Z
- **Tokens**: 5,200 in / 8,400 out
- **Cost**: $0.12

## Conversation

### System
You are an expert software architect...

### User
Generate a technical specification for TK-421...

### Assistant
# Technical Specification: User Authentication
...
```

### Diff View

```
Comparing:
  A: run-001 (completed)
  B: run-002 (completed)

Turns: 5 vs 6
Tokens In: 3200 vs 3400
Tokens Out: 5100 vs 5800

Assistant Output Comparison:
  Turn 3: different (1200 vs 1400 chars)
  Turn 5: identical
```

## CLI Commands

```bash
# View full transcript
devflow transcript view run-2025-01-15-001

# View summary
devflow transcript view --summary run-2025-01-15-001

# Export to markdown
devflow transcript export --format md run-2025-01-15-001 > transcript.md

# Compare runs
devflow transcript diff run-001 run-002
```

## Example

```go
store, _ := devflow.NewFileTranscriptStore(".devflow")
viewer := devflow.NewTranscriptViewer(true)

// List recent runs
runs, _ := store.List(devflow.ListFilter{
    FlowID: "ticket-to-pr",
    Limit:  5,
})

fmt.Println("Recent runs:")
for _, run := range runs {
    fmt.Printf("  %s: %s (%s)\n", run.RunID, run.Status, run.EndedAt.Format("15:04"))
}

// View specific run
if len(runs) > 0 {
    transcript, _ := store.Load(runs[0].RunID)
    viewer.ViewFull(os.Stdout, transcript)
}

// Export failed run for debugging
failedRuns, _ := store.List(devflow.ListFilter{
    Status: devflow.RunStatusFailed,
    Limit:  1,
})

if len(failedRuns) > 0 {
    transcript, _ := store.Load(failedRuns[0].RunID)
    f, _ := os.Create("failed-run.md")
    viewer.ExportMarkdown(f, transcript)
    f.Close()
    fmt.Println("Exported failed run to failed-run.md")
}
```

## Testing

```go
func TestTranscriptViewer(t *testing.T) {
    transcript := &devflow.Transcript{
        RunID: "test-run",
        Metadata: devflow.TranscriptMeta{
            FlowID: "test",
            Status: devflow.RunStatusCompleted,
        },
        Turns: []devflow.Turn{
            {ID: 1, Role: "user", Content: "Hello"},
            {ID: 2, Role: "assistant", Content: "Hi!"},
        },
    }

    viewer := devflow.NewTranscriptViewer(false)

    var buf bytes.Buffer
    err := viewer.ViewSummary(&buf, transcript)
    require.NoError(t, err)
    assert.Contains(t, buf.String(), "test-run")
    assert.Contains(t, buf.String(), "completed")
}
```

## References

- ADR-013: Transcript Replay
- Feature: Transcript Recording
