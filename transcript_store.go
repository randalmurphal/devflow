package devflow

import (
	"encoding/json"
	"fmt"
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

// NewTranscriptManager creates a TranscriptManager using file storage
func NewTranscriptManager(config TranscriptConfig) (TranscriptManager, error) {
	return NewFileTranscriptStore(config.BaseDir)
}

// TranscriptConfig holds configuration for transcript management
type TranscriptConfig struct {
	BaseDir string
}

// StartRun begins a new transcript
func (s *FileTranscriptStore) StartRun(runID string, meta RunMetadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.active[runID]; exists {
		return ErrRunAlreadyExists
	}

	// Check if run already exists on disk
	runDir := filepath.Join(s.baseDir, "runs", runID)
	if _, err := os.Stat(runDir); err == nil {
		return ErrRunAlreadyExists
	}

	if err := os.MkdirAll(runDir, 0755); err != nil {
		return err
	}

	transcript := &Transcript{
		RunID: runID,
		Metadata: TranscriptMeta{
			RunID:     runID,
			FlowID:    meta.FlowID,
			NodeID:    meta.NodeID,
			Input:     meta.Input,
			StartedAt: time.Now(),
			Status:    RunStatusRunning,
		},
		Turns: make([]Turn, 0),
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
		return ErrRunNotStarted
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

	active.transcript.Metadata.TurnCount = len(active.transcript.Turns)

	return nil
}

// RecordToolCall adds a tool call to the last turn of an active transcript
func (s *FileTranscriptStore) RecordToolCall(runID string, tc ToolCall) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	active, ok := s.active[runID]
	if !ok {
		return ErrRunNotStarted
	}

	if len(active.transcript.Turns) == 0 {
		return fmt.Errorf("no turns to add tool call to")
	}

	last := &active.transcript.Turns[len(active.transcript.Turns)-1]
	last.ToolCalls = append(last.ToolCalls, tc)

	return nil
}

// AddCost adds cost to an active transcript
func (s *FileTranscriptStore) AddCost(runID string, cost float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	active, ok := s.active[runID]
	if !ok {
		return ErrRunNotStarted
	}

	active.transcript.Metadata.TotalCost += cost
	return nil
}

// EndRun completes a transcript
func (s *FileTranscriptStore) EndRun(runID string, status RunStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	active, ok := s.active[runID]
	if !ok {
		return ErrRunNotStarted
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

	delete(s.active, runID)
	return nil
}

// EndRunWithError completes a transcript with an error
func (s *FileTranscriptStore) EndRunWithError(runID string, err error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	active, ok := s.active[runID]
	if !ok {
		return ErrRunNotStarted
	}

	active.transcript.Metadata.Status = RunStatusFailed
	active.transcript.Metadata.EndedAt = time.Now()
	if err != nil {
		active.transcript.Metadata.Error = err.Error()
	}

	// Save full transcript
	if saveErr := active.transcript.Save(s.baseDir); saveErr != nil {
		return saveErr
	}

	// Update metadata
	if writeErr := s.writeMetadata(runID, &active.transcript.Metadata); writeErr != nil {
		return writeErr
	}

	delete(s.active, runID)
	return nil
}

// Load retrieves a complete transcript
func (s *FileTranscriptStore) Load(runID string) (*Transcript, error) {
	// Check if it's an active run
	s.mu.RLock()
	if active, ok := s.active[runID]; ok {
		s.mu.RUnlock()
		// Return a copy to prevent concurrent modification
		data, err := json.Marshal(active.transcript)
		if err != nil {
			return nil, err
		}
		var t Transcript
		if err := json.Unmarshal(data, &t); err != nil {
			return nil, err
		}
		return &t, nil
	}
	s.mu.RUnlock()

	return LoadTranscript(s.baseDir, runID)
}

// LoadMetadata retrieves just the metadata
func (s *FileTranscriptStore) LoadMetadata(runID string) (*TranscriptMeta, error) {
	// Check if it's an active run
	s.mu.RLock()
	if active, ok := s.active[runID]; ok {
		s.mu.RUnlock()
		meta := active.transcript.Metadata
		return &meta, nil
	}
	s.mu.RUnlock()

	path := filepath.Join(s.baseDir, "runs", runID, "metadata.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrRunNotFound
		}
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
		if os.IsNotExist(err) {
			return nil, nil
		}
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
		if !filter.Before.IsZero() && meta.StartedAt.After(filter.Before) {
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
	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from active if present
	delete(s.active, runID)

	runDir := filepath.Join(s.baseDir, "runs", runID)
	if err := os.RemoveAll(runDir); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

// GetActive returns an active transcript (for monitoring)
func (s *FileTranscriptStore) GetActive(runID string) (*Transcript, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active, ok := s.active[runID]
	if !ok {
		return nil, false
	}

	return active.transcript, true
}

// ListActive returns all active run IDs
func (s *FileTranscriptStore) ListActive() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.active))
	for id := range s.active {
		ids = append(ids, id)
	}
	return ids
}

func (s *FileTranscriptStore) writeMetadata(runID string, meta *TranscriptMeta) error {
	path := filepath.Join(s.baseDir, "runs", runID, "metadata.json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// BaseDir returns the base directory for the store
func (s *FileTranscriptStore) BaseDir() string {
	return s.baseDir
}
