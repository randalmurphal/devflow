# ADR-012: Transcript Storage

## Status

Accepted

## Context

Transcripts need to be stored persistently for:

1. Post-run analysis
2. Debugging failures
3. Audit trails
4. Cost tracking

We need to decide where and how to store transcripts.

## Decision

### 1. File-Based Storage

Transcripts are stored as files in a predictable directory structure:

```
.devflow/
└── runs/
    ├── 2025-01-15-ticket-to-pr-TK421/
    │   ├── metadata.json
    │   ├── transcript.json
    │   └── artifacts/
    ├── 2025-01-15-ticket-to-pr-TK422/
    │   ├── metadata.json
    │   ├── transcript.json.gz    # Compressed
    │   └── artifacts/
    └── index.json                 # Optional index
```

### 2. Directory per Run

Each run gets its own directory containing:

| File | Purpose |
|------|---------|
| `metadata.json` | Quick-access run info |
| `transcript.json` | Full conversation |
| `artifacts/` | Generated files |

### 3. Metadata File (Fast Access)

Separate metadata file for quick listing without loading full transcript:

```json
{
  "runId": "2025-01-15-ticket-to-pr-TK421",
  "flowId": "ticket-to-pr",
  "status": "completed",
  "startedAt": "2025-01-15T10:30:00Z",
  "endedAt": "2025-01-15T10:45:32Z",
  "totalTokensIn": 5200,
  "totalTokensOut": 8400,
  "totalCost": 0.12,
  "turnCount": 8
}
```

### 4. TranscriptManager Interface

```go
type TranscriptManager interface {
    // Lifecycle
    StartRun(runID string, metadata RunMetadata) error
    RecordTurn(runID string, turn Turn) error
    EndRun(runID string, status RunStatus) error

    // Retrieval
    Load(runID string) (*Transcript, error)
    LoadMetadata(runID string) (*TranscriptMeta, error)
    List(filter ListFilter) ([]TranscriptMeta, error)

    // Maintenance
    Delete(runID string) error
    Archive(runID string) error
}
```

### 5. Optional Index File

For quick listing, maintain an index:

```json
{
  "runs": [
    {
      "runId": "2025-01-15-ticket-to-pr-TK421",
      "flowId": "ticket-to-pr",
      "status": "completed",
      "startedAt": "2025-01-15T10:30:00Z"
    },
    {
      "runId": "2025-01-15-ticket-to-pr-TK422",
      "flowId": "ticket-to-pr",
      "status": "failed",
      "startedAt": "2025-01-15T11:00:00Z"
    }
  ],
  "lastUpdated": "2025-01-15T12:00:00Z"
}
```

Index is rebuilt on demand if missing.

## Alternatives Considered

### Alternative 1: SQLite Database

Store transcripts in SQLite.

**Deferred:**
- Better for complex queries
- Can add as enhancement
- Files work for MVP

### Alternative 2: Single Large File

Append all transcripts to one file.

**Rejected because:**
- Hard to manage/query
- Can't delete individual runs
- File grows unbounded

### Alternative 3: Cloud Storage

Store in S3/GCS.

**Deferred:**
- Good for task-keeper
- devflow should work offline
- Can add as backend option

## Consequences

### Positive

- **Simple**: Just files
- **Portable**: Easy to backup/move
- **Debuggable**: Can inspect with any text editor
- **Offline**: No external services needed

### Negative

- **Scale**: May get slow with many runs
- **Query**: Complex queries need full scan
- **Concurrent**: File locking for concurrent writes

### Mitigations

1. **Index file**: Fast listing
2. **Cleanup policy**: Remove old runs
3. **Archive**: Move old runs to archive directory

## Code Example

