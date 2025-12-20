package transcript

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

// FileStore stores transcripts as files
type FileStore struct {
	baseDir string
	mu      sync.RWMutex
	active  map[string]*activeRun
}

type activeRun struct {
	transcript *Transcript
}

// NewFileStore creates a file-based transcript store
func NewFileStore(config StoreConfig) (*FileStore, error) {
	runsDir := filepath.Join(config.BaseDir, "runs")
	if err := os.MkdirAll(runsDir, 0755); err != nil {
		return nil, err
	}

	return &FileStore{
		baseDir: config.BaseDir,
		active:  make(map[string]*activeRun),
	}, nil
}

// StoreConfig holds configuration for transcript storage
type StoreConfig struct {
	BaseDir string
}

// StartRun begins a new transcript
func (s *FileStore) StartRun(runID string, meta RunMetadata) error {
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
		Metadata: Meta{
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
func (s *FileStore) RecordTurn(runID string, turn Turn) error {
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
func (s *FileStore) RecordToolCall(runID string, tc ToolCall) error {
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
func (s *FileStore) AddCost(runID string, cost float64) error {
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
func (s *FileStore) EndRun(runID string, status RunStatus) error {
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
func (s *FileStore) EndRunWithError(runID string, err error) error {
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
func (s *FileStore) Load(runID string) (*Transcript, error) {
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

	return Load(s.baseDir, runID)
}

// LoadMetadata retrieves just the metadata
func (s *FileStore) LoadMetadata(runID string) (*Meta, error) {
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

	var meta Meta
	if err := json.Unmarshal(data, &meta); err != nil {
		return nil, err
	}

	return &meta, nil
}

// List returns metadata for runs matching filter
func (s *FileStore) List(filter ListFilter) ([]Meta, error) {
	runsDir := filepath.Join(s.baseDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var results []Meta

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
func (s *FileStore) Delete(runID string) error {
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
func (s *FileStore) GetActive(runID string) (*Transcript, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active, ok := s.active[runID]
	if !ok {
		return nil, false
	}

	return active.transcript, true
}

// ListActive returns all active run IDs
func (s *FileStore) ListActive() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := make([]string, 0, len(s.active))
	for id := range s.active {
		ids = append(ids, id)
	}
	return ids
}

func (s *FileStore) writeMetadata(runID string, meta *Meta) error {
	path := filepath.Join(s.baseDir, "runs", runID, "metadata.json")
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// BaseDir returns the base directory for the store
func (s *FileStore) BaseDir() string {
	return s.baseDir
}
