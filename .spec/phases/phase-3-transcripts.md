# Phase 3: Transcript Management

## Overview

Implement transcript recording and storage for AI conversations.

**Duration**: Week 3
**Dependencies**: Phase 2 (Claude CLI)
**Deliverables**: `TranscriptManager` with recording, storage, and search

---

## Goals

1. Record all conversation turns during execution
2. Store transcripts as JSON files with compression
3. Support search and listing
4. Enable replay for debugging

---

## Components

### TranscriptManager

Central type for transcript operations:

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
}
```

### Transcript

Complete conversation record:

```go
type Transcript struct {
    RunID    string          `json:"runId"`
    Metadata TranscriptMeta  `json:"metadata"`
    Turns    []Turn          `json:"turns"`
}
```

### Turn

Single conversation turn:

```go
type Turn struct {
    ID        int        `json:"id"`
    Role      string     `json:"role"`
    Content   string     `json:"content"`
    TokensIn  int        `json:"tokensIn,omitempty"`
    TokensOut int        `json:"tokensOut,omitempty"`
    Timestamp time.Time  `json:"timestamp"`
    ToolCalls []ToolCall `json:"toolCalls,omitempty"`
}
```

---

## Implementation Tasks

### Task 3.1: Transcript Type

```go
type Transcript struct {
    RunID    string          `json:"runId"`
    Metadata TranscriptMeta  `json:"metadata"`
    Turns    []Turn          `json:"turns"`
}

func NewTranscript(runID, flowID string) *Transcript
func (t *Transcript) AddTurn(role, content string, tokens int) *Turn
func (t *Transcript) Complete()
func (t *Transcript) Fail(err error)
```

**Acceptance Criteria**:
- [ ] Tracks all turns with timestamps
- [ ] Accumulates token counts
- [ ] Serializes to JSON correctly

### Task 3.2: FileTranscriptStore

```go
type FileTranscriptStore struct {
    baseDir string
    active  map[string]*activeRun
}

func NewFileTranscriptStore(baseDir string) (*FileTranscriptStore, error)
```

**Acceptance Criteria**:
- [ ] Creates directory structure
- [ ] Handles concurrent access
- [ ] Stores metadata separately for quick listing

### Task 3.3: Recording Lifecycle

```go
func (s *FileTranscriptStore) StartRun(runID string, meta RunMetadata) error
func (s *FileTranscriptStore) RecordTurn(runID string, turn Turn) error
func (s *FileTranscriptStore) EndRun(runID string, status RunStatus) error
```

**Acceptance Criteria**:
- [ ] StartRun creates directories
- [ ] RecordTurn appends to in-memory transcript
- [ ] EndRun saves to disk and cleans up

### Task 3.4: Compression

```go
func (t *Transcript) Save(baseDir string) error
```

**Acceptance Criteria**:
- [ ] Compresses if > 100KB
- [ ] Uses gzip format
- [ ] Decompresses transparently on load

### Task 3.5: Listing and Filtering

```go
type ListFilter struct {
    FlowID string
    Status RunStatus
    After  time.Time
    Limit  int
}

func (s *FileTranscriptStore) List(filter ListFilter) ([]TranscriptMeta, error)
```

**Acceptance Criteria**:
- [ ] Filters by flow ID
- [ ] Filters by status
- [ ] Filters by date range
- [ ] Returns sorted by date (newest first)

### Task 3.6: Search

```go
type TranscriptSearcher struct {
    baseDir string
}

func (s *TranscriptSearcher) SearchContent(query string, opts SearchOptions) ([]SearchResult, error)
```

**Acceptance Criteria**:
- [ ] Uses ripgrep or grep for content search
- [ ] Returns matching run IDs and context
- [ ] Handles large transcript directories

### Task 3.7: Viewer

```go
type TranscriptViewer struct {
    colorEnabled bool
}

func (v *TranscriptViewer) ViewFull(w io.Writer, t *Transcript) error
func (v *TranscriptViewer) ViewSummary(w io.Writer, t *Transcript) error
func (v *TranscriptViewer) ExportMarkdown(w io.Writer, t *Transcript) error
```

**Acceptance Criteria**:
- [ ] Full view shows all turns
- [ ] Summary shows metadata and turn count
- [ ] Markdown export is well-formatted

---

## Directory Structure

```
.devflow/runs/
└── 2025-01-15-ticket-to-pr-TK421/
    ├── metadata.json      # Quick-access metadata
    ├── transcript.json    # < 100KB
    └── transcript.json.gz # >= 100KB, compressed
```

### metadata.json

```json
{
  "runId": "2025-01-15-ticket-to-pr-TK421",
  "flowId": "ticket-to-pr",
  "status": "completed",
  "startedAt": "2025-01-15T10:30:00Z",
  "endedAt": "2025-01-15T10:45:32Z",
  "totalTokensIn": 5200,
  "totalTokensOut": 8400,
  "turnCount": 8
}
```

---

## Testing Strategy

### Unit Tests

| Test | Description |
|------|-------------|
| `TestTranscript_AddTurn` | Adds turns correctly |
| `TestTranscript_TokenCounting` | Accumulates tokens |
| `TestTranscript_Serialize` | JSON roundtrip |
| `TestFileTranscriptStore_Lifecycle` | Start, record, end |
| `TestFileTranscriptStore_List` | Filtering works |

### Integration Tests

```go
func TestFileTranscriptStore_Integration(t *testing.T) {
    dir := t.TempDir()
    store, _ := NewFileTranscriptStore(dir)

    // Start run
    store.StartRun("run-001", RunMetadata{FlowID: "test"})

    // Record turns
    store.RecordTurn("run-001", Turn{Role: "user", Content: "Hello"})
    store.RecordTurn("run-001", Turn{Role: "assistant", Content: "Hi!"})

    // End run
    store.EndRun("run-001", RunStatusCompleted)

    // Verify
    loaded, _ := store.Load("run-001")
    assert.Len(t, loaded.Turns, 2)
}
```

---

## Error Handling

| Error | Condition | Recovery |
|-------|-----------|----------|
| `ErrRunNotFound` | Run ID doesn't exist | Check run ID |
| `ErrRunNotStarted` | Recording before start | Call StartRun |
| `ErrRunAlreadyEnded` | Recording after end | Start new run |

---

## File Structure

```
devflow/
├── transcript.go        # Transcript type
├── transcript_store.go  # FileTranscriptStore
├── transcript_search.go # TranscriptSearcher
├── transcript_view.go   # TranscriptViewer
└── transcript_test.go   # Tests
```

---

## Completion Criteria

- [ ] All tasks implemented
- [ ] Unit test coverage > 80%
- [ ] Integration tests pass
- [ ] Compression works correctly
- [ ] Search functional
- [ ] Viewer formats nicely

---

## References

- ADR-011: Transcript Format
- ADR-012: Transcript Storage
- ADR-013: Transcript Replay
- ADR-014: Transcript Search
