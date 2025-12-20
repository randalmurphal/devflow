# transcript package

AI conversation transcript recording, storage, and search.

## Quick Reference

| Type | Purpose |
|------|---------|
| `Transcript` | Complete conversation record |
| `Turn` | Single message in conversation |
| `Meta` | Transcript metadata (loaded separately) |
| `Manager` | Interface for transcript operations |
| `FileStore` | File-based Manager implementation |
| `Searcher` | Grep-based transcript search |
| `Viewer` | Display and export transcripts |

## Manager Interface

```go
type Manager interface {
    StartRun(runID string, meta RunMetadata) error
    RecordTurn(runID string, turn Turn) error
    EndRun(runID string, status RunStatus) error
    Load(runID string) (*Transcript, error)
    List(filter ListFilter) ([]Meta, error)
}
```

## Run Lifecycle

```go
store, _ := transcript.NewFileStore(transcript.StoreConfig{
    BaseDir: ".devflow/runs",
})

// Start run
store.StartRun("run-123", transcript.RunMetadata{
    FlowID: "ticket-to-pr",
    Input:  map[string]any{"ticket": "TK-421"},
})

// Record turns
store.RecordTurn("run-123", transcript.Turn{
    Role:    "assistant",
    Content: "I'll implement this...",
})

// End run
store.EndRun("run-123", transcript.RunStatusCompleted)
```

## Run Status

| Status | When |
|--------|------|
| `RunStatusRunning` | Active run |
| `RunStatusCompleted` | Finished successfully |
| `RunStatusFailed` | Finished with error |
| `RunStatusCanceled` | User canceled |

## Search

```go
searcher := transcript.NewSearcher(baseDir)

// Find by flow ID
results, _ := searcher.FindByFlow("ticket-to-pr")

// Find by status
results, _ := searcher.FindByStatus(transcript.RunStatusCompleted)

// Custom filter
results, _ := searcher.Find(func(m transcript.Meta) bool {
    return m.TurnCount > 10
})
```

## View/Export

```go
viewer := transcript.NewViewer(true) // color output

// Summary view
viewer.ViewSummary(os.Stdout, transcript)

// Full transcript
viewer.ViewFull(os.Stdout, transcript)

// Export to markdown
viewer.ExportMarkdown(os.Stdout, transcript)
```

## File Structure

```
transcript/
├── transcript.go  # Core types (Transcript, Turn, Meta)
├── manager.go     # Manager interface, ListFilter
├── store.go       # FileStore implementation
├── search.go      # Searcher
└── view.go        # Viewer
```
