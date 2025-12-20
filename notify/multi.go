package notify

import (
	"context"
	"log/slog"
)

// =============================================================================
// MultiNotifier
// =============================================================================

// MultiNotifier sends notifications to multiple notifiers.
type MultiNotifier struct {
	Notifiers []Notifier
	Logger    *slog.Logger
}

// NewMultiNotifier creates a notifier that fans out to multiple notifiers.
// Errors from individual notifiers are logged but don't stop other notifications.
func NewMultiNotifier(notifiers ...Notifier) *MultiNotifier {
	return &MultiNotifier{
		Notifiers: notifiers,
		Logger:    slog.Default(),
	}
}

// Notify implements Notifier.
func (n *MultiNotifier) Notify(ctx context.Context, event Event) error {
	var lastErr error
	for _, notifier := range n.Notifiers {
		if err := notifier.Notify(ctx, event); err != nil {
			lastErr = err
			if n.Logger != nil {
				n.Logger.Warn("notifier failed",
					"error", err,
					"event_type", event.Type,
				)
			}
		}
	}
	return lastErr // Return last error, if any
}

// =============================================================================
// NopNotifier
// =============================================================================

// NopNotifier is a no-op notifier that discards all notifications.
// Useful for testing or when notifications are disabled.
type NopNotifier struct{}

// Notify implements Notifier.
func (NopNotifier) Notify(ctx context.Context, event Event) error {
	return nil
}
