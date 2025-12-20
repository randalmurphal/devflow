package notify

import (
	"context"
	"log/slog"
)

// =============================================================================
// LogNotifier
// =============================================================================

// LogNotifier logs notifications using slog (for testing/debugging).
type LogNotifier struct {
	Logger *slog.Logger
}

// NewLogNotifier creates a notifier that logs to the given logger.
// If logger is nil, uses the default slog logger.
func NewLogNotifier(logger *slog.Logger) *LogNotifier {
	if logger == nil {
		logger = slog.Default()
	}
	return &LogNotifier{Logger: logger}
}

// Notify implements Notifier.
func (n *LogNotifier) Notify(ctx context.Context, event Event) error {
	level := slog.LevelInfo
	switch event.Severity {
	case SeverityWarning:
		level = slog.LevelWarn
	case SeverityError:
		level = slog.LevelError
	}

	n.Logger.Log(ctx, level, event.Message,
		"type", event.Type,
		"run_id", event.RunID,
		"flow_id", event.FlowID,
		"node_id", event.NodeID,
		"metadata", event.Metadata,
	)
	return nil
}
