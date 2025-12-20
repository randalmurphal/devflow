package notify

import (
	"context"
	"time"
)

// =============================================================================
// Notification Types
// =============================================================================

// EventType represents the type of workflow event.
type EventType string

// Event type constants.
const (
	EventRunStarted    EventType = "run_started"
	EventRunCompleted  EventType = "run_completed"
	EventRunFailed     EventType = "run_failed"
	EventNodeStarted   EventType = "node_started"
	EventNodeCompleted EventType = "node_completed"
	EventNodeFailed    EventType = "node_failed"
	EventReviewNeeded  EventType = "review_needed"
	EventPRCreated     EventType = "pr_created"
)

// Severity constants for notifications and findings.
// These are shared between notification events and review findings.
const (
	SeverityCritical = "critical"
	SeverityError    = "error"
	SeverityWarning  = "warning"
	SeverityInfo     = "info"
)

// Event describes a workflow event for notification.
type Event struct {
	Type      EventType      `json:"type"`
	RunID     string         `json:"run_id"`
	FlowID    string         `json:"flow_id"`
	NodeID    string         `json:"node_id,omitempty"`
	Message   string         `json:"message"`
	Severity  string         `json:"severity"` // SeverityInfo, SeverityWarning, SeverityError
	Timestamp time.Time      `json:"timestamp"`
	Metadata  map[string]any `json:"metadata,omitempty"`
}

// =============================================================================
// Notifier Interface
// =============================================================================

// Notifier sends notifications about workflow events.
type Notifier interface {
	// Notify sends a notification. Implementations should be non-blocking
	// and handle errors gracefully (log, don't crash).
	Notify(ctx context.Context, event Event) error
}

// =============================================================================
// Context Injection
// =============================================================================

type serviceContextKey string

const notifierServiceKey serviceContextKey = "devflow.notifier"

// WithNotifier adds a Notifier to the context.
func WithNotifier(ctx context.Context, n Notifier) context.Context {
	return context.WithValue(ctx, notifierServiceKey, n)
}

// NotifierFromContext extracts the Notifier from context.
// Returns nil if no notifier is configured.
func NotifierFromContext(ctx context.Context) Notifier {
	if n, ok := ctx.Value(notifierServiceKey).(Notifier); ok {
		return n
	}
	return nil
}

// MustNotifierFromContext extracts the Notifier or panics.
func MustNotifierFromContext(ctx context.Context) Notifier {
	n := NotifierFromContext(ctx)
	if n == nil {
		panic("devflow: Notifier not found in context")
	}
	return n
}
