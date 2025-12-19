# ADR-011: Transcript Format

## Status

Accepted

## Context

devflow needs to record AI conversations for:

1. Debugging failed runs
2. Auditing automated changes
3. Replaying for analysis
4. Cost tracking

We need to define the transcript format.

## Decision

### 1. JSON Format

Transcripts are stored as JSON files:

```json
{
  "runId": "2025-01-15-ticket-to-pr-TK421",
  "metadata": {
    "flowId": "ticket-to-pr",
    "nodeId": "generate-spec",
    "input": {"ticketId": "TK-421"},
    "startedAt": "2025-01-15T10:30:00Z",
    "endedAt": "2025-01-15T10:45:32Z",
    "status": "completed",
    "totalTokensIn": 5200,
    "totalTokensOut": 8400,
    "totalCost": 0.12
  },
  "turns": [
    {
      "id": 1,
      "role": "system",
      "content": "You are an expert software architect...",
      "timestamp": "2025-01-15T10:30:00Z"
    },
    {
      "id": 2,
      "role": "user",
      "content": "Generate a spec for TK-421...",
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
        {
          "name": "read_file",
          "input": {"path": "api/handler.go"},
          "output": "package api..."
        }
      ]
    }
  ]
}
```

### 2. Type Definitions

```go
// Transcript represents a full conversation record
type Transcript struct {
    RunID    string           `json:"runId"`
    Metadata TranscriptMeta   `json:"metadata"`
    Turns    []Turn           `json:"turns"`
}

// TranscriptMeta contains run metadata
type TranscriptMeta struct {
    FlowID        string         `json:"flowId"`
    NodeID        string         `json:"nodeId,omitempty"`
    Input         map[string]any `json:"input,omitempty"`
    StartedAt     time.Time      `json:"startedAt"`
    EndedAt       time.Time      `json:"endedAt,omitempty"`
    Status        RunStatus      `json:"status"`
    TotalTokensIn  int           `json:"totalTokensIn"`
    TotalTokensOut int           `json:"totalTokensOut"`
    TotalCost     float64        `json:"totalCost"`
    Error         string         `json:"error,omitempty"`
}

// Turn represents a conversation turn
type Turn struct {
    ID        int        `json:"id"`
    Role      string     `json:"role"` // system, user, assistant
    Content   string     `json:"content"`
    TokensIn  int        `json:"tokensIn,omitempty"`
    TokensOut int        `json:"tokensOut,omitempty"`
    Timestamp time.Time  `json:"timestamp"`
    ToolCalls []ToolCall `json:"toolCalls,omitempty"`
    Duration  int64      `json:"durationMs,omitempty"`
}

// ToolCall represents a tool/function call
type ToolCall struct {
    ID     string         `json:"id,omitempty"`
    Name   string         `json:"name"`
    Input  map[string]any `json:"input"`
    Output string         `json:"output,omitempty"`
    Error  string         `json:"error,omitempty"`
}

// RunStatus indicates the run outcome
type RunStatus string

const (
    RunStatusRunning   RunStatus = "running"
    RunStatusCompleted RunStatus = "completed"
    RunStatusFailed    RunStatus = "failed"
    RunStatusCanceled  RunStatus = "canceled"
)
```

### 3. Turn Roles

| Role | Description |
|------|-------------|
| `system` | System prompt |
| `user` | User/developer prompt |
| `assistant` | Claude's response |
| `tool_result` | Tool execution result |

### 4. Compression

Large transcripts are compressed with gzip:

| Size | Storage |
|------|---------|
| < 100KB | `transcript.json` |
| >= 100KB | `transcript.json.gz` |

### 5. Run ID Format

```
{date}-{flow}-{identifier}[-{suffix}]
```

Examples:
- `2025-01-15-ticket-to-pr-TK421`
- `2025-01-15-review-pr-123`
- `2025-01-15-manual-run-001`

## Alternatives Considered

### Alternative 1: JSONL (Line-Delimited)

Store each turn as a separate JSON line.

**Rejected because:**
- Harder to read/debug
- No natural grouping
- Metadata scattered

### Alternative 2: Protocol Buffers

Use protobuf for efficiency.

**Rejected because:**
- Harder to inspect manually
- Overkill for file storage
- JSON is human-readable

### Alternative 3: SQLite Database

Store transcripts in SQLite.

**Deferred:**
- Good idea for search/query
- Can add as secondary index
- JSON files remain primary

### Alternative 4: Markdown Format

Store transcripts as markdown.

**Rejected because:**
- Harder to parse programmatically
- Tool calls awkward to represent
- JSON better for structured data

## Consequences

### Positive

- **Human-readable**: JSON is easy to inspect
- **Structured**: Easy to parse and analyze
- **Complete**: Captures all conversation data
- **Portable**: Files can be shared/archived

### Negative

- **Size**: JSON is verbose
- **Single file**: Can't append incrementally
- **Search**: Need to load file to search

### Mitigations