```go
package devflow

import (
    "encoding/json"
    "os"
    "path/filepath"
    "sort"
    "sync"
    "time"
)

// FileTranscriptStore stores transcripts as files
type FileTranscriptStore struct {
    baseDir string
    mu      sync.RWMutex
    active  map[string]*activeRun
}

type activeRun struct {
    transcript *Transcript
    file       *os.File
}

// NewFileTranscriptStore creates a file-based transcript store
func NewFileTranscriptStore(baseDir string) (*FileTranscriptStore, error) {
    runsDir := filepath.Join(baseDir, "runs")
    if err := os.MkdirAll(runsDir, 0755); err != nil {
        return nil, err
    }

    return &FileTranscriptStore{
        baseDir: baseDir,
        active:  make(map[string]*activeRun),
    }, nil
}

// StartRun begins a new transcript
func (s *FileTranscriptStore) StartRun(runID string, meta RunMetadata) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    if _, exists := s.active[runID]; exists {
        return fmt.Errorf("run already exists: %s", runID)
    }

    runDir := filepath.Join(s.baseDir, "runs", runID)
    if err := os.MkdirAll(runDir, 0755); err != nil {
        return err
    }

    transcript := &Transcript{
        RunID: runID,
        Metadata: TranscriptMeta{
            FlowID:    meta.FlowID,
            NodeID:    meta.NodeID,
            Input:     meta.Input,
            StartedAt: time.Now(),
            Status:    RunStatusRunning,
        },
    }

    // Write initial metadata
    if err := s.writeMetadata(runID, &transcript.Metadata); err != nil {
        return err
    }

    s.active[runID] = &activeRun{
        transcript: transcript,
    }

    return nil
}

// RecordTurn adds a turn to an active transcript
func (s *FileTranscriptStore) RecordTurn(runID string, turn Turn) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    active, ok := s.active[runID]
    if !ok {
        return fmt.Errorf("run not found: %s", runID)
    }

    turn.ID = len(active.transcript.Turns) + 1
    if turn.Timestamp.IsZero() {
        turn.Timestamp = time.Now()
    }

    active.transcript.Turns = append(active.transcript.Turns, turn)

    // Update token counts
    switch turn.Role {
    case "user", "system":
        active.transcript.Metadata.TotalTokensIn += turn.TokensIn
    case "assistant":
        active.transcript.Metadata.TotalTokensOut += turn.TokensOut
    }

    return nil
}

// EndRun completes a transcript
func (s *FileTranscriptStore) EndRun(runID string, status RunStatus) error {
    s.mu.Lock()
    defer s.mu.Unlock()

    active, ok := s.active[runID]
    if !ok {
        return fmt.Errorf("run not found: %s", runID)
    }

    active.transcript.Metadata.Status = status
    active.transcript.Metadata.EndedAt = time.Now()

    // Save full transcript
    if err := active.transcript.Save(s.baseDir); err != nil {
        return err
    }

    // Update metadata
    if err := s.writeMetadata(runID, &active.transcript.Metadata); err != nil {
        return err
    }

    // Update index
    s.updateIndex()

    delete(s.active, runID)
    return nil
}

// Load retrieves a complete transcript
func (s *FileTranscriptStore) Load(runID string) (*Transcript, error) {
    return LoadTranscript(s.baseDir, runID)
}

// LoadMetadata retrieves just the metadata
func (s *FileTranscriptStore) LoadMetadata(runID string) (*TranscriptMeta, error) {
    path := filepath.Join(s.baseDir, "runs", runID, "metadata.json")
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }

    var meta TranscriptMeta
    if err := json.Unmarshal(data, &meta); err != nil {
        return nil, err
    }

    return &meta, nil
}

// List returns metadata for runs matching filter
func (s *FileTranscriptStore) List(filter ListFilter) ([]TranscriptMeta, error) {
    runsDir := filepath.Join(s.baseDir, "runs")
    entries, err := os.ReadDir(runsDir)
    if err != nil {
        return nil, err
    }

    var results []TranscriptMeta

    for _, entry := range entries {
        if !entry.IsDir() {
            continue
        }

        meta, err := s.LoadMetadata(entry.Name())
        if err != nil {
            continue
        }

        // Apply filters
        if filter.FlowID != "" && meta.FlowID != filter.FlowID {
            continue
        }
        if filter.Status != "" && meta.Status != filter.Status {
            continue
        }
        if !filter.After.IsZero() && meta.StartedAt.Before(filter.After) {
            continue
        }

        results = append(results, *meta)
    }

    // Sort by start time (newest first)
    sort.Slice(results, func(i, j int) bool {
        return results[i].StartedAt.After(results[j].StartedAt)
    })

    // Apply limit
    if filter.Limit > 0 && len(results) > filter.Limit {
        results = results[:filter.Limit]
    }

    return results, nil
}

// Delete removes a run
func (s *FileTranscriptStore) Delete(runID string) error {
    runDir := filepath.Join(s.baseDir, "runs", runID)
    if err := os.RemoveAll(runDir); err != nil {
        return err
    }
    s.updateIndex()
    return nil
}

func (s *FileTranscriptStore) writeMetadata(runID string, meta *TranscriptMeta) error {
    path := filepath.Join(s.baseDir, "runs", runID, "metadata.json")
    data, err := json.MarshalIndent(meta, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, 0644)
}

func (s *FileTranscriptStore) updateIndex() {
    // Best effort - index is optional
    // Implementation would rebuild index from metadata files
}

// ListFilter filters transcript listing
type ListFilter struct {
    FlowID string
    Status RunStatus
    After  time.Time
    Limit  int
}
```

### Usage

```go
// Create store
store, err := devflow.NewFileTranscriptStore(".devflow")
if err != nil {
    return err
}

// Start recording
err = store.StartRun("2025-01-15-ticket-to-pr-TK421", devflow.RunMetadata{
    FlowID: "ticket-to-pr",
    Input:  map[string]any{"ticketId": "TK-421"},
})

// Record turns
store.RecordTurn(runID, devflow.Turn{
    Role:     "user",
    Content:  userPrompt,
    TokensIn: 1200,
})

store.RecordTurn(runID, devflow.Turn{
    Role:      "assistant",
    Content:   result.Output,
    TokensOut: result.TokensOut,
})

// End run
store.EndRun(runID, devflow.RunStatusCompleted)

// List recent runs
runs, _ := store.List(devflow.ListFilter{
    FlowID: "ticket-to-pr",
    Limit:  10,
})
for _, run := range runs {
    fmt.Printf("%s: %s (%s)\n", run.RunID, run.Status, run.EndedAt)
}

// Load full transcript
transcript, _ := store.Load("2025-01-15-ticket-to-pr-TK421")
for _, turn := range transcript.Turns {
    fmt.Printf("[%s] %s\n", turn.Role, turn.Content[:100])
}
```

## References

- ADR-011: Transcript Format
- ADR-015: Artifact Structure (shares directory)
