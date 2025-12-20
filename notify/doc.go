// Package notify provides notification services for workflow events.
//
// Core types:
//   - Notifier: Interface for sending notifications
//   - Event: Notification event with type, message, and metadata
//   - EventType: Type of event (started, completed, failed, etc.)
//
// Implementations:
//   - SlackNotifier: Sends notifications to Slack webhooks
//   - WebhookNotifier: Sends notifications to generic webhooks
//   - LogNotifier: Logs notifications (for testing/debugging)
//   - MultiNotifier: Combines multiple notifiers
//   - NopNotifier: No-op notifier (for testing)
//
// Example usage:
//
//	notifier := notify.NewSlack(webhookURL,
//	    notify.WithChannel("#dev-alerts"),
//	    notify.WithUsername("devflow-bot"),
//	)
//	err := notifier.Notify(ctx, notify.Event{
//	    Type:    notify.EventCompleted,
//	    Message: "Workflow completed successfully",
//	})
package notify
