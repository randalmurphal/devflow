package devflow

import (
	"compress/gzip"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"
)

// Transcript errors
var (
	ErrRunNotFound      = errors.New("run not found")
	ErrRunAlreadyExists = errors.New("run already exists")
	ErrRunNotStarted    = errors.New("run not started")
	ErrRunAlreadyEnded  = errors.New("run already ended")
)

// RunStatus indicates the status of a run
type RunStatus string

const (
	RunStatusRunning   RunStatus = "running"
	RunStatusCompleted RunStatus = "completed"
	RunStatusFailed    RunStatus = "failed"
	RunStatusCanceled  RunStatus = "canceled"
)

// Transcript represents a complete conversation record
type Transcript struct {
	RunID    string         `json:"runId"`
	Metadata TranscriptMeta `json:"metadata"`
	Turns    []Turn         `json:"turns"`
}

// TranscriptMeta contains run metadata
type TranscriptMeta struct {
	RunID          string         `json:"runId,omitempty"`
	FlowID         string         `json:"flowId"`
	NodeID         string         `json:"nodeId,omitempty"`
	Input          map[string]any `json:"input,omitempty"`
	StartedAt      time.Time      `json:"startedAt"`
	EndedAt        time.Time      `json:"endedAt,omitempty"`
	Status         RunStatus      `json:"status"`
	TotalTokensIn  int            `json:"totalTokensIn"`
	TotalTokensOut int            `json:"totalTokensOut"`
	TotalCost      float64        `json:"totalCost"`
	TurnCount      int            `json:"turnCount"`
	Error          string         `json:"error,omitempty"`
}

// Turn represents a conversation turn
type Turn struct {
	ID         int        `json:"id"`
	Role       string     `json:"role"` // system, user, assistant, tool_result
	Content    string     `json:"content"`
	TokensIn   int        `json:"tokensIn,omitempty"`
	TokensOut  int        `json:"tokensOut,omitempty"`
	Timestamp  time.Time  `json:"timestamp"`
	ToolCalls  []ToolCall `json:"toolCalls,omitempty"`
	DurationMs int64      `json:"durationMs,omitempty"`
}

// ToolCall represents a tool/function call
type ToolCall struct {
	ID     string         `json:"id,omitempty"`
	Name   string         `json:"name"`
	Input  map[string]any `json:"input"`
	Output string         `json:"output,omitempty"`
	Error  string         `json:"error,omitempty"`
}

// RunMetadata is input for starting a new run
type RunMetadata struct {
	FlowID string
	NodeID string
	Input  map[string]any
}

// NewTranscript creates a new transcript
func NewTranscript(runID, flowID string) *Transcript {
	return &Transcript{
		RunID: runID,
		Metadata: TranscriptMeta{
			RunID:     runID,
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
	t.Metadata.TurnCount = len(t.Turns)
	return &t.Turns[len(t.Turns)-1]
}

// AddTurnWithDetails adds a turn with full control over fields
func (t *Transcript) AddTurnWithDetails(turn Turn) *Turn {
	turn.ID = len(t.Turns) + 1
	if turn.Timestamp.IsZero() {
		turn.Timestamp = time.Now()
	}

	// Update token counts
	switch turn.Role {
	case "user", "system":
		t.Metadata.TotalTokensIn += turn.TokensIn
	case "assistant":
		t.Metadata.TotalTokensOut += turn.TokensOut
	}

	t.Turns = append(t.Turns, turn)
	t.Metadata.TurnCount = len(t.Turns)
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

// AddToolCallError adds a failed tool call
func (t *Transcript) AddToolCallError(name string, input map[string]any, err error) {
	if len(t.Turns) == 0 {
		return
	}

	last := &t.Turns[len(t.Turns)-1]
	if last.Role != "assistant" {
		return
	}

	last.ToolCalls = append(last.ToolCalls, ToolCall{
		Name:  name,
		Input: input,
		Error: err.Error(),
	})
}

// SetCost sets the total cost
func (t *Transcript) SetCost(cost float64) {
	t.Metadata.TotalCost = cost
}

// AddCost adds to the total cost
func (t *Transcript) AddCost(cost float64) {
	t.Metadata.TotalCost += cost
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
	if err != nil {
		t.Metadata.Error = err.Error()
	}
}

// Cancel marks the transcript as canceled
func (t *Transcript) Cancel() {
	t.Metadata.Status = RunStatusCanceled
	t.Metadata.EndedAt = time.Now()
}

// Duration returns the run duration
func (t *Transcript) Duration() time.Duration {
	if t.Metadata.EndedAt.IsZero() {
		return time.Since(t.Metadata.StartedAt)
	}
	return t.Metadata.EndedAt.Sub(t.Metadata.StartedAt)
}

// IsActive returns true if the run is still in progress
func (t *Transcript) IsActive() bool {
	return t.Metadata.Status == RunStatusRunning
}

// LastTurn returns the last turn or nil
func (t *Transcript) LastTurn() *Turn {
	if len(t.Turns) == 0 {
		return nil
	}
	return &t.Turns[len(t.Turns)-1]
}

// TurnsByRole returns all turns with the given role
func (t *Transcript) TurnsByRole(role string) []Turn {
	var result []Turn
	for _, turn := range t.Turns {
		if turn.Role == role {
			result = append(result, turn)
		}
	}
	return result
}

// compressionThreshold is the size above which transcripts are compressed
const compressionThreshold = 100 * 1024 // 100KB

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
	if len(data) > compressionThreshold {
		return t.saveCompressed(runDir, data)
	}

	// Remove compressed version if it exists
	os.Remove(filepath.Join(runDir, "transcript.json.gz"))

	return os.WriteFile(filepath.Join(runDir, "transcript.json"), data, 0644)
}

func (t *Transcript) saveCompressed(runDir string, data []byte) error {
	// Remove uncompressed version if it exists
	os.Remove(filepath.Join(runDir, "transcript.json"))

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
			if os.IsNotExist(err) {
				return nil, ErrRunNotFound
			}
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

// TranscriptManager is the interface for transcript operations
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

// ListFilter filters transcript listing
type ListFilter struct {
	FlowID string
	Status RunStatus
	After  time.Time
	Before time.Time
	Limit  int
}