1. **Compression**: Gzip for large transcripts
2. **Streaming write**: Write incrementally, finalize at end
3. **Index file**: Separate summary/index for search

## Code Example

```go
package devflow

import (
    "compress/gzip"
    "encoding/json"
    "os"
    "path/filepath"
    "time"
)

// Transcript represents a complete conversation record
type Transcript struct {
    RunID    string          `json:"runId"`
    Metadata TranscriptMeta  `json:"metadata"`
    Turns    []Turn          `json:"turns"`
}

// NewTranscript creates a new transcript
func NewTranscript(runID string, flowID string) *Transcript {
    return &Transcript{
        RunID: runID,
        Metadata: TranscriptMeta{
            FlowID:    flowID,
            StartedAt: time.Now(),
            Status:    RunStatusRunning,
        },
        Turns: make([]Turn, 0),
    }
}

// AddTurn adds a turn to the transcript
func (t *Transcript) AddTurn(role, content string, tokens int) *Turn {
    turn := Turn{
        ID:        len(t.Turns) + 1,
        Role:      role,
        Content:   content,
        Timestamp: time.Now(),
    }

    switch role {
    case "user", "system":
        turn.TokensIn = tokens
        t.Metadata.TotalTokensIn += tokens
    case "assistant":
        turn.TokensOut = tokens
        t.Metadata.TotalTokensOut += tokens
    }

    t.Turns = append(t.Turns, turn)
    return &t.Turns[len(t.Turns)-1]
}

// AddToolCall adds a tool call to the last assistant turn
func (t *Transcript) AddToolCall(name string, input map[string]any, output string) {
    if len(t.Turns) == 0 {
        return
    }

    last := &t.Turns[len(t.Turns)-1]
    if last.Role != "assistant" {
        return
    }

    last.ToolCalls = append(last.ToolCalls, ToolCall{
        Name:   name,
        Input:  input,
        Output: output,
    })
}

// Complete marks the transcript as completed
func (t *Transcript) Complete() {
    t.Metadata.Status = RunStatusCompleted
    t.Metadata.EndedAt = time.Now()
}

// Fail marks the transcript as failed
func (t *Transcript) Fail(err error) {
    t.Metadata.Status = RunStatusFailed
    t.Metadata.EndedAt = time.Now()
    t.Metadata.Error = err.Error()
}

// Save writes the transcript to disk
func (t *Transcript) Save(baseDir string) error {
    runDir := filepath.Join(baseDir, "runs", t.RunID)
    if err := os.MkdirAll(runDir, 0755); err != nil {
        return err
    }

    data, err := json.MarshalIndent(t, "", "  ")
    if err != nil {
        return err
    }

    // Compress if large
    if len(data) > 100*1024 {
        return t.saveCompressed(runDir, data)
    }

    return os.WriteFile(filepath.Join(runDir, "transcript.json"), data, 0644)
}

func (t *Transcript) saveCompressed(runDir string, data []byte) error {
    f, err := os.Create(filepath.Join(runDir, "transcript.json.gz"))
    if err != nil {
        return err
    }
    defer f.Close()

    gz := gzip.NewWriter(f)
    defer gz.Close()

    _, err = gz.Write(data)
    return err
}

// LoadTranscript loads a transcript from disk
func LoadTranscript(baseDir, runID string) (*Transcript, error) {
    runDir := filepath.Join(baseDir, "runs", runID)

    // Try compressed first
    data, err := loadCompressed(filepath.Join(runDir, "transcript.json.gz"))
    if err != nil {
        // Try uncompressed
        data, err = os.ReadFile(filepath.Join(runDir, "transcript.json"))
        if err != nil {
            return nil, err
        }
    }

    var t Transcript
    if err := json.Unmarshal(data, &t); err != nil {
        return nil, err
    }

    return &t, nil
}

func loadCompressed(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    gz, err := gzip.NewReader(f)
    if err != nil {
        return nil, err
    }
    defer gz.Close()

    return io.ReadAll(gz)
}
```

### Usage

```go
// Create transcript
transcript := devflow.NewTranscript("2025-01-15-ticket-to-pr-TK421", "ticket-to-pr")

// Record system prompt
transcript.AddTurn("system", systemPrompt, 500)

// Record user turn
transcript.AddTurn("user", userPrompt, 1200)

// Run Claude
result, err := claude.Run(ctx, userPrompt)

// Record assistant response
turn := transcript.AddTurn("assistant", result.Output, result.TokensOut)

// Record any tool calls
for _, tc := range result.ToolCalls {
    transcript.AddToolCall(tc.Name, tc.Input, tc.Output)
}

// Complete and save
transcript.Complete()
if err := transcript.Save(".devflow"); err != nil {
    log.Error("save transcript", "error", err)
}
```

## References

- ADR-012: Transcript Storage
- ADR-013: Transcript Replay
- [OpenAI Chat Format](https://platform.openai.com/docs/guides/chat)
