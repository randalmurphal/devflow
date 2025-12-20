package transcript

import "time"

// Manager is the interface for transcript operations
type Manager interface {
	// Lifecycle
	StartRun(runID string, metadata RunMetadata) error
	RecordTurn(runID string, turn Turn) error
	EndRun(runID string, status RunStatus) error

	// Retrieval
	Load(runID string) (*Transcript, error)
	LoadMetadata(runID string) (*Meta, error)
	List(filter ListFilter) ([]Meta, error)

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
