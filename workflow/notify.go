package workflow

import (
	"time"

	"github.com/randalmurphal/devflow/notify"
	"github.com/randalmurphal/flowgraph/pkg/flowgraph"
)

// NotifyNode sends a notification based on current state.
//
// This node is typically placed at the end of a workflow to notify
// interested parties of completion or failure. If no notifier is
// configured in the context, this is a no-op.
//
// Updates: None (only sends notification)
func NotifyNode(ctx flowgraph.Context, state State) (State, error) {
	notifier := notify.NotifierFromContext(ctx)
	if notifier == nil {
		return state, nil // No-op if no notifier
	}

	event := notify.Event{
		Type:      determineEventType(state),
		RunID:     state.RunID,
		FlowID:    state.FlowID,
		Timestamp: time.Now(),
		Metadata:  buildMetadata(state),
	}

	// Set severity based on state
	if state.Error != "" {
		event.Severity = notify.SeverityError
		event.Message = state.Error
	} else {
		event.Severity = notify.SeverityInfo
		event.Message = "Workflow completed successfully"
	}

	// Notify but don't fail the workflow on notification errors
	_ = notifier.Notify(ctx, event)

	return state, nil
}

// determineEventType determines the event type from state
func determineEventType(state State) notify.EventType {
	if state.Error != "" {
		return notify.EventRunFailed
	}
	return notify.EventRunCompleted
}

// buildMetadata builds notification metadata from state
func buildMetadata(state State) map[string]any {
	meta := make(map[string]any)

	if state.TicketID != "" {
		meta["ticketId"] = state.TicketID
	}
	if state.Branch != "" {
		meta["branch"] = state.Branch
	}
	if state.PR != nil {
		meta["prUrl"] = state.PR.URL
	}
	if state.Review != nil {
		meta["reviewApproved"] = state.Review.Approved
	}

	meta["tokensIn"] = state.TotalTokensIn
	meta["tokensOut"] = state.TotalTokensOut
	meta["cost"] = state.TotalCost

	return meta
}
