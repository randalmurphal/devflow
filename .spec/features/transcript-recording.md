# Feature: Transcript Recording

## Overview

Record AI conversations during workflow execution for debugging and auditing.

## Use Cases

1. **Debug failures**: See what Claude was asked and responded
2. **Audit trail**: Track all AI interactions
3. **Cost tracking**: Monitor token usage
4. **Quality analysis**: Review AI outputs over time

## API

### Start Recording

```go
store, _ := devflow.NewFileTranscriptStore(".devflow")

err := store.StartRun("run-2025-01-15-001", devflow.RunMetadata{
    FlowID: "ticket-to-pr",
    Input:  map[string]any{"ticketId": "TK-421"},
})
```

### Record Turns

```go
// User turn
store.RecordTurn(runID, devflow.Turn{
    Role:     "user",
    Content:  userPrompt,
    TokensIn: 1200,
})

// Assistant turn
store.RecordTurn(runID, devflow.Turn{
    Role:      "assistant",
    Content:   result.Output,
    TokensOut: result.TokensOut,
    ToolCalls: []devflow.ToolCall{
        {Name: "read_file", Input: map[string]any{"path": "api.go"}},
    },
})
```

### End Recording

```go
err := store.EndRun(runID, devflow.RunStatusCompleted)
// or
err := store.EndRun(runID, devflow.RunStatusFailed)
```

## Transcript Structure

```json
{
  "runId": "2025-01-15-ticket-to-pr-TK421",
  "metadata": {
    "flowId": "ticket-to-pr",
    "input": {"ticketId": "TK-421"},
    "startedAt": "2025-01-15T10:30:00Z",
    "endedAt": "2025-01-15T10:45:32Z",
    "status": "completed",
    "totalTokensIn": 5200,
    "totalTokensOut": 8400
  },
  "turns": [
    {
      "id": 1,
      "role": "system",
      "content": "You are an expert...",
      "timestamp": "2025-01-15T10:30:00Z"
    },
    {
      "id": 2,
      "role": "user",
      "content": "Generate a spec...",
      "tokensIn": 1500,
      "timestamp": "2025-01-15T10:30:01Z"
    },
    {
      "id": 3,
      "role": "assistant",
      "content": "# Technical Specification...",
      "tokensOut": 2500,
      "timestamp": "2025-01-15T10:30:45Z",
      "toolCalls": [
        {"name": "read_file", "input": {"path": "api.go"}}
      ]
    }
  ]
}
```

## Storage

### Directory Structure

```
.devflow/runs/
└── 2025-01-15-ticket-to-pr-TK421/
    ├── metadata.json      # Quick-access metadata
    └── transcript.json    # Full conversation
```

### Compression

- Files < 100KB: Stored as JSON
- Files >= 100KB: Compressed with gzip

## Integration with Claude

```go
// Auto-record Claude calls
func recordingRun(
    claude *ClaudeCLI,
    store TranscriptManager,
    runID string,
    prompt string,
    opts ...RunOption,
) (*RunResult, error) {
    // Record user turn
    store.RecordTurn(runID, Turn{
        Role:    "user",
        Content: prompt,
    })

    // Run Claude
    result, err := claude.Run(ctx, prompt, opts...)
    if err != nil {
        return nil, err
    }

    // Record assistant turn
    store.RecordTurn(runID, Turn{
        Role:      "assistant",
        Content:   result.Output,
        TokensOut: result.TokensOut,
    })

    return result, nil
}
```

## Node Wrapper

```go
// Wrap nodes with automatic transcript recording
func WithTranscript(node NodeFunc[DevState]) NodeFunc[DevState] {
    return func(ctx flowgraph.Context, state DevState) (DevState, error) {
        store := TranscriptsFromContext(ctx)
        if store == nil {
            return node(ctx, state)
        }

        // Record before
        // ... execute node ...
        // Record after

        return node(ctx, state)
    }
}
```

## Example

```go
store, _ := devflow.NewFileTranscriptStore(".devflow")
claude, _ := devflow.NewClaudeCLI(devflow.ClaudeConfig{})

runID := fmt.Sprintf("%s-manual", time.Now().Format("2006-01-02"))

// Start
store.StartRun(runID, devflow.RunMetadata{
    FlowID: "manual-run",
})

// Record system prompt
store.RecordTurn(runID, devflow.Turn{
    Role:    "system",
    Content: "You are an expert Go developer.",
})

// Run and record
prompt := "Explain how to write a web server in Go"
store.RecordTurn(runID, devflow.Turn{
    Role:    "user",
    Content: prompt,
})

result, _ := claude.Run(ctx, prompt,
    devflow.WithSystemPrompt("You are an expert Go developer."),
)

store.RecordTurn(runID, devflow.Turn{
    Role:      "assistant",
    Content:   result.Output,
    TokensOut: result.TokensOut,
})

// End
store.EndRun(runID, devflow.RunStatusCompleted)

// View later
transcript, _ := store.Load(runID)
fmt.Printf("Run had %d turns\n", len(transcript.Turns))
```

## Testing

```go
func TestTranscriptRecording(t *testing.T) {
    dir := t.TempDir()
    store, _ := devflow.NewFileTranscriptStore(dir)

    // Start run
    store.StartRun("test-run", devflow.RunMetadata{FlowID: "test"})

    // Record turns
    store.RecordTurn("test-run", devflow.Turn{Role: "user", Content: "Hello"})
    store.RecordTurn("test-run", devflow.Turn{Role: "assistant", Content: "Hi!"})

    // End
    store.EndRun("test-run", devflow.RunStatusCompleted)

    // Verify
    transcript, _ := store.Load("test-run")
    assert.Len(t, transcript.Turns, 2)
    assert.Equal(t, "completed", string(transcript.Metadata.Status))
}
```

## References

- ADR-011: Transcript Format
- ADR-012: Transcript Storage
